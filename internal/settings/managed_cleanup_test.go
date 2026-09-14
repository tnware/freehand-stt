package settings

import (
	"errors"
	"reflect"
	"runtime"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage"
)

func TestManagedCleanupCommitReloadAndRequest(t *testing.T) {
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
	s, log, _, keys := transactionalService(false)
	s.store, s.cfg = store, cfg
	i := managedruntime.Instance{ID: "cleanup", Name: "Local cleanup", Provider: managedruntime.LlamaCPP, Model: "s1-mini"}
	if err := SaveManagedInstances(s, []managedruntime.Instance{i}); err != nil {
		t.Fatal(err)
	}
	draft := s.current()
	draft.PostProcessing.Enabled = true
	got, err := s.SaveSettings(SaveSettingsRequest{Settings: draft, ConnectionChange: &savedconnection.Change{Action: savedconnection.Select, ID: savedconnection.BuiltInID(i.ID), Purpose: savedconnection.Cleanup}})
	if err != nil {
		t.Fatal(err)
	}
	draft = got.Settings
	draft.PostProcessing.Enabled = true
	got, err = s.SaveSettings(SaveSettingsRequest{Settings: draft})
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	p := reloaded.PostProcessing
	if !p.Enabled || p.ManagedInstanceID != i.ID || p.Model != "s1-mini" || p.Preset != config.PostProcessingPresetS1Mini || p.BaseURL != "" || p.AllowInsecureHTTP || !reflect.DeepEqual(p, got.PostProcessing) {
		t.Fatal("enabled managed cleanup did not persist without transport")
	}
	if err := config.Validate(reloaded); err != nil {
		t.Fatal(err)
	}
	s.cfg = reloaded
	s.processKeys = keys
	keys.getErr = errors.New("locked")
	*log = nil
	WithManagedRuntimes(readyManagedEndpoint, nil)(s)
	profile, err := RequestProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if profile.PostProcessingUnavailable != nil || profile.PostProcessingCredential != "" || len(*log) != 0 {
		t.Fatal("managed cleanup unavailable or touched manual credentials")
	}
	resolved := profile.Settings.PostProcessing
	if resolved.BaseURL != "http://127.0.0.1:43210/v1" || resolved.Model != "served-model" || !resolved.AllowInsecureHTTP || resolved.ManagedInstanceID != i.ID {
		t.Fatal("request did not resolve managed cleanup")
	}
	if err := config.ValidatePostProcessing(resolved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.current().PostProcessing, p) {
		t.Fatal("request resolution mutated durable cleanup")
	}
}
