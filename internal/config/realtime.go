package config

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
	"maps"
	"strings"
)

// VoiceTranscriptionSettings owns the one active microphone provider/model.
// Realtime selects a transport for that same combination; files are independent.
type VoiceTranscriptionSettings struct {
	ManagedInstanceID    string               `json:"managedInstanceID,omitempty"`
	Realtime             bool                 `json:"realtime"`
	CompatibilityProfile compatibility.ID     `json:"compatibilityProfile"`
	ModelProfile         modelprofile.ID      `json:"modelProfile"`
	BaseURL              string               `json:"baseURL"`
	AllowInsecureHTTP    bool                 `json:"allowInsecureHTTP"`
	AuthenticationMode   AuthenticationMode   `json:"authenticationMode"`
	Model                string               `json:"model"`
	Language             string               `json:"language"`
	HealthPath           string               `json:"healthPath"`
	Headers              map[string]string    `json:"headers"`
	TimeoutSeconds       int                  `json:"timeoutSeconds"`
	TranscriptionOptions TranscriptionOptions `json:"transcriptionOptions"`
	Captions             bool                 `json:"captions"`
	realtimeOptions      modelprofile.NemotronOptions
}

func DefaultVoiceTranscription() VoiceTranscriptionSettings {
	return VoiceTranscriptionSettings{CompatibilityProfile: compatibility.Generic, ModelProfile: modelprofile.Generic, AuthenticationMode: AuthenticationModeNone, Language: "auto", Headers: map[string]string{}, TimeoutSeconds: DefaultTranscriptionTimeoutSeconds, Captions: true}
}
func ValidateVoiceTranscription(v VoiceTranscriptionSettings) error {
	return validateVoiceTranscription(v, false)
}

func validateVoiceTranscription(v VoiceTranscriptionSettings, stored bool) error {
	if err := validateVoiceOptions(v, stored); err != nil {
		return err
	}
	return validatePersistedSTTSettings(WithVoiceTranscription(Settings{VoiceTranscription: v}))
}

func validateVoiceOptions(v VoiceTranscriptionSettings, stored bool) error {
	validateTranscription := modelprofile.ValidateTranscription
	if stored {
		validateTranscription = modelprofile.ValidateStoredTranscription
	}
	if err := validateTranscription(v.ModelProfile, v.CompatibilityProfile, v.Language, v.TranscriptionOptions.Inference()); err != nil {
		return err
	}

	if v.ManagedInstanceID == "" && v.BaseURL == "" && (v.AuthenticationMode != AuthenticationModeNone || v.Model != "") {
		return errors.New("voice transcription requires a connection")
	}
	if err := speechlanguage.Validate(v.Language); err != nil {
		return err
	}
	if v.TimeoutSeconds < MinRequestTimeoutSeconds || v.TimeoutSeconds > MaxRequestTimeoutSeconds {
		return errors.New("voice transcription timeout is out of range")
	}
	if v.ModelProfile == modelprofile.Nemotron35 {
		var err error
		if stored {
			err = modelprofile.ValidateNemotronOptions(v.RealtimeOptions())
		} else {
			err = modelprofile.ValidateNemotron(v.Language, v.RealtimeOptions())
		}
		if err != nil {
			return err
		}
	} else if v.RealtimeOptions() != (modelprofile.NemotronOptions{}) {
		return errors.New("vocabulary strength requires the Nemotron model profile")
	}
	if v.Realtime {
		if _, err := modelprofile.Resolve(v.ModelProfile, v.CompatibilityProfile, compatibility.Realtime); err != nil {
			return err
		}
		if (v.ManagedInstanceID == "" && v.BaseURL == "") || v.Model == "" {
			return errors.New("choose a connection and loaded model before enabling realtime transcription")
		}
	}
	return nil
}

// WithVoiceTranscription adapts the immutable voice profile into the existing
// completed transcription workflow without changing file settings in storage.
func WithVoiceTranscription(v Settings) Settings {
	s := v.VoiceTranscription
	v.ManagedInstanceID = s.ManagedInstanceID
	v.CompatibilityProfile = s.CompatibilityProfile
	v.ModelProfile = s.ModelProfile
	v.BaseURL = s.BaseURL
	v.AllowInsecureHTTP = s.AllowInsecureHTTP
	v.AuthenticationMode = s.AuthenticationMode
	v.Model = s.Model
	v.Language = s.Language
	v.HealthPath = s.HealthPath
	v.Headers = maps.Clone(s.Headers)
	v.TranscriptionOptions = s.TranscriptionOptions
	v.TranscriptionTimeoutSeconds = s.TimeoutSeconds
	return v
}

func VoiceRealtimeEligible(v VoiceTranscriptionSettings) bool {
	c, err := modelprofile.Resolve(v.ModelProfile, v.CompatibilityProfile, compatibility.Transcription)
	return err == nil && c.Capabilities.Realtime
}

// ValidateVoiceRecording admits an already resolved request, not a durable
// managed reference. Runtime identity/ownership is resolved by settings first.
func ValidateVoiceRecording(v VoiceTranscriptionSettings) error {
	if err := validateVoiceOptions(v, false); err != nil {
		return err
	}
	if v.ManagedInstanceID != "" {
		if err := validateResolvedManagedTransport(v.BaseURL, v.AuthenticationMode, v.HealthPath, v.Headers); err != nil {
			return err
		}
	}
	if err := ValidateSTTConnection(v.BaseURL, v.AllowInsecureHTTP, v.AuthenticationMode, v.Model, v.HealthPath, v.Headers); err != nil {
		return err
	}
	c, err := modelprofile.Resolve(v.ModelProfile, v.CompatibilityProfile, compatibility.Transcription)
	if err != nil {
		return err
	}
	if v.BaseURL == "" || (strings.TrimSpace(v.Model) == "" && !c.Capabilities.ServerLoadedModel) {
		return errors.New("choose a Voice transcription connection and model before recording")
	}
	return nil
}
