package settings

import (
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/managedruntime"
)

func handshakeWait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("publication barrier timed out")
	}
}

func TestManagedSavePublishesAuthoritativeHandshake(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "failed-save"
		}
		t.Run(name, func(t *testing.T) {
			s, _, _, _ := transactionalService(false)
			initial := s.current().ManagedRuntime
			proposed := initial
			proposed.Enabled = true
			publicationEntered, releasePublication := make(chan struct{}), make(chan struct{})
			saveEntered, releaseSave := make(chan struct{}), make(chan struct{})
			var runtime *managedruntime.Service
			runtime = managedruntime.NewService(managedruntime.Options{Preferences: initial, SavePreferences: func(p managedruntime.Preferences) error {
				close(saveEntered)
				<-releaseSave
				return SaveManagedPreferences(s, p)
			}})
			if !runtime.GetStatus().Supported {
				t.Skip("real managed runtime handshake requires Windows x64")
			}
			first := true
			WithManagedRuntime(runtime.ResolveFor, func(p managedruntime.Preferences) {
				// Reading settings proves the publication is outside saveMu.
				_ = s.GetSettings()
				if first {
					first = false
					close(publicationEntered)
					<-releasePublication
				}
				managedruntime.ApplyPreferences(runtime, p)
			})(s)
			// An ordinary settings publication has committed P0 but is paused before
			// ApplyPreferences. SetPreferences then enters its lock-free saving phase.
			published := make(chan struct{})
			publicationResult := make(chan error, 1)
			go func() {
				_, err := s.SaveSettings(request(s.current(), ""))
				publicationResult <- err
				close(published)
			}()
			handshakeWait(t, publicationEntered)
			saved := make(chan error, 1)
			go func() { saved <- runtime.SetPreferences(proposed) }()
			handshakeWait(t, saveEntered)
			close(releasePublication)
			handshakeWait(t, published) // P0 is now staged and publicationMu is free.
			if err := <-publicationResult; err != nil {
				t.Fatal(err)
			}
			s.store.(*storeFake).fail = fail
			close(releaseSave)
			var err error
			select {
			case err = <-saved:
			case <-time.After(2 * time.Second):
				t.Fatal("save handshake deadlocked")
			}
			expected := proposed
			if fail {
				expected = initial
				if err == nil {
					t.Fatal("failed persistence accepted")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if got := runtime.GetPreferences(); got != expected {
				t.Fatalf("runtime applied stale publication: got %+v want %+v", got, expected)
			}
			if got := s.current().ManagedRuntime; got != expected {
				t.Fatalf("durable snapshot differs: %+v", got)
			}
		})
	}
}
