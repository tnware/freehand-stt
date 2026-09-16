package tts

import (
	"context"
	"errors"
	"time"
)

func (s *Service) monitor(ctx context.Context, generation uint64) {
	started := time.Now()
	// The monitor owns playback diagnostics, including restarted sessions and
	// stale generations that can no longer publish a status transition.
	s.logger.Info("speech playback started", "generation", generation)
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("speech playback cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
			return
		case <-ticker.C:
			s.control.Lock()
			if s.closed.Load() || ctx.Err() != nil || !s.isCurrent(generation) {
				s.control.Unlock()
				s.logger.Info("speech playback cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
				return
			}
			position, duration, done := s.player.Position()
			if done {
				s.finishLocked(generation, Completed, "Playback complete", "")
				s.control.Unlock()
				s.logger.Info("speech playback completed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "played")
				return
			}
			s.mu.Lock()
			s.status.PositionMilliseconds = position
			s.status.DurationMilliseconds = duration
			status := s.status
			s.mu.Unlock()
			s.publish(status)
			s.control.Unlock()
		}
	}
}

func (s *Service) Pause() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	generation := s.status.Generation
	valid := s.status.Phase == Playing
	s.mu.Unlock()
	if !valid {
		return errors.New("speech playback is not playing")
	}
	if err := s.player.Pause(); err != nil {
		return err
	}
	position, duration, _ := s.player.Position()
	s.mu.Lock()
	if s.status.Generation == generation {
		s.status.Phase = Paused
		s.status.PositionMilliseconds = position
		s.status.DurationMilliseconds = duration
		s.status.Message = "Playback paused"
		s.status.CanPause = false
		s.status.CanResume = true
		status := s.status
		s.mu.Unlock()
		s.publish(status)
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) Resume() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	generation := s.status.Generation
	valid := s.status.Phase == Paused
	s.mu.Unlock()
	if !valid {
		return errors.New("speech playback is not paused")
	}
	if err := s.player.Play(); err != nil {
		return err
	}
	s.mu.Lock()
	if s.status.Generation == generation {
		s.status.Phase = Playing
		s.status.Message = "Playing transcript"
		s.status.CanPause = true
		s.status.CanResume = false
		status := s.status
		s.mu.Unlock()
		s.publish(status)
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) Restart() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	valid := s.status.CanRestart
	if !valid {
		s.mu.Unlock()
		return errors.New("speech playback cannot be restarted")
	}
	if s.operation != nil {
		s.operation()
	}
	ctx, cancel := s.operationContext()
	s.generation++
	generation := s.generation
	s.status.Generation = generation
	s.operation = cancel
	s.mu.Unlock()
	if err := s.player.Rewind(); err != nil {
		cancel()
		return err
	}
	// Native Stop inside Rewind may have outlived shutdown's wait budget.
	// Play is a separate step and must not be admitted after cancellation.
	if s.closed.Load() {
		cancel()
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		cancel()
		return err
	}
	if err := s.player.Play(); err != nil {
		cancel()
		return err
	}

	s.mu.Lock()
	if s.status.Generation == generation {
		s.status.Phase = Playing
		s.status.PositionMilliseconds = 0
		s.status.Message = "Playing transcript"
		s.status.CanPause = true
		s.status.CanResume = false
		s.status.CanStop = true
		status := s.status
		s.mu.Unlock()
		s.publish(status)
		s.workers.Add(1)
		go func() { defer s.workers.Done(); s.monitor(ctx, generation) }()
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) Stop() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	if s.operation != nil {
		s.operation()
		s.operation = nil
	}
	active := s.status.Phase == Generating || s.status.Phase == Playing || s.status.Phase == Paused || s.status.Phase == Completed
	s.generation++
	generation := s.generation
	s.status.Generation = generation
	s.mu.Unlock()
	if err := s.player.Stop(); err != nil {
		return err
	}
	_ = s.player.Unload()
	if active {
		s.finishLocked(generation, Cancelled, "Speech playback stopped", "")
	}
	return nil
}

// ClearAudio explicitly releases the retained PCM session and returns the
// player to idle. Completed audio otherwise remains available for Restart.
func (s *Service) ClearAudio() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	if s.operation != nil {
		s.operation()
		s.operation = nil
	}
	canClear := s.status.CanClear
	s.mu.Unlock()
	if !canClear {
		return errors.New("there is no generated speech to clear")
	}
	if err := s.player.Stop(); err != nil {
		return err
	}
	if err := s.player.Unload(); err != nil {
		return err
	}
	s.mu.Lock()
	s.generation++
	generation := s.generation
	s.status = Status{Generation: generation, Phase: Idle}
	status := s.status
	s.mu.Unlock()
	s.publish(status)
	s.logger.Info("speech audio cleared", "generation", generation, "outcome", "released")
	return nil
}
