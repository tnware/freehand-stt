package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
)

type memoryVault struct {
	values              map[string]string
	failSet, failDelete bool
}

func (v *memoryVault) Get(a string) (string, error) {
	if s, ok := v.values[a]; ok {
		return s, nil
	}
	return "", credential.ErrNotFound
}
func (v *memoryVault) Set(a, s string) error {
	if v.failSet {
		return errors.New("vault unavailable")
	}
	v.values[a] = s
	return nil
}
func (v *memoryVault) Delete(a string) error {
	if v.failDelete {
		return errors.New("vault unavailable")
	}
	delete(v.values, a)
	return nil
}
func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s := newStore(filepath.Join(dir, "settings.db"), filepath.Join(dir, "settings.json"), &memoryVault{values: map[string]string{}})
	t.Cleanup(func() { s.Close() })
	return s
}
func loadStore(t *testing.T, s *Store) config.Settings {
	t.Helper()
	v, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func reopen(t *testing.T, s *Store) *Store {
	t.Helper()
	s.Close()
	next := newStore(s.path, s.legacy, s.vault)
	t.Cleanup(func() { next.Close() })
	return next
}
func writeLegacy(t *testing.T, s *Store, v config.Settings) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(s.legacy, b, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestFreshDatabaseAndRoundTrip(t *testing.T) {
	s := testStore(t)
	want := config.Default()
	if got := loadStore(t, s); !reflect.DeepEqual(got, want) {
		t.Fatalf("defaults changed: %#v", got)
	}
	want.Language = "ja"
	want.Headers = map[string]string{"X-Request-Mode": "unicode-æ—¥æœ¬èªž"}
	want.MicrophoneID = "device-id"
	want.OverlayEnabled = false
	want.ShowWindowOnLaunch = false
	want.TranscriptionOptions.Prompt = "Names and punctuation"
	want.PostProcessing.Styling = "formal"
	want.TextToSpeech.Speed = 1.5
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, want) {
		t.Fatal("settings round trip lost a value")
	}
	if _, err := os.Stat(s.legacy); !os.IsNotExist(err) {
		t.Fatal("created a parallel JSON store")
	}
	if err := checkIntegrity(context.Background(), s.db); err != nil {
		t.Fatal(err)
	}
}
func TestLegacyImportOnceAndInvalidImportRetry(t *testing.T) {
	s := testStore(t)
	if err := os.WriteFile(s.legacy, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); config.LoadFailureFor(err).Kind != "legacy_invalid" {
		t.Fatalf("wrong failure: %v", err)
	}
	if _, err := os.Stat(s.path); !os.IsNotExist(err) {
		t.Fatal("failed import published a database")
	}
	want := config.Default()
	want.Language = "es"
	writeLegacy(t, s, want)
	original, _ := os.ReadFile(s.legacy)
	s.vault.(*memoryVault).values[credential.STTAccount] = "legacy-secret"
	if got := loadStore(t, s); got.Language != "es" {
		t.Fatal("legacy selection lost")
	}
	if got, err := s.STTCredentials().Get(); err != nil || got != "legacy-secret" {
		t.Fatal("legacy credential reference lost")
	}
	if current, _ := os.ReadFile(s.legacy); string(current) != string(original) {
		t.Fatal("legacy file modified")
	}
	want.Language = "fr"
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if got := loadStore(t, s); got.Language != "fr" {
		t.Fatal("legacy file imported again")
	}
}
func TestUnknownLegacyFieldsBlockImport(t *testing.T) {
	s := testStore(t)
	os.WriteFile(s.legacy, []byte(`{"futureSetting":true}`), 0600)
	if _, err := s.Load(); config.LoadFailureFor(err).Kind != "legacy_newer" {
		t.Fatalf("unknown setting discarded: %v", err)
	}
}
func TestFailedSaveIsAtomic(t *testing.T) {
	s := testStore(t)
	old := loadStore(t, s)
	// Fault after preferences are written: the whole save must roll back.
	_, err := s.db.Exec(`CREATE TRIGGER fail_cleanup BEFORE UPDATE ON cleanup_settings BEGIN SELECT RAISE(ABORT, 'fixture failure'); END;`)
	if err != nil {
		t.Fatal(err)
	}
	next := old
	next.Language = "fr"
	next.ShowWindowOnLaunch = false
	if err = s.Save(next); err == nil {
		t.Fatal("save should fail")
	}
	if got := loadStore(t, s); !reflect.DeepEqual(got, old) {
		t.Fatal("partial save published")
	}
}
func TestReadOnlyAndDiskFullSavePreserveSettings(t *testing.T) {
	for _, mode := range []string{"readonly", "full"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			old := loadStore(t, s)
			if mode == "readonly" {
				s.db.Exec("PRAGMA query_only=ON")
			} else {
				if _, err := s.db.Exec("VACUUM"); err != nil {
					t.Fatal(err)
				}
				var pages int
				s.db.QueryRow("PRAGMA page_count").Scan(&pages)
				s.db.Exec("PRAGMA max_page_count=" + fmtInt(pages))
			}
			next := old
			next.Language = "fr"
			next.PostProcessing.SystemPrompt = strings.Repeat("x", 8192)
			if mode == "full" {
				next.Headers = map[string]string{"X-Large-One": strings.Repeat("x", 4096), "X-Large-Two": strings.Repeat("y", 4096), "X-Large-Three": strings.Repeat("z", 4096)}
			}
			if err := s.Save(next); err == nil {
				t.Fatal("save should fail")
			}
			if got := loadStore(t, s); !reflect.DeepEqual(got, old) {
				t.Fatal("failed save changed settings")
			}
		})
	}
}
func fmtInt(v int) string { return fmt.Sprintf("%d", v) }
func TestDatabaseLockAndBoundedSQLBusy(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	other := newStore(s.path, s.legacy, s.vault)
	defer other.Close()
	if _, err := other.Load(); config.LoadFailureFor(err).Kind != "locked" {
		t.Fatalf("missing ownership lock: %v", err)
	}
	outsider, err := openDatabase(s.path, "rw")
	if err != nil {
		t.Fatal(err)
	}
	defer outsider.Close()
	tx, err := outsider.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	start := time.Now()
	if err = s.Save(v); err == nil {
		t.Fatal("write ignored lock")
	}
	if time.Since(start) > 4*time.Second {
		t.Fatal("lock wait unbounded")
	}
}
func TestForeignAndNewerDatabaseNotMutated(t *testing.T) {
	for _, mode := range []string{"foreign", "newer", "corrupt"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			loadStore(t, s)
			switch mode {
			case "foreign":
				s.db.Exec("PRAGMA application_id=99")
			case "newer":
				s.db.Exec("INSERT INTO goose_db_version(version_id,is_applied) VALUES(99,1)")
			}
			s.Close()
			if mode == "corrupt" {
				os.WriteFile(s.path, []byte("not a sqlite database"), 0600)
			}
			before, _ := os.ReadFile(s.path)
			s = reopen(t, s)
			if _, err := s.Load(); err == nil {
				t.Fatal("bad database accepted")
			}
			after, _ := os.ReadFile(s.path)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected database modified")
			}
		})
	}
}
func TestCredentialsCommitRollbackAndCrashCleanup(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	vault := s.vault.(*memoryVault)
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("first-secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.STTCredentials().Get(); !errors.Is(err, credential.ErrNotFound) {
		t.Fatal("uncommitted credential exposed")
	}
	v.Language = "en"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s.DiscardCredentialChanges()
	old := s.refs["stt"]
	s.BeginCredentialChanges()
	s.STTCredentials().Set("replacement-secret")
	if got, _ := s.STTCredentials().Get(); got != "first-secret" {
		t.Fatal("replacement exposed before commit")
	}
	s.DiscardCredentialChanges()
	if len(vault.values) != 1 || vault.values[old] != "first-secret" {
		t.Fatal("rollback lost original or leaked replacement")
	}
	s.BeginCredentialChanges()
	s.STTCredentials().Set("crashed-secret")
	s = reopen(t, s)
	loadStore(t, s)
	if len(vault.values) != 1 || vault.values[old] != "first-secret" {
		t.Fatal("startup did not reclaim orphan credential")
	}
	s.BeginCredentialChanges()
	s.STTCredentials().Set("final-secret")
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.STTCredentials().Get(); got != "final-secret" {
		t.Fatal("committed credential not selected")
	}
	if len(vault.values) != 1 {
		t.Fatal("obsolete credential not reclaimed")
	}
	raw, _ := os.ReadFile(s.path)
	if strings.Contains(string(raw), "secret") {
		t.Fatal("secret reached database")
	}
}
func TestSchemaConstraintsAndHeaders(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	for _, statement := range []string{"UPDATE preferences_settings SET auto_insert=2", "UPDATE preferences_settings SET max_duration_seconds='invalid'", "UPDATE speech_settings SET speed=5", "DELETE FROM preferences_settings WHERE id=1"} {
		if _, err := s.db.Exec(statement); err == nil {
			t.Fatalf("constraint missing: %s", statement)
		}
	}
}
func withUpgrade(s *Store, sql string) {
	migrations := fstest.MapFS{}
	entries, _ := embeddedMigrations.ReadDir("migrations")
	for _, entry := range entries {
		data, _ := embeddedMigrations.ReadFile("migrations/" + entry.Name())
		migrations[entry.Name()] = &fstest.MapFile{Data: data}
	}
	migrations["00012_fixture.sql"] = &fstest.MapFile{Data: []byte("-- +goose Up\n" + sql)}
	s.migrations = migrations
}
func TestUpgradeBackupRollbackAndRestore(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure-%t", fail), func(t *testing.T) {
			s := testStore(t)
			want := loadStore(t, s)
			want.Language = "ja"
			s.Save(want)
			s = reopen(t, s)
			migration := "CREATE TABLE upgrade_fixture(id INTEGER PRIMARY KEY) STRICT;"
			if fail {
				migration += "INSERT INTO missing_table VALUES(1);"
			}
			withUpgrade(s, migration)
			got, err := s.Load()
			if (err != nil) != fail {
				t.Fatalf("upgrade outcome %v", err)
			}
			if !fail && got.Language != "ja" {
				t.Fatal("upgrade lost settings")
			}
			backups, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "backups", "settings-*.db"))
			if len(backups) != 1 {
				t.Fatal("missing pre-upgrade backup")
			}
			s.Close()
			db, _ := openDatabase(s.path, "rw")
			var current int
			db.QueryRow("SELECT max(version_id) FROM goose_db_version").Scan(&current)
			db.Close()
			if fail && current != 11 {
				t.Fatal("failed migration advanced version")
			}
			restore := newStore(s.path, s.legacy, s.vault)
			defer restore.Close()
			if err = restore.RestoreBackup(backups[0]); err != nil {
				t.Fatal(err)
			}
			if loadStore(t, restore).Language != "ja" {
				t.Fatal("restore lost settings")
			}
		})
	}
}
func TestBackupRetentionAndResetPreserveEvidence(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	for range 5 {
		if _, err := s.backup(context.Background(), s.db); err != nil {
			t.Fatal(err)
		}
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "backups", "*.db"))
	if len(files) != retainedBackups {
		t.Fatal("backup retention unbounded")
	}
	s.BeginCredentialChanges()
	s.STTCredentials().Set("keep-secret")
	s.Save(config.Default())
	s.DiscardCredentialChanges()
	if err := s.Reset(config.Default()); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.STTCredentials().Get(); got != "keep-secret" {
		t.Fatal("reset lost credential reference")
	}
	archived, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "settings-recovery-*", "settings.db"))
	if len(archived) != 1 {
		t.Fatal("reset did not preserve previous database")
	}
	s = reopen(t, s)
	loadStore(t, s)
	if got, _ := s.STTCredentials().Get(); got != "keep-secret" {
		t.Fatal("reset credential reference was not durable")
	}
}
func TestClosedStoreRejectsWork(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	s.Close()
	if err := s.Save(config.Default()); err == nil {
		t.Fatal("save after close")
	}
	if _, err := s.Load(); err == nil {
		t.Fatal("load after close")
	}
}

func TestZeroSpeedForDisabledUnconfiguredSpeechIsPreserved(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.TextToSpeech.Speed = 0
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	if got := loadStore(t, s); got.TextToSpeech.Speed != 0 {
		t.Fatal("inactive speech value lost")
	}
}
func TestInvalidCredentialReferenceBlocksRestoreBeforeReplacement(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.Language = "ja"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	backup, err := s.backup(context.Background(), s.db)
	if err != nil {
		t.Fatal(err)
	}
	db, err := openDatabase(backup, "rw")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("UPDATE credential_refs SET account='unrelated-native-account' WHERE purpose='stt'")
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RestoreBackup(backup); err == nil {
		t.Fatal("invalid credential reference accepted")
	}
	if got := loadStore(t, s); got.Language != "ja" {
		t.Fatal("failed restore replaced settings")
	}
}

func TestVersionOneUpgradePreservesSettingsAndReferences(t *testing.T) {
	s := testStore(t)
	want := loadStore(t, s)
	want.Language = "ja"
	want.VADActivitySilenceMS = 500
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("v1-fixture-key"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	// Recreate the retained v1 layout, including its original speech constraint.
	initial, err := embeddedMigrations.ReadFile("migrations/00001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(initial)
	start := strings.Index(sql, "CREATE TABLE speech_settings (")
	end := strings.Index(sql[start:], ") STRICT;") + len(") STRICT;")
	_, err = s.db.Exec(`ALTER TABLE speech_settings DROP COLUMN speech_language; ALTER TABLE speech_settings DROP COLUMN speech_instructions; ALTER TABLE remembered_models DROP COLUMN speech_language; ALTER TABLE remembered_models DROP COLUMN speech_instructions; DROP TABLE vocabulary_settings; DROP TABLE voice_transcription_settings; DROP TABLE voice_request_headers; DELETE FROM selected_connections WHERE purpose='voice'; DELETE FROM saved_connection_uses WHERE purpose='voice'; DELETE FROM credential_refs WHERE purpose='voice'; DROP TABLE remembered_models; ALTER TABLE transcription_settings DROP COLUMN model_profile;
 ALTER TABLE preferences_settings RENAME COLUMN vad_enabled TO vadenabled;
 ALTER TABLE preferences_settings RENAME COLUMN vad_mode TO vadmode;
 ALTER TABLE preferences_settings RENAME COLUMN vad_activity_silence_ms TO vadactivity_silence_ms;
 ALTER TABLE speech_settings RENAME TO speech_fixture;` + sql[start:start+end] + `
 INSERT INTO speech_settings(id,compatibility_profile,enabled,base_url,allow_insecure_http,authentication_mode,model,voice,speed,timeout_seconds)
 SELECT id,compatibility_profile,enabled,base_url,allow_insecure_http,authentication_mode,model,voice,speed,timeout_seconds FROM speech_fixture;
 DROP TABLE speech_fixture; DROP TABLE saved_connection_headers; DROP TABLE selected_connections; DROP TABLE saved_connection_uses; DROP TABLE saved_connections; DELETE FROM goose_db_version WHERE version_id>=2;`)
	if err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	want.VoiceTranscription = config.VoiceFromCompleted(want)
	if got := loadStore(t, s); !reflect.DeepEqual(got, want) {
		t.Fatal("v1 upgrade lost settings")
	}
	if key, err := s.STTCredentials().Get(); err != nil || key != "v1-fixture-key" {
		t.Fatal("v1 upgrade lost credential reference")
	}
	backups, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "backups", "*.db"))
	if len(backups) != 1 {
		t.Fatal("v1 upgrade missing backup")
	}
}

func TestVersionThreeUpgradeKeepsRuntimeModelsAndConnectionKeys(t *testing.T) {
	s := testStore(t)
	initial := config.Default()
	initial.BaseURL = "https://stt.example.test/v1"
	initial.Model = "current-stt"
	initial.PostProcessing.BaseURL = "https://cleanup.example.test/v1"
	initial.PostProcessing.Model = "current-cleanup"
	writeLegacy(t, s, initial)
	want := loadStore(t, s)
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("migration-fixture"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	before := s.ConnectionCatalog()
	restoreConnectionSchema(t, s, 3)
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, want) {
		t.Fatal("upgrade changed runtime choices")
	}
	if !reflect.DeepEqual(s.ConnectionCatalog(), before) {
		t.Fatal("upgrade changed connection selections")
	}
	if key, err := s.STTCredentials().Get(); err != nil || key != "migration-fixture" {
		t.Fatal("upgrade lost key reference")
	}
}

func restoreConnectionSchema(t *testing.T, s *Store, version int) {
	t.Helper()
	old, err := embeddedMigrations.ReadFile("migrations/00003_saved_connections.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`ALTER TABLE speech_settings DROP COLUMN speech_language; ALTER TABLE speech_settings DROP COLUMN speech_instructions; ALTER TABLE remembered_models DROP COLUMN speech_language; ALTER TABLE remembered_models DROP COLUMN speech_instructions; DROP TABLE vocabulary_settings; DROP TABLE voice_transcription_settings; DROP TABLE voice_request_headers; DELETE FROM selected_connections WHERE purpose='voice'; DELETE FROM saved_connection_uses WHERE purpose='voice'; DELETE FROM credential_refs WHERE purpose='voice'; DROP TABLE remembered_models; ALTER TABLE transcription_settings DROP COLUMN model_profile; ALTER TABLE speech_settings DROP COLUMN model_profile; DROP TABLE selected_connections; DROP TABLE saved_connection_headers; DROP TABLE saved_connection_uses; DROP TABLE saved_connections;` + string(old)); err != nil {
		t.Fatal(err)
	}
	// Recreate the configured-only catalog used by this fixture, without v3 bootstrap rows.
	if _, err = s.db.Exec(`DELETE FROM selected_connections WHERE connection_id IN(SELECT id FROM saved_connections WHERE base_url=''); DELETE FROM saved_connections WHERE base_url='';`); err != nil {
		t.Fatal(err)
	}
	if version == 4 {
		next, _ := embeddedMigrations.ReadFile("migrations/00004_connection_manager.sql")
		if _, err = s.db.Exec(string(next)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.db.Exec("DELETE FROM goose_db_version WHERE version_id>?", version); err != nil {
		t.Fatal(err)
	}
}
func TestVersionFourUpgradePreservesConnectionUses(t *testing.T) {
	s := testStore(t)
	initial := config.Default()
	initial.BaseURL = "https://speech.example.test/v1"
	initial.Model = "chosen-model"
	writeLegacy(t, s, initial)
	want := loadStore(t, s)
	before := s.ConnectionCatalog()
	restoreConnectionSchema(t, s, 4)
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, want) {
		t.Fatal("upgrade changed runtime settings")
	}
	if !reflect.DeepEqual(s.ConnectionCatalog(), before) {
		t.Fatal("upgrade changed original connection use or selection")
	}
}
