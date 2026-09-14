package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const initialMigration = "-- +goose Up\nCREATE TABLE settings (id INTEGER PRIMARY KEY);\n"
const initialPath = "internal/storage/schema/00001_initial.sql"

func writeMigrationFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func fixtureGit(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestMigrationLineage(t *testing.T) {
	for _, action := range []string{"unchanged", "changed", "removed", "renamed", "appended"} {
		t.Run(action, func(t *testing.T) {
			t.Chdir(t.TempDir())
			fixtureGit(t, "init", "-q")
			writeMigrationFixture(t, initialPath, initialMigration)
			fixtureGit(t, "add", ".")
			fixtureGit(t, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "publish baseline")
			base := fixtureGit(t, "rev-parse", "HEAD")
			want := ""
			switch action {
			case "changed":
				writeMigrationFixture(t, initialPath, initialMigration+"-- changed\n")
				want = "published migration changed"
			case "removed", "renamed":
				if err := os.Remove(initialPath); err != nil {
					t.Fatal(err)
				}
				// Retain a valid next migration so absence of the published
				// baseline cannot hide behind an empty-directory error.
				writeMigrationFixture(t, "internal/storage/schema/00002_next.sql", initialMigration)
				if action == "renamed" {
					writeMigrationFixture(t, "internal/storage/schema/00001_renamed.sql", initialMigration)
				}
				want = "published migration missing"
			case "appended":
				writeMigrationFixture(t, "internal/storage/schema/00002_next.sql", "-- +goose Up\nALTER TABLE settings ADD COLUMN name TEXT;\n")
			}
			err := checkMigrations(base)
			if want == "" && err != nil || want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
				t.Fatalf("checkMigrations() = %v, want %q", err, want)
			}
		})
	}
}

func TestMigrationValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"empty", nil, "no schema migrations"},
		{"invalid filename", map[string]string{"1_initial.sql": initialMigration}, "invalid migration filename"},
		{"zero version", map[string]string{"00000_initial.sql": initialMigration}, "invalid or duplicate migration version"},
		{"duplicate version", map[string]string{"00001_initial.sql": initialMigration, "00001_other.sql": initialMigration}, "invalid or duplicate migration version"},
		{"missing Up", map[string]string{"00001_initial.sql": "SELECT 1;\n"}, "migration must use goose transactions"},
		{"fake Up", map[string]string{"00001_initial.sql": "-- +goose Upgrade\nSELECT 1;\n"}, "migration must use goose transactions"},
		{"commented Up", map[string]string{"00001_initial.sql": "-- example: -- +goose Up\nSELECT 1;\n"}, "migration must use goose transactions"},
		{"nontransactional", map[string]string{"00001_initial.sql": "-- +goose NO TRANSACTION\n" + initialMigration}, "migration must use goose transactions"},
		{"nontransactional whitespace", map[string]string{"00001_initial.sql": "-- +goose NO\tTRANSACTION\n" + initialMigration}, "migration must use goose transactions"},
		{"nontransactional lowercase", map[string]string{"00001_initial.sql": "-- +goose no transaction\n" + initialMigration}, "migration must use goose transactions"},
		{"nontransactional mixed case", map[string]string{"00001_initial.sql": "-- +goose No Transaction\n" + initialMigration}, "migration must use goose transactions"},
		{"transactional", map[string]string{"00001_initial.sql": initialMigration}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.MkdirAll("internal/storage/schema", 0700); err != nil {
				t.Fatal(err)
			}
			for name, content := range tc.files {
				writeMigrationFixture(t, filepath.Join("internal/storage/schema", name), content)
			}
			err := checkMigrations("")
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("checkMigrations() = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestMigrationLineagePreservesPublishedGaps(t *testing.T) {
	for _, action := range []string{"unchanged gap", "append after gap", "backfill", "backfill and append"} {
		t.Run(action, func(t *testing.T) {
			t.Chdir(t.TempDir())
			fixtureGit(t, "init", "-q")
			writeMigrationFixture(t, initialPath, initialMigration)
			writeMigrationFixture(t, "internal/storage/schema/00003_published.sql", "-- +goose Up\nALTER TABLE settings ADD COLUMN published TEXT;\n")
			fixtureGit(t, "add", ".")
			fixtureGit(t, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "publish versions 1 and 3")
			base := fixtureGit(t, "rev-parse", "HEAD")
			backfill := action == "backfill" || action == "backfill and append"
			if backfill {
				writeMigrationFixture(t, "internal/storage/schema/00002_backfill.sql", "-- +goose Up\nALTER TABLE settings ADD COLUMN backfill TEXT;\n")
			}
			if action == "append after gap" || action == "backfill and append" {
				writeMigrationFixture(t, "internal/storage/schema/00004_next.sql", "-- +goose Up\nALTER TABLE settings ADD COLUMN next TEXT;\n")
			}
			err := checkMigrations(base)
			if backfill {
				if err == nil || !strings.Contains(err.Error(), "new migration must follow published version 00003") {
					t.Fatalf("backfilled migration passed append-only policy: %v", err)
				}
			} else if err != nil {
				t.Fatalf("unchanged published gap rejected: %v", err)
			}
		})
	}
}

func TestMigrationLineageAllowsExplicitAlphaReset(t *testing.T) {
	t.Chdir(t.TempDir())
	fixtureGit(t, "init", "-q")
	legacy := "internal/storage/migrations/00001_settings.sql"
	writeMigrationFixture(t, legacy, initialMigration)
	fixtureGit(t, "add", ".")
	fixtureGit(t, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "publish alpha")
	base := fixtureGit(t, "rev-parse", "HEAD")
	if err := os.RemoveAll("internal/storage/migrations"); err != nil {
		t.Fatal(err)
	}
	writeMigrationFixture(t, initialPath, initialMigration)
	if err := checkMigrations(base); err != nil {
		t.Fatalf("explicit new lineage rejected: %v", err)
	}
}
