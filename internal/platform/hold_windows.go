//go:build windows

package platform

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

const (
	whKeyboardLL       = 13
	wmKeyDown          = 0x0100
	wmKeyUp            = 0x0101
	wmSysKeyDown       = 0x0104
	wmSysKeyUp         = 0x0105
	llkhfLowerInjected = 0x00000002
	llkhfInjected      = 0x00000010
)

var setWindowsHookEx = user32.NewProc("SetWindowsHookExW")
var unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
var callNextHookEx = user32.NewProc("CallNextHookEx")
var getMessage = user32.NewProc("GetMessageW")
var postThreadMessage = user32.NewProc("PostThreadMessageW")
var getCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")

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
	mu        sync.Mutex
	reducer   hotkey.Reducer
	started   bool
	threadID  uint32
	edges     chan hotkey.Edge
	overflow  chan struct{}
	done      chan struct{}
	press     func()
	release   func()
	cancel    func()
	available atomic.Bool
}

func NewHoldHook(press, release, cancel func()) *HoldHook {
	return &HoldHook{edges: make(chan hotkey.Edge, 8), overflow: make(chan struct{}, 1), done: make(chan struct{}), press: press, release: release, cancel: cancel}
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
	ready := make(chan error, 1)
	go h.consume()
	go h.loop(chord, ready)
	if err = <-ready; err != nil {
		return err
	}
	h.available.Store(true)
	return nil
}

func (h *HoldHook) loop(chord hotkey.Chord, ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	tid, _, _ := getCurrentThreadID.Call()
	h.mu.Lock()
	h.threadID = uint32(tid)
	h.reducer = hotkey.Reducer{Chord: chord}
	h.mu.Unlock()
	callback := syscall.NewCallback(func(code int, wparam, lparam uintptr) uintptr {
		if code >= 0 {
			data := (*keyboardData)(unsafe.Pointer(lparam))
			if !shortcutCaptureActive.Load() && data.Flags&(llkhfInjected|llkhfLowerInjected) == 0 {
				down := wparam == wmKeyDown || wparam == wmSysKeyDown
				up := wparam == wmKeyUp || wparam == wmSysKeyUp
				if down || up {
					h.mu.Lock()
					edge := h.reducer.Event(data.VKCode, down)
					h.mu.Unlock()
					if edge != hotkey.NoEdge {
						select {
						case h.edges <- edge:
						default:
							select {
							case h.overflow <- struct{}{}:
							default:
							}
						}
					}
				}
			}
		}
		next, _, _ := callNextHookEx.Call(0, uintptr(code), wparam, lparam)
		return next
	})
	hook, _, callErr := setWindowsHookEx.Call(whKeyboardLL, callback, 0, 0)
	if hook == 0 {
		ready <- errors.New("low-level keyboard hook could not start: " + callErr.Error())
		close(h.done)
		return
	}
	h.mu.Lock()
	h.started = true
	h.mu.Unlock()
	ready <- nil
	var msg nativeMessage
	for {
		result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) <= 0 {
			break
		}
	}
	unhookWindowsHookEx.Call(hook)
	h.available.Store(false)
	h.mu.Lock()
	edge := h.reducer.ForceRelease()
	h.started = false
	h.mu.Unlock()
	if edge == hotkey.Released && h.release != nil {
		h.release()
	}
	if h.cancel != nil {
		h.cancel()
	}
	close(h.done)
}

func (h *HoldHook) consume() {
	for {
		select {
		case edge := <-h.edges:
			if edge == hotkey.Pressed && h.press != nil {
				h.press()
			} else if edge == hotkey.Released && h.release != nil {
				h.release()
			}
		case <-h.overflow:
			h.mu.Lock()
			h.reducer.ForceRelease()
			h.mu.Unlock()
			if h.cancel != nil {
				h.cancel()
			}
		case <-h.done:
			return
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
	if !h.started {
		if value == "" {
			return nil
		}
		return errors.New("hold-to-talk hook is unavailable")
	}
	if h.reducer.ForceRelease() == hotkey.Released && h.release != nil {
		go h.release()
	}
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
	started, tid := h.started, h.threadID
	h.mu.Unlock()
	if !started {
		return nil
	}
	if r, _, err := postThreadMessage.Call(uintptr(tid), 0x0012, 0, 0); r == 0 {
		return err
	}
	<-h.done
	return nil
}

func HoldAvailability() (bool, string) {
	return false, "Hold-to-talk availability is determined after the Windows keyboard hook starts."
}
