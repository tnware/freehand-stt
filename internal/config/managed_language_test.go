package config

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"testing"
)

func TestManagedLanguageIsPerSelectedTask(t *testing.T) {
	v := Default()
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
	i, c, e := ManagedContract(v, "local", compatibility.Transcription)
	if e != nil {
		t.Fatal(e)
	}
	v.ManagedInstanceID = i.ID
	v.Model = i.Model
	v.ModelProfile = c.ModelProfile
	v.CompatibilityProfile = c.CompatibilityProfile
	v.Language = "en-US"
	v.VoiceTranscription.Language = "en"
	if e = Validate(v); e != nil {
		t.Fatal(e)
	}
	v.Language = "en"
	if Validate(v) == nil {
		t.Fatal("unsupported managed language accepted")
	}
	v.Language = "en-US"
	v.Model = "tampered"
	if Validate(v) == nil {
		t.Fatal("task model overrode runtime")
	}
}
