//go:build darwin && cgo

package platform

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/insertion"
)

// Input values share a single native owner. Targets are comparable values, not
// AX pointers. Only the latest capture is retained, and Close is idempotent.
type Input struct{ state *darwinInputState }
type darwinInputBackend interface {
	capture() (uint32, uint64, error)
	valid(uint32, uint64) bool
	rejection(insertion.Stage) error
	modifiersReleased() bool
	send([]uint16) bool
	resetTarget()
	copyText(context.Context, <-chan struct{}, string) error
	close()
}
type darwinInputState struct {
	mu      darwinInputGate
	backend darwinInputBackend
	target  insertion.Target
	closed  bool
	stop    chan struct{}
	once    sync.Once
}

// A channel gate admits cancellable clipboard callers without waiter goroutines.
type darwinInputGate chan struct{}

func (m darwinInputGate) Lock()   { m <- struct{}{} }
func (m darwinInputGate) Unlock() { <-m }
func (m darwinInputGate) lockContext(ctx context.Context, stop <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stop:
		return insertion.ErrCopyRequired
	case m <- struct{}{}:
		return nil
	}
}

var darwinInputGeneration atomic.Uint64

func newDarwinInput(b darwinInputBackend) Input {
	return Input{state: &darwinInputState{backend: b, mu: make(darwinInputGate, 1), stop: make(chan struct{})}}
}
func (i Input) CaptureTarget() (insertion.Target, error) {
	if i.state == nil {
		return insertion.Target{}, insertion.ErrCopyRequired
	}
	s := i.state
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetTarget()
	if s.closed {
		return s.target, insertion.ErrCopyRequired
	}
	pid, started, err := s.backend.capture()
	if err != nil || pid == 0 || pid == uint32(os.Getpid()) || started == 0 {
		if err == nil {
			err = insertion.NewRejection(insertion.Capture, "target_invalid")
		}
		s.resetTarget()
		return s.target, insertion.CopyRequired(err)
	}
	token := darwinInputGeneration.Add(1)
	if token == 0 {
		s.resetTarget()
		return s.target, insertion.ErrCopyRequired
	}
	s.target = insertion.Target{DarwinToken: token, ProcessID: pid, ProcessCreationTime: started}
	return s.target, nil
}
func (s *darwinInputState) resetTarget() {
	s.target = insertion.Target{}
	s.backend.resetTarget()
}
func (s *darwinInputState) validate(target insertion.Target) error {
	if s.closed || !target.Valid() || target.DarwinToken == 0 || target != s.target {
		return insertion.NewRejection(insertion.Validate, "target_invalid")
	}
	if !s.backend.valid(target.ProcessID, target.ProcessCreationTime) {
		// Read the bounded reason before reset releases native diagnostic state.
		err := insertion.CopyRequired(s.backend.rejection(insertion.Validate))
		s.resetTarget()
		return err
	}
	return nil
}
func (i Input) Foreground() (insertion.Target, error) {
	if i.state == nil {
		return insertion.Target{}, insertion.ErrCopyRequired
	}
	s := i.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validate(s.target); err != nil {
		return insertion.Target{}, err
	}
	return s.target, nil
}

// Quartz Unicode payloads are deliberately small. A surrogate pair is always
// posted together. A dispatched target is consumed: Quartz cannot acknowledge
// application delivery, so the caller must never replay an ambiguous attempt.
const darwinUnicodeChunkUnits = 20
const darwinInputMaxBytes = 1 << 20

func (i Input) InsertUnicode(ctx context.Context, target insertion.Target, text string) error {
	if i.state == nil {
		return insertion.ErrCopyRequired
	}
	s := i.state
	s.mu.Lock()
	defer s.mu.Unlock()
	// Only this generation may release its native target. Stale attempts must
	// not revoke a later capture. Validation itself can release on rejection.
	defer func() {
		if target == s.target && target.DarwinToken != 0 {
			s.resetTarget()
		}
	}()
	if err := s.validate(target); err != nil {
		return err
	}
	if len(text) > darwinInputMaxBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return insertion.ErrCopyRequired
	}
	if err := s.interrupted(ctx); err != nil {
		return err
	}
	units := utf16.Encode([]rune(text))
	for offset := 0; offset < len(units); {
		if err := s.interrupted(ctx); err != nil {
			return err
		}
		if err := s.waitForModifiers(ctx, target); err != nil {
			return err
		}
		if err := s.validate(target); err != nil {
			return err
		}
		if err := s.interrupted(ctx); err != nil {
			return err
		}
		end := min(offset+darwinUnicodeChunkUnits, len(units))
		if end < len(units) && units[end-1] >= 0xD800 && units[end-1] <= 0xDBFF {
			end--
		}
		if !s.backend.send(units[offset:end]) {
			return insertion.CopyRequired(s.backend.rejection(insertion.Send))
		}
		offset = end
	}
	return nil
}

// Never synthesize modifier-up events: they would corrupt the user's keyboard
// state. Wait briefly for physical release, checking identity throughout.
func (s *darwinInputState) waitForModifiers(ctx context.Context, target insertion.Target) error {
	deadline := time.NewTimer(250 * time.Millisecond)
	defer deadline.Stop()
	for {
		if err := s.interrupted(ctx); err != nil {
			return err
		}
		if err := s.validate(target); err != nil {
			return err
		}
		if s.backend.modifiersReleased() {
			return nil
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-s.stop:
			timer.Stop()
			return insertion.ErrCopyRequired
		case <-deadline.C:
			timer.Stop()
			return insertion.NewRejection(insertion.Send, "modifiers_held")
		case <-timer.C:
		}
	}
}
func (s *darwinInputState) interrupted(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-s.stop:
		return insertion.ErrCopyRequired
	default:
		return nil
	}
}
func (i Input) Copy(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	if i.state == nil || len(text) > darwinInputMaxBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return insertion.ErrCopyNotPerformed
	}
	s := i.state
	if err := s.mu.lockContext(ctx, s.stop); err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, ctx.Err())
	}
	defer s.mu.Unlock()
	if err := s.interrupted(ctx); err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, ctx.Err())
	}
	if s.closed {
		return insertion.ErrCopyNotPerformed
	}
	return s.backend.copyText(ctx, s.stop, text)
}
func (i Input) Close() error {
	if i.state == nil {
		return nil
	}
	s := i.state
	s.once.Do(func() { close(s.stop) })
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		s.target = insertion.Target{}
		s.backend.close()
	}
	return nil
}
