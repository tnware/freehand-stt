//go:build linux && cgo

package keyboard

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/hotkey"
)

type holdSource interface {
	keyboardSource
	ungrabActive()
	released() int
	held(uint32) int
	modifierMasks() [4]uint16
}

func (k *nativeKeyboard) modifierMasks() [4]uint16 { return k.mods }
func openHold(chord hotkey.Chord) (holdSource, error) {
	k, err := newNativeKeyboard()
	if err != nil {
		return nil, err
	}
	if k.released() != 1 {
		_ = k.close()
		return nil, errors.New("Release all keyboard keys before enabling hold-to-talk.")
	}
	var mask uint16
	for i, mod := range []hotkey.Modifier{hotkey.Ctrl, hotkey.Alt, hotkey.Shift, hotkey.Meta} {
		if chord.Modifiers&mod != 0 {
			mask |= k.mods[i]
		}
	}
	if err = k.grab(chord.Key, mask); err != nil {
		_ = k.close()
		return nil, err
	}
	if k.released() != 1 {
		_ = k.close()
		return nil, errors.New("Release all keyboard keys before enabling hold-to-talk.")
	}
	return k, nil
}

type holdKeyboardRun struct {
	source holdSource
	stop   chan struct{}
	cancel atomic.Bool
	done   chan struct{}
}
type HoldHook struct {
	mu                     sync.Mutex
	closed                 bool
	toggle                 bool
	run                    *holdKeyboardRun
	chord                  hotkey.Chord
	failed                 atomic.Bool
	press, release, cancel func()
	open                   func(hotkey.Chord) (holdSource, error)
}

func NewHoldHook(press, release, cancel func()) *HoldHook {
	return &HoldHook{press: press, release: release, cancel: cancel, open: openHold}
}
func (h *HoldHook) Start(value string) error { return h.Configure(value) }
func (h *HoldHook) Configure(value string) error {
	chord, err := hotkey.ParseHold(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	if chord != (hotkey.Chord{}) && chord.ModifierOnly() {
		return errors.New("X11 hold-to-talk requires a primary key; modifier-only holds are unavailable.")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return errors.New("hold-to-talk hook is closed")
	}
	if chord == h.chord && h.run != nil && !h.failed.Load() {
		return nil
	}
	if chord == (hotkey.Chord{}) {
		if err = h.stopLocked(false); err != nil {
			return err
		}
		h.chord = chord
		h.failed.Store(false)
		return nil
	}
	// Stage a distinct replacement. BadAccess leaves the saved working grab alive.
	source, err := h.open(chord)
	if err != nil {
		if h.run == nil {
			h.failed.Store(true)
		}
		return err
	}
	if err = h.stopLocked(false); err != nil {
		return errors.Join(err, source.close())
	}
	run := &holdKeyboardRun{source: source, stop: make(chan struct{}, 1), done: make(chan struct{})}
	h.run, h.chord = run, chord
	h.failed.Store(false)
	go h.consume(run, chord)
	return nil
}
func (h *HoldHook) consume(run *holdKeyboardRun, chord hotkey.Chord) {
	defer close(run.done)
	defer run.source.close()
	r := hotkey.Reducer{Chord: chord}
	masks := run.source.modifierMasks()
	ticker := time.NewTicker(keyboardPollInterval)
	defer ticker.Stop()
	var activeSince time.Time
	var lastCheck time.Time
	missingRelease := false
	primaryHeld := false
	call := func(f func()) {
		if f != nil {
			f()
		}
	}
	stop := func(cancel bool) {
		if r.ForceRelease() == hotkey.Released {
			if cancel {
				call(h.cancel)
			} else {
				call(h.release)
			}
		}
		primaryHeld = false
		activeSince = time.Time{}
	}
	lost := func() {
		h.failed.Store(true)
		// Drop every native grab before an application cancellation callback.
		_ = run.source.close()
		stop(true)
	}
	for {
		select {
		case <-run.stop:
			stop(run.cancel.Load())
			return
		case now := <-ticker.C:
			if shortcutCaptureActive.Load() {
				run.source.ungrabActive()
				stop(true)
			}
			// A stale/missing release must not leave capture running. Normal edges use
			// the shared reducer; this watchdog cancels rather than transcribes.
			if !activeSince.IsZero() && now.Sub(lastCheck) >= 100*time.Millisecond {
				lastCheck = now
				held := run.source.held(chord.Key)
				if held < 0 || (held == 0 && missingRelease) || now.Sub(activeSince) > 2*time.Minute {
					lost()
					return
				}
				missingRelease = held == 0
			}
			for i := 0; i < 256; i++ {
				select {
				case <-run.stop:
					stop(run.cancel.Load())
					return
				default:
				}
				e, ok := run.source.read()
				if !ok {
					break
				}
				if e.lost {
					lost()
					return
				}
				if shortcutCaptureActive.Load() {
					continue
				}
				if e.code < 8 {
					continue
				}
				if h.toggle && e.wire != chord.Key {
					continue
				}
				// Passive grabs begin on primary-down, after modifiers were pressed.
				// Seed only from that server event's modifier state, never a late snapshot.
				if e.wire == chord.Key && e.down && !primaryHeld {
					for j, key := range []uint32{0x11, 0x12, 0x10, 0x5b} {
						r.Event(key, e.state&masks[j] != 0)
					}
					primaryHeld = true
				}
				edge := r.Event(e.wire, e.down)
				if e.wire == chord.Key && !e.down {
					primaryHeld = false
				}
				switch edge {
				case hotkey.Pressed:
					activeSince = now
					lastCheck = now
					missingRelease = false
					call(h.press)
				case hotkey.Released:
					activeSince = time.Time{}
					call(h.release)
					// Releasing either required modifier ends a hold conservatively, even
					// if its other physical side is still down. Do not keep swallowing keys.
					if e.wire != chord.Key {
						run.source.ungrabActive()
						r.ForceRelease()
						primaryHeld = false
					}
				}
			}
		}
	}
}
func (h *HoldHook) stopLocked(cancel bool) error {
	if h.run == nil {
		return nil
	}
	run := h.run
	if cancel {
		run.cancel.Store(true)
	}
	select {
	case run.stop <- struct{}{}:
	default:
	}
	// Release the native grab before waiting on arbitrary application callbacks.
	err := run.source.close()
	select {
	case <-run.done:
		h.run = nil
		return err
	case <-time.After(keyboardCloseBound):
		h.failed.Store(true)
		return errors.Join(err, errors.New("hold-to-talk callback shutdown timed out"))
	}
}
func (h *HoldHook) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	return h.stopLocked(true)
}
func (h *HoldHook) Available() (bool, string) {
	h.mu.Lock()
	closed := h.closed
	h.mu.Unlock()
	if closed {
		return false, "Hold-to-talk is closed."
	}
	if h.failed.Load() {
		return false, "The X11 keyboard source stopped or its key map changed. Release the keys and retry hold-to-talk."
	}
	return HoldAvailability()
}
func HoldAvailability() (bool, string) {
	if !X11Session() {
		return false, linuxKeyboardHelp
	}
	return true, "X11 primary-key holds use passive grabs with real press/release edges. Availability is checked when a shortcut is enabled; modifier-only holds are unavailable."
}
