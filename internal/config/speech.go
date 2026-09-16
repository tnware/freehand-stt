package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

const (
	MaxTTSInputBytes      = 256 * 1024
	MaxTTSInputCharacters = 4096
)

type TextToSpeechSettings struct {
	ManagedInstanceID    string                     `json:"managedInstanceID,omitempty"`
	Options              modelprofile.SpeechOptions `json:"options"`
	ModelProfile         modelprofile.ID            `json:"modelProfile"`
	CompatibilityProfile compatibility.ID           `json:"compatibilityProfile"`
	Enabled              bool                       `json:"enabled"`
	BaseURL              string                     `json:"baseURL"`
	AllowInsecureHTTP    bool                       `json:"allowInsecureHTTP"`
	AuthenticationMode   AuthenticationMode         `json:"authenticationMode"`
	Model                string                     `json:"model"`
	Voice                string                     `json:"voice"`
	Speed                float64                    `json:"speed"`
	TimeoutSeconds       int                        `json:"timeoutSeconds"`
}

func ValidateTextToSpeech(s TextToSpeechSettings, requireConnection bool) error {
	return validateTextToSpeech(s, requireConnection, false)
}

func validateTextToSpeech(s TextToSpeechSettings, requireConnection, managedPreferences bool) error {
	if s.ModelProfile == modelprofile.MagpieTTS && s.Speed != 0 && s.Speed != 1 {
		return fieldError("textToSpeech.speed", "MagpieTTS uses normal speed. Set speaking speed to 1.0 before choosing this model or connection.", errors.New("MagpieTTS does not support adjustable speaking speed"))
	}
	if err := modelprofile.ValidateSpeech(s.ModelProfile, s.CompatibilityProfile, s.Speed); err != nil {
		return fieldError("textToSpeech.modelProfile", "Choose a compatible speech model profile and speaking speed.", err)
	}
	if err := modelprofile.ValidateSpeechOptions(s.ModelProfile, s.CompatibilityProfile, s.Voice, s.Options); err != nil {
		return fieldError("textToSpeech.options", "Check the speech language, voice, and style instructions.", err)
	}
	if _, err := compatibility.Resolve(s.CompatibilityProfile, compatibility.Speech); err != nil {
		return fieldError("textToSpeech.compatibilityProfile", "Choose a supported speech server profile.", err)
	}
	if err := validateTimeout("speech generation", s.TimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return fieldError("textToSpeech.timeoutSeconds", fmt.Sprintf("Enter a speech timeout from %d to %d seconds.", MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds), err)
	}
	if (requireConnection && strings.TrimSpace(s.Voice) == "") || len(s.Voice) > 200 {
		return fieldError("textToSpeech.voice", "Choose a speech voice of at most 200 characters.", errors.New("speech playback voice is required and must be at most 200 characters"))
	}
	if managedPreferences {
		if err := validateManagedTransport(s.BaseURL, s.AllowInsecureHTTP, s.AuthenticationMode, "", nil); err != nil {
			return err
		}
		if s.Speed < 0.25 || s.Speed > 4 {
			return fieldError("textToSpeech.speed", "Enter a speaking speed from 0.25 to 4.", errors.New("speech playback speed must be between 0.25 and 4"))
		}
		return nil
	}
	if !requireConnection && strings.TrimSpace(s.BaseURL) == "" && strings.TrimSpace(s.Model) == "" {
		if s.Speed == 0 {
			return nil
		}
		if s.Speed < 0.25 || s.Speed > 4 {
			return fieldError("textToSpeech.speed", "Enter a speaking speed from 0.25 to 4.", errors.New("speech playback speed must be between 0.25 and 4"))
		}
		return nil
	}
	if err := validateTextToSpeechConnection(s.BaseURL, s.AllowInsecureHTTP, s.AuthenticationMode, s.Model, requireConnection); err != nil {
		return fieldError("textToSpeech.baseURL", "Check the speech connection, authentication mode, and model.", err)
	}
	if s.Speed < 0.25 || s.Speed > 4 {
		return fieldError("textToSpeech.speed", "Enter a speaking speed from 0.25 to 4.", errors.New("speech playback speed must be between 0.25 and 4"))
	}
	return nil
}
