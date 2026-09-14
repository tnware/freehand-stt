package input

import (
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"github.com/tnware/freehand-stt/internal/shortcut"
)

type conflictGlobals struct{ active map[string]bool }

func (g *conflictGlobals) Register(value string, _ func()) error {
	if value == "Ctrl+Shift+Space" || value == "" || g.active[value] {
		return errors.New("unavailable")
	}
	g.active[value] = true
	return nil
}
func (g *conflictGlobals) Unregister(value string) error {
	if !g.active[value] {
		return errors.New("not registered")
	}
	delete(g.active, value)
	return nil
}

func TestCaptureReplacementThroughServiceAfterStartupConflict(t *testing.T) {
	globals := &conflictGlobals{active: map[string]bool{}}
	controller := shortcut.New(globals, nil, func() {}, func() {})
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+Space"
	if err := controller.Start(cfg); err == nil {
		t.Fatal("missing startup warning")
	}
	chord, err := hotkey.Parse("Ctrl+Alt+K")
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{shortcutCapture: &shortcutFake{chord: chord}, shortcutGuard: controller}
	for _, clearFirst := range []bool{false, true} {
		if clearFirst {
			cfg.ToggleShortcut = ""
			if err := controller.Configure(cfg); err != nil {
				t.Fatal(err)
			}
		}
		result, err := service.CaptureShortcut(ShortcutCaptureRequest{
			Action:      hotkey.ToggleRecording,
			Assignments: hotkey.ShortcutAssignments{ToggleRecording: cfg.ToggleShortcut},
		})
		if err != nil || result.Outcome != ShortcutCaptured || result.Shortcut != "Ctrl+Alt+K" {
			t.Fatalf("clearFirst=%v: result=%+v error=%v", clearFirst, result, err)
		}
	}
	cfg.ToggleShortcut = "Ctrl+Alt+K"
	if err := controller.Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if !globals.active[cfg.ToggleShortcut] {
		t.Fatal("captured replacement not registered")
	}
}
