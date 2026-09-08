package storage

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestSharedVocabularyMigrationAndRoundTrip(t *testing.T) {
	s := testStore(t)
	before := loadStore(t, s)
	// Recreate v9 active hint fields without changing old released migrations.
	if _, err := s.db.Exec(`DROP TABLE vocabulary_settings; DELETE FROM goose_db_version WHERE version_id=10;
UPDATE voice_transcription_settings SET compatibility_profile='nemo-speech-v1',model_profile='nemotron-3.5-streaming',vocabulary='Freehand',hotwords='東京',boost=2.5;
UPDATE transcription_settings SET transcription_options_hotwords='File term';`); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	got := loadStore(t, s)
	want := config.VocabularySettings{Terms: "Freehand\n東京\nFile term", Voice: true, Files: true, Boost: 2.5}
	if got.Vocabulary != want || got.TranscriptionOptions.Hotwords != "" || got.VoiceTranscription.Options.Vocabulary != "" || got.VoiceTranscription.TranscriptionOptions.Hotwords != "" {
		t.Fatal("migration lost or duplicated active hint authority")
	}
	if got.BaseURL != before.BaseURL || got.Model != before.Model {
		t.Fatal("vocabulary migration changed connection selection")
	}
	got.Vocabulary.Files = false
	got.Vocabulary.Terms = "Updated\n日本語"
	if err := s.Save(got); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if loaded := loadStore(t, s); !reflect.DeepEqual(loaded, got) {
		t.Fatal("vocabulary save did not survive restart")
	}
}
