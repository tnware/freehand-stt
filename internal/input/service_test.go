package input

import (
	"context"
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/hotkey"
)

type shortcutFake struct {
	chord    hotkey.Chord
	canceled bool
	err      error
	cancel   bool
	entered  chan struct{}
	wait     <-chan struct{}
	progress hotkey.Chord
}

func (f *shortcutFake) Capture(_ context.Context, _ hotkey.ShortcutPolicy, changed func(hotkey.Chord)) (hotkey.Chord, bool, error) {
	if f.entered != nil {
		close(f.entered)
	}
	if changed != nil && f.progress.String() != "" {
		changed(f.progress)
	}
	if f.wait != nil {
		<-f.wait
	}
	return f.chord, f.canceled, f.err
}
func (f *shortcutFake) Cancel()      { f.cancel = true }
func (f *shortcutFake) Close() error { return nil }

type guardFake struct{ suspended, resumed bool }

func (f *guardFake) Suspend() error { f.suspended = true; return nil }
func (f *guardFake) Resume() error  { f.resumed = true; return nil }

func captureRequest(action hotkey.ShortcutAction) ShortcutCaptureRequest {
	return ShortcutCaptureRequest{
		Action: action,
		Assignments: hotkey.ShortcutAssignments{
			ToggleRecording: "Ctrl+Shift+Space",
			ShowFreehand:    "Ctrl+Shift+D",
		},
	}
}

func TestCaptureShortcutReturnsNormalizedResultAndCancellation(t *testing.T) {
	chord, _ := hotkey.Parse("Shift+Ctrl+K")
	capture := &shortcutFake{chord: chord}
	guard := &guardFake{}
	service := &Service{shortcutCapture: capture, shortcutGuard: guard}
	result, err := service.CaptureShortcut(captureRequest(hotkey.HoldToTalk))
	if err != nil || result.Shortcut != "Ctrl+Shift+K" || result.Outcome != ShortcutCaptured {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	if !guard.suspended || !guard.resumed {
		t.Fatalf("capture guard did not bracket capture: %#v", guard)
	}
	capture.canceled = true
	result, err = service.CaptureShortcut(captureRequest(hotkey.HoldToTalk))
	if err != nil || result.Outcome != ShortcutCancelled || result.Shortcut != "" {
		t.Fatalf("canceled result = %#v, err = %v", result, err)
	}
	service.CancelShortcutCapture()
	if !capture.cancel {
		t.Fatal("capture cancellation was not forwarded")
	}
}

func TestCaptureShortcutRejectsConcurrentOwner(t *testing.T) {
	chord, _ := hotkey.Parse("Ctrl+D")
	wait := make(chan struct{})
	capture := &shortcutFake{chord: chord, entered: make(chan struct{}), wait: wait}
	service := &Service{shortcutCapture: capture, shortcutGuard: &guardFake{}}
	first := make(chan error, 1)
	go func() {
		_, err := service.CaptureShortcut(captureRequest(hotkey.ToggleRecording))
		first <- err
	}()
	<-capture.entered
	if result, err := service.CaptureShortcut(captureRequest(hotkey.ShowFreehand)); err != nil || result.RejectionKind != hotkey.RejectionUnavailable {
		t.Fatalf("concurrent capture result = %#v, err = %v", result, err)
	}
	close(wait)
	if err := <-first; err != nil {
		t.Fatalf("first capture failed: %v", err)
	}
}

func TestCaptureShortcutResumesBindingsAfterCaptureError(t *testing.T) {
	guard := &guardFake{}
	service := &Service{shortcutCapture: &shortcutFake{err: errors.New("hook failed")}, shortcutGuard: guard}
	if _, err := service.CaptureShortcut(captureRequest(hotkey.ToggleRecording)); err == nil {
		t.Fatal("capture error was not returned")
	}
	if !guard.suspended || !guard.resumed {
		t.Fatalf("capture guard was not restored after failure: %#v", guard)
	}
}

func TestCaptureShortcutReturnsStructuredExpectedRejections(t *testing.T) {
	service := &Service{
		shortcutCapture: &shortcutFake{err: hotkey.NewRejection(hotkey.RejectionReserved, hotkey.ToggleRecording, "F12 is reserved")},
		shortcutGuard:   &guardFake{},
	}
	result, err := service.CaptureShortcut(captureRequest(hotkey.ToggleRecording))
	if err != nil || result.Outcome != ShortcutRejected || result.RejectionKind != hotkey.RejectionReserved {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

func TestCaptureShortcutRejectsDuplicateDraftAndPublishesProgress(t *testing.T) {
	chord, _ := hotkey.Parse("Ctrl+Shift+D")
	var progress ShortcutCaptureProgress
	service := &Service{
		shortcutCapture: &shortcutFake{chord: chord, progress: chord},
		shortcutGuard:   &guardFake{},
		shortcutChanged: func(next ShortcutCaptureProgress) { progress = next },
	}
	result, err := service.CaptureShortcut(captureRequest(hotkey.ToggleRecording))
	if err != nil || result.Outcome != ShortcutRejected || result.RejectionKind != hotkey.RejectionDuplicate || result.ConflictingAction != hotkey.ShowFreehand {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	if progress.Action != hotkey.ToggleRecording || progress.Shortcut != "Ctrl+Shift+D" {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestCaptureShortcutReportsAlreadyAssignedWithoutCreatingADirtyDraft(t *testing.T) {
	chord, _ := hotkey.Parse("Ctrl+Shift+Space")
	service := &Service{shortcutCapture: &shortcutFake{chord: chord}, shortcutGuard: &guardFake{}}
	result, err := service.CaptureShortcut(captureRequest(hotkey.ToggleRecording))
	if err != nil || result.Outcome != ShortcutCaptured || result.Changed || result.Shortcut != "Ctrl+Shift+Space" {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

func TestShortcutPoliciesExposeTheNativeActionMatrix(t *testing.T) {
	policies := (&Service{}).ShortcutPolicies()
	if len(policies) != 3 || !policies[0].Required || policies[2].ModifierOnlyMinimum != 2 {
		t.Fatalf("policies = %#v", policies)
	}
}
