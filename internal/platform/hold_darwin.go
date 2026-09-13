//go:build darwin

package platform

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type holdKeyboardRun struct {
	source keyboardSource
	stop   chan bool
	done   chan struct{}
}
type HoldHook struct {
	mu                     sync.Mutex
	closed                 bool
	run                    *holdKeyboardRun
	failed                 atomic.Bool
	press, release, cancel func()
	authorized             func() bool
	keysReleased           func() bool
	open                   func(bool) (keyboardSource, error)
}

func NewHoldHook(press, release, cancel func()) *HoldHook {
	return &HoldHook{press: press, release: release, cancel: cancel, authorized: KeyboardAuthorization, keysReleased: keyboardKeysReleased, open: openKeyboard}
}
func (h *HoldHook) Start(value string) error { return h.Configure(value) }
func (h *HoldHook) Configure(value string) error {
	value = strings.TrimSpace(value)
	chord, err := hotkey.ParseHold(value)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return errors.New("hold-to-talk hook is closed")
	}
	if value != "" && !h.authorized() {
		if h.run == nil {
			h.failed.Store(true)
		}
		return errors.New(keyboardPermissionHelp)
	}
	if value != "" && !h.keysReleased() {
		if h.run == nil {
			h.failed.Store(true)
		}
		return errors.New("Release all keyboard keys, then retry hold-to-talk.")
	}
	if value == "" {
		if err := h.stopLocked(false); err != nil {
			return err
		}
		h.failed.Store(false)
		return nil
	}
	// Stage the replacement before retiring the working hook. A failed settings
	// transaction must not silently disable the previous saved assignment.
	source, err := h.open(false)
	if err != nil {
		if h.run == nil {
			h.failed.Store(true)
		}
		return err
	}
	// Tap creation is asynchronous: keys may have gone down after preflight.
	// Do not seed an empty reducer from an already-held physical chord.
	if !h.keysReleased() {
		if h.run == nil {
			h.failed.Store(true)
		}
		return errors.Join(errors.New("Release all keyboard keys, then retry hold-to-talk."), source.close())
	}
	if err := h.stopLocked(false); err != nil {
		return errors.Join(err, source.close())
	}
	h.failed.Store(false)
	run := &holdKeyboardRun{source: source, stop: make(chan bool, 1), done: make(chan struct{})}
	h.run = run
	go h.consume(run, chord)
	return nil
}
func (h *HoldHook) consume(run *holdKeyboardRun, chord hotkey.Chord) {
	defer close(run.done)
	r := hotkey.Reducer{Chord: chord}
	var adapter keyboardAdapter
	ticker := time.NewTicker(keyboardPollInterval)
	defer ticker.Stop()
	call := func(f func()) {
		if f != nil {
			f()
		}
	}
	for {
		select {
		case cancel := <-run.stop:
			active := r.ForceRelease() == hotkey.Released
			if active {
				if cancel {
					call(h.cancel)
				} else {
					call(h.release)
				}
			}
			return
		case <-ticker.C:
			if shortcutCaptureActive.Load() {
				adapter = keyboardAdapter{}
				if r.ForceRelease() == hotkey.Released {
					call(h.cancel)
				}
			}
			// Bound each drain as well as the native ring; configuration/close cannot
			// starve behind a continuous stream of physical keyboard input.
			for i := 0; i < 256; i++ {
				select {
				case cancel := <-run.stop:
					if r.ForceRelease() == hotkey.Released {
						if cancel {
							call(h.cancel)
						} else {
							call(h.release)
						}
					}
					return
				default:
				}
				event, ok := run.source.read()
				if !ok {
					break
				}
				if event.lost {
					h.failed.Store(true)
					active := r.ForceRelease() == hotkey.Released
					_ = run.source.close()
					if active {
						call(h.cancel)
					}
					return
				}
				if shortcutCaptureActive.Load() {
					adapter = keyboardAdapter{}
					if r.ForceRelease() == hotkey.Released {
						call(h.cancel)
					}
					continue
				}
				key, down, ok := adapter.event(event)
				if !ok {
					continue
				}
				switch r.Event(key, down) {
				case hotkey.Pressed:
					call(h.press)
				case hotkey.Released:
					call(h.release)
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
	select {
	case run.stop <- cancel:
	default:
	}
	err := run.source.close() // native teardown does not depend on user callbacks
	select {
	case <-run.done:
		h.run = nil
		return err
	case <-time.After(keyboardCloseBound):
		h.failed.Store(true)
		return errors.Join(err, errors.New("hold-to-talk callback shutdown timed out"))
	}
}
func (h *HoldHook) Available() (bool, string) {
	if h.failed.Load() {
		return false, "The macOS keyboard event tap stopped; release the keys and click Retry hold-to-talk after unlocking the session or exiting Secure Input."
	}
	h.mu.Lock()
	closed := h.closed
	h.mu.Unlock()
	if closed {
		return false, "Hold-to-talk is closed."
	}
	if !h.authorized() {
		return false, keyboardPermissionHelp
	}
	return true, "True hold-to-talk uses a macOS keyboard event tap with key press and release edges."
}
func (h *HoldHook) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	return h.stopLocked(true)
}
func HoldAvailability() (bool, string) {
	if !KeyboardAuthorization() {
		return false, keyboardPermissionHelp
	}
	return true, "macOS Input Monitoring is authorized. Hold-to-talk starts a native key press/release event tap when configured."
}
