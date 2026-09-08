package storage

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestSpeechOptionsUpgradeRoundTripAndRollback(t *testing.T) {
	s := testStore(t)
	before := loadStore(t, s)
	// Reconstruct alpha.4's v10 schema, then exercise the real startup upgrade.
	if _, err := s.db.Exec(`ALTER TABLE speech_settings DROP COLUMN speech_language; ALTER TABLE speech_settings DROP COLUMN speech_instructions; ALTER TABLE remembered_models DROP COLUMN speech_language; ALTER TABLE remembered_models DROP COLUMN speech_instructions; DELETE FROM goose_db_version WHERE version_id=11;`); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	got := loadStore(t, s)
	if !reflect.DeepEqual(got, before) {
		t.Fatal("upgrade changed existing settings")
	}
	got.TextToSpeech.CompatibilityProfile = compatibility.VLLMOmni
	got.TextToSpeech.ModelProfile = modelprofile.Qwen3TTS
	got.TextToSpeech.BaseURL = "https://speech.example.test/v1"
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
