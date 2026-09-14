package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupPublishesOnlyCompletedSnapshots(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	// Hold the only connection so the backup cannot copy a single page yet.
	held, err := s.db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	waits := s.db.Stats().WaitCount
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := s.backup(ctx, s.db)
		done <- err
	}()
	// Pool admission provides a barrier after the candidate file is created.
	for s.db.Stats().WaitCount == waits {
		select {
		case err := <-done:
			t.Fatalf("backup returned before waiting for the source: %v", err)
		case <-ctx.Done():
			t.Fatal("backup did not wait for the occupied source connection")
		case <-time.After(time.Millisecond):
		}
	}
	dir := filepath.Join(filepath.Dir(s.path), "freehand-backups")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".db") {
			t.Errorf("unfinished backup published as %q", entry.Name())
		}
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled backup = %v", err)
	}
	entries, err = os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("canceled backup left files: %v, %v", entries, err)
	}
}

func TestBackupRetentionPreservesCompletedSnapshots(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	var previous []string
	for i := range retainedBackups {
		path, err := s.backup(t.Context(), s.db)
		if err != nil {
			t.Fatal(err)
		}
		previous = append(previous, path)
		stamp := time.Unix(int64(i+1), 0)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Dir(previous[0])
	// Interrupted work and unrelated files must not consume successful-backup slots.
	for _, name := range []string{"freehand-interrupted.tmp", "other.db"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("preserve"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	v.Language = "de"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	latest, err := s.backup(t.Context(), s.db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(previous[0]); !os.IsNotExist(err) {
		t.Fatalf("oldest backup was not pruned: %v", err)
	}
	for _, path := range append(previous[1:], latest, filepath.Join(dir, "freehand-interrupted.tmp"), filepath.Join(dir, "other.db")) {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("retention removed a needed file %q: %v", filepath.Base(path), err)
		}
	}
	restored := testStore(t)
	loadStore(t, restored)
	if err := restored.RestoreBackup(latest); err != nil {
		t.Fatal(err)
	}
	if got := loadStore(t, restored); got.Language != "de" {
		t.Fatalf("published backup did not preserve the latest settings: %q", got.Language)
	}
}
