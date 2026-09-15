package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/tnware/freehand-stt/internal/modelprofile"
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
			defer db.Close()
			ctx := t.Context()
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
			if err = writeVersionTwoSettings(ctx, db, v); err != nil {
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

// writeVersionTwoSettings seeds the historical schema before NeMo columns existed.
func writeVersionTwoSettings(ctx context.Context, db *sql.DB, v config.Settings) error {
	q := dbgen.New(db)
	if err := q.PutVocabulary(ctx, dbgen.PutVocabularyParams{Terms: v.Vocabulary.Terms, Voice: boolean(v.Vocabulary.Voice), Files: boolean(v.Vocabulary.Files), Boost: v.Vocabulary.Boost}); err != nil {
		return err
	}
	r := v.VoiceTranscription
	if _, err := db.ExecContext(ctx, `INSERT INTO voice_transcription_settings(id,realtime,model_profile,model,language,timeout_seconds,prompt,temperature_override,temperature,captions) VALUES(1,?,?,?,?,?,?,?,?,?)`, boolean(r.Realtime), string(r.ModelProfile), r.Model, r.Language, r.TimeoutSeconds, r.TranscriptionOptions.Prompt, boolean(r.TranscriptionOptions.TemperatureOverride), r.TranscriptionOptions.Temperature, boolean(r.Captions)); err != nil {
		return err
	}

	if err := q.PutPreferences(ctx, dbgen.PutPreferencesParams{
		ToggleShortcut:          v.ToggleShortcut,
		ShowShortcut:            v.ShowShortcut,
		HoldShortcut:            v.HoldShortcut,
		MicrophoneID:            v.MicrophoneID,
		MaxDurationSeconds:      int64(v.MaxDurationSeconds),
		AutoInsert:              boolean(v.AutoInsert),
		StartWithWindows:        boolean(v.StartWithWindows),
		ShowWindowOnLaunch:      boolean(v.ShowWindowOnLaunch),
		CheckForUpdates:         boolean(v.CheckForUpdates),
		SetupCompleted:          boolean(v.SetupCompleted),
		UseMica:                 boolean(v.UseMica),
		AppearanceMode:          string(v.AppearanceMode),
		OverlayEnabled:          boolean(v.OverlayEnabled),
		OverlaySizePercent:      int64(v.OverlaySizePercent),
		OverlayOpacityPercent:   int64(v.OverlayOpacityPercent),
		OverlayTopOffset:        int64(v.OverlayTopOffset),
		OverlayGlowPercent:      int64(v.OverlayGlowPercent),
		OverlayLayout:           string(v.OverlayLayout),
		OverlayAnchor:           string(v.OverlayAnchor),
		OverlayVisibility:       string(v.OverlayVisibility),
		OverlayMotion:           string(v.OverlayMotion),
		OverlaySurface:          string(v.OverlaySurface),
		OverlayVisualizer:       string(v.OverlayVisualizer),
		HistoryEnabled:          boolean(v.HistoryEnabled),
		VadEnabled:              boolean(v.VADEnabled),
		VadMode:                 string(v.VADMode),
		VadActivitySilenceMs:    int64(v.VADActivitySilenceMS),
		SilenceTrimming:         boolean(v.SilenceTrimming),
		SpeechPaddingMs:         int64(v.SpeechPaddingMS),
		AutoStopEnabled:         boolean(v.AutoStopEnabled),
		AutoStopSilenceMs:       int64(v.AutoStopSilenceMS),
		AutoStopMinimumSpeechMs: int64(v.AutoStopMinimumSpeechMS),
		SilenceSplitting:        boolean(v.SilenceSplitting),
		SegmentSeconds:          int64(v.SegmentSeconds),
		SegmentSilenceMs:        int64(v.SegmentSilenceMS),
	}); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO transcription_settings(id,model_profile,model,language,transcription_timeout_seconds,file_transcription_timeout_seconds,transcription_options_prompt,transcription_options_temperature_override,transcription_options_temperature) VALUES(1,?,?,?,?,?,?,?,?)`, string(modelprofile.Effective(v.ModelProfile)), v.Model, v.Language, v.TranscriptionTimeoutSeconds, v.FileTranscriptionTimeoutSeconds, v.TranscriptionOptions.Prompt, boolean(v.TranscriptionOptions.TemperatureOverride), v.TranscriptionOptions.Temperature); err != nil {
		return err
	}
	if err := q.PutCleanup(ctx, dbgen.PutCleanupParams{
		GenerationOptionsLimitOutputTokens: boolean(v.PostProcessing.GenerationOptions.LimitOutputTokens),
		GenerationOptionsMaxOutputTokens:   int64(v.PostProcessing.GenerationOptions.MaxOutputTokens),
		GenerationOptionsDisableReasoning:  boolean(v.PostProcessing.GenerationOptions.DisableReasoning),
		Enabled:                            boolean(v.PostProcessing.Enabled),
		Model:                              v.PostProcessing.Model,
		Preset:                             string(v.PostProcessing.Preset),
		SystemPrompt:                       v.PostProcessing.SystemPrompt,
		Styling:                            v.PostProcessing.Styling,
		Structure:                          v.PostProcessing.Structure,
		Context:                            v.PostProcessing.Context,
		TimeoutSeconds:                     int64(v.PostProcessing.TimeoutSeconds),
	}); err != nil {
		return err
	}
	if err := q.PutSpeech(ctx, dbgen.PutSpeechParams{
		SpeechLanguage:     v.TextToSpeech.Options.Language,
		SpeechInstructions: v.TextToSpeech.Options.Instructions,
		ModelProfile:       string(modelprofile.Effective(v.TextToSpeech.ModelProfile)),
		Enabled:            boolean(v.TextToSpeech.Enabled),
		Model:              v.TextToSpeech.Model,
		Voice:              v.TextToSpeech.Voice,
		Speed:              v.TextToSpeech.Speed,
		TimeoutSeconds:     int64(v.TextToSpeech.TimeoutSeconds),
	}); err != nil {
		return err
	}
	return nil
}
