package config

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestUnavailableCompatibilityProfilesFailEvenWhenFeatureDisabled(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.LocalAI, compatibility.ID("future-profile")} {
		for _, role := range []string{"stt", "processing", "speech"} {
			settings := Default()
			switch role {
			case "stt":
				settings.CompatibilityProfile = id
			case "processing":
				settings.PostProcessing.CompatibilityProfile = id
			case "speech":
				settings.TextToSpeech.CompatibilityProfile = id
			}
			if Validate(settings) == nil {
				t.Fatalf("accepted %s/%s", role, id)
			}
		}
	}
	settings := Default()
	settings.CompatibilityProfile = compatibility.LlamaCPP
	if Validate(settings) == nil {
		t.Fatal("chat profile accepted for STT")
	}
}

func TestWhisperCPPDoesNotRequireAClientModel(t *testing.T) {
	s := DefaultVoiceTranscription()
	s.BaseURL = "http://127.0.0.1:8081"
	s.AllowInsecureHTTP = true
	s.AuthenticationMode = AuthenticationModeNone
	s.CompatibilityProfile = compatibility.WhisperCPP
	s.Model = ""
	if err := ValidateVoiceRecording(s); err != nil {
		t.Fatal(err)
	}
	s.CompatibilityProfile = compatibility.VLLM
	if err := ValidateVoiceRecording(s); err == nil {
		t.Fatal("vLLM accepted missing model")
	}
}
