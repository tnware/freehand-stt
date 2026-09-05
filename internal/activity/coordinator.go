// Package activity owns cross-feature start admission, not feature lifetimes.
package activity

import (
	"errors"
	"sync"
)

// Sources read the feature owners' authoritative state. They are fixed at
// composition time, called only after construction, and must not reenter
// admission. StopPlayback must release playback before returning success.
type Sources struct {
	DictationActive func() bool
	FileActive      func() bool
	StopPlayback    func() error
}

// Coordinator serializes check -> preemption -> start publication. It never
// caches feature phases or retains a reservation for a running operation.
// Callers release admission on every return, including failed starts, and
// continue to own cancellation, generation fencing and shutdown of their work.
// Acquire admission before feature control locks, never while holding one.
type Coordinator struct {
	sources   Sources
	gate      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func New(sources Sources) *Coordinator {
	return &Coordinator{sources: sources, gate: make(chan struct{}, 1), done: make(chan struct{})}
}

var ErrClosed = errors.New("application is shutting down")

// Close rejects and wakes waiting starts without joining an admitted start.
// Feature shutdown still cancels/fences its own in-flight resources.
func (c *Coordinator) Close() {
	if c != nil {
		c.closeOnce.Do(func() { close(c.done) })
	}
}

func (c *Coordinator) enter() (func(), error) {
	select {
	case <-c.done:
		return nil, ErrClosed
	case c.gate <- struct{}{}:
	}
	var once sync.Once
	release := func() { once.Do(func() { <-c.gate }) }
	select {
	case <-c.done:
		release()
		return nil, ErrClosed
	default:
		return release, nil
	}
}

func active(read func() bool) bool { return read != nil && read() }

// BeginRecording excludes files and synchronously preempts speech. A failed
// preemption cannot authorize microphone capture. Release after start returns.
func (c *Coordinator) BeginRecording() (func(), error) {
	release, err := c.enter()
	if err != nil {
		return nil, err
	}
	if active(c.sources.FileActive) {
		release()
		return nil, errors.New("an audio file is being transcribed")
	}
	if c.sources.StopPlayback != nil {
		if err := c.sources.StopPlayback(); err != nil {
			release()
			return nil, errors.New("speech playback could not be stopped before recording")
		}
	}
	// Preemption may have waited for the player control lock while shutdown
	// closed admission. Do not grant capture after that wait.
	select {
	case <-c.done:
		release()
		return nil, ErrClosed
	default:
		return release, nil
	}
}

func (c *Coordinator) BeginFileTranscription() (func(), error) {
	release, err := c.enter()
	if err != nil {
		return nil, err
	}
	if active(c.sources.DictationActive) {
		release()
		return nil, errors.New("finish or cancel voice dictation before transcribing a file")
	}
	// File transcription does not preempt already-running playback.
	return release, nil
}

func (c *Coordinator) BeginPlayback() (func(), error) {
	release, err := c.enter()
	if err != nil {
		return nil, err
	}
	if active(c.sources.DictationActive) || active(c.sources.FileActive) {
		release()
		return nil, errors.New("finish the active transcription before starting speech playback")
	}
	return release, nil
}

// CheckShortcutCapture shares the activity vocabulary, not the native capture
// lifetime. The input/shortcut owners still serialize capture and suspend and
// restore native hotkeys; no admission token is held during the user prompt.
func (c *Coordinator) CheckShortcutCapture() error {
	if c == nil {
		return nil
	}
	release, err := c.enter()
	if err != nil {
		return err
	}
	defer release()
	if active(c.sources.FileActive) {
		return errors.New("Finish the active audio-file transcription before recording a shortcut.")
	}
	if active(c.sources.DictationActive) {
		return errors.New("Finish the active dictation before recording a shortcut.")
	}
	return nil
}
