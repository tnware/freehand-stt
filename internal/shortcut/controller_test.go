package shortcut

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
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
	cfg.HoldShortcut = "Control+Command"
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if !f.active["F13"] || !f.active["Super+F24"] || h.value != "Ctrl+Super" {
		t.Fatalf("normalized bindings = globals=%v hold=%q", f.active, h.value)
	}
}

func TestCaptureSuspendAndResumeRestoreWorkingBindings(t *testing.T) {
	f := &globalFake{active: map[string]bool{}}
	h := &holdFake{}
	c := New(f, h, func() {}, func() {})
	cfg := config.Default()
	cfg.HoldShortcut = "Ctrl+Space"
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
	cfg.HoldShortcut = "Ctrl+Space"
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
}
