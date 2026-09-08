package modelprofile

import (
	"errors"
	"slices"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

const Qwen3TTS ID = "qwen3-tts-customvoice"

type SpeechOptions struct {
	Language     string `json:"language"`
	Instructions string `json:"instructions"`
}

func qwenTTSProfile() Profile {
	return Profile{ID: Qwen3TTS, Voices: []string{"vivian", "serena", "uncle_fu", "dylan", "eric", "ryan", "aiden", "ono_anna", "sohee"}, Name: "Qwen3-TTS 1.7B CustomVoice", Description: "Preset speakers, ten languages, and style instructions for the 1.7B CustomVoice checkpoint on vLLM-Omni.", Capabilities: compatibility.Capabilities{SpeechSpeed: true, SpeechInstructions: true, SpeechLanguage: true}, Languages: []speechlanguage.Option{
		{Code: "auto", Label: "Automatic"}, {Code: "zh", Label: "Chinese"}, {Code: "en", Label: "English"},
		{Code: "ja", Label: "Japanese"}, {Code: "ko", Label: "Korean"}, {Code: "de", Label: "German"},
		{Code: "fr", Label: "French"}, {Code: "ru", Label: "Russian"}, {Code: "pt", Label: "Portuguese"},
		{Code: "es", Label: "Spanish"}, {Code: "it", Label: "Italian"},
	}}
}

func SpeechLanguageName(code string) string {
	if code == "" || code == "auto" {
		return "Auto"
	}
	for _, language := range qwenTTSProfile().Languages {
		if code == language.Code {
			return language.Label
		}
	}
	return ""
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
