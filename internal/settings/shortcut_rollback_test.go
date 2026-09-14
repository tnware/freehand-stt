package settings

import (
	"errors"
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/shortcut"
)

type rollbackGlobals struct {
	active   map[string]bool
	rejected string
	attempts []string
}

func (g *rollbackGlobals) Register(value string, _ func()) error {
	g.attempts = append(g.attempts, value)
	if value == g.rejected || g.active[value] {
		return errors.New("OS conflict")
	}
	g.active[value] = true
	return nil
}

func (g *rollbackGlobals) Unregister(value string) error {
	delete(g.active, value)
	return nil
}

type rollbackHold struct {
	value    string
	starts   int
	rejected string
}

func (h *rollbackHold) Start(value string) error {
	h.starts++
	return h.Configure(value)
}

func (h *rollbackHold) Configure(value string) error {
	if value != "" && value == h.rejected {
		return errors.New("hold unavailable")
	}
	h.value = value
	return nil
}

type shortcutRollbackStore struct {
	saved         config.Settings
	beforeFailure func(config.Settings)
	failure       error
}

func (s *shortcutRollbackStore) Save(next config.Settings) error {
	if s.failure != nil {
		s.beforeFailure(next)
		return s.failure
	}
	s.saved = next
	return nil
}

func TestSaveSettingsReplacementFailureRestoresDegradedShortcutSnapshot(t *testing.T) {
	testShortcutSnapshotRollback(t, false, true, false)
}

func TestSaveSettingsReplacementFailureRestoresUnavailableHoldSnapshot(t *testing.T) {
	testShortcutSnapshotRollback(t, false, false, true)
}

func TestSaveSettingsClearFailureRestoresShortcutSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		toggleUnavailable, holdUnavailable bool
	}{
		{"degraded-toggle", true, false},
		{"degraded-hold", false, true},
		{"fully-bound", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testShortcutSnapshotRollback(t, true, tc.toggleUnavailable, tc.holdUnavailable)
		})
	}
}

func testShortcutSnapshotRollback(t *testing.T, clear, toggleUnavailable, holdUnavailable bool) {
	t.Helper()
	old := config.Default()
	old.ToggleShortcut = "Ctrl+Shift+Space"
	old.ShowShortcut = "Ctrl+Shift+D"
	old.HoldShortcut = "Ctrl+Super"
	globals := &rollbackGlobals{active: map[string]bool{}}
	hold := &rollbackHold{}
	if toggleUnavailable {
		globals.rejected = old.ToggleShortcut
	}
	if holdUnavailable {
		hold.rejected = old.HoldShortcut
	}
	controller := shortcut.New(globals, hold, func() {}, func() {})
	if err := controller.Start(old); (err != nil) != (toggleUnavailable || holdUnavailable) {
		t.Fatalf("unexpected startup result: %v", err)
	}
	priorGlobals := map[string]bool{old.ShowShortcut: true}
	if !toggleUnavailable {
		priorGlobals[old.ToggleShortcut] = true
	}
	priorHold := old.HoldShortcut
	if holdUnavailable {
		priorHold = ""
	}
	if !reflect.DeepEqual(globals.active, priorGlobals) || hold.value != priorHold {
		t.Fatal("startup did not preserve independent globals and hold")
	}

	next := old
	next.ToggleShortcut = "Ctrl+Alt+A"
	next.ShowShortcut = "Ctrl+Alt+D"
	next.HoldShortcut = "Ctrl+Alt"
	wantAppliedGlobals := map[string]bool{next.ToggleShortcut: true, next.ShowShortcut: true}
	if clear {
		next.ToggleShortcut, next.ShowShortcut, next.HoldShortcut = "", "", ""
		wantAppliedGlobals = map[string]bool{}
	}
	next.StartWithWindows = !old.StartWithWindows
	next.HistoryEnabled = !old.HistoryEnabled
	next.Model = "replacement-model"
	next.BaseURL = "https://example.test/v1"
	failure := errors.New("disk full")
	store := &shortcutRollbackStore{saved: old, failure: failure}
	store.beforeFailure = func(applied config.Settings) {
		if !reflect.DeepEqual(applied, next) || !reflect.DeepEqual(globals.active, wantAppliedGlobals) || hold.value != next.HoldShortcut {
			t.Fatal("persistence failure did not occur after native replacement")
		}
	}
	log := &[]string{}
	startup := &startupFake{log: log, on: old.StartWithWindows}
	published := 0
	service := NewService(store, old, &keyFake{log: log}, &keyFake{log: log}, startup,
		func() (bool, string) { return !holdUnavailable, hold.rejected }, controller.ConfigureWithRollback,
		func(config.Settings) { published++ }, func(bool) { published++ },
		func(config.Settings) { published++ }, func(SettingsDTO) { published++ }, nil)
	before := service.GetSettings()
	globals.attempts = nil
	if _, err := service.SaveSettings(SaveSettingsRequest{Settings: next}); !errors.Is(err, failure) {
		t.Fatalf("save error = %v, want persistence failure", err)
	}
	if !reflect.DeepEqual(globals.active, priorGlobals) || hold.value != priorHold {
		t.Errorf("failed save leaked native state: globals=%v hold=%q; want globals=%v hold=%q", globals.active, hold.value, priorGlobals, priorHold)
	}
	if !reflect.DeepEqual(service.GetSettings(), before) || !reflect.DeepEqual(store.saved, old) || startup.on != old.StartWithWindows || published != 0 {
		t.Error("failed save changed prior settings, persistence, startup, or publication")
	}
	for _, value := range globals.attempts {
		if toggleUnavailable && value == old.ToggleShortcut {
			t.Error("rollback retried a saved chord that was never bound")
		}
	}
	if err := controller.Configure(old); err != nil {
		t.Errorf("controller did not restore saved preferences: %v", err)
	}
	if err := controller.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := controller.Resume(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(globals.active, priorGlobals) || hold.value != priorHold || hold.starts != 1 {
		t.Error("capture did not preserve the restored effective snapshot")
	}
	// The next successful save must use a fresh transaction, not stale history.
	store.failure = nil
	if _, err := service.SaveSettings(SaveSettingsRequest{Settings: next}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(service.GetSettings().Settings, next) || !reflect.DeepEqual(store.saved, next) || !reflect.DeepEqual(globals.active, wantAppliedGlobals) || hold.value != next.HoldShortcut {
		t.Error("subsequent save did not commit the requested settings and bindings")
	}
}
