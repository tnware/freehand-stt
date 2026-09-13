package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage"
	"runtime"
	"testing"
)

// A fresh temporary database has no native credential references. Never use the
// user's application database or mutate their credential vault in this test.
func TestManagedConnectionsCommitReloadAndMetadata(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("temporary LOCALAPPDATA storage harness is Windows-specific")
	}
	t.Setenv("LOCALAPPDATA", t.TempDir())
	store, err := storage.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	s, _, _, _ := transactionalService(false)
	s.store = store
	s.cfg = cfg
	i := testInstance()
	if err := SaveManagedInstances(s, []managedruntime.Instance{i}); err != nil {
		t.Fatal(err)
	}
	d := savedconnection.Details{ManagedInstanceID: i.ID, AuthenticationMode: config.AuthenticationModeNone, Headers: map[string]string{}}
	got, err := s.SaveSettings(SaveSettingsRequest{Settings: s.current(), ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Name: "Local", Uses: []savedconnection.Purpose{savedconnection.Transcription}, ActivateFor: savedconnection.Transcription, Details: &d}})
	if err != nil {
		t.Fatal(err)
	}
	id := got.SavedConnections.Selected[savedconnection.Transcription]
	if id == "" || got.ManagedInstanceID != i.ID || got.Model != i.Model || got.BaseURL != "" {
		t.Fatal("managed selection was not projected durably")
	}
	stale := got.Settings
	i.Model = "parakeet-tdt"
	if err := SaveManagedInstances(s, []managedruntime.Instance{i}); err != nil {
		t.Fatal(err)
	}
	got, err = s.SaveSettings(SaveSettingsRequest{Settings: stale})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != i.Model {
		t.Fatal("stale task draft overrode runtime model")
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Model != i.Model || reloaded.ManagedInstanceID != i.ID || reloaded.BaseURL != "" {
		t.Fatal("reload lost managed authority")
	}
	if err := SaveManagedInstances(s, nil); err == nil {
		t.Fatal("deleted referenced instance")
	}
	WithManagedRuntimes(readyManagedEndpoint, nil)(s)
	c, key, err := ConnectionResolver(s).ResolveSavedConnection(id)
	if err != nil {
		t.Fatal(err)
	}
	if c.Details.ManagedInstanceID != "" || c.Details.BaseURL != "http://127.0.0.1:43210/v1" || key != "" || c.HasCredential {
		t.Fatal("metadata resolution did not create credential-free transport")
	}
	stored, key, err := store.ResolveSavedConnection(id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Details.ManagedInstanceID != i.ID || stored.Details.BaseURL != "" || key != "" {
		t.Fatal("metadata resolution mutated saved details")
	}
	WithManagedRuntimes(nil, nil)(s)
	if _, _, err := ConnectionResolver(s).ResolveSavedConnection(id); !errors.Is(err, ErrManagedUnavailable) {
		t.Fatalf("stopped metadata fallback: %v", err)
	}
}
