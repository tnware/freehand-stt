package storage

import (
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"testing"
)

func TestManagedInstanceConnectionRoundTrip(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	if len(v.ManagedRuntimes) != 0 {
		t.Fatal("fresh inventory must be empty")
	}
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	d := savedconnection.Details{ManagedInstanceID: "local"}
	var err error
	v, err = s.BeginConnectionChange(savedconnection.Change{Action: savedconnection.Create, Name: "Local speech", Details: &d, Uses: []savedconnection.Purpose{savedconnection.Voice, savedconnection.Transcription}, ActivateFor: savedconnection.Voice}, v)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.StageConnectionCredential("secret", false); err == nil {
		t.Fatal("managed credential accepted")
	}
	if err = s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	got := loadStore(t, s)
	if got.VoiceTranscription.ManagedInstanceID != "local" || got.VoiceTranscription.Model != "nemotron-3.5" || got.VoiceTranscription.BaseURL != "" || got.ManagedInstanceID != "" {
		t.Fatalf("bad independent projection: %#v", got)
	}
	got.ManagedRuntimes = nil
	if err = s.Save(got); err == nil {
		t.Fatal("referenced runtime deleted")
	}
	if len(loadStore(t, s).ManagedRuntimes) != 1 {
		t.Fatal("failed save changed inventory")
	}
}
