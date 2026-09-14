//go:build windows

package storage

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/settings"
)

func TestSettingsRejectsIndirectManagedAliases(t *testing.T) {
	for _, action := range []savedconnection.Action{savedconnection.Update, savedconnection.Duplicate} {
		t.Run(string(action), func(t *testing.T) {
			s, svc := connectionService(t)
			i := managedruntime.Instance{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}
			if err := settings.SaveManagedInstances(svc, []managedruntime.Instance{i}); err != nil {
				t.Fatal(err)
			}
			v := loadStore(t, s)
			d := savedconnection.Extract(v, savedconnection.Voice)
			d.BaseURL = "https://example.test/v1"
			if action == savedconnection.Duplicate {
				d = savedconnection.Details{ManagedInstanceID: i.ID}
			}
			v = createSelectedConnection(t, s, v, "Existing", savedconnection.Voice, d)
			id := s.ConnectionCatalog().Selected[savedconnection.Voice]
			if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Select, ID: id, Purpose: savedconnection.Voice}}); err != nil {
				t.Fatal(err)
			}
			d = savedconnection.Details{ManagedInstanceID: i.ID}
			_, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: action, ID: id, Name: "New alias", Details: &d, Uses: []savedconnection.Purpose{savedconnection.Voice}}})
			if err == nil {
				t.Fatal("renderer created a managed alias through", action)
			}
		})
	}
}

func TestSettingsRejectsNewManagedAliases(t *testing.T) {
	s, svc := connectionService(t)
	i := managedruntime.Instance{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}
	if err := settings.SaveManagedInstances(svc, []managedruntime.Instance{i}); err != nil {
		t.Fatal(err)
	}
	before := s.ConnectionCatalog()
	d := savedconnection.Details{ManagedInstanceID: i.ID}
	_, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Name: "Unnecessary alias", Details: &d, Uses: []savedconnection.Purpose{savedconnection.Voice}}})
	if err == nil {
		t.Fatal("settings still creates user-managed aliases instead of selecting the automatic connection")
	}
	if len(s.ConnectionCatalog().Entries) != len(before.Entries) {
		t.Fatal("rejected alias changed catalog")
	}
}
