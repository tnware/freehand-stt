// Package modelsettings defines non-secret preferences for a connection, use, and model.
package modelsettings

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

const MaxPerUse = 32

// Options intentionally excludes endpoints, credentials, enablement, and capture policy.
// Only fields belonging to Purpose are populated; Apply ignores all other fields.
type Options struct {
	Realtime      modelprofile.NemotronOptions       `json:"realtime"`
	Profile       modelprofile.ID                    `json:"profile"`
	Language      string                             `json:"language"`
	Transcription compatibility.TranscriptionOptions `json:"transcription"`
	Cleanup       compatibility.CleanupOptions       `json:"cleanup"`
	SystemPrompt  string                             `json:"systemPrompt"`
	Styling       string                             `json:"styling"`
	Structure     string                             `json:"structure"`
	Context       string                             `json:"context"`
	Voice         string                             `json:"voice"`
	Speed         float64                            `json:"speed"`
}
type Key struct {
	ConnectionID string                  `json:"connectionID"`
	Purpose      savedconnection.Purpose `json:"purpose"`
	Model        string                  `json:"model"`
}
type Edit struct {
	ConnectionID string                  `json:"connectionID"`
	Purpose      savedconnection.Purpose `json:"purpose"`
	Model        string                  `json:"model"`
	Options      Options                 `json:"options"`
}
type Entry struct {
	ConnectionID string                  `json:"connectionID"`
	Purpose      savedconnection.Purpose `json:"purpose"`
	Model        string                  `json:"model"`
	Options      Options                 `json:"options"`
	Selected     bool                    `json:"selected"`
}
type Catalog struct {
	Entries  []Entry                             `json:"entries"`
	Defaults map[savedconnection.Purpose]Options `json:"defaults"`
}

func Defaults() map[savedconnection.Purpose]Options {
	d := config.Default()
	return map[savedconnection.Purpose]Options{savedconnection.Transcription: Extract(d, savedconnection.Transcription), savedconnection.Cleanup: Extract(d, savedconnection.Cleanup), savedconnection.Speech: Extract(d, savedconnection.Speech), savedconnection.Realtime: Extract(d, savedconnection.Realtime)}
}
func Model(v config.Settings, p savedconnection.Purpose) string {
	switch p {
	case savedconnection.Realtime:
		return v.Realtime.Model
	case savedconnection.Transcription:
		return v.Model
	case savedconnection.Cleanup:
		return v.PostProcessing.Model
	case savedconnection.Speech:
		return v.TextToSpeech.Model
	}
	return ""
}
func Extract(v config.Settings, p savedconnection.Purpose) Options {
	switch p {
	case savedconnection.Realtime:
		return Options{Profile: v.Realtime.ModelProfile, Language: v.Realtime.Language, Realtime: v.Realtime.Options}
	case savedconnection.Transcription:
		return Options{Profile: modelprofile.Effective(v.ModelProfile), Language: v.Language, Transcription: v.TranscriptionOptions}
	case savedconnection.Cleanup:
		c := v.PostProcessing
		return Options{Profile: modelprofile.Effective(modelprofile.ID(c.Preset)), Cleanup: c.GenerationOptions, SystemPrompt: c.SystemPrompt, Styling: c.Styling, Structure: c.Structure, Context: c.Context}
	case savedconnection.Speech:
		c := v.TextToSpeech
		return Options{Profile: modelprofile.Effective(c.ModelProfile), Voice: c.Voice, Speed: c.Speed}
	}
	return Options{}
}
func Apply(v config.Settings, p savedconnection.Purpose, model string, o Options) config.Settings {
	switch p {
	case savedconnection.Realtime:
		v.Realtime.Model = model
		v.Realtime.ModelProfile = o.Profile
		v.Realtime.Language = o.Language
		v.Realtime.Options = o.Realtime
	case savedconnection.Transcription:
		v.Model = model
		v.ModelProfile = o.Profile
		v.Language = o.Language
		v.TranscriptionOptions = o.Transcription
	case savedconnection.Cleanup:
		c := &v.PostProcessing
		c.Model = model
		c.Preset = config.PostProcessingPreset(o.Profile)
		c.GenerationOptions = o.Cleanup
		c.SystemPrompt = o.SystemPrompt
		c.Styling = o.Styling
		c.Structure = o.Structure
		c.Context = o.Context
	case savedconnection.Speech:
		c := &v.TextToSpeech
		c.Model = model
		c.ModelProfile = o.Profile
		c.Voice = o.Voice
		c.Speed = o.Speed
	}
	return v
}
func Validate(e Entry, d savedconnection.Details) error {
	if !savedconnection.ValidPurpose(e.Purpose) || len(e.Model) > 200 || !utf8.ValidString(e.Model) || strings.TrimSpace(e.Model) != e.Model {
		return errors.New("invalid remembered model")
	}
	for _, r := range e.Model {
		if unicode.IsControl(r) {
			return errors.New("invalid remembered model")
		}
	}
	if e.Model == "" && !(e.Purpose == savedconnection.Transcription && d.CompatibilityProfile == compatibility.WhisperCPP) {
		return errors.New("remembered model ID is required")
	}
	v := Apply(savedconnection.Apply(config.Default(), e.Purpose, d), e.Purpose, e.Model, e.Options)
	if Extract(v, e.Purpose) != e.Options {
		return errors.New("remembered options contain fields for another feature")
	}
	switch e.Purpose {
	case savedconnection.Realtime:
		return config.ValidateRealtime(v.Realtime)
	case savedconnection.Transcription:
		return config.Validate(v)
	case savedconnection.Cleanup:
		return config.ValidatePostProcessing(v.PostProcessing)
	case savedconnection.Speech:
		return config.ValidateTextToSpeech(v.TextToSpeech, false)
	}
	return errors.New("invalid remembered model purpose")
}

// Select restores engine behavior without replacing the user's current task.
// Apply remains the full snapshot decoder for validating the existing SQLite
// format; historical task fields in remembered rows are not selection authority.
func Select(v config.Settings, p savedconnection.Purpose, model string, o Options) config.Settings {
	next := Apply(v, p, model, o)
	next.Language = v.Language
	next.Realtime.Language = v.Realtime.Language
	next.PostProcessing.SystemPrompt = v.PostProcessing.SystemPrompt
	next.PostProcessing.Styling = v.PostProcessing.Styling
	next.PostProcessing.Structure = v.PostProcessing.Structure
	next.PostProcessing.Context = v.PostProcessing.Context
	next.TextToSpeech.Speed = v.TextToSpeech.Speed
	return next
}
