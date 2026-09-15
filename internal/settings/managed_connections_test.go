package settings

import (
	"errors"
	"reflect"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage"
	"runtime"
	"testing"
)

func TestManagedModelSwitchReconcilesVoiceMode(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("temporary LOCALAPPDATA storage harness is Windows-specific")
	}
	for _, tc := range []struct {
		name         string
		managedVoice bool
		realtime     bool
	}{
		{"managed realtime", true, true},
		{"managed completed", true, false},
		{"independent manual realtime", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			s.store, s.cfg = store, cfg
			i := testInstance()
			if err := SaveManagedInstances(s, []managedruntime.Instance{i}); err != nil {
				t.Fatal(err)
			}
			for _, purpose := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Voice} {
				change := savedconnection.Change{Action: savedconnection.Select, ID: savedconnection.BuiltInID(i.ID), Purpose: purpose}
				if purpose == savedconnection.Voice && !tc.managedVoice {
					change = savedconnection.Change{
						Action: savedconnection.Create, Name: "Independent Voice",
						Uses: []savedconnection.Purpose{purpose}, ActivateFor: purpose,
						Details: &savedconnection.Details{BaseURL: "https://voice.example.test/v1", CompatibilityProfile: compatibility.NeMoSpeechV1, AuthenticationMode: config.AuthenticationModeNone},
					}
				}
				if _, err := s.SaveSettings(SaveSettingsRequest{Settings: s.current(), ConnectionChange: &change}); err != nil {
					t.Fatal(err)
				}
			}
			next := s.current()
			next.VoiceTranscription.Realtime = tc.realtime
			next.VoiceTranscription.Captions = false
			next.VoiceTranscription.TimeoutSeconds = 123
			next.TranscriptionTimeoutSeconds = 234
			if !tc.managedVoice {
				next.VoiceTranscription.Language = "fr-FR"
				next.VoiceTranscription.Model = "manual-model"
				next.VoiceTranscription.ModelProfile = modelprofile.Nemotron35
			}
			if _, err := s.SaveSettings(SaveSettingsRequest{Settings: next}); err != nil {
				t.Fatal(err)
			}
			before := s.current()
			var saveErr error
			manager := managedruntime.NewManager(managedruntime.ManagerOptions{
				Directory: t.TempDir(), Instances: before.ManagedRuntimes,
				SaveInstances: func(instances []managedruntime.Instance) error {
					saveErr = SaveManagedInstances(s, instances)
					return saveErr
				},
			})
			defer manager.ServiceShutdown()
			WithManagedInventory(func(instances []managedruntime.Instance) (*managedruntime.InventoryReservation, error) {
				return managedruntime.ReserveInstances(manager, instances)
			})(s)
			i.Name = "Renamed speech"
			if err := manager.SetInstance(i); err != nil {
				t.Fatal(err)
			}
			if s.current().VoiceTranscription.Realtime != tc.realtime {
				t.Fatal("unrelated runtime edit changed Voice mode")
			}
			for _, model := range []string{"parakeet-tdt", "nemotron-3.5"} {
				i.Model = model
				if err := manager.SetInstance(i); err != nil {
					t.Fatalf("switch to %s: %v; persistence: %v", model, err, saveErr)
				}
				wantVoice := before.VoiceTranscription
				if tc.managedVoice {
					_, contract, err := config.ManagedContract(s.current(), i.ID, compatibility.Transcription)
					if err != nil {
						t.Fatal(err)
					}
					wantVoice.Model, wantVoice.ModelProfile = model, contract.ModelProfile
					wantVoice.Realtime = false
				}
				reloaded, err := store.Load()
				if err != nil {
					t.Fatal(err)
				}
				for _, got := range []config.Settings{s.current(), reloaded} {
					if !reflect.DeepEqual(got.VoiceTranscription, wantVoice) || got.Language != before.Language || got.TranscriptionTimeoutSeconds != before.TranscriptionTimeoutSeconds || got.Model != model || got.ManagedInstanceID != i.ID {
						t.Fatalf("model switch lost task settings: voice=%+v files=%s/%s", got.VoiceTranscription, got.Model, got.Language)
					}
				}
				if manager.GetInstances()[0].Instance != i {
					t.Fatal("runtime inventory did not publish the committed model")
				}
			}
		})
	}
}

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
	got, err := s.SaveSettings(SaveSettingsRequest{Settings: s.current(), ConnectionChange: &savedconnection.Change{Action: savedconnection.Select, ID: savedconnection.BuiltInID(i.ID), Purpose: savedconnection.Transcription}})
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
