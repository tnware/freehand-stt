package shortcut

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

type startupHold struct {
	holdFake
	starts int
}

func (h *startupHold) Start(value string) error {
	h.starts++
	return h.Configure(value)
}

func TestStartupInitializesHoldEvenWhenToggleConflicts(t *testing.T) {
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	cfg.HoldShortcut = "Ctrl+Super"
	h := &startupHold{}
	c := New(&globalFake{active: map[string]bool{}, fail: cfg.ToggleShortcut}, h, func() {}, func() {})
	if err := c.Start(cfg); err == nil {
		t.Fatal("missing toggle warning")
	}
	if h.starts != 1 || h.value != cfg.HoldShortcut {
		t.Fatal("hold hook was not initialized")
	}
	cfg.ToggleShortcut = ""
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	if h.starts != 1 {
		t.Fatal("hold hook was started more than once")
	}
}

func TestStartupInitializesEmptyHoldOnceBeforeConfigure(t *testing.T) {
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	cfg.HoldShortcut = ""
	h := &startupHold{}
	c := New(&globalFake{active: map[string]bool{}, fail: cfg.ToggleShortcut}, h, func() {}, func() {})
	if err := c.Start(cfg); err == nil {
		t.Fatal("missing toggle warning")
	}
	if h.starts != 1 || h.value != "" {
		t.Fatal("empty hold did not initialize the adapter exactly once")
	}
	cfg.ToggleShortcut = ""
	cfg.HoldShortcut = "Ctrl+Super"
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if h.starts != 1 || h.value != cfg.HoldShortcut {
		t.Fatal("later hold assignment did not use Configure on the initialized adapter")
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	if h.starts != 1 || h.value != cfg.HoldShortcut {
		t.Fatal("capture restarted or lost the initially empty hold adapter")
	}
}

func TestClearUnavailableHoldStillConfiguresEmptyAssignment(t *testing.T) {
	cfg := config.Default()
	cfg.HoldShortcut = "Ctrl+Super"
	h := &failingHold{fail: true}
	c := New(&globalFake{active: map[string]bool{}}, h, func() {}, func() {})
	if err := c.Start(cfg); err == nil {
		t.Fatal("missing hold warning")
	}
	before := h.calls
	cfg.HoldShortcut = ""
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if h.calls != before+1 || h.value != "" {
		t.Fatal("explicit Clear did not configure the native hold adapter")
	}
}
