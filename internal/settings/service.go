package settings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SettingsDTO is the renderer-safe settings snapshot. It reports credential
// presence and native capability state but never returns credential values.
type SettingsDTO struct {
	Platform               string                  `json:"platform"`
	RememberedModels       modelsettings.Catalog   `json:"rememberedModels"`
	ModelProfiles          modelprofile.Catalog    `json:"modelProfiles"`
	SavedConnections       savedconnection.Catalog `json:"savedConnections"`
	RealtimeLanguages      []speechlanguage.Option `json:"realtimeLanguages"`
	TranscriptionLanguages []speechlanguage.Option `json:"transcriptionLanguages"`
	CompatibilityProfiles  compatibility.Catalog   `json:"compatibilityProfiles"`
	config.Settings
	Configuration                      ConfigurationStatus   `json:"configuration"`
	CredentialConfigured               bool                  `json:"credentialConfigured"`
	PostProcessingCredentialConfigured bool                  `json:"postProcessingCredentialConfigured"`
	TextToSpeechCredentialConfigured   bool                  `json:"textToSpeechCredentialConfigured"`
	HoldAvailable                      bool                  `json:"holdAvailable"`
	HoldAvailabilityReason             string                `json:"holdAvailabilityReason"`
	MicaActive                         bool                  `json:"micaActive"`
	AppearanceModeActive               config.AppearanceMode `json:"appearanceModeActive"`
}

// ConfigurationStatus describes whether the durable settings database is
// usable. Ordinary mutations are blocked while RecoveryRequired is true.
type ConfigurationStatus struct {
	RecoveryRequired bool   `json:"recoveryRequired"`
	ErrorKind        string `json:"errorKind,omitempty"`
	Message          string `json:"message,omitempty"`
}

// SaveSettingsRequest groups the persisted settings and transient credential
// changes into one binding argument. Credential drafts are never returned.
type SaveSettingsRequest struct {
	ModelEdits                    []modelsettings.Edit               `json:"modelEdits,omitempty"`
	ForgetModel                   *modelsettings.Key                 `json:"forgetModel,omitempty"`
	ConnectionCredentialDraft     string                             `json:"connectionCredentialDraft,omitempty"`
	ClearConnectionCredential     bool                               `json:"clearConnectionCredential"`
	ExpectedConnections           map[savedconnection.Purpose]string `json:"expectedConnections,omitempty"`
	ConnectionChange              *savedconnection.Change            `json:"connectionChange,omitempty"`
	Settings                      config.Settings                    `json:"settings"`
	STTCredentialDraft            string                             `json:"sttCredentialDraft,omitempty"`
	ClearSTTCredential            bool                               `json:"clearSTTCredential"`
	PostProcessingCredentialDraft string                             `json:"postProcessingCredentialDraft,omitempty"`
	ClearPostProcessingCredential bool                               `json:"clearPostProcessingCredential"`
	TextToSpeechCredentialDraft   string                             `json:"textToSpeechCredentialDraft,omitempty"`
	ClearTextToSpeechCredential   bool                               `json:"clearTextToSpeechCredential"`
}

type HoldInfo func() (bool, string)
type Startup interface {
	Set(bool) error
}
type ConfigStore interface{ Save(config.Settings) error }
type ConfigLoader interface {
	ConfigStore
	Load() (config.Settings, error)
}

type Option func(*Service)

// WithConfigurationLoad connects the startup load result to the bound
// settings service. It is optional so isolated service tests and consumers
// that only need persistence do not acquire a recovery dependency.
func WithConfigurationLoad(loader ConfigLoader, failure *config.LoadFailure) Option {
	return func(service *Service) {
		service.loader = loader
		if failure != nil {
			service.configuration.RecoveryRequired = true
			service.configuration.ErrorKind = failure.Kind
			service.configuration.Message = failure.Message
		}
	}
}

func WithVoiceCredential(store credential.Store) Option {
	return func(service *Service) { service.voiceKeys = store }
}

func WithTextToSpeechCredential(store credential.Store) Option {
	return func(service *Service) { service.ttsKeys = store }
}

// WithUpdateChecks applies the persisted update-check preference after the
// settings transaction commits. Update checks are runtime policy rather than
// part of the native settings rollback transaction.
func WithUpdateChecks(apply func(bool)) Option {
	return func(service *Service) { service.updateChecksChanged = apply }
}

type Service struct {
	managedResolve         func(managedruntime.Instance, compatibility.Role) (managedruntime.ResolvedEndpoint, error)
	managedChanged         func([]managedruntime.Instance)
	managedReserve         func([]managedruntime.Instance) (*managedruntime.InventoryReservation, error)
	mu                     sync.RWMutex
	publicationMu          sync.Mutex // Serializes commits through runtime publication; callbacks may read settings.
	saveMu                 sync.Mutex
	cfg                    config.Settings
	store                  ConfigStore
	loader                 ConfigLoader
	configuration          ConfigurationStatus
	keys                   credential.Store
	processKeys            credential.Store
	ttsKeys                credential.Store
	voiceKeys              credential.Store
	startup                Startup
	hold                   HoldInfo
	holdRetry              func() error
	shortcutChanged        func(config.Settings) (func() error, error)
	overlaySettingsChanged func(config.Settings)
	historyEnabledChanged  func(bool)
	fileSettingsChanged    func(config.Settings)
	updateChecksChanged    func(bool)
	settingsChanged        func(SettingsDTO)
	logger                 *slog.Logger
	micaActive             bool
	appearanceModeActive   config.AppearanceMode
	closed                 atomic.Bool
}

// shortcutChanged returns the rollback for its pre-change native snapshot, even
// on failure if its own partial changes require recovery. Only this settings
// transaction invokes it; saved preferences are not a native binding snapshot.
func NewService(st ConfigStore, cfg config.Settings, k credential.Store, processKeys credential.Store, start Startup, hold HoldInfo, shortcutChanged func(config.Settings) (func() error, error), overlaySettingsChanged func(config.Settings), historyEnabledChanged func(bool), fileSettingsChanged func(config.Settings), settingsChanged func(SettingsDTO), logger *slog.Logger, options ...Option) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	service := &Service{
		store: st, cfg: cfg, keys: k, processKeys: processKeys,
		startup: start, hold: hold, shortcutChanged: shortcutChanged,
		overlaySettingsChanged: overlaySettingsChanged,
		historyEnabledChanged:  historyEnabledChanged,
		fileSettingsChanged:    fileSettingsChanged,
		settingsChanged:        settingsChanged,
		logger:                 logger.With("component", "settings"),
		micaActive:             cfg.UseMica && runtime.GOOS == "windows",
		appearanceModeActive:   config.EffectiveAppearanceMode(cfg.UseMica && runtime.GOOS == "windows", cfg.AppearanceMode),
	}
	for _, option := range options {
		option(service)
	}
	return service
}

const MaxAPIKeyBytes = 2048

type rollbackStep struct {
	name string
	run  func() error
}

type rollbackFailure struct {
	name  string
	cause error
}

func (e rollbackFailure) Error() string { return e.name + " rollback failed" }
func (e rollbackFailure) Unwrap() error { return e.cause }

func rollback(primary error, steps ...rollbackStep) error {
	errs := []error{primary}
	for _, step := range steps {
		if step.run == nil {
			continue
		}
		if err := step.run(); err != nil {
			errs = append(errs, rollbackFailure{name: step.name, cause: err})
		}
	}
	return errors.Join(errs...)
}

// Settings and profile bindings. This section owns the single transaction for
// durable settings, native shortcuts, startup registration, and credentials.

// WithHoldRetry injects a bounded native rearm operation; never runs on mount/save.
func WithHoldRetry(retry func() error) Option {
	return func(s *Service) { s.holdRetry = retry }
}

// RetryHoldShortcut retries the saved assignment without editing or persisting it.
func (s *Service) RetryHoldShortcut() (SettingsDTO, error) {
	if s.closed.Load() {
		return SettingsDTO{}, errors.New("application is shutting down")
	}
	if !s.publicationMu.TryLock() {
		return SettingsDTO{}, errors.New("settings are busy; try again")
	}
	defer s.publicationMu.Unlock()
	if !s.saveMu.TryLock() {
		return SettingsDTO{}, errors.New("settings are busy; try again")
	}
	if s.holdRetry == nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("hold-to-talk retry is unavailable")
	}
	err := s.holdRetry()
	result := s.settingsSnapshotLocked()
	s.saveMu.Unlock()
	if s.settingsChanged != nil {
		s.settingsChanged(result)
	}
	return result, err
}

// GetSettings returns a renderer-safe snapshot of the active runtime profile.
func (s *Service) GetSettings() SettingsDTO {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	return s.settingsSnapshotLocked()
}

func (s *Service) settingsSnapshotLocked() SettingsDTO {
	v := s.current()
	ok, reason := s.hold()
	processingCredentialConfigured := s.processKeys != nil && s.processKeys.Configured()
	ttsCredentialConfigured := s.ttsKeys != nil && s.ttsKeys.Configured()
	catalog := savedconnection.Catalog{Entries: []savedconnection.Connection{}, Selected: map[savedconnection.Purpose]string{}}
	if store, ok := s.store.(interface {
		ConnectionCatalog() savedconnection.Catalog
	}); ok {
		catalog = store.ConnectionCatalog()
	}
	models := modelsettings.Catalog{Entries: []modelsettings.Entry{}, Defaults: modelsettings.Defaults()}
	if store, ok := s.store.(interface{ RememberedModels() modelsettings.Catalog }); ok {
		models = store.RememberedModels()
	}
	return SettingsDTO{
		Platform:                           runtime.GOOS,
		RememberedModels:                   models,
		SavedConnections:                   catalog,
		CompatibilityProfiles:              compatibility.Profiles(),
		RealtimeLanguages:                  modelprofile.NemotronLanguages(),
		ModelProfiles:                      modelCatalog(v),
		TranscriptionLanguages:             speechlanguage.Options(),
		Settings:                           v,
		Configuration:                      s.configuration,
		CredentialConfigured:               s.keys.Configured(),
		PostProcessingCredentialConfigured: processingCredentialConfigured,
		TextToSpeechCredentialConfigured:   ttsCredentialConfigured,
		HoldAvailable:                      ok,
		HoldAvailabilityReason:             reason,
		MicaActive:                         s.micaActive,
		AppearanceModeActive:               s.appearanceModeActive,
	}
}

// GetPostProcessingProfiles returns backend-owned request behavior metadata so
// the renderer can present the exact built-in instruction and editable custom
// path without duplicating model-specific protocol text.
func (s *Service) GetPostProcessingProfiles() []postprocess.ProfileDescriptor {
	return postprocess.Profiles()
}

// PreviewVocabulary evaluates a bounded renderer draft without saving or inference.
func (s *Service) PreviewVocabulary(request config.VocabularyPreviewRequest) config.VocabularyPreview {
	return config.InspectVocabulary(request)
}

// SaveSettings atomically applies one complete settings and credential change
// request, rolling back native changes if persistence fails.
func (s *Service) SaveSettings(request SaveSettingsRequest) (result SettingsDTO, err error) {
	s.publicationMu.Lock()
	defer s.publicationMu.Unlock()
	started := time.Now()
	s.log().Info("settings save started",
		"stt_credential_action", credentialLogAction(request.STTCredentialDraft, request.ClearSTTCredential),
		"postprocess_credential_action", credentialLogAction(request.PostProcessingCredentialDraft, request.ClearPostProcessingCredential),
		"tts_credential_action", credentialLogAction(request.TextToSpeechCredentialDraft, request.ClearTextToSpeechCredential),
	)
	defer func() {
		if err != nil {
			s.log().Warn("settings save failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
			return
		}
		s.log().Info("settings save completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "applied")
	}()
	v := request.Settings
	newAPIKey := request.STTCredentialDraft
	clearKey := request.ClearSTTCredential
	newProcessingAPIKey := request.PostProcessingCredentialDraft
	clearProcessingKey := request.ClearPostProcessingCredential
	newTTSAPIKey := request.TextToSpeechCredentialDraft
	clearTTSKey := request.ClearTextToSpeechCredential
	var old config.Settings
	result, err = func() (SettingsDTO, error) {
		s.saveMu.Lock()
		defer s.saveMu.Unlock()
		if s.closed.Load() {
			return SettingsDTO{}, errors.New("application is shutting down")
		}
		if s.configuration.RecoveryRequired {
			return SettingsDTO{}, errors.New("saved settings must be recovered before changes can be applied")
		}
		if len(newAPIKey) > MaxAPIKeyBytes {
			return SettingsDTO{}, fmt.Errorf("API key must be at most %d bytes", MaxAPIKeyBytes)
		}
		if clearKey && strings.TrimSpace(newAPIKey) != "" {
			return SettingsDTO{}, errors.New("cannot set and clear the API key together")
		}
		if len(newProcessingAPIKey) > MaxAPIKeyBytes {
			return SettingsDTO{}, fmt.Errorf("post-processing API key must be at most %d bytes", MaxAPIKeyBytes)
		}
		if clearProcessingKey && strings.TrimSpace(newProcessingAPIKey) != "" {
			return SettingsDTO{}, errors.New("cannot set and clear the post-processing API key together")
		}
		if len(newTTSAPIKey) > MaxAPIKeyBytes {
			return SettingsDTO{}, fmt.Errorf("speech playback API key must be at most %d bytes", MaxAPIKeyBytes)
		}
		if clearTTSKey && strings.TrimSpace(newTTSAPIKey) != "" {
			return SettingsDTO{}, errors.New("cannot set and clear the speech playback API key together")
		}
		// Preserve authoritative inventory before connection projection and validation.
		v.ManagedRuntimes = s.current().ManagedRuntimes
		if request.ExpectedConnections != nil {
			store, ok := s.store.(interface {
				ConnectionCatalog() savedconnection.Catalog
			})
			if !ok {
				return SettingsDTO{}, errors.New("saved connections are unavailable")
			}
			selected := store.ConnectionCatalog().Selected
			if len(request.ExpectedConnections) != len(selected) {
				return SettingsDTO{}, errors.New("connections changed; reload settings before saving")
			}
			for p, id := range selected {
				if request.ExpectedConnections[p] != id {
					return SettingsDTO{}, errors.New("connections changed; reload settings before saving")
				}
			}
		}

		if request.ForgetModel != nil && request.ConnectionChange != nil {
			return SettingsDTO{}, errors.New("model and connection changes must be separate")
		}
		if change := request.ConnectionChange; change != nil {
			managedAlias := change.Action == savedconnection.Create && change.Details != nil && change.Details.ManagedInstanceID != ""
			if change.Action == savedconnection.Update || change.Action == savedconnection.Duplicate {
				if catalog, ok := s.store.(interface {
					ConnectionCatalog() savedconnection.Catalog
				}); ok {
					for _, c := range catalog.ConnectionCatalog().Entries {
						if c.ID != change.ID {
							continue
						}
						if change.Action == savedconnection.Duplicate {
							managedAlias = c.Details.ManagedInstanceID != ""
						} else if change.Details != nil {
							managedAlias = change.Details.ManagedInstanceID != "" && change.Details.ManagedInstanceID != c.Details.ManagedInstanceID
						}
						break
					}
				}
			}
			if managedAlias {
				return SettingsDTO{}, errors.New("managed runtimes provide built-in connections automatically; select the runtime connection instead")
			}
			store, ok := s.store.(interface {
				BeginConnectionChange(savedconnection.Change, config.Settings) (config.Settings, error)
				StageConnectionCredential(string, bool) error
				DiscardCredentialChanges()
			})
			if !ok {
				return SettingsDTO{}, errors.New("saved connections are unavailable")
			}
			if newAPIKey != "" || newProcessingAPIKey != "" || newTTSAPIKey != "" || clearKey || clearProcessingKey || clearTTSKey {
				return SettingsDTO{}, errors.New("connection edits use their own credential draft")
			}
			if len(request.ConnectionCredentialDraft) > MaxAPIKeyBytes || (request.ClearConnectionCredential && strings.TrimSpace(request.ConnectionCredentialDraft) != "") {
				return SettingsDTO{}, errors.New("invalid connection credential draft")
			}
			var prepareErr error
			v, prepareErr = store.BeginConnectionChange(*change, s.current())
			if prepareErr != nil {
				return SettingsDTO{}, prepareErr
			}
			defer store.DiscardCredentialChanges()
			if change.Action == savedconnection.Create || change.Action == savedconnection.Update {
				if err := store.StageConnectionCredential(request.ConnectionCredentialDraft, request.ClearConnectionCredential); err != nil {
					return SettingsDTO{}, err
				}
			} else if request.ConnectionCredentialDraft != "" || request.ClearConnectionCredential {
				return SettingsDTO{}, errors.New("credentials require an explicit connection edit")
			}
		} else {
			if request.ConnectionCredentialDraft != "" || request.ClearConnectionCredential {
				return SettingsDTO{}, errors.New("credentials require an explicit connection edit")
			}
			if store, ok := s.store.(interface {
				ApplySelectedConnections(config.Settings) config.Settings
			}); ok {
				v = store.ApplySelectedConnections(v)
			}
			if staged, ok := s.store.(interface {
				BeginCredentialChanges() error
				DiscardCredentialChanges()
			}); ok {
				if err := staged.BeginCredentialChanges(); err != nil {
					return SettingsDTO{}, err
				}
				defer staged.DiscardCredentialChanges()
			}
		}
		if key := request.ForgetModel; key != nil {
			store, ok := s.store.(interface {
				BeginForgetModel(modelsettings.Key, config.Settings) (config.Settings, error)
			})
			if !ok {
				return SettingsDTO{}, errors.New("remembered models are unavailable")
			}
			var forgetErr error
			v, forgetErr = store.BeginForgetModel(*key, s.current())
			if forgetErr != nil {
				return SettingsDTO{}, forgetErr
			}
		}
		if len(request.ModelEdits) > 0 {
			if request.ConnectionChange != nil || request.ForgetModel != nil {
				return SettingsDTO{}, errors.New("model edits must be saved separately from connection changes")
			}
			store, ok := s.store.(interface {
				BeginModelEdits([]modelsettings.Edit) error
			})
			if !ok {
				return SettingsDTO{}, errors.New("remembered models are unavailable")
			}
			if err := store.BeginModelEdits(request.ModelEdits); err != nil {
				return SettingsDTO{}, err
			}
		}
		// Runtime inventory has already been restored before task projection.
		v.Model = strings.TrimSpace(v.Model)
		v.PostProcessing.Model = strings.TrimSpace(v.PostProcessing.Model)
		v.TextToSpeech.Model = strings.TrimSpace(v.TextToSpeech.Model)
		v.VoiceTranscription.Model = strings.TrimSpace(v.VoiceTranscription.Model)
		if validateErr := config.Validate(v); validateErr != nil {
			return SettingsDTO{}, validateErr
		}

		old = s.current()
		shortcutsChanged := old.ToggleShortcut != v.ToggleShortcut || old.ShowShortcut != v.ShowShortcut || old.HoldShortcut != v.HoldShortcut
		rollbackShortcuts := rollbackStep{name: "shortcuts"}
		if shortcutsChanged && s.shortcutChanged != nil {
			var shortcutErr error
			rollbackShortcuts.run, shortcutErr = s.shortcutChanged(v)
			if shortcutErr != nil {
				return SettingsDTO{}, rollback(fmt.Errorf("shortcuts were not changed: %w", shortcutErr), rollbackShortcuts)
			}
		}

		startupChanged := old.StartWithWindows != v.StartWithWindows
		rollbackStartup := rollbackStep{name: "startup", run: func() error {
			if startupChanged {
				return s.startup.Set(old.StartWithWindows)
			}
			return nil
		}}
		if startupChanged {
			if startupErr := s.startup.Set(v.StartWithWindows); startupErr != nil {
				return SettingsDTO{}, rollback(fmt.Errorf("startup setting was not changed: %w", startupErr), rollbackStartup, rollbackShortcuts)
			}
		}

		credentialChanged := clearKey || strings.TrimSpace(newAPIKey) != ""
		oldKey := ""
		oldKeyPresent := false
		rollbackCredential := rollbackStep{name: "STT credential", run: func() error {
			defer func() { oldKey = "" }()
			if !credentialChanged {
				return nil
			}
			if oldKeyPresent {
				return s.keys.Set(oldKey)
			}
			return s.keys.Delete()
		}}
		if credentialChanged {
			var credentialErr error
			oldKey, credentialErr = s.keys.Get()
			if credentialErr == nil {
				oldKeyPresent = true
			} else if !errors.Is(credentialErr, credential.ErrNotFound) {
				return SettingsDTO{}, rollback(errors.New("stored credential could not be read"), rollbackStartup, rollbackShortcuts)
			}
			if clearKey {
				credentialErr = s.keys.Delete()
			} else {
				credentialErr = s.keys.Set(newAPIKey)
			}
			if credentialErr != nil {
				return SettingsDTO{}, rollback(errors.New("credential could not be changed"), rollbackCredential, rollbackStartup, rollbackShortcuts)
			}
		}

		processingCredentialChanged := clearProcessingKey || strings.TrimSpace(newProcessingAPIKey) != ""
		oldProcessingKey := ""
		oldProcessingKeyPresent := false
		rollbackProcessingCredential := rollbackStep{name: "post-processing credential", run: func() error {
			defer func() { oldProcessingKey = "" }()
			if !processingCredentialChanged || s.processKeys == nil {
				return nil
			}
			if oldProcessingKeyPresent {
				return s.processKeys.Set(oldProcessingKey)
			}
			return s.processKeys.Delete()
		}}
		if processingCredentialChanged {
			if s.processKeys == nil {
				return SettingsDTO{}, rollback(errors.New("post-processing credential storage is unavailable"), rollbackCredential, rollbackStartup, rollbackShortcuts)
			}
			var processingCredentialErr error
			oldProcessingKey, processingCredentialErr = s.processKeys.Get()
			if processingCredentialErr == nil {
				oldProcessingKeyPresent = true
			} else if !errors.Is(processingCredentialErr, credential.ErrNotFound) {
				return SettingsDTO{}, rollback(errors.New("stored post-processing credential could not be read"), rollbackCredential, rollbackStartup, rollbackShortcuts)
			}
			if clearProcessingKey {
				processingCredentialErr = s.processKeys.Delete()
			} else {
				processingCredentialErr = s.processKeys.Set(newProcessingAPIKey)
			}
			if processingCredentialErr != nil {
				return SettingsDTO{}, rollback(errors.New("post-processing credential could not be changed"), rollbackProcessingCredential, rollbackCredential, rollbackStartup, rollbackShortcuts)
			}
		}

		ttsCredentialChanged := clearTTSKey || strings.TrimSpace(newTTSAPIKey) != ""
		oldTTSKey := ""
		oldTTSKeyPresent := false
		rollbackTTSCredential := rollbackStep{name: "speech playback credential", run: func() error {
			defer func() { oldTTSKey = "" }()
			if !ttsCredentialChanged || s.ttsKeys == nil {
				return nil
			}
			if oldTTSKeyPresent {
				return s.ttsKeys.Set(oldTTSKey)
			}
			return s.ttsKeys.Delete()
		}}
		if ttsCredentialChanged {
			if s.ttsKeys == nil {
				return SettingsDTO{}, rollback(errors.New("speech playback credential storage is unavailable"), rollbackProcessingCredential, rollbackCredential, rollbackStartup, rollbackShortcuts)
			}
			var ttsCredentialErr error
			oldTTSKey, ttsCredentialErr = s.ttsKeys.Get()
			if ttsCredentialErr == nil {
				oldTTSKeyPresent = true
			} else if !errors.Is(ttsCredentialErr, credential.ErrNotFound) {
				return SettingsDTO{}, rollback(errors.New("stored speech playback credential could not be read"), rollbackProcessingCredential, rollbackCredential, rollbackStartup, rollbackShortcuts)
			}
			if clearTTSKey {
				ttsCredentialErr = s.ttsKeys.Delete()
			} else {
				ttsCredentialErr = s.ttsKeys.Set(newTTSAPIKey)
			}
			if ttsCredentialErr != nil {
				return SettingsDTO{}, rollback(errors.New("speech playback credential could not be changed"), rollbackTTSCredential, rollbackProcessingCredential, rollbackCredential, rollbackStartup, rollbackShortcuts)
			}
		}

		if persistErr := s.store.Save(v); persistErr != nil {
			if failure := config.LoadFailureFor(persistErr); failure.Kind == "commit_uncertain" {
				s.configuration = ConfigurationStatus{RecoveryRequired: true, ErrorKind: failure.Kind, Message: failure.Message}
			}
			return SettingsDTO{}, rollback(fmt.Errorf("settings were not persisted: %w", persistErr), rollbackTTSCredential, rollbackProcessingCredential, rollbackCredential, rollbackStartup, rollbackShortcuts)
		}
		oldKey = ""
		oldProcessingKey = ""
		oldTTSKey = ""
		s.mu.Lock()
		s.cfg = v
		s.mu.Unlock()
		return s.settingsSnapshotLocked(), nil
	}()
	if err != nil {
		// An uncertain commit must reach every renderer even though Save rejected.
		if s.settingsChanged != nil {
			snapshot := s.GetSettings()
			if snapshot.Configuration.RecoveryRequired {
				s.settingsChanged(snapshot)
			}
		}
		return SettingsDTO{}, err
	}
	s.publishSettingsChange(old, v, result)
	return result, nil
}

// RetryConfiguration reloads the authoritative saved configuration. An expected
// validation failure is returned as renderer-safe status rather than a rejected
// promise so the recovery dialog can update without losing its actions.
func (s *Service) RetryConfiguration() (result SettingsDTO, err error) {
	s.publicationMu.Lock()
	defer s.publicationMu.Unlock()
	started := time.Now()
	s.log().Info("settings recovery retry started")
	s.saveMu.Lock()
	if s.closed.Load() {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("application is shutting down")
	}
	if s.loader == nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("settings recovery is unavailable")
	}
	next, loadErr := s.loader.Load()
	if loadErr != nil {
		failure := config.LoadFailureFor(loadErr)
		s.configuration = ConfigurationStatus{
			RecoveryRequired: true,
			ErrorKind:        failure.Kind,
			Message:          failure.Message,
		}
		result = s.settingsSnapshotLocked()
		s.saveMu.Unlock()
		s.log().Warn("settings recovery retry failed",
			"duration_ms", time.Since(started).Milliseconds(),
			"error_kind", failure.Kind,
		)
		return result, nil
	}
	reservation, err := s.reserveManaged(next.ManagedRuntimes)
	if err != nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, err
	}
	defer reservation.Finish(false)
	result, old, err := s.applyRecoveredSettingsLocked(next, false)
	s.saveMu.Unlock()
	if err != nil {
		s.log().Warn("settings recovery apply failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return SettingsDTO{}, err
	}
	reservation.Finish(true)
	s.publishSettingsChange(old, next, result)
	s.log().Info("settings recovery completed",
		"duration_ms", time.Since(started).Milliseconds(),
		"outcome", "reloaded",
	)
	return result, nil
}

// ResetConfiguration is the only recovery operation that replaces the saved
// document. Credentials remain in the native credential store and are never
// copied into the replacement configuration.
func (s *Service) ResetConfiguration() (result SettingsDTO, err error) {
	s.publicationMu.Lock()
	defer s.publicationMu.Unlock()
	started := time.Now()
	s.log().Info("settings recovery reset started")
	s.saveMu.Lock()
	if s.closed.Load() {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("application is shutting down")
	}
	if !s.configuration.RecoveryRequired {
		result = s.settingsSnapshotLocked()
		s.saveMu.Unlock()
		return result, nil
	}
	next := config.Default()
	reservation, err := s.reserveManaged(next.ManagedRuntimes)
	if err != nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, err
	}
	defer reservation.Finish(false)
	result, old, err := s.applyRecoveredSettingsLocked(next, true)
	s.saveMu.Unlock()
	if err != nil {
		s.log().Warn("settings recovery reset failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return SettingsDTO{}, err
	}
	reservation.Finish(true)
	s.publishSettingsChange(old, next, result)
	s.log().Info("settings recovery completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "reset")
	return result, nil
}

func (s *Service) applyRecoveredSettingsLocked(next config.Settings, persist bool) (SettingsDTO, config.Settings, error) {
	validate := config.ValidateStored
	if persist {
		validate = config.Validate
	}
	if err := validate(next); err != nil {
		return SettingsDTO{}, config.Settings{}, err
	}
	old := s.current()
	// Reconcile native state even when a failed rollback left the same runtime snapshot.
	rollbackShortcuts := rollbackStep{name: "shortcuts"}
	if s.shortcutChanged != nil {
		var err error
		rollbackShortcuts.run, err = s.shortcutChanged(next)
		if err != nil {
			return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("shortcuts were not changed: %w", err), rollbackShortcuts)
		}
	}
	rollbackStartup := rollbackStep{name: "startup", run: func() error {
		return s.startup.Set(old.StartWithWindows)
	}}
	if err := s.startup.Set(next.StartWithWindows); err != nil {
		return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("startup setting was not changed: %w", err), rollbackStartup, rollbackShortcuts)
	}
	if persist {
		save := s.store.Save
		if resetter, ok := s.store.(interface{ Reset(config.Settings) error }); ok {
			save = resetter.Reset
		}
		if err := save(next); err != nil {
			return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("settings were not reset: %w", err), rollbackStartup, rollbackShortcuts)
		}
	}
	s.mu.Lock()
	s.cfg = next
	s.mu.Unlock()
	s.configuration = ConfigurationStatus{}
	return s.settingsSnapshotLocked(), old, nil
}

func (s *Service) publishSettingsChange(old, next config.Settings, result SettingsDTO) {
	if s.managedChanged != nil {
		s.managedChanged(slices.Clone(next.ManagedRuntimes))
	}
	if overlaySettingsDiffer(old, next) && s.overlaySettingsChanged != nil {
		s.overlaySettingsChanged(next)
	}
	if s.historyEnabledChanged != nil {
		s.historyEnabledChanged(next.HistoryEnabled)
	}
	if s.fileSettingsChanged != nil {
		s.fileSettingsChanged(next)
	}
	if s.updateChecksChanged != nil {
		s.updateChecksChanged(next.CheckForUpdates)
	}
	if s.settingsChanged != nil {
		s.settingsChanged(result)
	}
}

func overlaySettingsDiffer(old, next config.Settings) bool {
	return old.ToggleShortcut != next.ToggleShortcut ||
		old.HoldShortcut != next.HoldShortcut ||
		old.OverlayEnabled != next.OverlayEnabled ||
		old.OverlaySizePercent != next.OverlaySizePercent ||
		old.OverlayOpacityPercent != next.OverlayOpacityPercent ||
		old.OverlayTopOffset != next.OverlayTopOffset ||
		old.OverlayGlowPercent != next.OverlayGlowPercent ||
		old.OverlayLayout != next.OverlayLayout ||
		old.OverlayAnchor != next.OverlayAnchor ||
		old.OverlayVisibility != next.OverlayVisibility ||
		old.OverlayMotion != next.OverlayMotion ||
		old.OverlaySurface != next.OverlaySurface ||
		old.OverlayVisualizer != next.OverlayVisualizer
}

func credentialLogAction(draft string, clear bool) string {
	switch {
	case clear:
		return "clear"
	case strings.TrimSpace(draft) != "":
		return "set"
	default:
		return "unchanged"
	}
}

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return diagnostics.DiscardLogger()
}

func (s *Service) ServiceStartup(context.Context, application.ServiceOptions) error {
	s.closed.Store(false)
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.closed.Store(true)
	if closer, ok := s.store.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

func modelCatalog(v config.Settings) modelprofile.Catalog {
	c := modelprofile.Profiles(v.CompatibilityProfile, v.PostProcessing.CompatibilityProfile, v.TextToSpeech.CompatibilityProfile)
	c.VoiceTranscription = modelprofile.VoiceProfiles(v.VoiceTranscription.CompatibilityProfile)
	return c
}
