package modelprofile

import (
	"errors"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

const Nemotron35 ID = "nemotron-3.5-streaming"

const (
	NemotronVocabularyBytes = 2048
	NemotronPhraseBytes     = 128
	NemotronPhraseCount     = 32
)

// NemotronOptions are request-level recognition hints, never instructions to
// rewrite a transcript. Phrases are newline-separated to preserve spaces.
type NemotronOptions struct {
	Vocabulary string  `json:"vocabulary"`
	Boost      float64 `json:"boost"`
}

// Excludes the eight adaptation-only locales that require model training.
var nemotronLocales = []string{"auto", "en-US", "en-GB", "es-US", "es-ES", "fr-FR", "fr-CA", "it-IT", "pt-BR", "pt-PT", "nl-NL", "de-DE", "tr-TR", "ru-RU", "ar-AR", "hi-IN", "ja-JP", "ko-KR", "vi-VN", "uk-UA", "pl-PL", "sv-SE", "cs-CZ", "nb-NO", "da-DK", "bg-BG", "fi-FI", "hr-HR", "sk-SK", "zh-CN", "hu-HU", "ro-RO", "et-EE"}

func NemotronLanguages() []speechlanguage.Option {
	out := make([]speechlanguage.Option, 0, len(nemotronLocales))
	for _, code := range nemotronLocales {
		label := code
		for _, option := range speechlanguage.Options() {
			if option.Code == strings.Split(code, "-")[0] {
				label = option.Label + " (" + code + ")"
				break
			}
		}
		if code == "auto" {
			label = "Automatic"
		}
		out = append(out, speechlanguage.Option{Code: code, Label: label})
	}
	return out
}

func ValidateNemotron(language string, options NemotronOptions) error {
	if !slices.Contains(nemotronLocales, language) {
		return errors.New("choose a language supported by the Nemotron base model")
	}
	if !utf8.ValidString(options.Vocabulary) || len(options.Vocabulary) > NemotronVocabularyBytes {
		return errors.New("vocabulary must be at most 2048 UTF-8 bytes")
	}
	if math.IsNaN(options.Boost) || math.IsInf(options.Boost, 0) || options.Boost < 0 || options.Boost > 5 {
		return errors.New("vocabulary boost must be between 0 and 5")
	}
	phrases := 0
	for _, line := range strings.Split(options.Vocabulary, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		phrases++
		if len(line) > NemotronPhraseBytes {
			return errors.New("each vocabulary phrase must be at most 128 bytes")
		}
		for _, r := range line {
			if unicode.IsControl(r) {
				return errors.New("vocabulary phrases cannot contain control characters")
			}
		}
	}
	if phrases > NemotronPhraseCount {
		return errors.New("use at most 32 vocabulary phrases")
	}
	return nil
}

// StripNemotronLanguageTag removes only the model's terminal language token.
// Ordinary text and angle brackets remain untouched.
func StripNemotronLanguageTag(text string) (string, string) {
	text = strings.TrimSpace(text)
	for _, code := range nemotronLocales[1:] {
		if strings.HasSuffix(text, "<"+code+">") {
			return strings.TrimSpace(strings.TrimSuffix(text, "<"+code+">")), code
		}
	}
	return text, ""
}
