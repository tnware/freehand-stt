package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
)

func TestCurrentDatabaseHasIndependentPath(t *testing.T) {
	s, err := NewStore() // Construction must not open storage or access the vault.
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if filepath.Base(s.path) != "freehand.db" {
		t.Fatalf("current database still shares alpha path: %s", filepath.Base(s.path))
	}
}

func TestFreshStartIgnoresAlphaArtifacts(t *testing.T) {
	dir := t.TempDir()
	vault := &memoryVault{values: map[string]string{"stt-api-key": "alpha-key", "sqlite-stt-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": "alpha-reference"}}
	artifacts := map[string][]byte{
		"settings.db":   []byte("old alpha database must not be opened"),
		"settings.json": []byte(`{"language":"ja","setupCompleted":true}`),
	}
	for name, data := range artifacts {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := newStore(filepath.Join(dir, "freehand.db"), vault)
	defer s.Close()
	got := loadStore(t, s)
	if !reflect.DeepEqual(got, config.Default()) {
		t.Fatal("fresh baseline imported alpha settings")
	}
	if len(s.ConnectionCatalog().Entries) != 0 {
		t.Fatal("fresh baseline seeded alpha connections")
	}
	for _, view := range []credential.Store{s.STTCredentials(), s.CleanupCredentials(), s.VoiceCredentials(), s.SpeechCredentials()} {
		if _, err := view.Get(); !errors.Is(err, credential.ErrNotFound) {
			t.Fatalf("fresh baseline reused alpha credential: %v", err)
		}
	}
	for name, want := range artifacts {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("alpha artifact %s modified: %v", name, err)
		}
	}
	if len(vault.values) != 2 {
		t.Fatal("alpha credentials altered")
	}
}

func TestCurrentDatabaseRejectsAlphaIdentity(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	if _, err := s.db.Exec("PRAGMA application_id=1179796804"); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if _, err := s.Load(); config.LoadFailureFor(err).Kind != "foreign_database" {
		t.Fatalf("alpha identity accepted: %v", err)
	}
}

func TestCredentialAccountsDoNotAcceptAlphaNamespaces(t *testing.T) {
	for _, account := range []string{"stt-api-key", "post-processing-api-key", "text-to-speech-api-key", "sqlite-connection-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sqlite-stt-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		if connectionAccount(account) {
			t.Errorf("alpha credential reference accepted: %s", account)
		}
	}
}

func TestUnselectedTaskCannotStageCredential(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	defer s.DiscardCredentialChanges()
	if err := s.STTCredentials().Set("unowned-key"); err == nil {
		t.Fatal("staged credential without a selected connection")
	}
	if len(s.vault.(*memoryVault).values) != 0 {
		t.Fatal("unowned credential reached vault")
	}
}

func TestBackupRetentionLeavesAlphaBackupsUntouched(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	dir := filepath.Join(filepath.Dir(s.path), "backups")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dir, "settings-alpha.db")
	if err := os.WriteFile(old, []byte("alpha backup"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := s.backup(context.Background(), s.db); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(old)
	if err != nil || string(data) != "alpha backup" {
		t.Fatalf("current retention touched alpha backup: %v", err)
	}
}
