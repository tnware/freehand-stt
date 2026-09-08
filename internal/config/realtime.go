package config

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// RealtimeSettings are independently selected from completed/file transcription.
// Enabling this capability changes microphone dictation only.
type RealtimeSettings struct {
	Enabled              bool                         `json:"enabled"`
	CompatibilityProfile compatibility.ID             `json:"compatibilityProfile"`
	ModelProfile         modelprofile.ID              `json:"modelProfile"`
	BaseURL              string                       `json:"baseURL"`
	AllowInsecureHTTP    bool                         `json:"allowInsecureHTTP"`
	AuthenticationMode   AuthenticationMode           `json:"authenticationMode"`
	Model                string                       `json:"model"`
	Language             string                       `json:"language"`
	Captions             bool                         `json:"captions"`
	Options              modelprofile.NemotronOptions `json:"options"`
}

func DefaultRealtime() RealtimeSettings {
	return RealtimeSettings{CompatibilityProfile: compatibility.NeMoSpeechV1, ModelProfile: modelprofile.Nemotron35, AuthenticationMode: AuthenticationModeNone, Language: "auto", Captions: true, Options: modelprofile.NemotronOptions{Boost: 3}}
}

func ValidateRealtime(v RealtimeSettings) error {
	if _, err := modelprofile.Resolve(v.ModelProfile, v.CompatibilityProfile, compatibility.Realtime); err != nil {
		return err
	}
	if v.BaseURL != "" {
		if err := ValidateSTTConnection(v.BaseURL, v.AllowInsecureHTTP, v.AuthenticationMode, v.Model, "", nil); err != nil {
			return err
		}
	} else if v.AuthenticationMode != AuthenticationModeNone {
		return errors.New("realtime authentication requires a connection")
	}
	if v.Enabled && (v.BaseURL == "" || v.Model == "") {
		return errors.New("choose a realtime connection and confirm its loaded model before enabling live transcription")
	}
	return modelprofile.ValidateNemotron(v.Language, v.Options)
}
