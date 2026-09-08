package modelprofile

import (
	"errors"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

const (
	ParakeetTDT      ID = "parakeet-tdt-v3"
	CohereTranscribe ID = "cohere-transcribe"
	VoxtralRealtime  ID = "voxtral-realtime"
)

func familyProfile(id ID, role compatibility.Role) (Profile, bool) {
	p := Profile{ID: id}
	switch id {
	case ParakeetTDT:
		if role != compatibility.Transcription {
			return p, false
		}
		p.Name = "Parakeet TDT v3"
		p.Description = "Transcribe recordings and audio files in 25 languages with automatic language detection and punctuation. NeMo-Speech.cpp uses completed audio for this model."
		p.Languages = []speechlanguage.Option{{Code: "auto", Label: "Automatic detection"}}
	case CohereTranscribe:
		if role != compatibility.Transcription {
			return p, false
		}
		p.Name = "Cohere Transcribe"
		p.Description = "Completed transcription in 14 languages. Select the spoken language; the server default is English. vLLM adds punctuation automatically."
		p.Capabilities = compatibility.Capabilities{LanguageHint: true, FileStreaming: true, TranscriptionTemperature: true}
		p.Languages = []speechlanguage.Option{
			{Code: "auto", Label: "Server default (English)"},
			{Code: "en", Label: "English"}, {Code: "fr", Label: "French"}, {Code: "de", Label: "German"},
			{Code: "it", Label: "Italian"}, {Code: "es", Label: "Spanish"}, {Code: "pt", Label: "Portuguese"},
			{Code: "el", Label: "Greek"}, {Code: "nl", Label: "Dutch"}, {Code: "pl", Label: "Polish"},
			{Code: "zh", Label: "Chinese"}, {Code: "ja", Label: "Japanese"}, {Code: "ko", Label: "Korean"},
			{Code: "vi", Label: "Vietnamese"}, {Code: "ar", Label: "Arabic"},
		}
	case VoxtralRealtime:
		if role != compatibility.Transcription && role != compatibility.Realtime {
			return p, false
		}
		p.Name = "Voxtral Mini Realtime"
		p.Description = "Voxtral Mini 4B Realtime on vLLM. Follow live results and optional overlay captions, or transcribe completed recordings with automatic language detection."
		p.Capabilities = compatibility.Capabilities{Realtime: true}
		p.Languages = []speechlanguage.Option{{Code: "auto", Label: "Automatic detection"}}
	default:
		return p, false
	}
	return p, true
}

func validateFamilyLanguage(id ID, selected string) error {
	if speechlanguage.Unspecified(selected) {
		return nil
	}
	p, ok := familyProfile(id, compatibility.Transcription)
	if !ok {
		return errors.New("unknown transcription model profile")
	}
	for _, language := range p.Languages {
		if language.Code == selected {
			return nil
		}
	}
	return errors.New("choose a language supported by the selected transcription profile")
}
