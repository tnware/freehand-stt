package storage

import (
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"testing"
)

// Transport and credential metadata belong to reusable connections, never tasks.
func TestCurrentSchemaHasSinglePersistenceOwner(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	forbidden := map[string][]string{
		"transcription_settings":       {"base_url", "compatibility_profile", "authentication_mode", "allow_insecure_http", "health_path", "transcription_options_hotwords"},
		"voice_transcription_settings": {"base_url", "compatibility_profile", "authentication_mode", "allow_insecure_http", "health_path", "hotwords", "vocabulary", "boost"},
		"cleanup_settings":             {"base_url", "compatibility_profile", "allow_insecure_http"},
		"speech_settings":              {"base_url", "compatibility_profile", "authentication_mode", "allow_insecure_http"},
		"remembered_models":            {"language", "hotwords", "vocabulary", "boost", "system_prompt", "styling", "structure", "context", "speed"},
	}
	for table, columns := range forbidden {
		for _, column := range columns {
			var count int
			if err := s.db.QueryRow(`SELECT count(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("%s duplicates %s ownership", table, column)
			}
		}
	}
	for _, table := range []string{"request_headers", "voice_request_headers", "credential_refs", "realtime_settings"} {
		var count int
		if err := s.db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("obsolete projection table %s remains", table)
		}
	}
}

func TestCurrentSchemaRejectsUnsupportedRememberedUse(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	d := savedconnection.Extract(v, savedconnection.Transcription)
	d.BaseURL = "https://schema.example.test/v1"
	createSelectedConnection(t, s, v, "Files", savedconnection.Transcription, d)
	id := s.ConnectionCatalog().Selected[savedconnection.Transcription]
	if _, err := s.db.Exec(`INSERT INTO remembered_models(connection_id,purpose,model,selected) VALUES(?, 'speech','wrong-use',0)`, id); err == nil {
		t.Fatal("remembered model accepted an unsupported connection use")
	}
}
