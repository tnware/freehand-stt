package managedruntime

import (
	"errors"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestManagerSaveCallbackCanReadStatus(t *testing.T) {
	initial := Instance{ID: "voice", Name: "Voice", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{initial}})
	m.save = func([]Instance) error { _ = m.GetInstances(); return nil }
	next := initial
	next.Name = "Renamed"
	done := make(chan error, 1)
	go func() { done <- m.SetInstance(next) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("save callback deadlocked on runtime mutex")
	}
}

func TestManagerPersistenceFailsClosed(t *testing.T) {
	initial := Instance{ID: "voice", Name: "Voice", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{initial}, SaveInstances: func([]Instance) error { return errors.New("private path") }})
	next := initial
	next.Model = "parakeet-tdt"
	if m.SetInstance(next) == nil {
		t.Fatal("ignored persistence failure")
	}
	if got := m.GetInstances(); len(got) != 1 || got[0].Instance != initial {
		t.Fatal("applied unpersisted instance")
	}
	if _, err := m.ResolveFor(next, compatibility.Transcription); err == nil {
		t.Fatal("resolved unpersisted instance")
	}
	m.save = nil
	if err := m.SetInstance(next); err != nil {
		t.Fatal(err)
	}
	if got := m.GetInstances(); len(got) != 1 || got[0].Instance != next {
		t.Fatal("lost persisted instance")
	}
	if _, err := m.ResolveFor(next, compatibility.Transcription); err == nil {
		t.Fatal("resolved unavailable runtime")
	}
}
