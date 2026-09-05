package settings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SettingsDTO is the renderer-safe settings snapshot. It reports credential
// presence and native capability state but never returns credential values.
type SettingsDTO struct {
	CompatibilityProfiles compatibility.Catalog `json:"compatibilityProfiles"`
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

// ConfigurationStatus describes whether the durable settings document is
// usable. Renderer code may show preserved newer fields, but must block all
// ordinary mutations while RecoveryRequired is true.
type ConfigurationStatus struct {
	RecoveryRequired    bool     `json:"recoveryRequired"`
	ErrorKind           string   `json:"errorKind,omitempty"`
	Message             string   `json:"message,omitempty"`
	PreservedFieldCount int      `json:"preservedFieldCount,omitempty"`
	PreservedFields     []string `json:"preservedFields,omitempty"`
}

// SaveSettingsRequest groups the persisted settings and transient credential
// changes into one binding argument. Credential drafts are never returned.
type SaveSettingsRequest struct {
	Settings                      config.Settings `json:"settings"`
	STTCredentialDraft            string          `json:"sttCredentialDraft,omitempty"`
	ClearSTTCredential            bool            `json:"clearSTTCredential"`
	PostProcessingCredentialDraft string          `json:"postProcessingCredentialDraft,omitempty"`
	ClearPostProcessingCredential bool            `json:"clearPostProcessingCredential"`
	TextToSpeechCredentialDraft   string          `json:"textToSpeechCredentialDraft,omitempty"`
	ClearTextToSpeechCredential   bool            `json:"clearTextToSpeechCredential"`
}

type HoldInfo func() (bool, string)
type Startup interface {
	Set(bool) error
}
type ConfigStore interface{ Save(config.Settings) error }
type ConfigLoader interface {
	ConfigStore
	Load() (config.Settings, error)
	LoadReport() config.LoadReport
}

type Option func(*Service)

// WithConfigurationLoad connects the startup load result to the bound
// settings service. It is optional so isolated service tests and consumers
// that only need persistence do not acquire a recovery dependency.
func WithConfigurationLoad(loader ConfigLoader, failure *config.LoadFailure, report config.LoadReport) Option {
	return func(service *Service) {
		service.loader = loader
		service.configuration.PreservedFieldCount = report.PreservedFieldCount
		service.configuration.PreservedFields = append([]string(nil), report.PreservedFields...)
		if failure != nil {
			service.configuration.RecoveryRequired = true
			service.configuration.ErrorKind = failure.Kind
			service.configuration.Message = failure.Message
			service.configuration.PreservedFieldCount = 0
			service.configuration.PreservedFields = nil
		}
	}
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
	mu                     sync.RWMutex
	saveMu                 sync.Mutex
	cfg                    config.Settings
	store                  ConfigStore
	loader                 ConfigLoader
	configuration          ConfigurationStatus
	keys                   credential.Store
	processKeys            credential.Store
	ttsKeys                credential.Store
	startup                Startup
	hold                   HoldInfo
	shortcutChanged        func(config.Settings) error
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

func NewService(st ConfigStore, cfg config.Settings, k credential.Store, processKeys credential.Store, start Startup, hold HoldInfo, shortcutChanged func(config.Settings) error, overlaySettingsChanged func(config.Settings), historyEnabledChanged func(bool), fileSettingsChanged func(config.Settings), settingsChanged func(SettingsDTO), logger *slog.Logger, options ...Option) *Service {
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
		micaActive:             cfg.UseMica,
		appearanceModeActive:   cfg.EffectiveAppearanceMode(),
	}
	for _, option := range options {
		option(service)
	}
	return service
}

// Source and ProfileSource are ordinary injected Go functions. They expose no
// additional Wails methods while allowing each feature to capture one coherent
// settings/credential snapshot at the start of an operation.
type Source func() config.Settings

func (source Source) Current() config.Settings { return source() }

type RequestProfile struct {
	Settings                 config.Settings
	STTCredential            string
	PostProcessingCredential string
}

type ProfileSource func() (RequestProfile, error)

func (source ProfileSource) Capture() (RequestProfile, error) { return source() }

func CurrentSource(service *Service) Source { return service.current }

func RequestProfiles(service *Service) ProfileSource {
	return service.captureRequestProfile
}

type TextToSpeechProfile struct {
	Settings   config.TextToSpeechSettings
	Credential string
}

type TextToSpeechProfileSource func() (TextToSpeechProfile, error)

func (source TextToSpeechProfileSource) Capture() (TextToSpeechProfile, error) { return source() }

func TextToSpeechProfiles(service *Service) TextToSpeechProfileSource {
	return service.captureTextToSpeechProfile
}

func (s *Service) current() config.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.cfg
	v.Headers = clone(v.Headers)
	return v
}
func clone(m map[string]string) map[string]string {
	o := map[string]string{}
	for k, v := range m {
		o[k] = v
	}
	return o
}

func (s *Service) captureRequestProfile() (RequestProfile, error) {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if s.closed.Load() {
		return RequestProfile{}, errors.New("application is shutting down")
	}
	if s.configuration.RecoveryRequired {
		return RequestProfile{}, errors.New("saved settings must be recovered before transcription can start")
	}
	profile := RequestProfile{Settings: s.current()}
	if profile.Settings.AuthenticationMode == config.AuthenticationModeAPIKey {
		if s.keys == nil {
			return RequestProfile{}, errors.New("API credential is not configured")
		}
		key, err := s.keys.Get()
		if err != nil {
			return RequestProfile{}, errors.New("API credential is not configured")
		}
		profile.STTCredential = key
	}
	if profile.Settings.PostProcessing.Enabled && s.processKeys != nil {
		key, err := s.processKeys.Get()
		switch {
		case err == nil:
			profile.PostProcessingCredential = key
		case errors.Is(err, credential.ErrNotFound):
		case err != nil:
			profile.STTCredential = ""
			return RequestProfile{}, errors.New("post-processing credential could not be read")
		}
	}
	return profile, nil
}

func (s *Service) captureTextToSpeechProfile() (TextToSpeechProfile, error) {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if s.closed.Load() {
		return TextToSpeechProfile{}, errors.New("application is shutting down")
	}
	if s.configuration.RecoveryRequired {
		return TextToSpeechProfile{}, errors.New("saved settings must be recovered before speech playback can start")
	}
	profile := TextToSpeechProfile{Settings: s.current().TextToSpeech}
	if !profile.Settings.Enabled {
		return TextToSpeechProfile{}, errors.New("speech playback is disabled")
	}
	if profile.Settings.AuthenticationMode == config.AuthenticationModeAPIKey {
		if s.ttsKeys == nil {
			return TextToSpeechProfile{}, errors.New("speech playback credential is not configured")
		}
		key, err := s.ttsKeys.Get()
		if err != nil {
			return TextToSpeechProfile{}, errors.New("speech playback credential is not configured")
		}
		profile.Credential = key
	}
	return profile, nil
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
// JSON settings, native shortcuts, startup registration, and credentials.

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
	return SettingsDTO{
		CompatibilityProfiles:              compatibility.Profiles(),
		Settings:                           v,
		Configuration:                      cloneConfigurationStatus(s.configuration),
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

// SaveSettings atomically applies one complete settings and credential change
// request, rolling back native changes if persistence fails.
func (s *Service) SaveSettings(request SaveSettingsRequest) (result SettingsDTO, err error) {
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
		if validateErr := config.Validate(v); validateErr != nil {
			return SettingsDTO{}, validateErr
		}

		old = s.current()
		shortcutsChanged := old.ToggleShortcut != v.ToggleShortcut || old.ShowShortcut != v.ShowShortcut || old.HoldShortcut != v.HoldShortcut
		rollbackShortcuts := rollbackStep{name: "shortcuts", run: func() error {
			if shortcutsChanged && s.shortcutChanged != nil {
				return s.shortcutChanged(old)
			}
			return nil
		}}
		if shortcutsChanged && s.shortcutChanged != nil {
			if shortcutErr := s.shortcutChanged(v); shortcutErr != nil {
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
		return SettingsDTO{}, err
	}
	if overlaySettingsDiffer(old, v) && s.overlaySettingsChanged != nil {
		s.overlaySettingsChanged(v)
	}
	if s.historyEnabledChanged != nil {
		s.historyEnabledChanged(v.HistoryEnabled)
	}
	if s.fileSettingsChanged != nil {
		s.fileSettingsChanged(v)
	}
	if s.updateChecksChanged != nil {
		s.updateChecksChanged(v.CheckForUpdates)
	}
	if s.settingsChanged != nil {
		s.settingsChanged(result)
	}
	return result, nil
}

// RetryConfiguration re-reads the original settings document. An expected
// validation failure is returned as renderer-safe status rather than a rejected
// promise so the recovery dialog can update without losing its actions.
func (s *Service) RetryConfiguration() (result SettingsDTO, err error) {
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
	result, old, err := s.applyRecoveredSettingsLocked(next, false)
	s.saveMu.Unlock()
	if err != nil {
		s.log().Warn("settings recovery apply failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return SettingsDTO{}, err
	}
	s.publishSettingsChange(old, next, result)
	s.log().Info("settings recovery completed",
		"duration_ms", time.Since(started).Milliseconds(),
		"preserved_field_count", result.Configuration.PreservedFieldCount,
		"outcome", "reloaded",
	)
	return result, nil
}

// ResetConfiguration is the only recovery operation that replaces the saved
// document. Credentials remain in the native credential store and are never
// copied into the replacement JSON file.
func (s *Service) ResetConfiguration() (result SettingsDTO, err error) {
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
	result, old, err := s.applyRecoveredSettingsLocked(next, true)
	s.saveMu.Unlock()
	if err != nil {
		s.log().Warn("settings recovery reset failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return SettingsDTO{}, err
	}
	s.publishSettingsChange(old, next, result)
	s.log().Info("settings recovery completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "reset")
	return result, nil
}

func (s *Service) applyRecoveredSettingsLocked(next config.Settings, persist bool) (SettingsDTO, config.Settings, error) {
	if err := config.Validate(next); err != nil {
		return SettingsDTO{}, config.Settings{}, err
	}
	old := s.current()
	shortcutsChanged := old.ToggleShortcut != next.ToggleShortcut || old.ShowShortcut != next.ShowShortcut || old.HoldShortcut != next.HoldShortcut
	rollbackShortcuts := rollbackStep{name: "shortcuts", run: func() error {
		if shortcutsChanged && s.shortcutChanged != nil {
			return s.shortcutChanged(old)
		}
		return nil
	}}
	if shortcutsChanged && s.shortcutChanged != nil {
		if err := s.shortcutChanged(next); err != nil {
			return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("shortcuts were not changed: %w", err), rollbackShortcuts)
		}
	}
	startupChanged := old.StartWithWindows != next.StartWithWindows
	rollbackStartup := rollbackStep{name: "startup", run: func() error {
		if startupChanged {
			return s.startup.Set(old.StartWithWindows)
		}
		return nil
	}}
	if startupChanged {
		if err := s.startup.Set(next.StartWithWindows); err != nil {
			return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("startup setting was not changed: %w", err), rollbackStartup, rollbackShortcuts)
		}
	}
	if persist {
		if err := s.store.Save(next); err != nil {
			return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("settings were not reset: %w", err), rollbackStartup, rollbackShortcuts)
		}
	}
	s.mu.Lock()
	s.cfg = next
	s.mu.Unlock()
	s.configuration = ConfigurationStatus{}
	if !persist && s.loader != nil {
		report := s.loader.LoadReport()
		s.configuration.PreservedFieldCount = report.PreservedFieldCount
		s.configuration.PreservedFields = append([]string(nil), report.PreservedFields...)
	}
	return s.settingsSnapshotLocked(), old, nil
}

func (s *Service) publishSettingsChange(old, next config.Settings, result SettingsDTO) {
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

func cloneConfigurationStatus(status ConfigurationStatus) ConfigurationStatus {
	status.PreservedFields = append([]string(nil), status.PreservedFields...)
	return status
}

func overlaySettingsDiffer(old, next config.Settings) bool {
	return old.OverlayEnabled != next.OverlayEnabled ||
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
	return nil
}
