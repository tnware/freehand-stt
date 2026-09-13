//go:build darwin

package platform

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeKeyboard struct {
	mu     sync.Mutex
	events []keyboardEvent
	closed bool
}

func (f *fakeKeyboard) read() (keyboardEvent, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		return keyboardEvent{}, false
	}
	e := f.events[0]
	f.events = f.events[1:]
	return e, true
}
func (f *fakeKeyboard) close() error { f.mu.Lock(); defer f.mu.Unlock(); f.closed = true; return nil }
func (f *fakeKeyboard) send(e keyboardEvent) {
	f.mu.Lock()
	f.events = append(f.events, e)
	f.mu.Unlock()
}
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

func TestDarwinHoldLazyAuthorizationAndDisable(t *testing.T) {
	events := make(chan string, 8)
	h := NewHoldHook(func() { events <- "press" }, func() { events <- "release" }, func() { events <- "cancel" })
	starts := 0
	f := &fakeKeyboard{}
	h.keysReleased = func() bool { return true }
	h.authorized = func() bool { return false }
	h.open = func(bool) (keyboardSource, error) { starts++; return f, nil }
	if err := h.Start(""); err != nil {
		t.Fatal(err)
	}
	if starts != 0 {
		t.Fatal("empty startup acquired keyboard")
	}
	if err := h.Configure("Ctrl+Alt"); err == nil || !strings.Contains(err.Error(), "Input Monitoring") {
		t.Fatalf("authorization: %v", err)
	}
	h.keysReleased = func() bool { return true }
	h.authorized = func() bool { return true }
	if ok, _ := h.Available(); ok {
		t.Fatal("restored authorization reported a hook ready without rearm")
	}
	if err := h.Configure("Ctrl+Alt"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 59, down: true})
	f.send(keyboardEvent{code: 58, down: true})
	expectKeyboardSignal(t, events, "press")
	if err := h.Configure(""); err != nil {
		t.Fatal(err)
	}
	expectKeyboardSignal(t, events, "release")
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if !closed {
		t.Fatal("disable did not release tap")
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if err := h.Configure("Ctrl+Alt"); err == nil {
		t.Fatal("closed hook restarted")
	}
}

func TestDarwinHoldIdleLossFailsAndStopsWithoutCanceling(t *testing.T) {
	signals := make(chan string, 8)
	f := &fakeKeyboard{}
	h := NewHoldHook(func() { signals <- "press" }, func() { signals <- "release" }, func() { signals <- "cancel" })
	h.keysReleased = func() bool { return true }
	h.authorized = func() bool { return true }
	h.open = func(bool) (keyboardSource, error) { return f, nil }
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	f.send(keyboardEvent{lost: true})
	f.send(keyboardEvent{code: 105, down: true})
	select {
	case <-h.run.done:
	case <-time.After(time.Second):
		t.Fatal("idle tap loss did not stop the consumer")
	}
	if ok, _ := h.Available(); ok {
		t.Error("failed idle tap reported available")
	}
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if !closed {
		t.Error("idle tap loss did not release the source")
	}
	select {
	case event := <-signals:
		t.Fatalf("idle tap loss triggered an unrelated dictation callback: %s", event)
	default:
	}
}

func TestDarwinHoldLossCancelsWithoutDeliveringQueuedRelease(t *testing.T) {
	signals := make(chan string, 8)
	f := &fakeKeyboard{}
	h := NewHoldHook(func() { signals <- "press" }, func() { signals <- "release" }, func() { signals <- "cancel" })
	h.keysReleased = func() bool { return true }
	h.authorized = func() bool { return true }
	h.open = func(bool) (keyboardSource, error) { return f, nil }
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	f.send(keyboardEvent{code: 105, down: true})
	expectKeyboardSignal(t, signals, "press")
	f.send(keyboardEvent{lost: true})
	f.send(keyboardEvent{code: 105, down: false})
	expectKeyboardSignal(t, signals, "cancel")
	if ok, _ := h.Available(); ok {
		t.Fatal("failed tap reported available")
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-signals:
		t.Fatalf("stale edge: %s", event)
	default:
	}
}

func TestDarwinHoldDoesNotRecordShortcutCapture(t *testing.T) {
	signals := make(chan string, 8)
	f := &fakeKeyboard{}
	h := NewHoldHook(func() { signals <- "press" }, func() { signals <- "release" }, func() { signals <- "cancel" })
	h.keysReleased = func() bool { return true }
	h.authorized = func() bool { return true }
	h.open = func(bool) (keyboardSource, error) { return f, nil }
	if err := h.Start("F13"); err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	shortcutCaptureActive.Store(true)
	defer shortcutCaptureActive.Store(false)
	f.send(keyboardEvent{code: 105, down: true})
	f.send(keyboardEvent{code: 105, down: false})
	deadline := time.Now().Add(time.Second)
	for {
		f.mu.Lock()
		empty := len(f.events) == 0
		f.mu.Unlock()
		if empty {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("capture input was not drained")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case got := <-signals:
		t.Fatalf("capture triggered hold: %s", got)
	default:
	}
	shortcutCaptureActive.Store(false)
	f.send(keyboardEvent{code: 105, down: true})
	expectKeyboardSignal(t, signals, "press")
}

func TestDarwinHoldStartFailure(t *testing.T) {
	h := NewHoldHook(nil, nil, nil)
	h.keysReleased = func() bool { return true }
	h.authorized = func() bool { return true }
	h.open = func(bool) (keyboardSource, error) { return nil, errors.New("tap unavailable") }
	if err := h.Start("F13"); err == nil {
		t.Fatal("missing tap failure")
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	_ = hotkey.NoEdge
}

func TestDarwinHoldExplicitRearmRequiresReleasedKeysAndFreshSource(t *testing.T) {
	signals := make(chan string, 8)
	h := NewHoldHook(func() { signals <- "press" }, func() { signals <- "release" }, func() { signals <- "cancel" })
	h.authorized = func() bool { return true }
	released := true
	h.keysReleased = func() bool { return released }
	source := &fakeKeyboard{}
	opens := 0
	h.open = func(bool) (keyboardSource, error) { opens++; return source, nil }
	if err := h.Configure("F13"); err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	source.send(keyboardEvent{code: 105, down: true})
	expectKeyboardSignal(t, signals, "press")
	source.send(keyboardEvent{lost: true})
	expectKeyboardSignal(t, signals, "cancel")
	<-h.run.done
	released = false
	if err := h.Configure("F13"); err == nil {
		t.Fatal("rearmed with keys down")
	}
	if opens != 1 {
		t.Fatal("opened unsafe source")
	}
	released = true
	source = &fakeKeyboard{}
	if err := h.Configure("F13"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := h.Available(); !ok {
		t.Fatal("failed state not cleared")
	}
	source.send(keyboardEvent{code: 105, down: false})
	source.send(keyboardEvent{code: 105, down: true})
	expectKeyboardSignal(t, signals, "press")
	source.send(keyboardEvent{code: 105, down: false})
	expectKeyboardSignal(t, signals, "release")
	if opens != 2 {
		t.Fatal("unchanged chord did not recreate source")
	}
}

func TestDarwinHoldReplacementFailureKeepsPreviousHook(t *testing.T) {
	signals := make(chan string, 8)
	h := NewHoldHook(func() { signals <- "press" }, func() { signals <- "release" }, nil)
	h.authorized = func() bool { return true }
	h.keysReleased = func() bool { return true }
	source := &fakeKeyboard{}
	h.open = func(bool) (keyboardSource, error) { return source, nil }
	if err := h.Configure("F13"); err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	h.open = func(bool) (keyboardSource, error) { return nil, errors.New("replacement tap unavailable") }
	if err := h.Configure("F14"); err == nil {
		t.Fatal("replacement failure hidden")
	}
	source.mu.Lock()
	closed := source.closed
	source.mu.Unlock()
	if closed {
		t.Fatal("failed replacement stopped the previous hold hook")
	}
	if ok, _ := h.Available(); !ok {
		t.Fatal("working previous hook marked unavailable")
	}
	source.send(keyboardEvent{code: 105, down: true})
	expectKeyboardSignal(t, signals, "press")
	source.send(keyboardEvent{code: 105, down: false})
	expectKeyboardSignal(t, signals, "release")
}

func TestDarwinHoldRechecksReleasedKeysAfterOpeningTap(t *testing.T) {
	h := NewHoldHook(nil, nil, nil)
	h.authorized = func() bool { return true }
	released := true
	h.keysReleased = func() bool { return released }
	source := &fakeKeyboard{}
	h.open = func(bool) (keyboardSource, error) {
		released = false // a physical key went down while tap creation was in flight
		return source, nil
	}
	defer h.Close()
	if err := h.Configure("F13"); err == nil {
		t.Fatal("rearmed with keys held during tap creation")
	}
	source.mu.Lock()
	closed := source.closed
	source.mu.Unlock()
	if !closed {
		t.Fatal("unsafe candidate tap leaked")
	}
	if ok, _ := h.Available(); ok {
		t.Fatal("unsafe rearm reported ready")
	}
	if h.run != nil {
		t.Fatal("unsafe candidate consumer started")
	}
}
