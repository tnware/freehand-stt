package storage

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"reflect"
	"testing"
)

func TestVersionEightUpgradeSelectsOneVoiceProvider(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed", true: "realtime"}[live], func(t *testing.T) {
			s := testStore(t)
			initial := config.Default()
			initial.BaseURL = "https://file.example.test/v1"
			initial.Model = "file-model"
			initial.Language = "de"
			initial.Headers["X-Route"] = "file"
			writeLegacy(t, s, initial)
			before := loadStore(t, s)
			// Reconstruct the retained v8 layout and configured realtime slot, then let
			// normal startup apply v9 with backup, integrity, and domain validation.
			old, err := embeddedMigrations.ReadFile("migrations/00008_realtime.sql")
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.db.Exec(`DROP TABLE vocabulary_settings; DROP TABLE voice_transcription_settings; DROP TABLE voice_request_headers;
   DELETE FROM selected_connections WHERE purpose='voice'; DELETE FROM saved_connection_uses WHERE purpose='voice'; DELETE FROM credential_refs WHERE purpose='voice'; DELETE FROM remembered_models WHERE purpose='voice';` + string(old) + `DELETE FROM goose_db_version WHERE version_id>=9;
   INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account) VALUES('live-fixture','Live fixture','nemo-speech-v1','https://live.example.test/v1',0,'none','','');
   INSERT INTO saved_connection_uses VALUES('live-fixture','realtime'); INSERT INTO selected_connections VALUES('realtime','live-fixture');
   UPDATE realtime_settings SET base_url='https://live.example.test/v1',model='live-model',language='fr-FR',vocabulary='Freehand',boost=2.5;
   INSERT INTO remembered_models(connection_id,purpose,model,selected,profile,language,vocabulary,boost) VALUES('live-fixture','realtime','live-model',1,'nemotron-3.5-streaming','fr-FR','Freehand',2.5);`)
			if err != nil {
				t.Fatal(err)
			}
			if live {
				if _, err = s.db.Exec(`UPDATE realtime_settings SET enabled=1`); err != nil {
					t.Fatal(err)
				}
			}
			s = reopen(t, s)
			got := loadStore(t, s)
			if got.BaseURL != before.BaseURL || got.Model != before.Model || !reflect.DeepEqual(got.Headers, before.Headers) {
				t.Fatal("migration changed audio-file selection")
			}
			selected := s.ConnectionCatalog().Selected[savedconnection.Voice]
			if live {
				if !got.SetupCompleted || !got.VoiceTranscription.Realtime || got.VoiceTranscription.Model != "live-model" || got.VoiceTranscription.Language != "fr-FR" || got.Vocabulary.Terms != "Freehand" || selected != "live-fixture" {
					t.Fatal("migration lost active realtime selection")
				}
			} else if !reflect.DeepEqual(got.VoiceTranscription, config.VoiceFromCompleted(before)) || selected != s.ConnectionCatalog().Selected[savedconnection.Transcription] {
				t.Fatal("migration lost completed microphone configuration")
			}
			found := false
			for _, e := range s.connections.models {
				if e.Purpose == savedconnection.Voice && e.Model == "live-model" {
					found = true
				}
			}
			if !found {
				t.Fatal("inactive realtime model choices were discarded")
			}
		})
	}
}
