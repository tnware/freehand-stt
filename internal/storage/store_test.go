package storage

import (
	"context"
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
	"github.com/tnware/freehand-stt/internal/savedconnection"
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
	s := newStore(filepath.Join(dir, "freehand.db"), &memoryVault{values: map[string]string{}})
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
	next := newStore(s.path, s.vault)
	t.Cleanup(func() { next.Close() })
	return next
}

// selectStorageConnection exercises the real saved-connection transaction before
// tests mutate transport settings or credentials; neither can have an orphan owner.
func selectStorageConnection(t *testing.T, s *Store, v config.Settings) config.Settings {
	t.Helper()
	details := savedconnection.Extract(v, savedconnection.Transcription)
	details.BaseURL = "https://speech.example.test/v1"
	details.AuthenticationMode = config.AuthenticationModeNone
	next, err := s.BeginConnectionChange(savedconnection.Change{
		Action: savedconnection.Create, Name: "Test speech", Uses: []savedconnection.Purpose{savedconnection.Transcription},
		ActivateFor: savedconnection.Transcription, Details: &details,
	}, v)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DiscardCredentialChanges()
	if err := s.Save(next); err != nil {
		t.Fatal(err)
	}
	return next
}

func TestFreshDatabaseAndRoundTrip(t *testing.T) {
	s := testStore(t)
	want := config.Default()
	if got := loadStore(t, s); !reflect.DeepEqual(got, want) {
		t.Fatalf("defaults changed: %#v", got)
	}
	want = selectStorageConnection(t, s, want)
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
		actual, expected := reflect.ValueOf(got), reflect.ValueOf(want)
		for i := 0; i < actual.NumField(); i++ {
			if !reflect.DeepEqual(actual.Field(i).Interface(), expected.Field(i).Interface()) {
				t.Errorf("round trip %s: got %#v, want %#v", actual.Type().Field(i).Name, actual.Field(i).Interface(), expected.Field(i).Interface())
			}
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.path), "settings.json")); !os.IsNotExist(err) {
		t.Fatal("created a parallel JSON store")
	}
	if err := checkIntegrity(context.Background(), s.db); err != nil {
		t.Fatal(err)
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
			old := selectStorageConnection(t, s, loadStore(t, s))
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
	other := newStore(s.path, s.vault)
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
	v := selectStorageConnection(t, s, loadStore(t, s))
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
	entries, _ := embeddedMigrations.ReadDir("schema")
	for _, entry := range entries {
		data, _ := embeddedMigrations.ReadFile("schema/" + entry.Name())
		migrations[entry.Name()] = &fstest.MapFile{Data: data}
	}
	migrations[fmt.Sprintf("%05d_fixture.sql", len(entries)+1)] = &fstest.MapFile{Data: []byte("-- +goose Up\n" + sql)}
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
			backups, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "freehand-backups", "freehand-*.db"))
			if len(backups) != 1 {
				t.Fatal("missing pre-upgrade backup")
			}
			s.Close()
			db, _ := openDatabase(s.path, "rw")
			var current int
			db.QueryRow("SELECT max(version_id) FROM goose_db_version").Scan(&current)
			db.Close()
			entries, _ := embeddedMigrations.ReadDir("schema")
			if fail && current != len(entries) {
				t.Fatal("failed migration advanced version")
			}
			restore := newStore(s.path, s.vault)
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
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "freehand-backups", "*.db"))
	if len(files) != retainedBackups {
		t.Fatal("backup retention unbounded")
	}
	v := selectStorageConnection(t, s, loadStore(t, s))
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("keep-secret"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s.DiscardCredentialChanges()
	account := s.refs["stt"]
	if err := s.Reset(config.Default()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.STTCredentials().Get(); !errors.Is(err, credential.ErrNotFound) {
		t.Fatal("reset retained an unowned credential reference")
	}
	if s.vault.(*memoryVault).values[account] != "keep-secret" {
		t.Fatal("reset deleted credential needed by the recovery archive")
	}
	archived, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "freehand-recovery-*", "freehand.db"))
	if len(archived) != 1 {
		t.Fatal("reset did not preserve previous database")
	}
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, config.Default()) {
		t.Fatal("reset defaults did not survive reopen")
	}
	if _, err := s.STTCredentials().Get(); !errors.Is(err, credential.ErrNotFound) {
		t.Fatal("reset reused a credential after reopen")
	}
	if err := s.RestoreBackup(archived[0]); err != nil {
		t.Fatal(err)
	}
	if got, err := s.STTCredentials().Get(); err != nil || got != "keep-secret" {
		t.Fatal("explicit restore lost archived credential reference")
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
	v := selectStorageConnection(t, s, loadStore(t, s))
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
	_, err = db.Exec("UPDATE saved_connections SET credential_account='unrelated-native-account'")
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
