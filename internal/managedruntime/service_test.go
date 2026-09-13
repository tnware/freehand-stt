package managedruntime

import (
	"errors"
	"testing"
	"time"
)

func TestSaveCallbackCanReadStatus(t *testing.T) {
	s := NewService(Options{Preferences: Defaults()})
	s.status.Supported = true
	s.save = func(p Preferences) error { _ = s.GetStatus(); return nil }
	done := make(chan error, 1)
	go func() { done <- s.SetPreferences(Defaults()) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("save callback deadlocked on runtime mutex")
	}
}
func TestPreferencesPersistenceFailClosed(t *testing.T) {
	s := NewService(Options{Directory: t.TempDir(), Preferences: Defaults(), SavePreferences: func(Preferences) error { return errors.New("secret path") }})
	s.status.Supported = true // Exercise the state machine independently of the host OS.
	if e, err := s.Resolve(); err != nil || e.Enabled {
		t.Fatalf("disabled: %+v %v", e, err)
	}
	p := Defaults()
	p.Enabled = true
	if s.SetPreferences(p) == nil {
		t.Fatal("ignored persistence failure")
	}
	if s.GetStatus().Enabled {
		t.Fatal("applied unpersisted preference")
	}
	s.save = nil
	if err := s.SetPreferences(p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(); err == nil {
		t.Fatal("enabled unavailable must fail closed")
	}
	st := s.GetStatus()
	if st.Models == nil || len(st.Models) != 0 || st.SelectedModel != "nemotron-3.5" || !st.Realtime {
		t.Fatalf("status %+v", st)
	}
}
