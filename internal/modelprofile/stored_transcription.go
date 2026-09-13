package modelprofile

import (
	"errors"
	"math"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

// ValidateStoredTranscription validates a qualified model/backend and bounded
// durable task preferences, not whether that model can execute those preferences.
// Migration may change the selected model without changing task language/options.
// Request admission must use ValidateTranscription after resolving the selection.
func ValidateStoredTranscription(id ID, backend compatibility.ID, language string, options compatibility.TranscriptionOptions) error {
	if _, err := Resolve(id, backend, compatibility.Transcription); err != nil {
		return err
	}
	if err := speechlanguage.Validate(language); err != nil {
		return err
	}
	validHint := func(value string, maximum int) bool {
		if !utf8.ValidString(value) || len(value) > maximum {
			return false
		}
		for _, r := range value {
			if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
				return false
			}
		}
		return true
	}
	// Keep the transport's text and numeric bounds without asserting capability
	// support or substituting a different backend to validate retained preferences.
	if !validHint(options.Prompt, compatibility.MaxTranscriptionPromptBytes) {
		return errors.New("transcription context must be valid text of at most 8192 UTF-8 bytes without control characters other than tabs and newlines")
	}
	if !validHint(options.Hotwords, compatibility.MaxTranscriptionHotwordsBytes) {
		return errors.New("transcription hotwords must be valid text of at most 2048 UTF-8 bytes without control characters other than tabs and newlines")
	}
	if math.IsNaN(options.Temperature) || math.IsInf(options.Temperature, 0) || options.Temperature < 0 || options.Temperature > 1 {
		return errors.New("transcription temperature must be a finite number between 0 and 1")
	}
	return ValidateNemotronOptions(NemotronOptions{Vocabulary: options.Vocabulary, Boost: options.VocabularyBoost})
}
