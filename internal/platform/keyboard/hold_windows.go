//go:build windows

package keyboard

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/hotkey"
	"golang.org/x/sys/windows"
)

const (
	whKeyboardLL       = 13
	wmKeyDown          = 0x0100
	wmKeyUp            = 0x0101
	wmSysKeyDown       = 0x0104
	wmSysKeyUp         = 0x0105
	llkhfLowerInjected = 0x00000002
	llkhfInjected      = 0x00000010
	keyboardCloseBound = 2 * time.Second
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var kernel32 = windows.NewLazySystemDLL("kernel32.dll")
var postQuitMessage = user32.NewProc("PostQuitMessage")

var setWindowsHookEx = user32.NewProc("SetWindowsHookExW")
var unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
var callNextHookEx = user32.NewProc("CallNextHookEx")
var getMessage = user32.NewProc("GetMessageW")
var postThreadMessage = user32.NewProc("PostThreadMessageW")
var getCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")
var peekMessage = user32.NewProc("PeekMessageW")

var nextKeyboardHook = func(code int32, wparam uintptr, data *keyboardData) uintptr {
	next, _, _ := callNextHookEx.Call(0, uintptr(code), wparam, uintptr(unsafe.Pointer(data)))
	return next
}

func stopKeyboardThread(tid uint32) error {
	if r, _, err := postThreadMessage.Call(uintptr(tid), 0x0012, 0, 0); r == 0 {
		return err
	}
	return nil
}

type keyboardData struct {
	VKCode    uint32
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type nativeMessage struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   struct{ X, Y int32 }
	Private uint32
}

type HoldHook struct {
	mu            sync.Mutex
	reducer       hotkey.Reducer
	started       bool
	initialized   bool
	closed        bool
	threadID      uint32
	edges         chan hotkey.Edge
	overflow      chan struct{}
	done          chan struct{}
	callbacksDone chan struct{}
	stop          chan struct{}
	postQuit      func(uint32) error
	press         func()
	release       func()
	cancel        func()
	available     atomic.Bool
}

func NewHoldHook(press, release, cancel func()) *HoldHook {
	return &HoldHook{
		edges: make(chan hotkey.Edge, 8), overflow: make(chan struct{}, 1),
		done: make(chan struct{}), callbacksDone: make(chan struct{}), stop: make(chan struct{}),
		postQuit: stopKeyboardThread, press: press, release: release, cancel: cancel,
	}
}

func (h *HoldHook) Start(value string) error {
	chord, err := hotkey.ParseHold(value)
	if value == "" {
		chord = hotkey.Chord{}
		err = nil
	}
	if err != nil {
		return err
	}
	h.mu.Lock()
	if h.closed || h.initialized {
		h.mu.Unlock()
		return errors.New("hold-to-talk hook is already started or closed")
	}
	h.initialized = true
	h.mu.Unlock()
	ready := make(chan error, 1)
	go h.consume()
	go h.loop(chord, ready)
	if err = <-ready; err != nil {
		return err
	}
	return nil
}

func (h *HoldHook) loop(chord hotkey.Chord, ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer h.sourceStopped()
	// Create the message queue before publishing its thread ID to Close.
	var msg nativeMessage
	peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
	tid, _, _ := getCurrentThreadID.Call()
	h.mu.Lock()
	h.threadID = uint32(tid)
	h.reducer = hotkey.Reducer{Chord: chord}
	closed := h.closed
	h.mu.Unlock()
	if closed {
		ready <- errors.New("hold-to-talk hook is closed")
		return
	}
	callback := syscall.NewCallback(h.callback)
	hook, _, callErr := setWindowsHookEx.Call(whKeyboardLL, callback, 0, 0)
	if hook == 0 {
		ready <- errors.New("low-level keyboard hook could not start: " + callErr.Error())
		return
	}
	defer unhookWindowsHookEx.Call(hook)
	h.mu.Lock()
	h.started = true
	h.available.Store(!h.closed)
	h.mu.Unlock()
	ready <- nil
	for {
		select {
		case <-h.stop:
			return
		default:
		}
		result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) <= 0 {
			break
		}
	}
}

// nCode is a Win32 32-bit int, including on amd64. The typed native pointer
// remains valid only during this callback and is never retained by Go.
func (h *HoldHook) callback(code int32, wparam uintptr, data *keyboardData) uintptr {
	if code >= 0 && !shortcutCaptureActive.Load() && data.Flags&(llkhfInjected|llkhfLowerInjected) == 0 {
		down := wparam == wmKeyDown || wparam == wmSysKeyDown
		up := wparam == wmKeyUp || wparam == wmSysKeyUp
		if down || up {
			h.mu.Lock()
			if !h.closed {
				if edge := h.reducer.Event(data.VKCode, down); edge != hotkey.NoEdge {
					h.enqueueLocked(edge)
				}
			}
			h.mu.Unlock()
		}
	}
	return nextKeyboardHook(code, wparam, data)
}

func (h *HoldHook) sourceStopped() {
	h.available.Store(false)
	h.mu.Lock()
	h.reducer.ForceRelease()
	h.started = false
	h.mu.Unlock()
	close(h.done)
}

func (h *HoldHook) consume() {
	defer close(h.callbacksDone)
	active := false
	cancel := func() {
		if active {
			active = false
			if h.cancel != nil {
				h.cancel()
			}
		}
	}
	for {
		// Source teardown never calls recorder code. The one tracked consumer
		// finishes any in-flight callback and then cancels without replaying edges.
		select {
		case <-h.stop:
			cancel()
			return
		case <-h.done:
			cancel()
			return
		default:
		}
		select {
		case <-h.overflow:
			cancel()
			continue
		default:
		}
		select {
		case edge := <-h.edges:
			h.mu.Lock()
			closed := h.closed || !h.started
			h.mu.Unlock()
			if closed {
				cancel()
				return
			}
			if edge == hotkey.Pressed && !active {
				active = true
				if h.press != nil {
					h.press()
				}
			} else if edge == hotkey.Released && active {
				active = false
				if h.release != nil {
					h.release()
				}
			}
		case <-h.overflow:
			cancel()
		case <-h.stop:
			cancel()
			return
		case <-h.done:
			cancel()
			return
		}
	}
}

func (h *HoldHook) drainLocked() {
	for {
		select {
		case <-h.edges:
		default:
			return
		}
	}
}

func (h *HoldHook) enqueueLocked(edge hotkey.Edge) {
	select {
	case h.edges <- edge:
	default:
		h.drainLocked()
		h.reducer.ForceRelease()
		select {
		case h.overflow <- struct{}{}:
		default:
		}
	}
}

func (h *HoldHook) Configure(value string) error {
	var chord hotkey.Chord
	var err error
	if value != "" {
		chord, err = hotkey.ParseHold(value)
		if err != nil {
			return err
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.started || h.closed {
		if value == "" {
			return nil
		}
		return errors.New("hold-to-talk hook is unavailable")
	}
	h.reducer.ForceRelease()
	h.drainLocked()
	// The consumer owns all callbacks, including a configuration release. This
	// also settles an already-consumed press whose physical release was queued.
	h.enqueueLocked(hotkey.Released)
	h.reducer = hotkey.Reducer{Chord: chord}
	return nil
}

func (h *HoldHook) Available() (bool, string) {
	if h.available.Load() {
		return true, "True hold-to-talk uses a Windows low-level keyboard press/release hook."
	}
	return false, "The Windows low-level keyboard hook is unavailable."
}

func (h *HoldHook) Close() error {
	h.mu.Lock()
	if !h.closed {
		h.closed = true
		close(h.stop)
	}
	initialized, tid := h.initialized, h.threadID
	h.available.Store(false)
	h.mu.Unlock()
	if !initialized {
		return nil
	}
	deadline := time.NewTimer(keyboardCloseBound)
	defer deadline.Stop()
	var stopErr error
	select {
	case <-h.done:
	default:
		if tid != 0 {
			stopErr = h.postQuit(tid)
		}
	}
	select {
	case <-h.done:
	case <-deadline.C:
		return errors.Join(stopErr, errors.New("hold-to-talk native shutdown timed out"))
	}
	select {
	case <-h.callbacksDone:
		return stopErr
	case <-deadline.C:
		// Retain both completion channels. A later Close can still observe the
		// tracked consumer finishing after the owning service cancels its work.
		return errors.Join(stopErr, errors.New("hold-to-talk callback shutdown timed out"))
	}
}

func HoldAvailability() (bool, string) {
	return false, "Hold-to-talk availability is determined after the Windows keyboard hook starts."
}
