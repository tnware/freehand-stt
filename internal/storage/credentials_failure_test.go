package storage

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

func credentialFailureStore(t *testing.T) (*Store, config.Settings, string) {
	t.Helper()
	s := testStore(t)
	v := loadStore(t, s)
	details := savedconnection.Extract(v, savedconnection.Transcription)
	details.BaseURL = "https://credential-fixture.example/v1"
	details.AuthenticationMode = config.AuthenticationModeAPIKey
	v = createSelectedConnection(t, s, v, "Credential fixture", savedconnection.Transcription, details)
	v.Model = "fixture-model"
	commitFixtureCredential(t, s, v, "original-key")
	return s, loadStore(t, s), s.ConnectionCatalog().Selected[savedconnection.Transcription]
}

func commitFixtureCredential(t *testing.T, s *Store, v config.Settings, value string) {
	t.Helper()
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	defer s.DiscardCredentialChanges()
	if err := s.STTCredentials().Set(value); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
}

func pendingCredentialCount(t *testing.T, s *Store) int64 {
	t.Helper()
	count, err := dbgen.New(s.db).CountCredentialGC(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func TestCredentialSetFailurePreservesCommittedConnection(t *testing.T) {
	s, before, id := credentialFailureStore(t)
	vault := s.vault.(*memoryVault)
	connection, key, err := s.ResolveSavedConnection(id)
	if err != nil {
		t.Fatal(err)
	}
	vault.failSet = true
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("rejected-key"); err == nil {
		t.Fatal("vault write failure was accepted")
	}
	if pendingCredentialCount(t, s) != 1 {
		t.Fatal("failed vault write lost its durable cleanup intent")
	}
	current, currentKey, err := s.ResolveSavedConnection(id)
	if err != nil || !reflect.DeepEqual(current, connection) || currentKey != key {
		t.Fatal("failed vault write changed the committed connection or key")
	}
	s.DiscardCredentialChanges()
	if pendingCredentialCount(t, s) != 0 || len(vault.values) != 1 {
		t.Fatal("discard did not reclaim the failed write's cleanup intent")
	}
	if got := loadStore(t, s); !reflect.DeepEqual(got, before) {
		t.Fatal("failed vault write changed durable settings")
	}
	vault.failSet = false
	commitFixtureCredential(t, s, before, "retry-key")
	if _, got, err := s.ResolveSavedConnection(id); err != nil || got != "retry-key" {
		t.Fatal("credential replacement did not recover after the vault became available")
	}
}

func TestCredentialDeleteFailureRetriesWithoutReclaimingSharedKeys(t *testing.T) {
	s, v, id := credentialFailureStore(t)
	vault := s.vault.(*memoryVault)
	sharedAccount := s.refs["stt"]
	duplicate, err := s.BeginConnectionChange(savedconnection.Change{
		Action: savedconnection.Duplicate, ID: id, Name: "Retained copy",
	}, v)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(duplicate); err != nil {
		t.Fatal(err)
	}
	s.DiscardCredentialChanges()
	copyID := ""
	for _, entry := range s.ConnectionCatalog().Entries {
		if entry.Name == "Retained copy" {
			copyID = entry.ID
		}
	}
	if copyID == "" {
		t.Fatal("duplicate connection was not committed")
	}
	commitFixtureCredential(t, s, v, "obsolete-key")
	obsoleteAccount := s.refs["stt"]
	if vault.values[sharedAccount] != "original-key" {
		t.Fatal("replacement reclaimed a key still used by an inactive connection")
	}
	vault.failDelete = true
	v.Language = "de"
	commitFixtureCredential(t, s, v, "current-key")
	if _, key, err := s.ResolveSavedConnection(id); err != nil || key != "current-key" {
		t.Fatal("cleanup failure prevented the committed credential from taking effect")
	}
	if pendingCredentialCount(t, s) != 1 || vault.values[obsoleteAccount] != "obsolete-key" {
		t.Fatal("failed deletion did not retain the obsolete key for cleanup retry")
	}
	vault.failDelete = false
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, v) {
		t.Fatal("cleanup retry changed committed settings")
	}
	if pendingCredentialCount(t, s) != 0 || len(vault.values) != 2 {
		t.Fatal("reopen did not reclaim exactly the unreferenced key")
	}
	if _, present := vault.values[obsoleteAccount]; present {
		t.Fatal("obsolete key survived successful cleanup")
	}
	for connectionID, want := range map[string]string{id: "current-key", copyID: "original-key"} {
		if _, key, err := s.ResolveSavedConnection(connectionID); err != nil || key != want {
			t.Fatal("cleanup changed a surviving connection's credential")
		}
	}
}

func TestCredentialCleanupBackpressureRecoversAfterVaultDeletion(t *testing.T) {
	s, before, id := credentialFailureStore(t)
	vault := s.vault.(*memoryVault)
	vault.failDelete = true
	for range maxPendingCredentials {
		if err := s.BeginCredentialChanges(); err != nil {
			t.Fatal(err)
		}
		if err := s.STTCredentials().Set("abandoned-key"); err != nil {
			t.Fatal(err)
		}
		s.DiscardCredentialChanges()
	}
	if pendingCredentialCount(t, s) != maxPendingCredentials {
		t.Fatal("fixture did not reach the pending cleanup limit")
	}
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("over-capacity-key"); err == nil {
		t.Fatal("credential staging exceeded the cleanup limit")
	}
	s.DiscardCredentialChanges()
	if len(vault.values) != maxPendingCredentials+1 || pendingCredentialCount(t, s) != maxPendingCredentials {
		t.Fatal("rejected staging grew the vault or cleanup queue")
	}
	if _, key, err := s.ResolveSavedConnection(id); err != nil || key != "original-key" {
		t.Fatal("cleanup backpressure changed the active credential")
	}
	vault.failDelete = false
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, before) {
		t.Fatal("cleanup backpressure changed durable settings")
	}
	if pendingCredentialCount(t, s) != 0 || len(vault.values) != 1 {
		t.Fatal("reopen did not clear the recovered cleanup backlog")
	}
	commitFixtureCredential(t, s, before, "recovered-key")
	if _, key, err := s.ResolveSavedConnection(id); err != nil || key != "recovered-key" {
		t.Fatal("credential staging remained blocked after cleanup recovered")
	}
}
