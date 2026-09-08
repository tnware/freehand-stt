package storage

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"reflect"
	"testing"
)

func TestVersionFiveModelProfileUpgradePreservesCleanupAndConnections(t *testing.T) {
	s := testStore(t)
	initial := config.Default()
	initial.BaseURL = "https://speech.example.test/v1"
	initial.Model = "server-model"
	initial.PostProcessing.BaseURL = "https://cleanup.example.test/v1"
	initial.PostProcessing.Model = "arbitrary-cleanup-alias"
	initial.PostProcessing.Preset = config.PostProcessingPresetS1Mini
	initial.PostProcessing.Styling = "formal"
	writeLegacy(t, s, initial)
	before := loadStore(t, s)
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("upgrade-test-key"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(before); err != nil {
		t.Fatal(err)
	}
	catalog := s.ConnectionCatalog()
	if _, err := s.db.Exec(`ALTER TABLE speech_settings DROP COLUMN speech_language; ALTER TABLE speech_settings DROP COLUMN speech_instructions; ALTER TABLE remembered_models DROP COLUMN speech_language; ALTER TABLE remembered_models DROP COLUMN speech_instructions; DROP TABLE vocabulary_settings; DROP TABLE voice_transcription_settings; DROP TABLE voice_request_headers; DELETE FROM selected_connections WHERE purpose='voice'; DELETE FROM saved_connection_uses WHERE purpose='voice'; DELETE FROM credential_refs WHERE purpose='voice'; DROP TABLE remembered_models; ALTER TABLE transcription_settings DROP COLUMN model_profile; ALTER TABLE speech_settings DROP COLUMN model_profile; DELETE FROM goose_db_version WHERE version_id>=6;`); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	after := loadStore(t, s)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(catalog, s.ConnectionCatalog()) {
		t.Fatal("upgrade changed existing choices")
	}
	if after.ModelProfile != modelprofile.Generic || after.TextToSpeech.ModelProfile != modelprofile.Generic || after.PostProcessing.Preset != config.PostProcessingPresetS1Mini {
		t.Fatal("model profile defaults or cleanup preset lost")
	}
	if key, err := s.STTCredentials().Get(); err != nil || key != "upgrade-test-key" {
		t.Fatal("upgrade changed credential reference")
	}
	after.PostProcessing.Preset = config.PostProcessingPresetGeneric
	if err := s.Save(after); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, after) {
		t.Fatal("profile selection did not survive restart")
	}
	// Corrupt/unknown IDs trigger recovery instead of silently choosing Generic.
	if _, err := s.db.Exec(`UPDATE transcription_settings SET model_profile='future' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if _, err := s.Load(); err == nil {
		t.Fatal("unknown stored model profile accepted")
	}
}
