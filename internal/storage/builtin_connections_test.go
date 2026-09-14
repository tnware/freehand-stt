package storage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func TestBuiltInConnectionCannotBeEditedOrDeleted(t *testing.T) {
	for _, action := range []savedconnection.Action{savedconnection.Update, savedconnection.Rename, savedconnection.Duplicate, savedconnection.Delete} {
		t.Run(string(action), func(t *testing.T) {
			s := testStore(t)
			v := loadStore(t, s)
			v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
			if err := s.Save(v); err != nil {
				t.Fatal(err)
			}
			c := s.ConnectionCatalog().Entries[0]
			if !c.BuiltIn {
				t.Fatal("automatic row lacks runtime ownership metadata")
			}
			d := savedconnection.Details{ManagedInstanceID: "local"}
			_, err := s.BeginConnectionChange(savedconnection.Change{Action: action, ID: c.ID, Name: "Edited", Details: &d, Uses: []savedconnection.Purpose{savedconnection.Voice}}, v)
			if err == nil {
				t.Fatalf("runtime-owned connection accepted %s", action)
			}
			if !reflect.DeepEqual(s.ConnectionCatalog().Entries[0], c) {
				t.Fatal("failed edit changed catalog")
			}
		})
	}
}

func TestBuiltInConnectionRemovalFollowsExplicitInventoryRemoval(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	id := s.ConnectionCatalog().Entries[0].ID
	v = selectConnection(t, s, v, savedconnection.Voice, id)
	removed := v
	removed.ManagedRuntimes = nil
	if err := s.Save(removed); err == nil {
		t.Fatal("selected runtime was removed")
	}
	v = selectConnection(t, s, v, savedconnection.Voice, "")
	v.ManagedRuntimes = nil
	if err := s.Save(v); err != nil {
		t.Fatalf("unused built-in prevented explicit inventory removal: %v", err)
	}
	loadStore(t, reopen(t, s))
	if len(s.ConnectionCatalog().Entries) != 0 {
		t.Fatal("removed runtime left a built-in row")
	}
}

func TestBuiltInsDoNotConsumeManualConnectionCapacity(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	d := savedconnection.Extract(v, savedconnection.Voice)
	d.BaseURL = "https://example.test/v1"
	for n := 0; n < savedconnection.Limit(savedconnection.Voice); n++ {
		var err error
		v, err = s.BeginConnectionChange(savedconnection.Change{Action: savedconnection.Create, Name: fmt.Sprintf("Manual %d", n), Details: &d, Uses: []savedconnection.Purpose{savedconnection.Voice}}, v)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Save(v); err != nil {
			t.Fatal(err)
		}
	}
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	v, err := s.Load()
	if err != nil {
		t.Fatalf("automatic row exceeded manual use limit on reload: %v", err)
	}
	v = selectConnection(t, s, v, savedconnection.Voice, savedconnection.BuiltInID("local"))
	if v.VoiceTranscription.ManagedInstanceID != "local" {
		t.Fatal("full manual catalog prevented built-in selection")
	}
}

func TestConfiguredRuntimeAutomaticallyJoinsConnectionCatalog(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	instance := managedruntime.Instance{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}
	v.ManagedRuntimes = []managedruntime.Instance{instance}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	catalog := s.ConnectionCatalog()
	if len(catalog.Entries) != 1 {
		t.Fatalf("configured runtime needs an automatic connection, got %#v", catalog)
	}
	c := catalog.Entries[0]
	if c.ID == "" || c.Details.ManagedInstanceID != instance.ID || !c.Supports(savedconnection.Voice) || !c.Supports(savedconnection.Transcription) || c.Supports(savedconnection.Cleanup) || c.HasCredential || c.Details.BaseURL != "" {
		t.Fatalf("incorrect automatic runtime connection: %#v", c)
	}
	if len(catalog.Selected) != 0 {
		t.Fatal("runtime setup selected a task implicitly")
	}
	v = selectConnection(t, s, v, savedconnection.Voice, c.ID)
	if v.VoiceTranscription.ManagedInstanceID != instance.ID || v.VoiceTranscription.Model != instance.Model || v.VoiceTranscription.BaseURL != "" {
		t.Fatal("automatic connection did not use ordinary selection projection")
	}
	s = reopen(t, s)
	loadStore(t, s)
	reloaded := s.ConnectionCatalog()
	if !reflect.DeepEqual(reloaded.Entries, catalog.Entries) || reloaded.Selected[savedconnection.Voice] != c.ID {
		t.Fatal("automatic identity or selection changed after reopen")
	}
}
