package tts

import (
	"github.com/tnware/freehand-stt/internal/history"
)

type Phase string
type Source string

const (
	Idle       Phase = "idle"
	Generating Phase = "generating"
	Playing    Phase = "playing"
	Paused     Phase = "paused"
	Completed  Phase = "completed"
	Cancelled  Phase = "cancelled"
	Failed     Phase = "failed"
)

const (
	SourceHistory Source = "history"
	SourceFile    Source = "audio-file"
	SourceVoice   Source = "voice"
	SourcePreview Source = "preview"
	SourceCompose Source = "compose"
)

type Status struct {
	Generation           uint64                     `json:"generation"`
	Phase                Phase                      `json:"phase"`
	Source               Source                     `json:"source,omitempty"`
	HistoryID            uint64                     `json:"historyID,omitempty"`
	HistoryVersion       history.HistoryTextVersion `json:"historyVersion,omitempty"`
	PositionMilliseconds int64                      `json:"positionMilliseconds"`
	DurationMilliseconds int64                      `json:"durationMilliseconds"`
	Message              string                     `json:"message,omitempty"`
	ErrorKind            string                     `json:"errorKind,omitempty"`
	CanPause             bool                       `json:"canPause"`
	CanResume            bool                       `json:"canResume"`
	CanSeek              bool                       `json:"canSeek"`
	CanRestart           bool                       `json:"canRestart"`
	CanStop              bool                       `json:"canStop"`
	CanSave              bool                       `json:"canSave"`
	CanClear             bool                       `json:"canClear"`
}

func (s *Service) CurrentStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) source(generation uint64) Source {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Generation == generation {
		return s.status.Source
	}
	return ""
}
func (s *Service) historyID(generation uint64) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Generation == generation {
		return s.status.HistoryID
	}
	return 0
}
func (s *Service) historyVersion(generation uint64) history.HistoryTextVersion {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Generation == generation {
		return s.status.HistoryVersion
	}
	return ""
}

func (s *Service) update(generation uint64, status Status) {
	s.mu.Lock()
	if s.closed.Load() || s.status.Generation != generation {
		s.mu.Unlock()
		return
	}
	s.status = status
	s.mu.Unlock()
	s.publish(status)
}

func (s *Service) isCurrent(generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status.Generation == generation
}

func (s *Service) finish(generation uint64, phase Phase, message, errorKind string) {
	s.control.Lock()
	defer s.control.Unlock()
	s.finishLocked(generation, phase, message, errorKind)
}

// Caller owns control; stale work must never touch the shared player.
func (s *Service) finishLocked(generation uint64, phase Phase, message, errorKind string) {
	if s.closed.Load() || !s.isCurrent(generation) {
		return
	}

	if phase == Completed {
		_ = s.player.Pause()
	}
	position, duration, _ := s.player.Position()
	s.mu.Lock()
	if s.closed.Load() || s.status.Generation != generation {
		s.mu.Unlock()
		return
	}
	terminal := s.status.Phase == Completed || s.status.Phase == Cancelled || s.status.Phase == Failed
	if terminal && phase != Cancelled {
		s.mu.Unlock()
		return
	}
	s.operation = nil
	s.status.Phase, s.status.Message, s.status.ErrorKind = phase, message, errorKind
	s.status.PositionMilliseconds, s.status.DurationMilliseconds = position, duration
	s.status.CanPause, s.status.CanResume, s.status.CanStop = false, false, false
	s.status.CanSeek = phase == Completed && duration > 0
	s.status.CanRestart = phase == Completed
	s.status.CanSave = phase == Completed
	s.status.CanClear = phase == Completed
	status := s.status
	s.mu.Unlock()
	s.publish(status)
}

func (s *Service) publish(status Status) {
	if s.changed != nil && !s.closed.Load() {
		s.changed(status)
	}
}
