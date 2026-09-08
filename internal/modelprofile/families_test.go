package modelprofile

import (
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestSpeechFamilyAdmission(t *testing.T) {
	for _, fixture := range []struct {
		id       ID
		backend  compatibility.ID
		realtime bool
	}{
		{ParakeetTDT, compatibility.NeMoSpeechV1, false},
		{CohereTranscribe, compatibility.VLLM, false},
		{VoxtralRealtime, compatibility.VLLM, true},
	} {
		t.Run(string(fixture.id), func(t *testing.T) {
			c, err := Resolve(fixture.id, fixture.backend, compatibility.Transcription)
			if err != nil || c.Capabilities.Realtime != fixture.realtime || c.Capabilities.TranscriptionPrompt || c.Capabilities.TranscriptionHotwords {
				t.Fatalf("wrong contract: %#v %v", c, err)
			}
			if _, err := Resolve(fixture.id, compatibility.Generic, compatibility.Transcription); err == nil {
				t.Fatal("wrong backend admitted")
			}
			if _, err := Resolve(fixture.id, fixture.backend, compatibility.Realtime); (err == nil) != fixture.realtime {
				t.Fatal("wrong realtime admission")
			}
			if VocabularyMode(fixture.id, fixture.backend, false) != "" {
				t.Fatal("unsupported vocabulary exposed")
			}
			if ValidateTranscription(fixture.id, fixture.backend, "auto", compatibility.TranscriptionOptions{Prompt: "unsupported"}) == nil {
				t.Fatal("unsupported prompt admitted")
			}
			if ValidateTranscription(fixture.id, fixture.backend, "xx", compatibility.TranscriptionOptions{}) == nil {
				t.Fatal("unsupported language admitted")
			}
		})
	}
	if err := ValidateLanguage(CohereTranscribe, compatibility.Transcription, "ja", nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLanguage(ParakeetTDT, compatibility.Transcription, "en", nil); err == nil {
		t.Fatal("Parakeet language hint admitted")
	}
}

func TestQwenSpeechControlsRequireBothContracts(t *testing.T) {
	o := SpeechOptions{Language: "ja", Instructions: "Warm delivery. 日本語"}
	if err := ValidateSpeechOptions(Qwen3TTS, compatibility.VLLMOmni, "ryan", o); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		id      ID
		backend compatibility.ID
		voice   string
		options SpeechOptions
	}{
		{Generic, compatibility.VLLMOmni, "ryan", o},
		{Qwen3TTS, compatibility.Generic, "ryan", o},
		{Qwen3TTS, compatibility.VLLMOmni, "uploaded-voice", o},
		{Qwen3TTS, compatibility.VLLMOmni, "ryan|aiden", o},
		{Qwen3TTS, compatibility.VLLMOmni, "ryan", SpeechOptions{Language: "xx"}},
		{Qwen3TTS, compatibility.VLLMOmni, "ryan", SpeechOptions{Instructions: strings.Repeat("あ", 700)}},
		{Qwen3TTS, compatibility.VLLMOmni, "ryan", SpeechOptions{Instructions: strings.Repeat("a", 501)}},
		{Qwen3TTS, compatibility.VLLMOmni, "ryan", SpeechOptions{Instructions: "bad\x00text"}},
	} {
		if ValidateSpeechOptions(fixture.id, fixture.backend, fixture.voice, fixture.options) == nil {
			t.Fatal("invalid speech options accepted")
		}
	}
	c, _ := Resolve(Qwen3TTS, compatibility.VLLMOmni, compatibility.Speech)
	c.Voices[0] = "changed"
	next, _ := Resolve(Qwen3TTS, compatibility.VLLMOmni, compatibility.Speech)
	if next.Voices[0] == "changed" {
		t.Fatal("profile metadata aliases state")
	}
}
