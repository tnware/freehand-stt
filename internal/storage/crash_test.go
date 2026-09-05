package storage

import (
	"context"
	"database/sql/driver"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
	"modernc.org/sqlite"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Exit without deferred cleanup, exercising recovery from an abruptly lost process.
func TestAbruptProcessRecovery(t *testing.T) {
	if mode := os.Getenv("FREEHAND_STORAGE_CRASH_FIXTURE"); mode != "" {
		path := os.Getenv("FREEHAND_STORAGE_FIXTURE_PATH")
		s := newStore(path, filepath.Join(filepath.Dir(path), "settings.json"), &memoryVault{values: map[string]string{}})
		if mode == "save" {
			v := loadStore(t, s)
			v.Language = "de"
			tx, err := s.db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err = writeSettings(context.Background(), dbgen.New(tx), v); err != nil {
				t.Fatal(err)
			}
			os.Exit(77)
		}
		err := sqlite.RegisterScalarFunction("fixture_crash", 0, func(*sqlite.FunctionContext, []driver.Value) (driver.Value, error) { os.Exit(77); return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
		withUpgrade(s, "CREATE TABLE interrupted_fixture(id INTEGER PRIMARY KEY) STRICT; SELECT fixture_crash();")
		if _, err = s.Load(); err != nil {
			t.Fatal(err)
		}
		t.Fatal("crash fixture unexpectedly returned")
	}
	for _, mode := range []string{"save", "upgrade", "import"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			if mode == "import" {
				v := configForCrash()
				writeLegacy(t, s, v)
			} else {
				v := loadStore(t, s)
				v.Language = "ja"
				if err := s.Save(v); err != nil {
					t.Fatal(err)
				}
			}
			s.Close()
			command := exec.Command(os.Args[0], "-test.run=^TestAbruptProcessRecovery$")
			command.Env = append(os.Environ(), "FREEHAND_STORAGE_CRASH_FIXTURE="+mode, "FREEHAND_STORAGE_FIXTURE_PATH="+s.path)
			output, err := command.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 77 {
				t.Fatalf("crash fixture failed: %v %s", err, output)
			}
			if mode == "import" {
				if _, err := os.Stat(s.path); !os.IsNotExist(err) {
					t.Fatal("interrupted import published incomplete database")
				}
			}
			s = newStore(s.path, s.legacy, s.vault)
			defer s.Close()
			if got := loadStore(t, s); got.Language != "ja" {
				t.Fatal("interrupted operation changed committed settings")
			}
			var count int
			if err := s.db.QueryRow("SELECT count(*) FROM sqlite_schema WHERE name='interrupted_fixture'").Scan(&count); err != nil || count != 0 {
				t.Fatal("interrupted migration left partial DDL")
			}
			if mode == "upgrade" {
				backups, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "backups", "*.db"))
				if len(backups) != 1 {
					t.Fatal("interrupted upgrade lost backup")
				}
			}
		})
	}
}

func configForCrash() config.Settings { v := config.Default(); v.Language = "ja"; return v }
