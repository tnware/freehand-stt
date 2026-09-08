package config

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"testing"
)

func TestRealtimeRequiresBackendAndExplicitModelProfile(t *testing.T) {
	v := DefaultVoiceTranscription()
	v.BaseURL = "https://voice.example.test/v1"
	v.Model = "arbitrary-alias"
	v.Realtime = true
	if ValidateVoiceTranscription(v) == nil || VoiceRealtimeEligible(v) {
		t.Fatal("generic unexpectedly supports realtime")
	}
	v.CompatibilityProfile = compatibility.NeMoSpeechV1
	if ValidateVoiceTranscription(v) == nil || VoiceRealtimeEligible(v) {
		t.Fatal("backend inferred a model profile")
	}
	v.ModelProfile = modelprofile.Nemotron35
	if err := ValidateVoiceTranscription(v); err != nil {
		t.Fatal(err)
	}
	if !VoiceRealtimeEligible(v) {
		t.Fatal("qualified pair is unavailable")
	}
	v.Realtime = false
	if err := ValidateVoiceRecording(v); err != nil {
		t.Fatal("same selection cannot use completed transcription", err)
	}
	v.CompatibilityProfile = compatibility.Generic
	if ValidateVoiceTranscription(v) == nil {
		t.Fatal("specialized profile accepted on wrong backend")
	}
}
