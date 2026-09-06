// Package savedconnection defines named endpoint configurations without credentials or SQL.
package savedconnection

import (
	"errors"
	"slices"
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
	// ActivateFor selects a newly created connection in the same settings transaction.
	// Library creation omits it and remains inactive.
	ActivateFor Purpose   `json:"activateFor,omitempty"`
	Action      Action    `json:"action"`
	Details     *Details  `json:"details,omitempty"`
	Uses        []Purpose `json:"uses,omitempty"`
	Purpose     Purpose   `json:"purpose,omitempty"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
}

// Details are connection-scoped. Model preferences and workflow settings are separate.
type Details struct {
	CompatibilityProfile compatibility.ID          `json:"compatibilityProfile"`
	BaseURL              string                    `json:"baseURL"`
	AllowInsecureHTTP    bool                      `json:"allowInsecureHTTP"`
	AuthenticationMode   config.AuthenticationMode `json:"authenticationMode"`
	HealthPath           string                    `json:"healthPath"`
	Headers              map[string]string         `json:"headers"`
}
type Connection struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Uses          []Purpose `json:"uses"`
	Details       Details   `json:"details"`
	HasCredential bool      `json:"hasCredential"`
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

// Apply selects endpoint details. The settings owner validates model options before committing.
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
	if err := config.ValidateSTTConnection(d.BaseURL, d.AllowInsecureHTTP, d.AuthenticationMode, "", d.HealthPath, d.Headers); err != nil {
		return err
	}
	switch p {
	case Transcription:
		return config.ValidateSTTConnection(d.BaseURL, d.AllowInsecureHTTP, d.AuthenticationMode, "", d.HealthPath, d.Headers)
	case Cleanup:
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

// Supports is the user's explicit declaration of an implemented use, not inferred server evidence.
func (c Connection) Supports(p Purpose) bool { return slices.Contains(c.Uses, p) }
func ValidateUses(uses []Purpose, d Details) error {
	if len(uses) == 0 || len(uses) > 3 {
		return errors.New("choose at least one supported use")
	}
	seen := map[Purpose]bool{}
	for _, p := range uses {
		if seen[p] {
			return errors.New("duplicate connection use")
		}
		seen[p] = true
		if err := Validate(p, d); err != nil {
			return err
		}
	}
	if !seen[Transcription] && (d.HealthPath != "" || len(d.Headers) != 0) {
		return errors.New("custom health paths and headers require transcription use")
	}
	return nil
}

// Project reflects the fields owned by one runtime; the connection keeps its full shared details.
func Project(d Details, p Purpose) Details {
	d = CloneDetails(d)
	if p != Transcription {
		d.HealthPath = ""
		d.Headers = map[string]string{}
	}
	if p == Cleanup {
		d.AuthenticationMode = config.AuthenticationModeNone
	}
	return d
}
