package modelprofile

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

const Qwen3TTS ID = "qwen3-tts-customvoice"

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
