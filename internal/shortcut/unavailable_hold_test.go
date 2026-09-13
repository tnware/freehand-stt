package shortcut

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestStartupUnavailableHoldDoesNotBlockReplacementCapture(t *testing.T) {
	hold := &failingHold{fail: true}
	globals := &globalFake{active: map[string]bool{}, fail: "Ctrl+Shift+Space"}
	c := New(globals, hold, func() {}, func() {})
	cfg := config.Default()
	cfg.ToggleShortcut, cfg.HoldShortcut = "Ctrl+Shift+Space", "Ctrl+Alt"
	if err := c.Start(cfg); err == nil {
		t.Fatal("missing startup warning")
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	cfg.ToggleShortcut = ""
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if c.active.HoldShortcut != "Ctrl+Alt" {
		t.Fatal("unavailable hold preference was lost")
	}
	hold.fail = false
	if err := c.RetryHold(); err != nil {
		t.Fatal(err)
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if hold.value != "" {
		t.Fatal("recovered hold remained active during capture")
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	if hold.value != "Ctrl+Alt" {
		t.Fatal("recovered hold was not restored")
	}
}
