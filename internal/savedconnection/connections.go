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
	Update    Action = "update"
	Select    Action = "select"
	Duplicate Action = "duplicate"
	Rename    Action = "rename"
	Delete    Action = "delete"
)
const MaxPerPurpose = 32

type Change struct {
	Action        Action   `json:"action"`
	Details       *Details `json:"details,omitempty"`
	Purpose       Purpose  `json:"purpose"`
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ReplacementID string   `json:"replacementID"`
}

// Details are connection-scoped. Language and general workflow preferences remain operation-scoped.
type Details struct {
	CompatibilityProfile compatibility.ID          `json:"compatibilityProfile"`
	BaseURL              string                    `json:"baseURL"`
	AllowInsecureHTTP    bool                      `json:"allowInsecureHTTP"`
	AuthenticationMode   config.AuthenticationMode `json:"authenticationMode"`
	HealthPath           string                    `json:"healthPath"`
	Headers              map[string]string         `json:"headers"`
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
		d.HealthPath = v.HealthPath
		for k, x := range v.Headers {
			d.Headers[k] = x
		}
	case Cleanup:
		d.CompatibilityProfile = v.PostProcessing.CompatibilityProfile
		d.BaseURL = v.PostProcessing.BaseURL
		d.AllowInsecureHTTP = v.PostProcessing.AllowInsecureHTTP
	case Speech:
		d.CompatibilityProfile = v.TextToSpeech.CompatibilityProfile
		d.BaseURL = v.TextToSpeech.BaseURL
		d.AllowInsecureHTTP = v.TextToSpeech.AllowInsecureHTTP
		d.AuthenticationMode = v.TextToSpeech.AuthenticationMode
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
		v.HealthPath = d.HealthPath
		v.Headers = CloneDetails(d).Headers
	case Cleanup:
		v.PostProcessing.CompatibilityProfile = d.CompatibilityProfile
		v.PostProcessing.BaseURL = d.BaseURL
		v.PostProcessing.AllowInsecureHTTP = d.AllowInsecureHTTP
	case Speech:
		v.TextToSpeech.CompatibilityProfile = d.CompatibilityProfile
		v.TextToSpeech.BaseURL = d.BaseURL
		v.TextToSpeech.AllowInsecureHTTP = d.AllowInsecureHTTP
		v.TextToSpeech.AuthenticationMode = d.AuthenticationMode
	}
	return v
}

// Validate checks the reusable connection contract independently from model and runtime options.
func Validate(p Purpose, d Details) error {
	var operation compatibility.Role
	switch p {
	case Transcription:
		operation = compatibility.Transcription
	case Cleanup:
		operation = compatibility.PostProcessing
	case Speech:
		operation = compatibility.Speech
	default:
		return errors.New("invalid connection purpose")
	}
	if _, err := compatibility.Resolve(d.CompatibilityProfile, operation); err != nil {
		return err
	}
	if p != Transcription && (d.HealthPath != "" || len(d.Headers) != 0) {
		return errors.New("custom headers and health paths belong to transcription connections")
	}
	switch p {
	case Transcription:
		return config.ValidateSTTConnection(d.BaseURL, d.AllowInsecureHTTP, d.AuthenticationMode, "", d.HealthPath, d.Headers)
	case Cleanup:
		if d.AuthenticationMode != config.AuthenticationModeNone {
			return errors.New("cleanup authentication uses an optional API key")
		}
		return config.ValidatePostProcessingConnection(d.BaseURL, d.AllowInsecureHTTP, "")
	default:
		return config.ValidateTextToSpeechConnection(d.BaseURL, d.AllowInsecureHTTP, d.AuthenticationMode, "")
	}
}

// ClearModel requires an explicit model choice after switching servers.
func ClearModel(v config.Settings, p Purpose) config.Settings {
	switch p {
	case Transcription:
		v.Model = ""
		v.SetupCompleted = false
	case Cleanup:
		v.PostProcessing.Model = ""
		v.PostProcessing.Enabled = false
	case Speech:
		v.TextToSpeech.Model = ""
		v.TextToSpeech.Enabled = false
	}
	return v
}
