//go:build darwin

package platform

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/hotkey"
)

var shortcutCaptureActive atomic.Bool
var errShortcutCaptureBusy = errors.New("shortcut capture is already in progress or its preview callback has not returned")

// ShortcutCapturer owns a temporary suppressing event tap. Preview callbacks
// never run on the native input thread or the capture/cancellation loop.
// A stuck callback prevents reuse rather than accumulating callback workers.
type ShortcutCapturer struct {
	mu          sync.Mutex
	closed      bool
	cancel      context.CancelFunc
	done        chan struct{}
	previewDone chan struct{}
	open        func(bool) (keyboardSource, error)
}

func (c *ShortcutCapturer) Capture(parent context.Context, policy hotkey.ShortcutPolicy, changed func(hotkey.Chord)) (chord hotkey.Chord, canceled bool, err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return hotkey.Chord{}, false, errors.New("shortcut capture is closed")
	}
	if c.previewDone != nil {
		select {
		case <-c.previewDone:
			c.previewDone = nil
		default:
			c.mu.Unlock()
			return hotkey.Chord{}, false, errShortcutCaptureBusy
		}
	}
	if c.done != nil || !shortcutCaptureActive.CompareAndSwap(false, true) {
		c.mu.Unlock()
		return hotkey.Chord{}, false, errShortcutCaptureBusy
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	c.cancel, c.done = cancel, done
	open := c.open
	if open == nil {
		open = openKeyboard
	}
	progress := make(chan hotkey.Chord, 1)
	if changed != nil {
		workerDone := make(chan struct{})
		c.previewDone = workerDone
		go func() {
			defer close(workerDone)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				select {
				case <-ctx.Done():
					return
				case chord := <-progress:
					if ctx.Err() != nil {
						return
					}
					changed(chord)
				}
			}
		}()
	}
	c.mu.Unlock()
	defer func() {
		cancel()
		c.mu.Lock()
		c.cancel = nil
		c.done = nil
		close(done)
		shortcutCaptureActive.Store(false)
		c.mu.Unlock()
	}()
	if ctx.Err() != nil {
		return captureContextResult(parent, policy.Action)
	}
	source, err := open(true)
	if err != nil {
		return hotkey.Chord{}, false, err
	}
	defer func() { err = errors.Join(err, source.close()) }()
	reducer := hotkey.CaptureReducer{Policy: policy}
	var adapter keyboardAdapter
	var preview hotkey.Chord
	ticker := time.NewTicker(keyboardPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return captureContextResult(parent, policy.Action)
		case <-ticker.C:
			for i := 0; i < 256; i++ {
				if ctx.Err() != nil {
					return captureContextResult(parent, policy.Action)
				}
				event, ok := source.read()
				if !ok {
					break
				}
				if event.lost {
					return hotkey.Chord{}, false, errors.New("shortcut capture lost keyboard input; release the keys and try again after unlocking the session or exiting Secure Input")
				}
				key, down, ok := adapter.event(event)
				if !ok {
					continue
				}
				result := reducer.Event(key, down)
				switch result.State {
				case hotkey.CaptureComplete:
					return result.Chord, false, nil
				case hotkey.CaptureCanceled:
					return hotkey.Chord{}, true, nil
				case hotkey.CaptureRejected:
					return hotkey.Chord{}, false, result.Err
				}
				if next := reducer.Preview(); changed != nil && next != preview {
					preview = next
					select {
					case progress <- next:
					default:
					}
				}
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

// Close releases the capture, but cannot wait for arbitrary UI callbacks.
// A blocked preview worker remains the sole worker and can never be replaced.
func (c *ShortcutCapturer) Close() error {
	c.mu.Lock()
	c.closed = true
	cancel, done := c.cancel, c.done
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * keyboardCloseBound):
			return errors.New("shortcut capture shutdown timed out")
		}
	}
	return nil
}
