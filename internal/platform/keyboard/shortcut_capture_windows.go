//go:build windows

package keyboard

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/hotkey"
)

var shortcutCaptureActive atomic.Bool

var errShortcutCaptureBusy = errors.New("shortcut capture is already in progress")

type shortcutCaptureOutcome struct {
	chord    hotkey.Chord
	canceled bool
	err      error
}

type shortcutCaptureLoop struct {
	recorder hotkey.CaptureReducer
	preview  hotkey.Chord
	progress chan hotkey.Chord
	ready    chan error
	result   chan shortcutCaptureOutcome
	done     chan struct{}
	once     sync.Once
	threadID atomic.Uint32
}

func newShortcutCaptureLoop(policy hotkey.ShortcutPolicy) *shortcutCaptureLoop {
	return &shortcutCaptureLoop{
		recorder: hotkey.CaptureReducer{Policy: policy},
		progress: make(chan hotkey.Chord, 8),
		ready:    make(chan error, 1),
		result:   make(chan shortcutCaptureOutcome, 1),
		done:     make(chan struct{}),
	}
}

func (l *shortcutCaptureLoop) finish(outcome shortcutCaptureOutcome) {
	l.once.Do(func() {
		l.result <- outcome
		postQuitMessage.Call(0)
	})
}

func (l *shortcutCaptureLoop) stop() {
	if threadID := l.threadID.Load(); threadID != 0 {
		postThreadMessage.Call(uintptr(threadID), 0x0012, 0, 0) // WM_QUIT
	}
}

func (l *shortcutCaptureLoop) run(ctx context.Context) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(l.done)

	var msg nativeMessage
	peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
	threadID, _, _ := getCurrentThreadID.Call()
	l.threadID.Store(uint32(threadID))
	defer l.threadID.Store(0)
	if err := ctx.Err(); err != nil {
		l.ready <- err
		return
	}
	callback := syscall.NewCallback(l.callback)
	hook, _, callErr := setWindowsHookEx.Call(whKeyboardLL, callback, 0, 0)
	if hook == 0 {
		l.ready <- errors.New("shortcut capture hook could not start: " + callErr.Error())
		return
	}
	defer unhookWindowsHookEx.Call(hook)
	l.ready <- nil

	for {
		if ctx.Err() != nil {
			return
		}
		result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) <= 0 {
			return
		}
	}
}

func (l *shortcutCaptureLoop) callback(code int32, wparam uintptr, data *keyboardData) uintptr {
	if code < 0 {
		return nextKeyboardHook(code, wparam, data)
	}
	if data.Flags&(llkhfInjected|llkhfLowerInjected) != 0 {
		return nextKeyboardHook(code, wparam, data)
	}
	down := wparam == wmKeyDown || wparam == wmSysKeyDown
	up := wparam == wmKeyUp || wparam == wmSysKeyUp
	if !down && !up {
		return nextKeyboardHook(code, wparam, data)
	}
	result := l.recorder.Event(data.VKCode, down)
	if result.State == hotkey.CaptureWaiting {
		preview := l.recorder.Preview()
		if preview != l.preview {
			l.preview = preview
			select {
			case l.progress <- preview:
			default:
			}
		}
	}
	switch result.State {
	case hotkey.CaptureComplete:
		l.finish(shortcutCaptureOutcome{chord: result.Chord})
	case hotkey.CaptureCanceled:
		l.finish(shortcutCaptureOutcome{canceled: true})
	case hotkey.CaptureRejected:
		l.finish(shortcutCaptureOutcome{err: result.Err})
	}
	return 1 // Suppress the complete physical edge while capture is active.
}

type ShortcutCapturer struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   <-chan struct{}
	loop   *shortcutCaptureLoop
	closed bool
}

// Capture records one chord using the backend-owned action policy.
func (c *ShortcutCapturer) Capture(parent context.Context, policy hotkey.ShortcutPolicy, changed func(hotkey.Chord)) (hotkey.Chord, bool, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return hotkey.Chord{}, false, errors.New("shortcut capture is closed")
	}
	if !shortcutCaptureActive.CompareAndSwap(false, true) {
		c.mu.Unlock()
		return hotkey.Chord{}, false, errShortcutCaptureBusy
	}
	defer shortcutCaptureActive.Store(false)

	ctx, cancel := context.WithCancel(parent)
	loop := newShortcutCaptureLoop(policy)
	done := make(chan struct{})
	c.cancel = cancel
	c.done = done
	c.loop = loop
	c.mu.Unlock()
	// Stop the native source even while a presentation callback is blocked.
	// Track the short cancellation callback through Capture's completion too.
	stopDone := make(chan struct{})
	stopOnCancel := context.AfterFunc(ctx, func() { defer close(stopDone); loop.stop() })
	defer func() {
		cancel()
		if !stopOnCancel() {
			<-stopDone
		}
		c.mu.Lock()
		c.cancel = nil
		c.done = nil
		c.loop = nil
		close(done)
		c.mu.Unlock()
	}()

	go loop.run(ctx)
	select {
	case err := <-loop.ready:
		if err != nil {
			<-loop.done
			if ctx.Err() != nil {
				return captureContextResult(parent, policy.Action)
			}
			return hotkey.Chord{}, false, err
		}
	case <-ctx.Done():
		err := <-loop.ready
		if err == nil {
			loop.stop()
		}
		<-loop.done
		return captureContextResult(parent, policy.Action)
	}

	for {
		select {
		case preview := <-loop.progress:
			if changed != nil {
				changed(preview)
			}
		case outcome := <-loop.result:
			<-loop.done
			return outcome.chord, outcome.canceled, outcome.err
		case <-ctx.Done():
			loop.stop()
			<-loop.done
			return captureContextResult(parent, policy.Action)
		case <-loop.done:
			if ctx.Err() != nil {
				return captureContextResult(parent, policy.Action)
			}
			select {
			case outcome := <-loop.result:
				return outcome.chord, outcome.canceled, outcome.err
			default:
				return hotkey.Chord{}, false, errors.New("shortcut capture ended unexpectedly")
			}
		}
	}
}

func captureContextResult(parent context.Context, action hotkey.ShortcutAction) (hotkey.Chord, bool, error) {
	if errors.Is(parent.Err(), context.DeadlineExceeded) {
		return hotkey.Chord{}, false, hotkey.NewRejection(hotkey.RejectionTimedOut, action, "Shortcut capture timed out. Try again when you are ready to press the chord.")
	}
	return hotkey.Chord{}, true, nil
}

func (c *ShortcutCapturer) Cancel() {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *ShortcutCapturer) Close() error {
	c.mu.Lock()
	c.closed = true
	cancel, done, loop := c.cancel, c.done, c.loop
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	deadline := time.NewTimer(keyboardCloseBound)
	defer deadline.Stop()
	if loop != nil {
		loop.stop()
		select {
		case <-loop.done:
		case <-deadline.C:
			return errors.New("shortcut capture native shutdown timed out")
		}
	}
	if done != nil {
		select {
		case <-done:
		case <-deadline.C:
			return errors.New("shortcut capture callback shutdown timed out")
		}
	}
	return nil
}
