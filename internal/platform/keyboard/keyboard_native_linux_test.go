//go:build linux && cgo

package keyboard

import (
	"context"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"os"
	"testing"
	"time"
)

// These probes use real XCB grabs. They NEVER synthesize input or change the
// desktop key map. Run only from an explicitly selected disposable X11 desktop.
func requireNativeX11(t *testing.T) {
	t.Helper()
	if os.Getenv("FREEHAND_X11_ACCEPTANCE") != "1" {
		t.Skip("set FREEHAND_X11_ACCEPTANCE=1 in an explicitly selected native X11 desktop")
	}
	if !X11Session() {
		t.Fatal(linuxKeyboardHelp)
	}
	time.Sleep(time.Second)
}
func TestNativeX11GrabLifecycle(t *testing.T) {
	requireNativeX11(t)
	chord, err := hotkey.ParseHold("Ctrl+Shift+F8")
	if err != nil {
		t.Fatal(err)
	}
	source, err := openHold(chord)
	if err != nil {
		t.Fatal(err)
	}
	defer source.close()
	other, err := openHold(chord)
	if err == nil {
		_ = other.close()
		t.Fatal("conflicting passive grab incorrectly admitted")
	}
	if err = source.close(); err != nil {
		t.Fatal(err)
	}
	other, err = openHold(chord)
	if err != nil {
		t.Fatalf("grab not released: %v", err)
	}
	_ = other.close()
	c := &ShortcutCapturer{}
	p, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, _, err = c.Capture(ctx, p, nil); err == nil {
		t.Fatal("expected bounded capture timeout")
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	// Verify capture relinquished its exclusive keyboard grab on cancellation.
	capture, err := openKeyboard(true)
	if err != nil {
		t.Fatalf("capture retained keyboard: %v", err)
	}
	_ = capture.close()
}
func TestNativeX11HoldEdges(t *testing.T) {
	requireNativeX11(t)
	events := make(chan string, 32)
	h := NewHoldHook(func() { events <- "press" }, func() { events <- "release" }, func() { events <- "cancel" })
	defer h.Close()
	if err := h.Start("Ctrl+Shift+F8"); err != nil {
		t.Fatal(err)
	}
	t.Log("READY: physically hold Ctrl+Shift+F8 for at least 2 seconds, then release all keys. No typing is retained.")
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	var pressed time.Time
	select {
	case edge := <-events:
		if edge != "press" {
			t.Fatalf("first edge %s", edge)
		}
		pressed = time.Now()
	case <-timer.C:
		t.Fatal("no physical press")
	}
	select {
	case edge := <-events:
		if edge != "release" {
			t.Fatalf("repeat/loss produced %s", edge)
		}
		if time.Since(pressed) < time.Second {
			t.Fatal("hold less than one second; repeat acceptance not established")
		}
	case <-timer.C:
		t.Fatal("no physical release")
	}
	select {
	case edge := <-events:
		t.Fatalf("extra edge %s", edge)
	case <-time.After(200 * time.Millisecond):
	}
}
func TestNativeX11CaptureChord(t *testing.T) {
	requireNativeX11(t)
	t.Log("READY: record Ctrl+Shift+F9, then release. Escape cancels. Capture releases the keyboard after 30 seconds even without input.")
	c := &ShortcutCapturer{}
	defer c.Close()
	p, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	chord, canceled, err := c.Capture(context.Background(), p, nil)
	if err != nil || canceled {
		t.Fatalf("capture: canceled=%v error=%v", canceled, err)
	}
	if chord != (hotkey.Chord{Modifiers: hotkey.Ctrl | hotkey.Shift, Key: 0x78}) {
		t.Fatal("captured chord did not match Ctrl+Shift+F9")
	}
}
func TestNativeX11CaptureEscape(t *testing.T) {
	requireNativeX11(t)
	t.Log("READY: press Escape to cancel the temporary capture.")
	c := &ShortcutCapturer{}
	defer c.Close()
	p, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	if _, canceled, err := c.Capture(context.Background(), p, nil); err != nil || !canceled {
		t.Fatalf("Escape canceled=%v error=%v", canceled, err)
	}
}

func TestNativeX11GlobalRepeat(t *testing.T) {
	requireNativeX11(t)
	g := NewGlobal()
	defer g.Close()
	calls := make(chan struct{}, 32)
	if err := g.Register("Ctrl+Shift+F8", func() { calls <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	t.Log("READY: hold Ctrl+Shift+F8 for at least 2 seconds, then release. Exactly one toggle is expected.")
	select {
	case <-calls:
	case <-time.After(30 * time.Second):
		t.Fatal("no native toggle")
	}
	select {
	case <-calls:
		t.Fatal("auto-repeat dispatched a second toggle")
	case <-time.After(4 * time.Second):
	}
}
