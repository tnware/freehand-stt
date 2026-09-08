package config

import (
	"fmt"
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

func TestVocabularyLineFeedbackMatchesQualifiedAdmission(t *testing.T) {
	nemo := VocabularySelection{Backend: compatibility.NeMoSpeechV1, ModelProfile: modelprofile.Nemotron35}
	prompt := VocabularySelection{Backend: compatibility.Generic, ModelProfile: modelprofile.Generic, Context: strings.Repeat("x", 8187)}
	request := VocabularyPreviewRequest{Vocabulary: VocabularySettings{Terms: "  One \r\n\nOne\n" + strings.Repeat("語", 43), Boost: 3}, Voice: nemo, Files: prompt}
	result := InspectVocabulary(request)
	if result.PhraseCount != 2 || result.DuplicateCount != 1 || len(result.Issues) != 2 {
		t.Fatalf("unexpected accounting: %+v", result)
	}
	if result.Issues[0].Line != 3 || result.Issues[0].DuplicateOf != 1 {
		t.Fatal("duplicate source lines were lost")
	}
	issue := result.Issues[1]
	if issue.Line != 4 || !strings.Contains(issue.VoiceProblem, "129 bytes") || issue.FilesProblem == "" {
		t.Fatalf("missing per-use UTF-8 feedback: %+v", issue)
	}
	if result.Voice.Problem == "" || result.Files.Problem == "" {
		t.Fatal("line feedback disagrees with admission")
	}
	request.Files.Realtime = true
	request.Files.ModelProfile = modelprofile.Qwen3ASR
	request.Files.Backend = compatibility.VLLM
	if got := InspectVocabulary(request); got.Issues[1].FilesProblem != "" {
		t.Fatal("unsupported workflow received line restrictions")
	}
}

func TestVocabularyCountAndBudgetIdentifyFirstExcessLine(t *testing.T) {
	for _, mode := range []string{"count", "total"} {
		t.Run(mode, func(t *testing.T) {
			lines := []string{}
			count := 33
			if mode == "total" {
				count = 17
			}
			for i := 1; i <= count; i++ {
				line := fmt.Sprintf("phrase-%02d", i)
				if mode == "total" {
					line += strings.Repeat("x", 120-len(line))
				}
				lines = append(lines, line)
			}
			request := VocabularyPreviewRequest{Vocabulary: VocabularySettings{Terms: strings.Join(lines, "\n"), Boost: 3}, Voice: VocabularySelection{Backend: compatibility.NeMoSpeechV1, ModelProfile: modelprofile.Nemotron35}}
			result := InspectVocabulary(request)
			if len(result.Issues) != 1 || result.Issues[0].Line != count || result.Voice.Problem == "" {
				t.Fatalf("incorrect first excess line: %+v", result)
			}
			request.Vocabulary.Terms = strings.Join(lines[:count-1], "\n")
			result = InspectVocabulary(request)
			if len(result.Issues) != 0 || result.Voice.Problem != "" {
				t.Fatalf("valid boundary rejected: %+v", result)
			}
		})
	}
}

func TestVocabularyOversizeDraftHasBoundedFeedback(t *testing.T) {
	result := InspectVocabulary(VocabularyPreviewRequest{Vocabulary: VocabularySettings{Terms: strings.Repeat("x\n", 16384)}})
	if len(result.Issues) != 0 || result.PhraseCount != 0 {
		t.Fatal("oversize draft was expanded into line feedback")
	}
}
