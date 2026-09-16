//go:build windows

package keyboard

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"testing/synctest"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/hotkey"
)

func TestNativeKeyboardCallbacksForwardNegative32BitCodeWithoutReadingData(t *testing.T) {
	oldNext := nextKeyboardHook
	t.Cleanup(func() { nextKeyboardHook = oldNext })
	forwarded := 0
	nextKeyboardHook = func(code int32, wparam uintptr, data *keyboardData) uintptr {
		if code != -1 || wparam != wmKeyDown || data != nil {
			t.Errorf("forwarded code=%d wparam=%x data=%p", code, wparam, data)
		}
		forwarded++
		return 37
	}
	h := NewHoldHook(nil, nil, nil)
	capture := newShortcutCaptureLoop(hotkey.ShortcutPolicy{})
	for _, callback := range []func(int32, uintptr, *keyboardData) uintptr{h.callback, capture.callback} {
		// Exercise the actual Windows callback trampoline, with unspecified upper
		// register bits zero. The old Go-int signature misread -1 as 4294967295.
		address := syscall.NewCallback(callback)
		got, _, _ := syscall.SyscallN(address, uintptr(uint32(0xffffffff)), wmKeyDown, 0)
		if got != 37 {
			t.Fatalf("callback returned %d", got)
		}
	}
	if forwarded != 2 || len(h.edges) != 0 || len(capture.progress) != 0 {
		t.Fatal("negative code was processed")
	}
}

func TestNativeKeyboardCallbackPreservesNativeDataLayout(t *testing.T) {
	if unsafe.Sizeof(keyboardData{}) != 24 || unsafe.Offsetof(keyboardData{}.ExtraInfo) != 16 {
		t.Fatal("KBDLLHOOKSTRUCT layout does not match Windows x64")
	}
	oldNext := nextKeyboardHook
	t.Cleanup(func() { nextKeyboardHook = oldNext })
	nextKeyboardHook = func(int32, uintptr, *keyboardData) uintptr { return 23 }
	h := NewHoldHook(nil, nil, nil)
	h.reducer = hotkey.Reducer{Chord: hotkey.Chord{Key: 0x41}}
	callback := syscall.NewCallback(h.callback)
	data := keyboardData{VKCode: 0x41}
	got, _, _ := syscall.SyscallN(callback, 0, wmKeyDown, uintptr(unsafe.Pointer(&data)))
	if got != 23 {
		t.Fatalf("next hook result=%d", got)
	}
	select {
	case edge := <-h.edges:
		if edge != hotkey.Pressed {
			t.Fatalf("edge=%v", edge)
		}
	default:
		t.Fatal("typed callback did not read the native key data")
	}
}

func fixtureHold(press, release, cancel func()) *HoldHook {
	h := NewHoldHook(press, release, cancel)
	h.initialized, h.started, h.threadID = true, true, 1
	h.postQuit = func(uint32) error { h.sourceStopped(); return nil }
	go h.consume()
	return h
}

func TestHoldCloseBoundsCallbacksAndRetainsCompletion(t *testing.T) {
	for _, phase := range []string{"press", "release", "cancel"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				entered := make(chan struct{})
				unblock := make(chan struct{})
				block := func() { close(entered); <-unblock }
				press, release, cancel := func() {}, func() {}, func() {}
				switch phase {
				case "press":
					press = block
				case "release":
					release = block
				case "cancel":
					cancel = block
				}
				h := fixtureHold(press, release, cancel)
				h.edges <- hotkey.Pressed
				synctest.Wait()
				if phase == "release" {
					h.edges <- hotkey.Released
					<-entered
				}
				start := time.Now()
				err := h.Close()
				if err == nil || !strings.Contains(err.Error(), "callback shutdown timed out") || time.Since(start) != keyboardCloseBound {
					t.Fatalf("error=%v elapsed=%s", err, time.Since(start))
				}
				select {
				case <-h.done:
				default:
					t.Fatal("native source is still owned by blocked recorder callback")
				}
				select {
				case <-h.callbacksDone:
					t.Fatal("lost in-flight callback tracking")
				default:
				}
				close(unblock)
				<-h.callbacksDone
				if err := h.Close(); err != nil {
					t.Fatalf("completion after timeout: %v", err)
				}
			})
		})
	}
}

func TestHoldCloseRejectsQueuedPressAfterBlockedCallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		unblock := make(chan struct{})
		presses, releases, cancels := 0, 0, 0
		h := fixtureHold(func() { presses++; <-unblock }, func() { releases++ }, func() { cancels++ })
		h.edges <- hotkey.Pressed
		synctest.Wait()
		h.edges <- hotkey.Released
		h.edges <- hotkey.Pressed
		if err := h.Close(); err == nil {
			t.Fatal("expected pending callback timeout")
		}
		close(unblock)
		<-h.callbacksDone
		if presses != 1 || releases != 0 || cancels != 1 {
			t.Fatalf("press=%d release=%d cancel=%d", presses, releases, cancels)
		}
	})
}

func TestHoldConfigurationReleaseStaysOnTrackedConsumer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		unblock := make(chan struct{})
		releases := 0
		h := fixtureHold(func() { <-unblock }, func() { releases++ }, nil)
		h.edges <- hotkey.Pressed
		synctest.Wait()
		if err := h.Configure(""); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if releases != 0 {
			t.Fatal("configuration spawned a concurrent release callback")
		}
		close(unblock)
		synctest.Wait()
		if releases != 1 {
			t.Fatalf("release count=%d", releases)
		}
		if err := h.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestHoldOverflowDiscardsQueuedPresses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		unblock := make(chan struct{})
		presses, cancels := 0, 0
		h := fixtureHold(func() { presses++; <-unblock }, nil, func() { cancels++ })
		h.edges <- hotkey.Pressed
		synctest.Wait()
		h.mu.Lock()
		for range cap(h.edges) {
			h.enqueueLocked(hotkey.Pressed)
		}
		h.enqueueLocked(hotkey.Released)
		h.mu.Unlock()
		close(unblock)
		synctest.Wait()
		if presses != 1 || cancels != 1 {
			t.Fatalf("press=%d cancel=%d", presses, cancels)
		}
		if err := h.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestHoldCloseBeforeThreadPublicationIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := NewHoldHook(nil, nil, nil)
		h.initialized = true // Start owns the source, but its OS thread has not published yet.
		go h.consume()
		if err := h.Close(); err == nil || !strings.Contains(err.Error(), "native shutdown timed out") {
			t.Fatalf("error=%v", err)
		}
		// The delayed startup observes closed and retires its source; no callback is abandoned.
		h.sourceStopped()
		if err := h.Close(); err != nil {
			t.Fatal(err)
		}
		if err := h.Start(""); err == nil {
			t.Fatal("closed owner restarted")
		}
	})
}

func TestShortcutCloseBoundsSourceAndProgressCompletion(t *testing.T) {
	for _, sourceBlocked := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			loop := newShortcutCaptureLoop(hotkey.ShortcutPolicy{})
			done := make(chan struct{})
			c := &ShortcutCapturer{cancel: cancel, done: done, loop: loop}
			if !sourceBlocked {
				close(loop.done)
			}
			start := time.Now()
			if err := c.Close(); err == nil || time.Since(start) != keyboardCloseBound {
				t.Fatalf("error=%v elapsed=%s", err, time.Since(start))
			}
			if ctx.Err() == nil {
				t.Fatal("capture context was not cancelled")
			}
			if c.done == nil || c.loop == nil {
				t.Fatal("timed-out capture owner was discarded")
			}
			if sourceBlocked {
				close(loop.done)
			}
			close(done)
			if err := c.Close(); err != nil {
				t.Fatal(err)
			}
			if _, _, err := c.Capture(t.Context(), hotkey.ShortcutPolicy{}, nil); err == nil {
				t.Fatal("closed capture accepted work")
			}
		})
	}
}
