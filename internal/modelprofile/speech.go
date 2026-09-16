package modelprofile

import (
	"errors"
	"slices"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

type SpeechOptions struct {
	Language     string `json:"language"`
	Instructions string `json:"instructions"`
}

func ValidateSpeechOptions(id ID, backend compatibility.ID, voice string, options SpeechOptions) error {
	c, err := Resolve(id, backend, compatibility.Speech)
	if err != nil {
		return err
	}
	if options.Instructions != "" && !c.Capabilities.SpeechInstructions {
		return errors.New("style instructions are unavailable for this speech profile")
	}
	if options.Language != "" && !c.Capabilities.SpeechLanguage {
		return errors.New("language selection is unavailable for this speech profile")
	}
	if !utf8.ValidString(options.Instructions) || utf8.RuneCountInString(options.Instructions) > 500 {
		return errors.New("speech instructions must be valid text of at most 500 characters")
	}
	for _, r := range options.Instructions {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return errors.New("speech instructions contain unsupported control characters")
		}
	}
	if id == MagpieTTS {
		return validateMagpieOptions(voice, options)
	}
	if id == Qwen3TTS {
		if SpeechLanguageName(options.Language) == "" {
			return errors.New("choose one of the ten Qwen3-TTS languages")
		}
		// CustomVoice uses the checkpoint's trained speaker IDs. Uploaded voices
		// invoke a different Base model task and are not this profile's contract.
		if voice != "" && !slices.Contains(c.Voices, voice) {
			return errors.New("choose a Qwen3-TTS CustomVoice speaker")
		}
	}
	return nil
}
