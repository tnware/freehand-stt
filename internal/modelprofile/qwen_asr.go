package modelprofile

import (
	"errors"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

const Qwen3ASR ID = "qwen3-asr"

// Model-card language names also identify structured realtime output headers.
var qwenLanguages = []speechlanguage.Option{
	{Code: "auto", Label: "Automatic"},
	{Code: "zh", Label: "Chinese"}, {Code: "en", Label: "English"},
	{Code: "yue", Label: "Cantonese"}, {Code: "ar", Label: "Arabic"},
	{Code: "de", Label: "German"}, {Code: "fr", Label: "French"},
	{Code: "es", Label: "Spanish"}, {Code: "pt", Label: "Portuguese"},
	{Code: "id", Label: "Indonesian"}, {Code: "it", Label: "Italian"},
	{Code: "ko", Label: "Korean"}, {Code: "ru", Label: "Russian"},
	{Code: "th", Label: "Thai"}, {Code: "vi", Label: "Vietnamese"},
	{Code: "ja", Label: "Japanese"}, {Code: "tr", Label: "Turkish"},
	{Code: "hi", Label: "Hindi"}, {Code: "ms", Label: "Malay"},
	{Code: "nl", Label: "Dutch"}, {Code: "sv", Label: "Swedish"},
	{Code: "da", Label: "Danish"}, {Code: "fi", Label: "Finnish"},
	{Code: "pl", Label: "Polish"}, {Code: "cs", Label: "Czech"},
	{Code: "fil", Label: "Filipino"}, {Code: "fa", Label: "Persian"},
	{Code: "el", Label: "Greek"}, {Code: "hu", Label: "Hungarian"},
	{Code: "mk", Label: "Macedonian"}, {Code: "ro", Label: "Romanian"},
}

func qwenProfile(role compatibility.Role) Profile {
	p := Profile{ID: Qwen3ASR, Name: "Qwen3-ASR", Description: "Qwen3-ASR on vLLM 0.28.0. Completed audio supports language and context hints. Realtime detects language automatically and supports live results and captions.", Capabilities: compatibility.Capabilities{Realtime: true}}
	// vLLM's Whisper-derived map lacks Cantonese and Filipino. Do not invent
	// a Tagalog alias that forces an unqualified model-language prefix.
	for _, language := range qwenLanguages {
		if language.Code != "fil" && language.Code != "yue" {
			p.Languages = append(p.Languages, language)
		}
	}
	if role == compatibility.Transcription {
		p.Capabilities.LanguageHint = true
		p.Capabilities.TranscriptionPrompt = true
		p.Capabilities.TranscriptionTemperature = true
		p.Capabilities.FileStreaming = true
	}
	return p
}

func validateQwenLanguage(language string) error {
	if speechlanguage.Unspecified(language) {
		return nil
	}
	for _, option := range qwenProfile(compatibility.Transcription).Languages {
		if option.Code == language {
			return nil
		}
	}
	return errors.New("choose a language supported by Qwen3-ASR on vLLM")
}

// QwenRealtimeText removes only recognized structured language headers. vLLM
// concatenates independently decoded segments; headers can span delta events.
// A possible trailing header stays provisional until it is completed or disproved.
func QwenRealtimeText(raw string, final bool) (string, string) {
	language := ""
	mixed := false
	for _, option := range qwenLanguages[1:] {
		header := "language " + option.Label + "<asr_text>"
		if strings.Contains(raw, header) {
			if language != "" && language != option.Code {
				mixed = true
			}
			language = option.Code
			raw = strings.ReplaceAll(raw, header, " ")
		}
		if !final {
			for n := len(header) - 1; n > 0; n-- {
				if strings.HasSuffix(raw, header[:n]) {
					raw = strings.TrimSuffix(raw, header[:n])
					break
				}
			}
		}
	}
	if mixed {
		language = "multilingual"
	}
	return strings.TrimSpace(raw), language
}
