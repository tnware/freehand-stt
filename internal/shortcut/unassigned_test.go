package shortcut

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestToggleCanBeClearedCapturedAndReassigned(t *testing.T) {
	globals := &globalFake{active: map[string]bool{}}
	c := New(globals, &holdFake{}, func() {}, func() {})
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	if err := c.Start(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.ToggleShortcut = ""
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if len(globals.active) != 0 {
		t.Fatal("cleared toggle is still registered")
	}
	globals.log = nil
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	if len(globals.log) != 0 {
		t.Fatalf("unassigned shortcuts reached the OS: %v", globals.log)
	}
	cfg.ToggleShortcut = "Ctrl+Alt+K"
	if err := c.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if !globals.active[cfg.ToggleShortcut] {
		t.Fatal("replacement is not registered")
	}
}
