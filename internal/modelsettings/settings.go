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

// TranscriptionOptions contains model behavior, never task-owned vocabulary.
// Runtime inference options remain in compatibility.TranscriptionOptions.
type TranscriptionOptions struct {
	Prompt              string  `json:"prompt"`
	TemperatureOverride bool    `json:"temperatureOverride"`
	Temperature         float64 `json:"temperature"`
}

func transcriptionOptions(o config.TranscriptionOptions) TranscriptionOptions {
	return TranscriptionOptions{Prompt: o.Prompt, TemperatureOverride: o.TemperatureOverride, Temperature: o.Temperature}
}

func (o TranscriptionOptions) apply(v *config.TranscriptionOptions) {
	v.Prompt = o.Prompt
	v.TemperatureOverride = o.TemperatureOverride
	v.Temperature = o.Temperature
}

// Options excludes task intent, endpoints, credentials, enablement, and capture policy.
// Only fields belonging to Purpose are populated; Apply ignores all other fields.
type Options struct {
	Speech        modelprofile.SpeechOptions   `json:"speech"`
	Profile       modelprofile.ID              `json:"profile"`
	Transcription TranscriptionOptions         `json:"transcription"`
	Cleanup       compatibility.CleanupOptions `json:"cleanup"`
	Voice         string                       `json:"voice"`
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
	return map[savedconnection.Purpose]Options{savedconnection.Transcription: Extract(d, savedconnection.Transcription), savedconnection.Cleanup: Extract(d, savedconnection.Cleanup), savedconnection.Speech: Extract(d, savedconnection.Speech), savedconnection.Voice: Extract(d, savedconnection.Voice)}
}
func Model(v config.Settings, p savedconnection.Purpose) string {
	switch p {
	case savedconnection.Voice:
		return v.VoiceTranscription.Model
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
	case savedconnection.Voice:
		return Options{Profile: v.VoiceTranscription.ModelProfile, Transcription: transcriptionOptions(v.VoiceTranscription.TranscriptionOptions)}
	case savedconnection.Transcription:
		return Options{Profile: modelprofile.Effective(v.ModelProfile), Transcription: transcriptionOptions(v.TranscriptionOptions)}
	case savedconnection.Cleanup:
		c := v.PostProcessing
		return Options{Profile: modelprofile.Effective(modelprofile.ID(c.Preset)), Cleanup: c.GenerationOptions}
	case savedconnection.Speech:
		c := v.TextToSpeech
		return Options{Profile: modelprofile.Effective(c.ModelProfile), Voice: c.Voice, Speech: c.Options}
	}
	return Options{}
}
func Apply(v config.Settings, p savedconnection.Purpose, model string, o Options) config.Settings {
	switch p {
	case savedconnection.Voice:
		v.VoiceTranscription.Model = model
		v.VoiceTranscription.ModelProfile = o.Profile
		o.Transcription.apply(&v.VoiceTranscription.TranscriptionOptions)
	case savedconnection.Transcription:
		v.Model = model
		v.ModelProfile = o.Profile
		o.Transcription.apply(&v.TranscriptionOptions)
	case savedconnection.Cleanup:
		c := &v.PostProcessing
		c.Model = model
		c.Preset = config.PostProcessingPreset(o.Profile)
		c.GenerationOptions = o.Cleanup
	case savedconnection.Speech:
		c := &v.TextToSpeech
		c.Model = model
		c.ModelProfile = o.Profile
		c.Options = o.Speech
		c.Voice = o.Voice
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
	if e.Model == "" && !((e.Purpose == savedconnection.Transcription || e.Purpose == savedconnection.Voice) && d.CompatibilityProfile == compatibility.WhisperCPP) {
		return errors.New("remembered model ID is required")
	}
	v := Apply(savedconnection.Apply(config.Default(), e.Purpose, d), e.Purpose, e.Model, e.Options)
	if Extract(v, e.Purpose) != e.Options {
		return errors.New("remembered options contain fields for another feature")
	}
	switch e.Purpose {
	case savedconnection.Voice:
		return config.ValidateVoiceTranscription(v.VoiceTranscription)
	case savedconnection.Transcription:
		return config.Validate(v)
	case savedconnection.Cleanup:
		return config.ValidatePostProcessing(v.PostProcessing)
	case savedconnection.Speech:
		return config.ValidateTextToSpeech(v.TextToSpeech, false)
	}
	return errors.New("invalid remembered model purpose")
}

// Select restores model behavior and gates realtime against the selected backend/profile.
// Task intent is preserved by construction: Options cannot represent it.
func Select(v config.Settings, p savedconnection.Purpose, model string, o Options) config.Settings {
	next := Apply(v, p, model, o)
	next.VoiceTranscription.Realtime = v.VoiceTranscription.Realtime && config.VoiceRealtimeEligible(next.VoiceTranscription)
	return next
}
