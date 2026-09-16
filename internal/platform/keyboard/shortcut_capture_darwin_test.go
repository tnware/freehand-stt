//go:build darwin

package keyboard

import (
	"context"
	"errors"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"testing"
	"time"
)

type failingCloseKeyboard struct{ fakeKeyboard }

func (f *failingCloseKeyboard) close() error { return errors.New("test teardown failure") }

func TestDarwinShortcutCaptureReportsTeardownFailure(t *testing.T) {
	f := &failingCloseKeyboard{}
	c := &ShortcutCapturer{open: func(bool) (keyboardSource, error) { return f, nil }}
	defer c.Close()
	p, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	f.send(keyboardEvent{code: 105, down: true})
	f.send(keyboardEvent{code: 105, down: false})
	if _, _, err := c.Capture(context.Background(), p, nil); err == nil {
		t.Fatal("capture hid teardown failure")
	}
}

func TestDarwinShortcutCaptureFullPhysicalChord(t *testing.T) {
	f := &fakeKeyboard{}
	c := &ShortcutCapturer{open: func(capture bool) (keyboardSource, error) {
		if !capture {
			t.Error("capture requested listen-only tap")
		}
		return f, nil
	}}
	p, _ := hotkey.PolicyFor(hotkey.ToggleRecording)
	f.send(keyboardEvent{code: 55, down: true})
	f.send(keyboardEvent{code: 0, down: true})
	f.send(keyboardEvent{code: 0, down: false})
	f.send(keyboardEvent{code: 55, down: false})
	chord, canceled, err := c.Capture(context.Background(), p, nil)
	if err != nil || canceled || chord != (hotkey.Chord{Modifiers: hotkey.Meta, Key: 'A'}) {
		t.Fatalf("%+v %v %v", chord, canceled, err)
	}
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if !closed {
		t.Fatal("capture retained tap")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDarwinShortcutCaptureCancelAndLoss(t *testing.T) {
	p, _ := hotkey.PolicyFor(hotkey.HoldToTalk)
	for _, loss := range []bool{false, true} {
		f := &fakeKeyboard{}
		c := &ShortcutCapturer{open: func(bool) (keyboardSource, error) { return f, nil }}
		ctx, cancel := context.WithCancel(context.Background())
		if loss {
			f.send(keyboardEvent{lost: true})
		} else {
			cancel()
		}
		_, canceled, err := c.Capture(ctx, p, nil)
		cancel()
		if loss && err == nil {
			t.Fatal("loss completed a capture")
		}
		if !loss && (!canceled || err != nil) {
			t.Fatalf("cancel: %v %v", canceled, err)
		}
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDarwinShortcutCapturePreviewDoesNotBlockCancellation(t *testing.T) {
	f := &fakeKeyboard{}
	c := &ShortcutCapturer{open: func(bool) (keyboardSource, error) { return f, nil }}
	p, _ := hotkey.PolicyFor(hotkey.HoldToTalk)
	entered, unblock, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	f.send(keyboardEvent{code: 59, down: true})
	go func() {
		defer close(done)
		_, _, _ = c.Capture(context.Background(), p, func(hotkey.Chord) { close(entered); <-unblock })
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("missing preview")
	}
	c.Cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("blocked preview blocked capture cancellation")
	}
	// A canceled capture must not spawn another worker over a stuck callback.
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, _, err := c.Capture(ctx, p, nil); err != errShortcutCaptureBusy {
			t.Fatalf("stuck preview admitted capture: %v", err)
		}
	}
	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked preview blocked Close")
	}
	close(unblock)
	if _, _, err := c.Capture(context.Background(), p, nil); err == nil {
		t.Fatal("closed capturer restarted")
	}
}
