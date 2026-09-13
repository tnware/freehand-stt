package shortcut

import (
	"errors"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestResumeHoldAndGlobalFailureRemainRecoverable(t *testing.T) {
	globals := &globalFake{active: map[string]bool{}}
	hold := &reviewHold{}
	c := New(globals, hold, func() {}, func() {})
	saved := config.Default()
	saved.ToggleShortcut = "Ctrl+Shift+Space"
	saved.ShowShortcut = "Ctrl+Shift+D"
	saved.HoldShortcut = "F13"
	if err := c.Start(saved); err != nil {
		t.Fatal(err)
	}
	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	hold.err = errors.New("release all keys")
	globals.fail = saved.ShowShortcut
	err := c.Resume()
	if !errors.Is(err, hold.err) || !strings.Contains(err.Error(), "could not be restored") {
		t.Errorf("lost restoration errors: %v", err)
	}
	if !c.suspended || !c.hasState || len(globals.active) != 0 || c.active.HoldShortcut != saved.HoldShortcut {
		t.Fatalf("bad rollback state: suspended=%v globals=%v", c.suspended, globals.active)
	}
	globals.fail = ""
	if err := c.Resume(); !errors.Is(err, hold.err) {
		t.Fatalf("expected degraded recovery warning: %v", err)
	}
	if c.suspended || !globals.active[saved.ToggleShortcut] || !globals.active[saved.ShowShortcut] {
		t.Fatal("global recovery failed")
	}
	hold.err = nil
	if err := c.RetryHold(); err != nil {
		t.Fatalf("retry after degraded resume: %v", err)
	}
}
