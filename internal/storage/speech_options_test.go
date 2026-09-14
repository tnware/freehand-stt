package storage

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"reflect"
	"testing"
)

func TestSpeechOptionsRoundTripAndRollback(t *testing.T) {
	s := testStore(t)
	got := loadStore(t, s)
	d := savedconnection.Extract(got, savedconnection.Speech)
	d.CompatibilityProfile = compatibility.VLLMOmni
	d.BaseURL = "https://speech.example.test/v1"
	got = createSelectedConnection(t, s, got, "Speech", savedconnection.Speech, d)
	got.TextToSpeech.ModelProfile = modelprofile.Qwen3TTS
	got.TextToSpeech.Model = "customvoice-alias"
	got.TextToSpeech.Voice = "ryan"
	got.TextToSpeech.Options = modelprofile.SpeechOptions{Language: "ja", Instructions: "穏やかに話す"}
	if err := s.Save(got); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if !reflect.DeepEqual(loadStore(t, s), got) {
		t.Fatal("speech options did not survive restart")
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_speech_options BEFORE UPDATE ON speech_settings BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	changed := got
	changed.TextToSpeech.Options.Instructions = "A different delivery"
	if err := s.Save(changed); err == nil {
		t.Fatal("failed transaction accepted")
	}
	s = reopen(t, s)
	if !reflect.DeepEqual(loadStore(t, s), got) {
		t.Fatal("failed transaction changed speech options")
	}
}
