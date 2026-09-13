package shortcut

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestStartupConflictKeepsIndependentBindingsAndAllowsCapture(t *testing.T) {
	for _, failed := range []string{"Ctrl+Shift+Space", "Ctrl+Shift+D"} {
		t.Run(failed, func(t *testing.T) {
			globals := &globalFake{active: map[string]bool{}, fail: failed}
			hold := &holdFake{}
			c := New(globals, hold, func() {}, func() {})
			cfg := config.Default()
			cfg.ToggleShortcut, cfg.ShowShortcut, cfg.HoldShortcut = "Ctrl+Shift+Space", "Ctrl+Shift+D", "Ctrl+Alt"
			if err := c.Start(cfg); err == nil {
				t.Fatal("startup conflict was hidden")
			}
			if len(globals.active) != 1 || globals.active[failed] || hold.value != cfg.HoldShortcut {
				t.Fatalf("independent bindings lost: globals=%v hold=%q", globals.active, hold.value)
			}
			if c.active.ToggleShortcut != cfg.ToggleShortcut || c.active.ShowShortcut != cfg.ShowShortcut {
				t.Fatal("saved preferences lost")
			}
			globals.log = nil
			if err := c.Suspend(); err != nil {
				t.Fatal(err)
			}
			if err := c.Resume(); err != nil {
				t.Fatal(err)
			}
			for _, call := range globals.log {
				if call == "register:"+failed || call == "unregister:"+failed {
					t.Fatal("capture touched a binding we never owned")
				}
			}
			if failed == cfg.ToggleShortcut {
				cfg.ToggleShortcut = ""
			} else {
				cfg.ShowShortcut = ""
			}
			if err := c.Configure(cfg); err != nil {
				t.Fatal(err)
			}
			if len(globals.active) != 1 {
				t.Fatal("clearing failed shortcut disturbed working binding")
			}
		})
	}
}
