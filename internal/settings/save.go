package settings

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

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
