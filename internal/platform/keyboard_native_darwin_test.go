//go:build darwin

package platform

import (
	"github.com/tnware/freehand-stt/internal/hotkey"
	"testing"
)

func TestDarwinCaptureEscapeCancelsHeldPrimary(t *testing.T) {
	q := newKeyboardTestQueue(true)
	defer q.close()
	q.feed(59, true, false, false)
	q.feed(0, true, false, false)
	if !q.feed(53, true, false, false) {
		t.Fatal("Escape was dropped after the captured primary key")
	}
	policy, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	reducer := hotkey.CaptureReducer{Policy: policy}
	var adapter keyboardAdapter
	for {
		event, ok := q.read()
		if !ok {
			break
		}
		key, down, ok := adapter.event(event)
		if ok && reducer.Event(key, down).State == hotkey.CaptureCanceled {
			return
		}
	}
	t.Fatal("Ctrl+A held followed by Escape did not cancel capture")
}

// This test seam allocates only the queue. It never acquires an event tap,
// posts keyboard input, or invokes a permission request.
func TestDarwinNativeQueueSafety(t *testing.T) {
	q := newKeyboardTestQueue(false)
	defer q.close()
	if q.feed(0, true, true, false) {
		t.Fatal("listen tap suppressed input")
	}
	if _, ok := q.read(); ok {
		t.Fatal("injected input was queued")
	}
	q.feed(0, true, false, true)
	if _, ok := q.read(); ok {
		t.Fatal("autorepeat was queued")
	}
	q.feed(0, true, false, false)
	e, ok := q.read()
	if !ok || e.code != 0 || !e.down {
		t.Fatalf("physical event: %+v %v", e, ok)
	}
	for i := 0; i < 300; i++ {
		q.feed(uint16(i%100), true, false, false)
	}
	e, ok = q.read()
	if !ok || !e.lost {
		t.Fatal("overflow must discard backlog and signal loss")
	}
	if _, ok = q.read(); ok {
		t.Fatal("stale queued input after overflow")
	}
}

func TestDarwinCaptureSuppressesOnlyOwnedPhysicalEdges(t *testing.T) {
	q := newKeyboardTestQueue(true)
	defer q.close()
	if q.feed(0, false, false, false) {
		t.Fatal("suppressed release of pre-existing key")
	}
	if !q.feed(59, true, false, false) || !q.feed(0, true, false, false) {
		t.Fatal("captured input not suppressed")
	}
	if q.feed(1, true, false, false) {
		t.Fatal("unrelated secondary key suppressed")
	}
	if q.feed(0, true, true, false) {
		t.Fatal("injected event suppressed")
	}
	if !q.feed(0, true, false, true) {
		t.Fatal("captured repeat leaked")
	}
	if !q.feed(0, false, false, false) || !q.feed(59, false, false, false) {
		t.Fatal("captured release leaked")
	}
}
