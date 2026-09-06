package modelprofile

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"testing"
)

func TestProfilesSeparateModelBehaviorFromBackend(t *testing.T) {
	catalog := Profiles(compatibility.WhisperCPP, compatibility.LlamaCPP, compatibility.Speaches)
	if len(catalog.Transcription) != 1 || len(catalog.PostProcessing) != 2 || len(catalog.Speech) != 1 {
		t.Fatal("unexpected qualified catalog")
	}
	stt := catalog.Transcription[0]
	if stt.ID != Generic || !stt.Capabilities.ServerLoadedModel || stt.Capabilities.FileStreaming || stt.Capabilities.TranscriptionHotwords {
		t.Fatal("Generic model widened whisper.cpp's contract")
	}
	s1 := catalog.PostProcessing[1]
	if s1.ID != S1Mini || s1.Language != "en" || s1.AssumeUnknownLanguage != "en" || !s1.ReasoningOffRequired || !s1.Capabilities.CleanupDisableReasoning {
		t.Fatal("S1-mini requirements missing")
	}
	genericServer, err := Resolve(S1Mini, compatibility.Generic, compatibility.PostProcessing)
	if err != nil || !genericServer.ReasoningOffRequired || genericServer.Capabilities.CleanupDisableReasoning {
		t.Fatal("Generic server gained an unsupported reasoning override")
	}
	if _, err = Resolve(S1Mini, compatibility.Speaches, compatibility.Transcription); err == nil {
		t.Fatal("cleanup profile accepted for STT")
	}
	if _, err = Resolve(S1Mini, compatibility.Speaches, compatibility.Speech); err == nil {
		t.Fatal("cleanup profile accepted for speech")
	}
	if _, err = Resolve(ID("model-name-or-future-profile"), compatibility.Generic, compatibility.PostProcessing); err == nil {
		t.Fatal("unqualified model profile accepted")
	}
}

func TestCapabilitiesRequireBothContracts(t *testing.T) {
	backend := compatibility.Capabilities{ServerLoadedModel: true, TypedTranscriptionEvents: true, FileStreaming: true, LanguageHint: true, TranscriptionPrompt: true, TranscriptionHotwords: true, TranscriptionTemperature: true, CleanupOutputLimit: true, CleanupDisableReasoning: true, SpeechSpeed: true}
	restricted := constrain(backend, compatibility.Capabilities{})
	if !restricted.ServerLoadedModel || !restricted.TypedTranscriptionEvents {
		t.Fatal("model changed transport facts")
	}
	restricted.ServerLoadedModel = false
	restricted.TypedTranscriptionEvents = false
	if restricted != (compatibility.Capabilities{}) {
		t.Fatal("model restrictions not enforced")
	}
	if got := constrain(compatibility.Capabilities{}, backend); got != (compatibility.Capabilities{}) {
		t.Fatal("model enabled fields absent from backend")
	}
}

func TestLanguageRequirementsAndIndependentCatalogCopies(t *testing.T) {
	for _, selected := range []string{"", "auto", "en", "en-US"} {
		if err := ValidateLanguage(S1Mini, compatibility.PostProcessing, selected, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, selected := range []string{"ja", "fr"} {
		if ValidateLanguage(S1Mini, compatibility.PostProcessing, selected, nil) == nil {
			t.Fatal("accepted non-English input")
		}
	}
	if ValidateLanguage(S1Mini, compatibility.PostProcessing, "", []string{"en", "Japanese"}) == nil {
		t.Fatal("ignored detected-language mismatch")
	}
	if err := ValidateLanguage(Generic, compatibility.Transcription, "ja", nil); err != nil {
		t.Fatal(err)
	}
	catalog := Profiles(compatibility.Generic, compatibility.Generic, compatibility.Generic)
	catalog.PostProcessing[1].Name = "modified"
	if Profiles(compatibility.Generic, compatibility.Generic, compatibility.Generic).PostProcessing[1].Name == "modified" {
		t.Fatal("catalog aliases mutable state")
	}
}
