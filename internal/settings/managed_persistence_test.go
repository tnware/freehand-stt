package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"reflect"
	"testing"
)

func TestGeneralSaveCannotOverwriteManagedInstances(t *testing.T) {
	s, _, _, _ := transactionalService(false)
	stale := s.current()
	s.cfg.ManagedRuntimes = []managedruntime.Instance{testInstance()}
	stale.HistoryEnabled = true
	got, err := s.SaveSettings(request(stale, ""))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ManagedRuntimes, s.cfg.ManagedRuntimes) || len(got.ManagedRuntimes) != 1 || !got.HistoryEnabled {
		t.Fatal("stale draft replaced inventory")
	}
}
func TestManagedInstanceCommitPublishesOutsideSaveLock(t *testing.T) {
	s, _, _, keys := transactionalService(false)
	keys.getErr = errors.New("locked")
	instances := []managedruntime.Instance{testInstance()}
	var published SettingsDTO
	WithManagedRuntimes(nil, func(v []managedruntime.Instance) { published = s.GetSettings(); v[0].Name = "mutated callback" })(s)
	if err := SaveManagedInstances(s, instances); err != nil {
		t.Fatal(err)
	}
	instances[0].Name = "mutated caller"
	if s.current().ManagedRuntimes[0].Name != "Local speech" || published.ManagedRuntimes[0].Name != "Local speech" {
		t.Fatal("inventory alias escaped")
	}
}
func TestManagedInstanceSaveFailureKeepsCommittedSnapshot(t *testing.T) {
	for _, mode := range []string{"disk", "recovery", "closed", "publication-busy", "save-busy", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _, _ := transactionalService(mode == "disk")
			before := s.current()
			instances := []managedruntime.Instance{testInstance()}
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
				instances[0].Model = "unknown"
			}
			if err := SaveManagedInstances(s, instances); err == nil {
				t.Fatal("unsafe save accepted")
			}
			if !reflect.DeepEqual(before, s.current()) {
				t.Fatal("failed commit published")
			}
		})
	}
}
func TestRecoveryReconcilesManagedInstances(t *testing.T) {
	s, log, _, _ := transactionalService(false)
	next := config.Default()
	next.ManagedRuntimes = []managedruntime.Instance{testInstance()}
	st := &recoveryStoreFake{storeFake: storeFake{log: log}, settings: next}
	s.store, s.loader = st, st
	var applied []managedruntime.Instance
	WithManagedRuntimes(nil, func(v []managedruntime.Instance) { applied = v; _ = s.GetSettings() })(s)
	if _, err := s.RetryConfiguration(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(applied, next.ManagedRuntimes) {
		t.Fatal("recovery did not publish inventory")
	}
	st.loadErr = errors.New("corrupt")
	_, _ = s.RetryConfiguration()
	if _, err := s.ResetConfiguration(); err != nil {
		t.Fatal(err)
	}
	if len(applied) != 0 {
		t.Fatal("reset kept runtime inventory")
	}
}
