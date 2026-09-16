package modelprofile

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"testing"
)

func TestQwenContractByMode(t *testing.T) {
	completed, err := Resolve(Qwen3ASR, compatibility.VLLM, compatibility.Transcription)
	if err != nil || !completed.Capabilities.Realtime || !completed.Capabilities.TranscriptionPrompt || !completed.Capabilities.FileStreaming {
		t.Fatal("missing completed contract", err)
	}
	live, err := Resolve(Qwen3ASR, compatibility.VLLM, compatibility.Realtime)
	if err != nil || live.Capabilities != (compatibility.Capabilities{Realtime: true}) || completed.RealtimeLanguageHint {
		t.Fatal("invented realtime controls", err)
	}
	if _, err := Resolve(Generic, compatibility.VLLM, compatibility.Realtime); err == nil {
		t.Fatal("Generic gained realtime")
	}
	if _, err := Resolve(Qwen3ASR, compatibility.Generic, compatibility.Transcription); err == nil {
		t.Fatal("unqualified backend accepted")
	}
	for _, code := range []string{"", "auto", "en", "zh", "ja", "ro"} {
		if err := ValidateTranscription(Qwen3ASR, compatibility.VLLM, code, compatibility.TranscriptionOptions{Prompt: "Freehand", TemperatureOverride: true}); err != nil {
			t.Fatal(code, err)
		}
	}
	for _, code := range []string{"en-US", "yue", "fil", "tl", "uk"} {
		if ValidateTranscription(Qwen3ASR, compatibility.VLLM, code, compatibility.TranscriptionOptions{}) == nil {
			t.Fatal("unsupported forced language", code)
		}
	}
	if VocabularyMode(Qwen3ASR, compatibility.VLLM, false) != "prompt" || VocabularyMode(Qwen3ASR, compatibility.VLLM, true) != "" {
		t.Fatal("vocabulary mode mismatch")
	}
	if ValidateTranscription(Qwen3ASR, compatibility.VLLM, "en", compatibility.TranscriptionOptions{Hotwords: "term"}) == nil {
		t.Fatal("unsupported hotwords")
	}
}

func TestQwenSegmentHeadersAndFragmentedPreview(t *testing.T) {
	for _, tc := range []struct {
		raw, want, language string
		final               bool
	}{
		{"language English<asr_text>Hello.language English<asr_text>World.", "Hello. World.", "en", true},
		{"language Japanese<asr_text>こんにちは", "こんにちは", "ja", true},
		{"language Cantonese<asr_text>你好", "你好", "yue", true},
		{"language English<asr_te", "", "", false},
		{"language English<asr_text>Hello. language Jap", "Hello.", "en", false},
		{"The language English is useful.", "The language English is useful.", "", true},
		{"ordinary final", "ordinary final", "", true},
	} {
		text, language := QwenRealtimeText(tc.raw, tc.final)
		if text != tc.want || language != tc.language {
			t.Errorf("%q: %q/%q", tc.raw, text, language)
		}
	}
}
