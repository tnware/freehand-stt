package settings

import (
	"errors"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

// Source and ProfileSource are ordinary injected Go functions. They expose no
// additional Wails methods while allowing each feature to capture one coherent
// settings/credential snapshot at the start of an operation.
type Source func() config.Settings

func (source Source) Current() config.Settings { return source() }

type RequestProfile struct {
	Settings                  config.Settings
	STTCredential             string
	VoiceCredential           string
	PostProcessingCredential  string
	PostProcessingUnavailable error
}

type ProfileSource func() (RequestProfile, error)

func (source ProfileSource) Capture() (RequestProfile, error) { return source() }

func CurrentSource(service *Service) Source { return service.effectiveCurrent }

func DictationProfiles(service *Service) ProfileSource {
	return func() (RequestProfile, error) { return service.captureProfile(true) }
}

func RequestProfiles(service *Service) ProfileSource {
	return service.captureRequestProfile
}

type TextToSpeechProfile struct {
	Settings   config.TextToSpeechSettings
	Credential string
}

// TextToSpeechPreview contains only non-secret, request-level draft options.
// Connection identity is checked against the active saved speech connection.
type TextToSpeechPreview struct {
	Options        modelprofile.SpeechOptions `json:"options"`
	ConnectionID   string                     `json:"connectionID"`
	Enabled        bool                       `json:"enabled"`
	ModelProfile   modelprofile.ID            `json:"modelProfile"`
	Model          string                     `json:"model"`
	Voice          string                     `json:"voice"`
	Speed          float64                    `json:"speed"`
	TimeoutSeconds int                        `json:"timeoutSeconds"`
}

type TextToSpeechProfileSource func(*TextToSpeechPreview) (TextToSpeechProfile, error)

func (source TextToSpeechProfileSource) Capture() (TextToSpeechProfile, error) { return source(nil) }

func (source TextToSpeechProfileSource) CapturePreview(draft *TextToSpeechPreview) (TextToSpeechProfile, error) {
	return source(draft)
}

func TextToSpeechProfiles(service *Service) TextToSpeechProfileSource {
	return service.captureTextToSpeechProfile
}

func (s *Service) current() config.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.cfg
	v.ManagedRuntimes = slices.Clone(v.ManagedRuntimes)
	v.Headers = cloneHeaders(v.Headers)
	v.VoiceTranscription.Headers = cloneHeaders(v.VoiceTranscription.Headers)
	return v
}
func cloneHeaders(m map[string]string) map[string]string {
	o := map[string]string{}
	maps.Copy(o, m)
	return o
}

func (s *Service) captureRequestProfile() (RequestProfile, error) { return s.captureProfile(false) }

func (s *Service) captureProfile(dictation bool) (RequestProfile, error) {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if s.closed.Load() {
		return RequestProfile{}, errors.New("application is shutting down")
	}
	if s.configuration.RecoveryRequired {
		return RequestProfile{}, errors.New("saved settings must be recovered before transcription can start")
	}
	profile := RequestProfile{Settings: s.current()}
	var managedErr error
	profile.Settings, managedErr = s.managedSettings(profile.Settings, dictation)
	if managedErr != nil {
		return RequestProfile{}, managedErr
	}
	if !dictation && profile.Settings.AuthenticationMode == config.AuthenticationModeAPIKey {
		if s.keys == nil {
			return RequestProfile{}, errors.New("API credential is not configured")
		}
		key, err := s.keys.Get()
		if err != nil {
			return RequestProfile{}, errors.New("API credential is not configured")
		}
		profile.STTCredential = key
	}
	if dictation && profile.Settings.VoiceTranscription.AuthenticationMode == config.AuthenticationModeAPIKey {
		if s.voiceKeys == nil {
			return RequestProfile{}, errors.New("voice transcription credential is not configured")
		}
		key, err := s.voiceKeys.Get()
		if err != nil {
			return RequestProfile{}, errors.New("voice transcription credential is not configured")
		}
		profile.VoiceCredential = key
		profile.STTCredential = key
	}
	profile.Settings, profile.PostProcessingUnavailable = s.managedCleanup(profile.Settings)
	if profile.Settings.PostProcessing.Enabled && profile.Settings.PostProcessing.ManagedInstanceID == "" && s.processKeys != nil {
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
	if dictation {
		var err error
		profile.Settings, err = config.WithVocabulary(profile.Settings, true)
		if err != nil {
			return RequestProfile{}, err
		}
		profile.Settings = config.WithVoiceTranscription(profile.Settings)
	} else {
		var err error
		profile.Settings, err = config.WithVocabulary(profile.Settings, false)
		if err != nil {
			return RequestProfile{}, err
		}
	}
	if (!dictation && profile.Settings.ManagedInstanceID != "") || (dictation && profile.Settings.VoiceTranscription.ManagedInstanceID != "") {
		if dictation {
			if err := config.ValidateVoiceRecording(profile.Settings.VoiceTranscription); err != nil {
				return RequestProfile{}, err
			}
		} else if err := modelprofile.ValidateTranscription(profile.Settings.ModelProfile, profile.Settings.CompatibilityProfile, profile.Settings.Language, profile.Settings.TranscriptionOptions.Inference()); err != nil {
			return RequestProfile{}, err
		}
	}
	return profile, nil
}

func (s *Service) captureTextToSpeechProfile(draft *TextToSpeechPreview) (TextToSpeechProfile, error) {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if s.closed.Load() {
		return TextToSpeechProfile{}, errors.New("application is shutting down")
	}
	if s.configuration.RecoveryRequired {
		return TextToSpeechProfile{}, errors.New("saved settings must be recovered before speech playback can start")
	}
	current := s.current()
	profile := TextToSpeechProfile{Settings: current.TextToSpeech}
	if draft != nil {
		store, ok := s.store.(interface {
			ConnectionCatalog() savedconnection.Catalog
		})
		if !ok || draft.ConnectionID == "" || draft.ConnectionID != store.ConnectionCatalog().Selected[savedconnection.Speech] {
			return TextToSpeechProfile{}, errors.New("speech connection changed; reopen settings before previewing")
		}
		if math.IsNaN(draft.Speed) || math.IsInf(draft.Speed, 0) {
			return TextToSpeechProfile{}, errors.New("speech preview speed must be finite")
		}
		if profile.Settings.ManagedInstanceID != "" && (strings.TrimSpace(draft.Model) != profile.Settings.Model || draft.ModelProfile != profile.Settings.ModelProfile) {
			return TextToSpeechProfile{}, errors.New("speech preview cannot change the managed instance model or profile")
		}
		profile.Settings.Options = draft.Options
		profile.Settings.Enabled = draft.Enabled
		profile.Settings.ModelProfile = draft.ModelProfile
		profile.Settings.Model = strings.TrimSpace(draft.Model)
		profile.Settings.Voice = strings.TrimSpace(draft.Voice)
		profile.Settings.Speed = draft.Speed
		profile.Settings.TimeoutSeconds = draft.TimeoutSeconds
		// Validate the actual resolved transport below, after the bounded draft.
	}
	if !profile.Settings.Enabled {
		return TextToSpeechProfile{}, errors.New("speech playback is disabled")
	}
	current.TextToSpeech = profile.Settings
	resolved, err := s.managedSpeech(current)
	if err != nil {
		return TextToSpeechProfile{}, err
	}
	profile.Settings = resolved.TextToSpeech
	if draft != nil || profile.Settings.ManagedInstanceID != "" {
		if err := config.ValidateTextToSpeech(profile.Settings, true); err != nil {
			return TextToSpeechProfile{}, err
		}
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
