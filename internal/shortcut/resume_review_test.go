package shortcut

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"testing"
)

type reviewHold struct{ err error }

func (h *reviewHold) Start(value string) error { return h.Configure(value) }
func (h *reviewHold) Configure(value string) error {
	if value != "" {
		return h.err
	}
	return nil
}

func TestReviewResumeHoldFailureMustNotTrapGlobalsAndRecovery(t *testing.T) {
	for _, failure := range []string{"Release all keyboard keys, then retry hold-to-talk.", "Input Monitoring denied"} {
		t.Run(failure, func(t *testing.T) {
			globals := &globalFake{active: map[string]bool{}}
			hold := &reviewHold{}
			controller := New(globals, hold, func() {}, func() {})
			saved := config.Default()
			saved.ToggleShortcut = "Ctrl+Shift+Space"
			saved.ShowShortcut = "Ctrl+Shift+D"
			saved.HoldShortcut = "F13"
			if err := controller.Start(saved); err != nil {
				t.Fatal(err)
			}
			if err := controller.Suspend(); err != nil {
				t.Fatal(err)
			}
			hold.err = errors.New(failure)
			resumeErr := controller.Resume()
			t.Logf("resume=%v suspended=%v globals=%v", resumeErr, controller.suspended, globals.active)
			if !globals.active[saved.ToggleShortcut] || !globals.active[saved.ShowShortcut] {
				t.Error("independent Carbon globals remain unregistered after capture ended")
			}
			if controller.suspended {
				t.Error("controller remains capture-suspended after capture ended")
			}
			if !errors.Is(resumeErr, hold.err) {
				t.Errorf("missing hold warning: %v", resumeErr)
			}
			if controller.active.HoldShortcut != saved.HoldShortcut {
				t.Error("saved hold preference lost")
			}
			before := len(globals.log)
			if err := controller.Resume(); err != nil {
				t.Errorf("second Resume: %v", err)
			}
			if len(globals.log) != before {
				t.Error("second Resume duplicated global registration")
			}
			clearedWhileDenied := saved
			clearedWhileDenied.HoldShortcut = ""
			if err := controller.Configure(clearedWhileDenied); err != nil {
				t.Errorf("Clear while still unavailable: %v", err)
			}
			hold.err = nil // user has now released keys/restored permission
			if err := controller.Configure(saved); err != nil {
				t.Fatal(err)
			}
			if err := controller.RetryHold(); err != nil {
				t.Errorf("explicit recovery remains blocked after cause resolved: %v", err)
			}
			cleared := saved
			cleared.HoldShortcut = ""
			if err := controller.Configure(cleared); err != nil {
				t.Errorf("saving Clear also remains blocked: %v", err)
			}
		})
	}
}

func TestReviewCaptureCompletesBeforeModifiersReleased(t *testing.T) {
	policy, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	reducer := hotkey.CaptureReducer{Policy: policy}
	reducer.Event(0x11, true) // Ctrl down, never released
	reducer.Event(0x10, true) // Shift down, never released
	reducer.Event(0x20, true) // Space down
	result := reducer.Event(0x20, false)
	if result.State != hotkey.CaptureComplete {
		t.Fatalf("capture did not complete on primary release: %+v", result)
	}
	t.Logf("capture completes with Ctrl+Shift still physically held: %s", result.Chord.String())
	cancelReducer := hotkey.CaptureReducer{Policy: policy}
	if result := cancelReducer.Event(0x1b, true); result.State != hotkey.CaptureCanceled {
		t.Fatalf("unexpected Escape behavior: %+v", result)
	}
	t.Log("Escape cancels on key-down, before Escape is released")
}
