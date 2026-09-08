package config

import (
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestVocabularyUsesQualifiedFieldsAndImmutableWorkflowChoices(t *testing.T) {
	for _, mode := range []string{"prompt", "hotwords", "completed-nemo", "realtime-nemo", "unsupported"} {
		t.Run(mode, func(t *testing.T) {
			v := Default()
			v.Vocabulary = VocabularySettings{Terms: " Freehand \n東京\nFreehand", Voice: true, Boost: 2.5}
			switch mode {
			case "prompt":
				v.VoiceTranscription.TranscriptionOptions.Prompt = "Project context"
			case "hotwords":
				v.VoiceTranscription.CompatibilityProfile = compatibility.Speaches
			default:
				v.VoiceTranscription.CompatibilityProfile = compatibility.NeMoSpeechV1
			}
			if mode == "completed-nemo" || mode == "realtime-nemo" {
				v.VoiceTranscription.ModelProfile = modelprofile.Nemotron35
			}
			v.VoiceTranscription.Realtime = mode == "realtime-nemo"
			got, err := WithVocabulary(v, true)
			if err != nil {
				t.Fatal(err)
			}
			if got.TranscriptionOptions != v.TranscriptionOptions || v.VoiceTranscription.Options.Vocabulary != "" {
				t.Fatal("projection mutated file settings or its input")
			}
			switch mode {
			case "prompt":
				if got.VoiceTranscription.TranscriptionOptions.Prompt != "Project context\n\nFreehand\n東京" {
					t.Fatal("context lost")
				}
			case "hotwords":
				if got.VoiceTranscription.TranscriptionOptions.Hotwords != "Freehand\n東京" {
					t.Fatal("missing hotwords")
				}
			case "completed-nemo":
				if got.VoiceTranscription.TranscriptionOptions.Vocabulary != "Freehand\n東京" || got.VoiceTranscription.TranscriptionOptions.VocabularyBoost != 2.5 {
					t.Fatal("missing completed speech contexts")
				}
			case "realtime-nemo":
				if got.VoiceTranscription.Options.Vocabulary != "Freehand\n東京" || got.VoiceTranscription.Options.Boost != 2.5 {
					t.Fatal("missing realtime speech contexts")
				}
			case "unsupported":
				if got.VoiceTranscription.Options.Vocabulary != "" || got.VoiceTranscription.TranscriptionOptions != v.VoiceTranscription.TranscriptionOptions {
					t.Fatal("unsupported profile received hints")
				}
			}
			file, err := WithVocabulary(v, false)
			if err != nil || file.TranscriptionOptions != v.TranscriptionOptions {
				t.Fatal("Voice opt-in enabled file hints")
			}
		})
	}
}

func TestVocabularyLimitsAreVisibleAndNotSilentlyTruncated(t *testing.T) {
	v := Default()
	v.Vocabulary = VocabularySettings{Terms: strings.Repeat("語", 43), Voice: true, Boost: 3}
	v.VoiceTranscription.CompatibilityProfile = compatibility.NeMoSpeechV1
	v.VoiceTranscription.ModelProfile = modelprofile.Nemotron35
	p := PreviewVocabulary(v.Vocabulary, VocabularySelection{Backend: compatibility.NeMoSpeechV1, ModelProfile: modelprofile.Nemotron35})
	if p.Mode != "speech-contexts" || p.Problem == "" {
		t.Fatal("UTF-8 phrase limit missing from preview")
	}
	if _, err := WithVocabulary(v, true); err == nil {
		t.Fatal("oversized phrase admitted")
	}
	v.Vocabulary.Voice = false
	if _, err := WithVocabulary(v, true); err != nil {
		t.Fatal("disabled hints blocked transcription")
	}
	if ValidateVocabulary(VocabularySettings{Terms: "private\x00value"}) == nil {
		t.Fatal("control character accepted")
	}
}
