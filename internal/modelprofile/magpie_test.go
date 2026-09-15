package modelprofile

import (
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestMagpieTTSQualifiedContract(t *testing.T) {
	c, err := Resolve(MagpieTTS, compatibility.NeMoSpeechV1, compatibility.Speech)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Capabilities.ServerLoadedModel || !c.Capabilities.VoiceDiscovery || !c.Capabilities.SpeechLanguage || c.Capabilities.SpeechSpeed || c.Capabilities.SpeechInstructions || len(c.Languages) != 9 || len(c.Voices) != 5 {
		t.Fatalf("incorrect Magpie capabilities: %+v", c)
	}
	for _, language := range c.Languages {
		if err := ValidateSpeechOptions(MagpieTTS, compatibility.NeMoSpeechV1, "default", SpeechOptions{Language: language.Code}); err != nil {
			t.Fatal(language, err)
		}
	}
	for _, voice := range []string{"", "default", "John", "magpietts.John", "0"} {
		if err := ValidateSpeechOptions(MagpieTTS, compatibility.NeMoSpeechV1, voice, SpeechOptions{}); err != nil {
			t.Fatal(voice, err)
		}
	}
	for _, options := range []SpeechOptions{{Language: "auto"}, {Language: "ar"}, {Language: "ko"}, {Language: "pt"}, {Language: "en-GB"}, {Instructions: "Cheerfully"}} {
		if ValidateSpeechOptions(MagpieTTS, compatibility.NeMoSpeechV1, "default", options) == nil {
			t.Fatalf("unsupported options admitted: %+v", options)
		}
	}
	for _, voice := range []string{"bad\nvoice", strings.Repeat("v", 201)} {
		if ValidateSpeechOptions(MagpieTTS, compatibility.NeMoSpeechV1, voice, SpeechOptions{}) == nil {
			t.Fatal("unsafe voice admitted")
		}
	}
	if ValidateSpeech(MagpieTTS, compatibility.NeMoSpeechV1, 1.25) == nil {
		t.Fatal("unsupported speed admitted")
	}
	if _, err := Resolve(MagpieTTS, compatibility.Generic, compatibility.Speech); err == nil {
		t.Fatal("wrong backend admitted")
	}
	if _, err := Resolve(MagpieTTS, compatibility.NeMoSpeechV1, compatibility.Transcription); err == nil {
		t.Fatal("wrong role admitted")
	}
	c.Languages[0].Code = "bad"
	next, _ := Resolve(MagpieTTS, compatibility.NeMoSpeechV1, compatibility.Speech)
	if next.Languages[0].Code != "en-US" {
		t.Fatal("mutable profile inventory")
	}
}
