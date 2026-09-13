package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"reflect"
	"testing"
)

func TestGeneralSaveCannotOverwriteManagedPreferences(t *testing.T) {
	s, _, _, _ := transactionalService(false)
	stale := s.current()
	s.cfg.ManagedRuntime.Enabled = true
	stale.HistoryEnabled = true
	got, err := s.SaveSettings(request(stale, ""))
	if err != nil {
		t.Fatal(err)
	}
	if !got.ManagedRuntime.Enabled || !got.HistoryEnabled {
		t.Fatal("general save overwrote managed preference from stale DTO")
	}
}

func TestManagedPreferencePersistenceIsNarrowAndPublishesCommittedPreferences(t *testing.T) {
	s, log, _, _ := transactionalService(false)
	before := s.current()
	var applied []managedruntime.Preferences
	WithManagedRuntime(nil, func(p managedruntime.Preferences) {
		applied = append(applied, p)
		_ = s.GetSettings()
	})(s)
	var published SettingsDTO
	s.settingsChanged = func(v SettingsDTO) { published = v; _ = s.GetSettings() }
	p := before.ManagedRuntime
	p.Enabled = true
	if err := SaveManagedPreferences(s, p); err != nil {
		t.Fatal(err)
	}
	expected := before
	expected.ManagedRuntime = p
	if !reflect.DeepEqual(s.current(), expected) || published.ManagedRuntime != p {
		t.Fatal("managed preference commit lost unrelated settings or publication")
	}
	if len(applied) != 1 || applied[0] != p {
		t.Fatal("persistence did not publish the authoritative committed preferences")
	}
	if !reflect.DeepEqual(*log, []string{"config:save"}) {
		t.Fatalf("unexpected native/credential writes: %v", *log)
	}
}

func TestManagedPreferencePersistenceGuardsAndFailure(t *testing.T) {
	for _, mode := range []string{"failure", "recovery", "closed", "publication-busy", "save-busy", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _, _ := transactionalService(mode == "failure")
			before := s.current()
			p := before.ManagedRuntime
			p.Enabled = true
			switch mode {
			case "recovery":
				s.configuration.RecoveryRequired = true
			case "closed":
				s.closed.Store(true)
			case "publication-busy":
				s.publicationMu.Lock()
				defer s.publicationMu.Unlock()
			case "save-busy":
				s.saveMu.Lock()
				defer s.saveMu.Unlock()
			case "invalid":
				p.Model = "invalid"
			}
			if err := SaveManagedPreferences(s, p); err == nil {
				t.Fatal("unsafe save accepted")
			}
			if !reflect.DeepEqual(s.current(), before) {
				t.Fatal("failed save changed active preferences")
			}
		})
	}
}

func TestRecoveryReconcilesManagedPreferences(t *testing.T) {
	s, log, _, _ := transactionalService(false)
	next := config.Default()
	next.ManagedRuntime.Enabled = true
	store := &recoveryStoreFake{storeFake: storeFake{log: log}, settings: next}
	s.store, s.loader = store, store
	var applied []managedruntime.Preferences
	WithManagedRuntime(nil, func(p managedruntime.Preferences) { applied = append(applied, p); _ = s.GetSettings() })(s)
	if _, err := s.RetryConfiguration(); err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0] != next.ManagedRuntime {
		t.Fatal("retry did not reconcile runtime")
	}
	store.loadErr = errors.New("corrupt")
	if _, err := s.RetryConfiguration(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResetConfiguration(); err != nil {
		t.Fatal(err)
	}
	if len(applied) != 2 || applied[1] != config.Default().ManagedRuntime {
		t.Fatal("reset did not disable runtime")
	}
}
