package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

func TestManagedUpgradePreservesManualState(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			s := testStore(t)
			db, err := openDatabase(s.path, "rwc")
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			p, err := goose.NewProvider(goose.DialectSQLite3, db, s.migrations, goose.WithLogger(goose.NopLogger()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = p.UpTo(ctx, 2); err != nil {
				t.Fatal(err)
			}
			v := config.Default()
			v.Language = "fr"
			v.VoiceTranscription.Language = "de"
			if err = writeSettings(ctx, dbgen.New(db), v); err != nil {
				t.Fatal(err)
			}
			statements := []string{
				fmt.Sprintf("PRAGMA application_id=%d", applicationID),
				fmt.Sprintf("UPDATE managed_runtime_preferences SET enabled=%d, realtime=1", boolean(enabled)),
				`INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account) VALUES('manual','Manual','generic','https://example.test/v1',0,'none','','')`,
				`INSERT INTO saved_connection_uses VALUES('manual','voice'),('manual','stt')`,
				`INSERT INTO selected_connections VALUES('voice','manual'),('stt','manual')`,
				`INSERT INTO saved_connection_headers VALUES('manual','X-Test','preserved')`,
				`INSERT INTO remembered_models(connection_id,purpose,model,selected) VALUES('manual','voice','manual-model',1)`,
			}
			for _, stmt := range statements {
				if _, err = db.Exec(stmt); err != nil {
					t.Fatal(err)
				}
			}
			db.Close()
			got, err := s.Load()
			if err != nil {
				t.Fatalf("upgrade: %v; cause: %v", err, err.(*storageError).cause)
			}
			if len(got.ManagedRuntimes) != 1 || got.ManagedRuntimes[0].AutoStart != enabled || got.ManagedRuntimes[0].ID != "nemo-default" {
				t.Fatalf("bad inventory: %#v", got.ManagedRuntimes)
			}
			if got.Language != "fr" || got.VoiceTranscription.Language != "de" {
				t.Fatal("task languages lost")
			}
			want := "manual"
			if enabled {
				want = "managed-nemo-default"
			}
			for _, role := range []savedconnection.Purpose{savedconnection.Voice, savedconnection.Transcription} {
				if s.ConnectionCatalog().Selected[role] != want {
					t.Fatal("selection not migrated")
				}
			}
			if enabled && (!got.VoiceTranscription.Realtime || got.VoiceTranscription.BaseURL != "" || got.BaseURL != "") {
				t.Fatal("managed projection retained manual transport")
			}
			manual, _, err := s.ResolveSavedConnection("manual")
			if err != nil || manual.Details.Headers["X-Test"] != "preserved" {
				t.Fatal("manual connection lost")
			}
			var count int
			if err = s.db.QueryRow(`SELECT count(*) FROM remembered_models WHERE connection_id='manual' AND model='manual-model'`).Scan(&count); err != nil || count != 1 {
				t.Fatal("manual model memory lost", err)
			}
			loadStore(t, reopen(t, s))
		})
	}
}

func TestManagedDatabaseIdentityAndTransportConstraints(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
	v = createSelectedConnection(t, s, v, "Local", savedconnection.Voice, savedconnection.Details{ManagedInstanceID: "local"})
	id := s.ConnectionCatalog().Selected[savedconnection.Voice]
	for n, stmt := range []string{
		`DELETE FROM managed_runtime_instances WHERE id=?`,
		`UPDATE managed_runtime_instances SET provider='other' WHERE id=?`,
		`UPDATE saved_connections SET credential_account='connection-secret' WHERE id=?`,
		`UPDATE saved_connections SET authentication_mode='api-key' WHERE id=?`,
		`INSERT INTO saved_connection_headers VALUES(?,'X-Test','manual')`,
	} {
		target := id
		if n < 2 {
			target = "local"
		}
		if _, err := s.db.Exec(stmt, target); err == nil {
			t.Fatalf("constraint accepted %s", stmt)
		}
	}
	v = selectConnection(t, s, v, savedconnection.Voice, "")
	v, err := s.BeginConnectionChange(savedconnection.Change{Action: savedconnection.Delete, ID: id}, v)
	if err != nil {
		t.Fatal(err)
	}
	v.ManagedRuntimes = nil
	if err = s.Save(v); err != nil {
		t.Fatal(err)
	}
	if got := loadStore(t, reopen(t, s)); len(got.ManagedRuntimes) != 0 {
		t.Fatal("unreferenced runtime not deleted")
	}
}

func TestManagedConversionClearsManualHeaders(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
	d := savedconnection.Extract(v, savedconnection.Voice)
	d.BaseURL = "https://example.test/v1"
	d.Headers = map[string]string{"X-Test": "manual"}
	v = createSelectedConnection(t, s, v, "Speech", savedconnection.Voice, d)
	id := s.ConnectionCatalog().Selected[savedconnection.Voice]
	d = savedconnection.Details{ManagedInstanceID: "local"}
	v, err := s.BeginConnectionChange(savedconnection.Change{Action: savedconnection.Update, ID: id, Name: "Speech", Details: &d, Uses: []savedconnection.Purpose{savedconnection.Voice}}, v)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.StageConnectionCredential("", true); err != nil {
		t.Fatal(err)
	}
	if err = s.Save(v); err != nil {
		t.Fatalf("manual to managed conversion failed: %v", err)
	}
	got := loadStore(t, reopen(t, s))
	if got.VoiceTranscription.ManagedInstanceID != "local" || len(got.VoiceTranscription.Headers) != 0 {
		t.Fatal("manual transport survived conversion")
	}
}
