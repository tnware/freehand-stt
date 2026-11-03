//go:build linux && cgo

package keyboard

import (
	"context"
	"errors"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"sync"
	"testing"
	"time"
)

type fakeKeyboard struct {
	mu       sync.Mutex
	events   []keyboardEvent
	closed   bool
	physical int
	ungrabs  int
}

func (f *fakeKeyboard) read() (keyboardEvent, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 || f.closed {
		return keyboardEvent{}, false
	}
	e := f.events[0]
	f.events = f.events[1:]
	return e, true
}
func (f *fakeKeyboard) close() error { f.mu.Lock(); defer f.mu.Unlock(); f.closed = true; return nil }
func (f *fakeKeyboard) send(e keyboardEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}
func (f *fakeKeyboard) released() int            { return 1 }
func (f *fakeKeyboard) held(uint32) int          { f.mu.Lock(); defer f.mu.Unlock(); return f.physical }
func (f *fakeKeyboard) ungrabActive()            { f.mu.Lock(); defer f.mu.Unlock(); f.ungrabs++ }
func (f *fakeKeyboard) modifierMasks() [4]uint16 { return [4]uint16{4, 32, 1, 64} }
func expectKeyboardSignal(t *testing.T, c <-chan string, want string) {
	t.Helper()
	select {
	case got := <-c:
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("missing " + want)
	}
}
func noKeyboardSignal(t *testing.T, c <-chan string) {
	t.Helper()
	select {
	case got := <-c:
		t.Fatalf("unexpected %s", got)
	case <-time.After(25 * time.Millisecond):
	}
}
func testHold(t *testing.T) (*HoldHook, *fakeKeyboard, chan string) {
	t.Helper()
	events := make(chan string, 32)
	f := &fakeKeyboard{physical: 1}
	h := NewHoldHook(func() { events <- "press" }, func() { events <- "release" }, func() { events <- "cancel" })
	h.open = func(hotkey.Chord) (holdSource, error) { return f, nil }
	t.Cleanup(func() { _ = h.Close() })
	return h, f, events
}
func TestLinuxSessionGatesWaylandAndRemote(t *testing.T) {
	for _, tc := range []struct {
		s, w, d string
		want    bool
	}{{"x11", "", ":0", true}, {"x11", "", ":1.0", true}, {"wayland", "", ":0", false}, {"x11", "wayland-0", ":0", false}, {"", "", ":0", false}, {"x11", "", "localhost:10.0", false}, {"x11", "", "", false}} {
		if got := x11Session(tc.s, tc.w, tc.d); got != tc.want {
			t.Fatalf("%+v got %v", tc, got)
		}
	}
}
func TestLinuxAdapterRepeatAndModifierSides(t *testing.T) {
	var a keyboardAdapter
	for _, tc := range []struct {
		e    keyboardEvent
		want bool
	}{{keyboardEvent{code: 37, wire: 0x11, down: true}, true}, {keyboardEvent{code: 37, wire: 0x11, down: true}, false}, {keyboardEvent{code: 105, wire: 0x11, down: true}, false}, {keyboardEvent{code: 37, wire: 0x11}, false}, {keyboardEvent{code: 105, wire: 0x11}, true}, {keyboardEvent{code: 105, wire: 0x11}, false}} {
		if _, _, ok := a.event(tc.e); ok != tc.want {
			t.Fatalf("%+v emitted %v", tc.e, ok)
		}
	}
}
func TestLinuxHoldRepeatAndNativeModifierMask(t *testing.T) {
	h, f, events := testHold(t)
	if err := h.Start("Ctrl+Alt+K"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		f.send(keyboardEvent{code: 45, wire: 'K', state: 4 | 32, down: true})
	}
	expectKeyboardSignal(t, events, "press")
	noKeyboardSignal(t, events)
	f.send(keyboardEvent{code: 45, wire: 'K', state: 4 | 32})
	expectKeyboardSignal(t, events, "release")
	noKeyboardSignal(t, events)
}
func TestLinuxHoldDoesNotAssumeAltMod1(t *testing.T) {
	h, f, events := testHold(t)
	if err := h.Start("Ctrl+Alt+K"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 45, wire: 'K', state: 4 | 8, down: true})
	noKeyboardSignal(t, events)
}
func TestLinuxHoldLossCancelsNotReleases(t *testing.T) {
	h, f, events := testHold(t)
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 191, wire: 0x7c, down: true})
	expectKeyboardSignal(t, events, "press")
	f.send(keyboardEvent{lost: true})
	f.send(keyboardEvent{code: 191, wire: 0x7c})
	expectKeyboardSignal(t, events, "cancel")
	noKeyboardSignal(t, events)
	if ok, _ := h.Available(); ok {
		t.Fatal("failed hook available")
	}
}
func TestLinuxHoldCloseCancelsAndDisableReleases(t *testing.T) {
	for _, cancel := range []bool{true, false} {
		h, f, events := testHold(t)
		if err := h.Start("F13"); err != nil {
			t.Fatal(err)
		}
		f.send(keyboardEvent{code: 191, wire: 0x7c, down: true})
		expectKeyboardSignal(t, events, "press")
		var err error
		if cancel {
			err = h.Close()
		} else {
			err = h.Configure("")
		}
		if err != nil {
			t.Fatal(err)
		}
		want := "release"
		if cancel {
			want = "cancel"
		}
		expectKeyboardSignal(t, events, want)
	}
}
func TestLinuxHoldFailedReplacementPreservesOldGrab(t *testing.T) {
	h, f, _ := testHold(t)
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	h.open = func(hotkey.Chord) (holdSource, error) { return nil, errors.New("conflict") }
	if err := h.Configure("F14"); err == nil {
		t.Fatal("expected failure")
	}
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if closed {
		t.Fatal("old grab closed on failed replacement")
	}
}
func TestLinuxToggleIgnoresRepeatAfterModifierRelease(t *testing.T) {
	h, f, events := testHold(t)
	h.toggle = true
	if err := h.Start("Ctrl+K"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 45, wire: 'K', state: 4, down: true})
	expectKeyboardSignal(t, events, "press")
	f.send(keyboardEvent{code: 37, wire: 0x11})
	f.send(keyboardEvent{code: 45, wire: 'K', down: true})
	noKeyboardSignal(t, events)
	f.send(keyboardEvent{code: 45, wire: 'K'})
	expectKeyboardSignal(t, events, "release")
}
func TestLinuxMissingReleaseCancels(t *testing.T) {
	h, f, events := testHold(t)
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 191, wire: 0x7c, down: true})
	expectKeyboardSignal(t, events, "press")
	f.mu.Lock()
	f.physical = 0
	f.mu.Unlock()
	expectKeyboardSignal(t, events, "cancel")
}
func TestLinuxCaptureEscapeAndDeadline(t *testing.T) {
	p, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	f := &fakeKeyboard{}
	c := &ShortcutCapturer{open: func(bool) (keyboardSource, error) { return f, nil }}
	f.send(keyboardEvent{code: 9, wire: 0x1b, down: true})
	if _, cancel, err := c.Capture(context.Background(), p, nil); !cancel || err != nil {
		t.Fatalf("Escape: %v %v", cancel, err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, _, err := c.Capture(ctx, p, nil); err == nil {
		t.Fatal("deadline not reported")
	}
}

func TestLinuxHoldBlockedCallbackCannotRetainNativeGrab(t *testing.T) {
	f := &fakeKeyboard{physical: 1}
	entered, unblock := make(chan struct{}), make(chan struct{})
	h := NewHoldHook(func() { close(entered); <-unblock }, nil, nil)
	h.open = func(hotkey.Chord) (holdSource, error) { return f, nil }
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 191, wire: 0x7c, down: true})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("callback not reached")
	}
	if err := h.Close(); err == nil {
		t.Error("blocked callback shutdown must report timeout")
	}
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if !closed {
		t.Error("blocked callback retained native grab")
	}
	close(unblock)
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
}
func TestLinuxCaptureCancelsActiveHold(t *testing.T) {
	h, f, events := testHold(t)
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 191, wire: 0x7c, down: true})
	expectKeyboardSignal(t, events, "press")
	shortcutCaptureActive.Store(true)
	defer shortcutCaptureActive.Store(false)
	expectKeyboardSignal(t, events, "cancel")
	f.send(keyboardEvent{code: 191, wire: 0x7c})
	noKeyboardSignal(t, events)
}

func TestLinuxCloseUpgradesPendingDisableToCancellation(t *testing.T) {
	f := &fakeKeyboard{physical: 1}
	entered, unblock := make(chan struct{}), make(chan struct{})
	events := make(chan string, 4)
	h := NewHoldHook(func() { close(entered); <-unblock }, func() { events <- "release" }, func() { events <- "cancel" })
	h.open = func(hotkey.Chord) (holdSource, error) { return f, nil }
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 191, wire: 0x7c, down: true})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("callback not reached")
	}
	if err := h.Configure(""); err == nil {
		t.Error("disable should report blocked callback")
	}
	if err := h.Close(); err == nil {
		t.Error("close should report blocked callback")
	}
	close(unblock)
	expectKeyboardSignal(t, events, "cancel")
	noKeyboardSignal(t, events)
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
}
