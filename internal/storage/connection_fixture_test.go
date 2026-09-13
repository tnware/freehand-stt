package storage

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func createSelectedConnection(t *testing.T, s *Store, v config.Settings, name string, purpose savedconnection.Purpose, d savedconnection.Details, uses ...savedconnection.Purpose) config.Settings {
	t.Helper()
	if len(uses) == 0 {
		uses = []savedconnection.Purpose{purpose}
	}
	next, err := s.BeginConnectionChange(savedconnection.Change{Action: savedconnection.Create, Name: name, Uses: uses, Details: &d, ActivateFor: purpose}, v)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(next); err != nil {
		t.Fatal(err)
	}
	return loadStore(t, s)
}

func selectConnection(t *testing.T, s *Store, v config.Settings, purpose savedconnection.Purpose, id string) config.Settings {
	t.Helper()
	next, err := s.BeginConnectionChange(savedconnection.Change{Action: savedconnection.Select, Purpose: purpose, ID: id}, v)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(next); err != nil {
		t.Fatal(err)
	}
	return loadStore(t, s)
}
