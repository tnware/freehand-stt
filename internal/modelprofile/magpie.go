package modelprofile

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

// MagpieTTS qualifies the v2602 checkpoint with NeMo-Speech.cpp v0.1.0.
// Its HTTP model inventory reports which optional tokenizers were compiled in.
const MagpieTTS ID = "magpie-tts-multilingual-357m"

func magpieTTSProfile() Profile {
	return Profile{ID: MagpieTTS, Voices: []string{"John", "Sofia", "Aria", "Jason", "Leo"}, Name: "MagpieTTS Multilingual 357M", Description: "NeMo-Speech.cpp v0.1.0 with the v2602 checkpoint. Buffered WAV, local voices, and nine model languages; Japanese and Mandarin require optional server tokenizers. Speed is fixed at 1.0.", Capabilities: compatibility.Capabilities{SpeechLanguage: true}, Languages: []speechlanguage.Option{
		{Code: "en-US", Label: "English"}, {Code: "es-ES", Label: "Spanish"}, {Code: "de-DE", Label: "German"},
		{Code: "fr-FR", Label: "French"}, {Code: "it-IT", Label: "Italian"}, {Code: "vi-VN", Label: "Vietnamese"},
		{Code: "hi-IN", Label: "Hindi"}, {Code: "zh-CN", Label: "Mandarin Chinese"}, {Code: "ja-JP", Label: "Japanese"},
	}}
}

// MagpieLanguage accepts the model's documented locale or its base-language
// alias. An empty selection means the configured server default, not detection.
func MagpieLanguage(code string) bool {
	if code == "" {
		return true
	}
	for _, language := range magpieTTSProfile().Languages {
		base, _, _ := strings.Cut(language.Code, "-")
		if code == language.Code || code == base {
			return true
		}
	}
	return false
}

func validateMagpieOptions(voice string, options SpeechOptions) error {
	if !MagpieLanguage(options.Language) {
		return errors.New("choose a supported MagpieTTS v2602 language; an empty selection uses the server default")
	}
	// Local names, model-qualified names, and numeric speaker IDs are resolved by
	// the loaded model. Discovery can report server-specific names; validation
	// preserves those bounded IDs rather than substituting hosted voice aliases.
	if !utf8.ValidString(voice) || len(voice) > 200 || strings.TrimSpace(voice) != voice {
		return errors.New("choose a valid MagpieTTS local voice")
	}
	for _, r := range voice {
		if unicode.IsControl(r) {
			return errors.New("choose a valid MagpieTTS local voice")
		}
	}
	return nil
}
