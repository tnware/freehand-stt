package insertion

import (
	"context"
	"errors"
	"sync"
)

var ErrSessionUnsupported = errors.New("platform does not support segment delivery")

// SegmentPlatform retains the original capture only after a successful segment
// dispatch. Errors revoke it. ReleaseTarget must never revoke a newer capture.
// This capability is separate from the one-shot Platform insertion contract.
type SegmentPlatform interface {
	InsertSegment(context.Context, Target, string) error
	ReleaseTarget(Target)
}

// Session permits new authoritative segments, never replay of an earlier
// attempt. It owns no transcript, never captures focus, and never uses Copy.
// Call Close when recording/delivery ends, including before replacing a run.
type Session struct {
	mu       sync.Mutex
	platform Platform
	segments SegmentPlatform
	target   Target
	ctx      context.Context
	cancel   context.CancelFunc
	stop     func() bool
	closed   bool
	last     uint64
}

// BeginSession adopts the target already captured at recording start. The run
// context must be canceled when the owning recording generation becomes stale.
func (p Policy) BeginSession(ctx context.Context, target Target, mode Mode) (*Session, error) {
	segments, ok := p.Platform.(SegmentPlatform)
	if !ok {
		return nil, ErrSessionUnsupported
	}
	if mode != DirectInput || !target.Valid() || ctx.Err() != nil {
		segments.ReleaseTarget(target)
		return nil, ErrCopyRequired
	}
	runCtx, cancel := context.WithCancel(ctx)
	s := &Session{platform: p.Platform, segments: segments, target: target, ctx: runCtx, cancel: cancel}
	s.mu.Lock()
	s.stop = context.AfterFunc(runCtx, s.Close)
	s.mu.Unlock()
	return s, nil
}

// Deliver accepts a strictly increasing, nonzero segment ID exactly once. Any
// rejection, cancellation or uncertain/partial native attempt ends the session.
// A successful dispatch is not an acknowledgement by the receiving application.
func (s *Session) Deliver(ctx context.Context, id uint64, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrCopyRequired
	}
	if s.ctx.Err() != nil || ctx.Err() != nil || id == 0 || id <= s.last {
		s.closeLocked()
		return ErrCopyRequired
	}
	s.last = id // consume before the first side effect; no failed attempt is retryable
	sendCtx, cancel := context.WithCancel(s.ctx)
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	defer cancel()
	got, err := s.platform.Foreground()
	if err != nil || !got.Valid() || got != s.target {
		s.closeLocked()
		return CopyRequired(err)
	}
	if err = sendCtx.Err(); err == nil && s.ctx.Err() == nil {
		if text != "" {
			err = s.segments.InsertSegment(sendCtx, s.target, text)
		}
	} else if err == nil {
		err = s.ctx.Err()
	}
	if err != nil || s.ctx.Err() != nil || ctx.Err() != nil {
		s.closeLocked()
		return CopyRequired(err)
	}
	return nil
}

// Close cancels an in-flight send before waiting for its serialized cleanup.
func (s *Session) Close() {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked()
}
func (s *Session) closeLocked() {
	if s.closed {
		return
	}
	s.closed = true
	s.cancel()
	if s.stop != nil {
		s.stop()
	}
	s.segments.ReleaseTarget(s.target)
}
