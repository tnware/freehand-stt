//go:build linux && cgo

package keyboard

/*
#cgo pkg-config: xcb
#include "keyboard_linux.h"
*/
import "C"

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

const keyboardCloseBound = time.Second
const keyboardPollInterval = 5 * time.Millisecond
const linuxKeyboardHelp = "Native shortcut capture and hold-to-talk require a local X11 session with XKB detectable autorepeat. Wayland and XWayland are not supported."

// X11Session is deliberately strict: a DISPLAY under Wayland is not permission
// to treat XWayland as a desktop-global keyboard source. Remote displays are
// excluded so capture cannot grab a keyboard on another computer.
func X11Session() bool {
	return x11Session(os.Getenv("XDG_SESSION_TYPE"), os.Getenv("WAYLAND_DISPLAY"), os.Getenv("DISPLAY"))
}
func x11Session(session, wayland, display string) bool {
	if strings.ToLower(session) != "x11" || wayland != "" {
		return false
	}
	if !strings.HasPrefix(display, ":") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(display, ":"), ".")
	if len(parts) > 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

type keyboardEvent struct {
	code       uint16
	wire       uint32
	state      uint16
	down, lost bool
}
type keyboardSource interface {
	read() (keyboardEvent, bool)
	close() error
}
type nativeKeyboard struct {
	mu   sync.Mutex
	ptr  *C.fh_xkeyboard
	mods [4]uint16
}

var nativeOpenSlot = make(chan struct{}, 1)

func newNativeKeyboard() (*nativeKeyboard, error) {
	if !X11Session() {
		return nil, errors.New(linuxKeyboardHelp)
	}
	// Limit a stalled local server handshake to one pending native owner. It
	// cannot acquire grabs, and closes its connection if admission times out.
	select {
	case nativeOpenSlot <- struct{}{}:
	default:
		return nil, errors.New("An X11 keyboard connection is still starting.")
	}
	result := make(chan *C.fh_xkeyboard)
	cancel := make(chan struct{})
	go func() {
		defer func() { <-nativeOpenSlot }()
		p := C.fh_xopen()
		select {
		case result <- p:
		case <-cancel:
			C.fh_xclose(p)
		}
	}()
	var p *C.fh_xkeyboard
	select {
	case p = <-result:
	case <-time.After(1500 * time.Millisecond):
		close(cancel)
		return nil, errors.New("X11 keyboard connection timed out.")
	}
	if p == nil {
		return nil, errors.New("X11 keyboard initialization failed; XKB detectable autorepeat and an unambiguous modifier map are required.")
	}
	k := &nativeKeyboard{ptr: p}
	for i := range k.mods {
		k.mods[i] = uint16(C.fh_xmod(p, C.uint(i)))
	}
	return k, nil
}
func openKeyboard(capture bool) (keyboardSource, error) {
	k, err := newNativeKeyboard()
	if err != nil {
		return nil, err
	}
	if !capture {
		_ = k.close()
		return nil, errors.New("X11 keyboard source requires a specific passive shortcut grab")
	}
	if C.fh_xreleased(k.ptr) != 1 {
		_ = k.close()
		return nil, errors.New("Release all keyboard keys before recording a shortcut.")
	}
	if C.fh_xcapture(k.ptr) == 0 {
		_ = k.close()
		return nil, errors.New("Another application owns the keyboard. Release the keys and retry shortcut capture.")
	}
	// Never seed a fresh reducer from keys held across admission.
	if C.fh_xreleased(k.ptr) != 1 {
		_ = k.close()
		return nil, errors.New("Release all keyboard keys before recording a shortcut.")
	}
	return k, nil
}
func (k *nativeKeyboard) read() (keyboardEvent, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ptr == nil {
		return keyboardEvent{}, false
	}
	var e C.fh_xevent
	if C.fh_xread(k.ptr, &e) == 0 {
		return keyboardEvent{}, false
	}
	return keyboardEvent{code: uint16(e.code), wire: uint32(C.fh_xwire(k.ptr, C.uint(e.code))), state: uint16(e.state), down: e.down != 0, lost: e.lost != 0}, true
}
func (k *nativeKeyboard) close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ptr != nil {
		C.fh_xclose(k.ptr)
		k.ptr = nil
	}
	return nil
}
func (k *nativeKeyboard) ungrabActive() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ptr != nil {
		C.fh_xungrab_active(k.ptr)
	}
}
func (k *nativeKeyboard) released() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ptr == nil {
		return -1
	}
	return int(C.fh_xreleased(k.ptr))
}
func (k *nativeKeyboard) grab(key uint32, mask uint16) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	code := C.fh_xcode(k.ptr, C.uint32_t(key))
	if code == 0 {
		return errors.New("The shortcut key has no unique mapping on this X11 keyboard.")
	}
	if C.fh_xgrab(k.ptr, code, C.uint(mask)) == 0 {
		return errors.New("The X11 shortcut is already in use or could not be grabbed.")
	}
	return nil
}

type keyboardAdapter struct{ held [256]uint32 }

func (a *keyboardAdapter) event(e keyboardEvent) (uint32, bool, bool) {
	if e.code < 8 || e.code >= 256 {
		return 0, false, false
	}
	if e.down {
		if a.held[e.code] != 0 {
			return 0, false, false
		}
		a.held[e.code] = e.wire
	} else {
		if a.held[e.code] == 0 {
			return 0, false, false
		}
		a.held[e.code] = 0
	}
	if e.wire == 0x10 || e.wire == 0x11 || e.wire == 0x12 || e.wire == 0x5b {
		for code, other := range a.held {
			if uint16(code) != e.code && other == e.wire {
				return 0, false, false
			}
		}
		// On key-down the just-stored side itself is present.
		if e.down {
			return e.wire, true, true
		}
	}
	return e.wire, e.down, true
}

func (k *nativeKeyboard) held(key uint32) int {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ptr == nil {
		return -1
	}
	return int(C.fh_xheld(k.ptr, C.uint32_t(key)))
}
