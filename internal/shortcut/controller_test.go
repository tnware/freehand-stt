package shortcut

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"runtime"
	"testing"
)

type globalFake struct {
	active map[string]bool
	fail   string
	log    []string
}

func (f *globalFake) Register(v string, _ func()) error {
	f.log = append(f.log, "register:"+v)
	if v == f.fail || f.active[v] {
		return errors.New("conflict")
	}
	f.active[v] = true
	return nil
}
func (f *globalFake) Unregister(v string) error {
	f.log = append(f.log, "unregister:"+v)
	delete(f.active, v)
	return nil
}

type holdFake struct{ value string }

func (h *holdFake) Start(v string) error     { return h.Configure(v) }
func (h *holdFake) Configure(v string) error { h.value = v; return nil }

func TestReplacementRegistersBeforeUnregisterAndPreservesOnConflict(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	h := &holdFake{}
	c := New(f, h, func() {}, func() {})
	old := config.Default()
	if err := c.Configure(old); err != nil {
		t.Fatal(err)
	}
	f.log = nil
	next := old
	next.ToggleShortcut = "Ctrl+Alt+A"
	if err := c.Configure(next); err != nil {
		t.Fatal(err)
	}
	if f.log[0] != "register:Ctrl+Alt+A" {
		t.Fatalf("replacement order: %v", f.log)
	}
	f.fail = "Ctrl+Alt+B"
	bad := next
	bad.ToggleShortcut = f.fail
	if err := c.Configure(bad); err == nil || !f.active["Ctrl+Alt+A"] {
		t.Fatalf("working binding not preserved: %v", f.active)
	}
}

func TestImpossibleAtomicSwapRejected(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	c := New(f, nil, func() {}, func() {})
	old := config.Default()
	if err := c.Configure(old); err != nil {
		t.Fatal(err)
	}
	next := old
	next.ToggleShortcut, next.ShowShortcut = old.ShowShortcut, old.ToggleShortcut
	if err := c.Configure(next); err == nil {
		t.Fatal("swap accepted")
	}
}

func TestDedicatedAndAliasedChordsNormalizeBeforeNativeRegistration(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	h := &holdFake{}
	c := New(f, h, func() {}, func() {})
	cfg := config.Default()
	cfg.ToggleShortcut = "F13"
	cfg.ShowShortcut = "Win+F24"
	expectedShow := "Super+F24"
	if runtime.GOOS == "darwin" {
		cfg.ShowShortcut = "Command+F20"
		expectedShow = "Super+F20"
	}
	cfg.HoldShortcut = "Control+Command"
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if !f.active["F13"] || !f.active[expectedShow] || h.value != "Ctrl+Super" {
		t.Fatalf("normalized bindings = globals=%v hold=%q", f.active, h.value)
	}
}

func TestCaptureSuspendAndResumeRestoreWorkingBindings(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	h := &holdFake{}
	c := New(f, h, func() {}, func() {})
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	cfg.HoldShortcut = "Ctrl+Space"
	cfg.ShowShortcut = "Ctrl+Shift+D"
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if len(f.active) != 0 || h.value != "" {
		t.Fatalf("bindings remained active during capture: %v", f.active)
	}
	if err := c.Configure(cfg); err == nil {
		t.Fatal("configuration changed while capture was suspended")
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	if !f.active["Ctrl+Shift+Space"] || !f.active["Ctrl+Shift+D"] || h.value != "Ctrl+Space" {
		t.Fatalf("bindings were not restored: globals=%v hold=%q", f.active, h.value)
	}
}

func TestCaptureResumeDoesNotLeavePartialBindingsOnConflict(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	h := &holdFake{}
	c := New(f, h, func() {}, func() {})
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	cfg.HoldShortcut = "Ctrl+Space"
	cfg.ShowShortcut = "Ctrl+Shift+D"
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	f.fail = "Ctrl+Shift+D"
	if err := c.Resume(); err == nil {
		t.Fatal("resume succeeded despite conflict")
	}
	if len(f.active) != 0 || h.value != "" {
		t.Fatalf("partial binding survived failed resume: globals=%v hold=%q", f.active, h.value)
	}
	if !c.suspended || !c.hasState || c.active.HoldShortcut != cfg.HoldShortcut {
		t.Fatal("failed global restoration lost recoverable suspension or preference")
	}
	f.fail = ""
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	if c.suspended || !f.active[cfg.ToggleShortcut] || !f.active[cfg.ShowShortcut] || h.value != cfg.HoldShortcut {
		t.Fatal("resume did not recover after global conflict resolved")
	}
}

func TestShowShortcutCanBeUnassignedAndCleared(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	c := New(f, &holdFake{}, func() {}, func() {})
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	if cfg.ShowShortcut != "" {
		t.Fatal("Show Freehand default must be unassigned")
	}
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if len(f.active) != 1 || f.active[""] {
		t.Fatal("registered an unassigned shortcut")
	}
	cfg.ShowShortcut = "Ctrl+Shift+D"
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.ShowShortcut = ""
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if f.active["Ctrl+Shift+D"] {
		t.Fatal("cleared shortcut remained registered")
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	if len(f.active) != 1 || !f.active["Ctrl+Shift+Space"] {
		t.Fatal("capture failed to preserve optional shortcut state")
	}
}

type failingHold struct {
	value string
	fail  bool
	calls int
}

func (h *failingHold) Start(v string) error { return h.Configure(v) }
func (h *failingHold) Configure(v string) error {
	h.calls++
	if h.fail && v != "" {
		return errors.New("Input Monitoring denied")
	}
	h.value = v
	return nil
}
func TestStartupHoldFailurePreservesIndependentGlobalsAndPreference(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	h := &failingHold{fail: true}
	c := New(f, h, func() {}, func() {})
	cfg := config.Default()
	cfg.HoldShortcut = "Ctrl+Alt"
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	cfg.ShowShortcut = "Ctrl+Shift+D"
	if err := c.Start(cfg); err == nil {
		t.Fatal("missing degraded startup error")
	}
	if !f.active[cfg.ToggleShortcut] || !f.active[cfg.ShowShortcut] {
		t.Fatalf("independent globals lost: %v", f.active)
	}
	if c.active.HoldShortcut != cfg.HoldShortcut {
		t.Fatal("saved hold preference lost")
	}
	next := cfg
	next.ToggleShortcut = "Ctrl+Alt+A"
	next.HoldShortcut = "F13"
	if err := c.Configure(next); err == nil {
		t.Fatal("explicit change accepted")
	}
	if !f.active[cfg.ToggleShortcut] || f.active[next.ToggleShortcut] {
		t.Fatal("transaction did not roll back")
	}
	h.fail = false
	if err := c.RetryHold(); err != nil {
		t.Fatal(err)
	}
	if h.value != cfg.HoldShortcut {
		t.Fatal("retry did not use unchanged saved preference")
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := c.RetryHold(); err == nil {
		t.Fatal("retry during capture accepted")
	}
}
