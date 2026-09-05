// Package savedconnection defines named endpoint configurations without credentials or SQL.
package savedconnection

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
)

type Purpose string

const (
	Transcription Purpose = "stt"
	Cleanup       Purpose = "cleanup"
	Speech        Purpose = "speech"
)

type Action string

const (
	Create    Action = "create"
	Select    Action = "select"
	Duplicate Action = "duplicate"
	Rename    Action = "rename"
	Delete    Action = "delete"
)
const MaxPerPurpose = 32

type Change struct {
	Action        Action  `json:"action"`
	Purpose       Purpose `json:"purpose"`
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ReplacementID string  `json:"replacementID"`
}

// Details are connection-scoped. Language and general workflow preferences remain operation-scoped.
type Details struct {
	CompatibilityProfile compatibility.ID            `json:"compatibilityProfile"`
	BaseURL              string                      `json:"baseURL"`
	AllowInsecureHTTP    bool                        `json:"allowInsecureHTTP"`
	AuthenticationMode   config.AuthenticationMode   `json:"authenticationMode"`
	Model                string                      `json:"model"`
	HealthPath           string                      `json:"healthPath"`
	Headers              map[string]string           `json:"headers"`
	CleanupPreset        config.PostProcessingPreset `json:"cleanupPreset"`
}
type Connection struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Purpose       Purpose `json:"purpose"`
	Details       Details `json:"details"`
	HasCredential bool    `json:"hasCredential"`
}
type Catalog struct {
	Entries  []Connection       `json:"entries"`
	Selected map[Purpose]string `json:"selected"`
}

func ValidPurpose(p Purpose) bool { return p == Transcription || p == Cleanup || p == Speech }
func ValidateName(name string) error {
	if strings.TrimSpace(name) != name || name == "" || len(name) > 80 || !utf8.ValidString(name) {
		return errors.New("connection name must be 1 to 80 UTF-8 bytes without surrounding whitespace")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return errors.New("connection name cannot contain control characters")
		}
	}
	return nil
}
func CloneDetails(v Details) Details {
	h := map[string]string{}
	for k, x := range v.Headers {
		h[k] = x
	}
	v.Headers = h
	return v
}
func Extract(v config.Settings, p Purpose) Details {
	d := Details{Headers: map[string]string{}, AuthenticationMode: config.AuthenticationModeNone}
	switch p {
	case Transcription:
		d.CompatibilityProfile = v.CompatibilityProfile
		d.BaseURL = v.BaseURL
		d.AllowInsecureHTTP = v.AllowInsecureHTTP
		d.AuthenticationMode = v.AuthenticationMode
		d.Model = v.Model
		d.HealthPath = v.HealthPath
		for k, x := range v.Headers {
			d.Headers[k] = x
		}
	case Cleanup:
		d.CompatibilityProfile = v.PostProcessing.CompatibilityProfile
		d.BaseURL = v.PostProcessing.BaseURL
		d.AllowInsecureHTTP = v.PostProcessing.AllowInsecureHTTP
		d.Model = v.PostProcessing.Model
		d.CleanupPreset = v.PostProcessing.Preset
	case Speech:
		d.CompatibilityProfile = v.TextToSpeech.CompatibilityProfile
		d.BaseURL = v.TextToSpeech.BaseURL
		d.AllowInsecureHTTP = v.TextToSpeech.AllowInsecureHTTP
		d.AuthenticationMode = v.TextToSpeech.AuthenticationMode
		d.Model = v.TextToSpeech.Model
	}
	return d
}

// Apply selects endpoint details. The settings owner validates shared options before committing.
func Apply(v config.Settings, p Purpose, d Details) config.Settings {
	switch p {
	case Transcription:
		v.CompatibilityProfile = d.CompatibilityProfile
		v.BaseURL = d.BaseURL
		v.AllowInsecureHTTP = d.AllowInsecureHTTP
		v.AuthenticationMode = d.AuthenticationMode
		v.Model = d.Model
		v.HealthPath = d.HealthPath
		v.Headers = CloneDetails(d).Headers
		if c, err := compatibility.Resolve(d.CompatibilityProfile, compatibility.Transcription); err == nil {
			if d.BaseURL == "" || (!c.Capabilities.ServerLoadedModel && d.Model == "") {
				v.SetupCompleted = false
			}
		}
	case Cleanup:
		v.PostProcessing.CompatibilityProfile = d.CompatibilityProfile
		v.PostProcessing.BaseURL = d.BaseURL
		v.PostProcessing.AllowInsecureHTTP = d.AllowInsecureHTTP
		v.PostProcessing.Model = d.Model
		v.PostProcessing.Preset = d.CleanupPreset
		if d.BaseURL == "" || d.Model == "" {
			v.PostProcessing.Enabled = false
		}
	case Speech:
		v.TextToSpeech.CompatibilityProfile = d.CompatibilityProfile
		v.TextToSpeech.BaseURL = d.BaseURL
		v.TextToSpeech.AllowInsecureHTTP = d.AllowInsecureHTTP
		v.TextToSpeech.AuthenticationMode = d.AuthenticationMode
		v.TextToSpeech.Model = d.Model
		if d.BaseURL == "" || d.Model == "" {
			v.TextToSpeech.Enabled = false
		}
	}
	return v
}
