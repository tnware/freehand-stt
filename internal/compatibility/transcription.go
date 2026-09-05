package compatibility

import (
	"errors"
	"math"
	"unicode"
	"unicode/utf8"
)

const (
	MaxTranscriptionPromptBytes   = 8192
	MaxTranscriptionHotwordsBytes = 2048
)

// TranscriptionOptions is a value-only snapshot of optional request controls.
// TemperatureOverride distinguishes an explicit zero from the server default.
type TranscriptionOptions struct {
	Prompt              string  `json:"prompt"`
	Hotwords            string  `json:"hotwords"`
	TemperatureOverride bool    `json:"temperatureOverride"`
	Temperature         float64 `json:"temperature"`
}

// ValidateTranscriptionOptions is shared by persisted settings and the transport.
// Errors describe the field, never the user-provided contents.
func ValidateTranscriptionOptions(id ID, options TranscriptionOptions) error {
	contract, err := Resolve(id, Transcription)
	if err != nil {
		return err
	}
	if !validHint(options.Prompt, MaxTranscriptionPromptBytes) {
		return errors.New("transcription context must be valid text of at most 8192 UTF-8 bytes without control characters other than tabs and newlines")
	}
	if !validHint(options.Hotwords, MaxTranscriptionHotwordsBytes) {
		return errors.New("transcription hotwords must be valid text of at most 2048 UTF-8 bytes without control characters other than tabs and newlines")
	}
	if options.Prompt != "" && !contract.Capabilities.TranscriptionPrompt {
		return errors.New("transcription context is unavailable for this profile")
	}
	if options.Hotwords != "" && !contract.Capabilities.TranscriptionHotwords {
		return errors.New("transcription hotwords require the Speaches profile; clear hotwords before choosing another profile")
	}
	if math.IsNaN(options.Temperature) || math.IsInf(options.Temperature, 0) || options.Temperature < 0 || options.Temperature > 1 {
		return errors.New("transcription temperature must be a finite number between 0 and 1")
	}
	if options.TemperatureOverride && !contract.Capabilities.TranscriptionTemperature {
		return errors.New("transcription temperature is unavailable for this profile")
	}
	return nil
}

func validHint(value string, maximum int) bool {
	if len(value) > maximum || !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) && char != '\n' && char != '\r' && char != '\t' {
			return false
		}
	}
	return true
}
