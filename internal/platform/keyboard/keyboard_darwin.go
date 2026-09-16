//go:build darwin

package keyboard

/*
#cgo CFLAGS: -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework ApplicationServices -framework Carbon
#include "keyboard_darwin.h"
#include <ApplicationServices/ApplicationServices.h>
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

const keyboardCloseBound = time.Second
const keyboardPollInterval = 5 * time.Millisecond
const keyboardPermissionHelp = "Enable Freehand in System Settings > Privacy & Security > Input Monitoring, then restart Freehand."

// KeyboardAuthorization checks only; it never opens a permission prompt.
func KeyboardAuthorization() bool        { return C.fh_keyboard_authorized() != 0 }
func keyboardCaptureAuthorization() bool { return C.fh_keyboard_capture_authorized() != 0 }

// A fresh reducer must not infer a press from keys held across a lost source.
// This is a passive query; it never requests access or posts input.
func keyboardKeysReleased() bool {
	for code := 0; code < 128; code++ {
		if code != 57 && C.CGEventSourceKeyState(C.kCGEventSourceStateCombinedSessionState, C.CGKeyCode(code)) {
			return false
		}
	}
	return true
}

type keyboardSource interface {
	read() (keyboardEvent, bool)
	close() error
}
type nativeKeyboard struct {
	mu   sync.Mutex
	ptr  *C.fh_keyboard
	done chan struct{}
}

func openKeyboard(capture bool) (keyboardSource, error) {
	if !KeyboardAuthorization() {
		return nil, errors.New(keyboardPermissionHelp)
	}
	if capture && !keyboardCaptureAuthorization() {
		return nil, errors.New("Shortcut capture also requires Accessibility. Enable Freehand in System Settings > Privacy & Security > Accessibility, then restart Freehand.")
	}
	k := allocateKeyboard(capture)
	if k == nil {
		return nil, errors.New("keyboard event queue allocation failed")
	}
	ptr := k.ptr
	C.fh_keyboard_retain(ptr) // owner keeps native state alive even on close timeout
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(k.done)
		defer C.fh_keyboard_release(ptr)
		C.fh_keyboard_run(ptr)
	}()
	ticker := time.NewTicker(keyboardPollInterval)
	defer ticker.Stop()
	timeout := time.NewTimer(keyboardCloseBound)
	defer timeout.Stop()
	for {
		switch C.fh_keyboard_ready(ptr) {
		case 1:
			return k, nil
		case -1:
			_ = k.close()
			return nil, errors.New("macOS keyboard event tap could not start; check Input Monitoring, Accessibility for shortcut capture, and unlock the session or exit Secure Input")
		}
		select {
		case <-ticker.C:
		case <-timeout.C:
			_ = k.close()
			return nil, errors.New("macOS keyboard event tap startup timed out")
		}
	}
}
func allocateKeyboard(capture bool) *nativeKeyboard {
	flag := C.int(0)
	if capture {
		flag = 1
	}
	ptr := C.fh_keyboard_new(flag)
	if ptr == nil {
		return nil
	}
	return &nativeKeyboard{ptr: ptr, done: make(chan struct{})}
}
func (k *nativeKeyboard) read() (keyboardEvent, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ptr == nil {
		return keyboardEvent{}, false
	}
	var e C.fh_keyboard_event
	if C.fh_keyboard_read(k.ptr, &e) == 0 {
		return keyboardEvent{}, false
	}
	return keyboardEvent{code: uint16(e.code), down: e.down != 0, lost: e.lost != 0}, true
}
func (k *nativeKeyboard) close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ptr == nil {
		return nil
	}
	C.fh_keyboard_stop(k.ptr)
	var err error
	select {
	case <-k.done:
	case <-time.After(keyboardCloseBound):
		err = errors.New("keyboard event tap shutdown timed out; native owner is stopping")
	}
	C.fh_keyboard_release(k.ptr)
	k.ptr = nil
	return err
}

// Queue-only test seam. It never creates a tap or synthesizes/posts an event.
func newKeyboardTestQueue(capture bool) *nativeKeyboard {
	k := allocateKeyboard(capture)
	close(k.done)
	return k
}
func (k *nativeKeyboard) feed(code uint16, down, injected, repeat bool) bool {
	flags := func(b bool) C.int {
		if b {
			return 1
		}
		return 0
	}
	return C.fh_keyboard_feed(k.ptr, C.uint16_t(code), flags(down), flags(injected), flags(repeat)) != 0
}
