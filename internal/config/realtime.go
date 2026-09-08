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
	Realtime             bool                               `json:"realtime"`
	CompatibilityProfile compatibility.ID                   `json:"compatibilityProfile"`
	ModelProfile         modelprofile.ID                    `json:"modelProfile"`
	BaseURL              string                             `json:"baseURL"`
	AllowInsecureHTTP    bool                               `json:"allowInsecureHTTP"`
	AuthenticationMode   AuthenticationMode                 `json:"authenticationMode"`
	Model                string                             `json:"model"`
	Language             string                             `json:"language"`
	HealthPath           string                             `json:"healthPath"`
	Headers              map[string]string                  `json:"headers"`
	TimeoutSeconds       int                                `json:"timeoutSeconds"`
	TranscriptionOptions compatibility.TranscriptionOptions `json:"transcriptionOptions"`
	Captions             bool                               `json:"captions"`
	Options              modelprofile.NemotronOptions       `json:"options"`
}

func DefaultVoiceTranscription() VoiceTranscriptionSettings {
	return VoiceTranscriptionSettings{CompatibilityProfile: compatibility.Generic, ModelProfile: modelprofile.Generic, AuthenticationMode: AuthenticationModeNone, Language: "auto", Headers: map[string]string{}, TimeoutSeconds: DefaultTranscriptionTimeoutSeconds, Captions: true}
}
func ValidateVoiceTranscription(v VoiceTranscriptionSettings) error {
	if err := modelprofile.ValidateTranscription(v.ModelProfile, v.CompatibilityProfile, v.Language, v.TranscriptionOptions); err != nil {
		return err
	}
	if err := validatePersistedSTTSettings(WithVoiceTranscription(Settings{VoiceTranscription: v})); err != nil {
		return err
	}
	if v.BaseURL == "" && (v.AuthenticationMode != AuthenticationModeNone || v.Model != "") {
		return errors.New("voice transcription requires a connection")
	}
	if err := speechlanguage.Validate(v.Language); err != nil {
		return err
	}
	if v.TimeoutSeconds < MinRequestTimeoutSeconds || v.TimeoutSeconds > MaxRequestTimeoutSeconds {
		return errors.New("voice transcription timeout is out of range")
	}
	if v.ModelProfile == modelprofile.Nemotron35 {
		if err := modelprofile.ValidateNemotron(v.Language, v.Options); err != nil {
			return err
		}
	} else if v.Options != (modelprofile.NemotronOptions{}) {
		return errors.New("vocabulary strength requires the Nemotron model profile")
	}
	if v.Realtime {
		if _, err := modelprofile.Resolve(v.ModelProfile, v.CompatibilityProfile, compatibility.Realtime); err != nil {
			return err
		}
		if v.BaseURL == "" || v.Model == "" {
			return errors.New("choose a connection and loaded model before enabling realtime transcription")
		}
	}
	return nil
}

// WithVoiceTranscription adapts the immutable voice profile into the existing
// completed transcription workflow without changing file settings in storage.
func WithVoiceTranscription(v Settings) Settings {
	s := v.VoiceTranscription
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

func VoiceFromCompleted(v Settings) VoiceTranscriptionSettings {
	s := DefaultVoiceTranscription()
	s.CompatibilityProfile = v.CompatibilityProfile
	s.ModelProfile = v.ModelProfile
	s.BaseURL = v.BaseURL
	s.AllowInsecureHTTP = v.AllowInsecureHTTP
	s.AuthenticationMode = v.AuthenticationMode
	s.Model = v.Model
	s.Language = v.Language
	s.HealthPath = v.HealthPath
	s.Headers = maps.Clone(v.Headers)
	s.TimeoutSeconds = v.TranscriptionTimeoutSeconds
	s.TranscriptionOptions = v.TranscriptionOptions
	return s
}

func VoiceRealtimeEligible(v VoiceTranscriptionSettings) bool {
	c, err := modelprofile.Resolve(v.ModelProfile, v.CompatibilityProfile, compatibility.Transcription)
	return err == nil && c.Capabilities.Realtime
}
func ValidateVoiceRecording(v VoiceTranscriptionSettings) error {
	if err := ValidateVoiceTranscription(v); err != nil {
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
