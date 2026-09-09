package tts

import (
	"context"
	"errors"
)

// SeekRequest identifies the retained audio the user was manipulating. A delayed
// drag must never reposition a replacement session.
type SeekRequest struct {
	Generation           uint64 `json:"generation"`
	PositionMilliseconds int64  `json:"positionMilliseconds"`
}

// Seek repositions retained audio, preserving playing/paused intent. Seeking a
// completed session leaves it paused; seeking to the end leaves it completed.
func (s *Service) Seek(request SeekRequest) (Status, error) {
	// Like pause/resume, seeking controls retained audio. Recording preemption
	// serializes Stop through this lock and invalidates the session before capture.
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return Status{}, context.Canceled
	}
	s.mu.Lock()
	previous := s.status
	s.mu.Unlock()
	if previous.Generation != request.Generation || !previous.CanSeek ||
		(previous.Phase != Playing && previous.Phase != Paused && previous.Phase != Completed) {
		return Status{}, errors.New("speech session is no longer available for seeking")
	}
	if request.PositionMilliseconds < 0 || request.PositionMilliseconds > previous.DurationMilliseconds {
		return Status{}, errors.New("seek position is outside the generated audio")
	}
	if err := s.player.SeekTo(request.PositionMilliseconds); err != nil {
		return Status{}, errors.New("speech playback could not seek to this position")
	}
	// Native Stop may block. Never restart the device after shutdown cancelled it.
	if s.closed.Load() {
		return Status{}, context.Canceled
	}
	ctx, cancel := s.operationContext()
	if err := ctx.Err(); err != nil {
		cancel()
		return Status{}, err
	}
	s.mu.Lock()
	if s.operation != nil {
		s.operation()
	}
	generation := previous.Generation
	s.operation = cancel
	s.status.Generation = generation
	s.mu.Unlock()

	position, duration, _ := s.player.Position()
	atEnd := request.PositionMilliseconds == previous.DurationMilliseconds
	phase, message := Paused, "Playback paused"
	if atEnd {
		phase, message = Completed, "Playback complete"
	}
	var playErr error
	if previous.Phase == Playing && !atEnd {
		if s.closed.Load() || ctx.Err() != nil {
			cancel()
			return Status{}, context.Canceled
		}
		if playErr = s.player.Play(); playErr == nil {
			phase, message = Playing, "Playing speech"
		}
	}
	if s.closed.Load() || ctx.Err() != nil {
		cancel()
		return Status{}, context.Canceled
	}
	status := previous
	status.Generation = generation
	status.Phase, status.Message = phase, message
	status.PositionMilliseconds, status.DurationMilliseconds = position, duration
	status.CanPause, status.CanResume, status.CanStop = phase == Playing, phase == Paused, !atEnd
	s.update(generation, status)
	if atEnd {
		cancel()
		s.mu.Lock()
		s.operation = nil
		s.mu.Unlock()
	} else {
		s.workers.Add(1)
		go func() { defer s.workers.Done(); s.monitor(ctx, generation) }()
	}
	if playErr != nil {
		return status, errors.New("audio was repositioned but playback could not resume")
	}
	return status, nil
}
