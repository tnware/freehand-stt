package history

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tnware/freehand-stt/internal/insertion"
)

// Store owns the bounded, process-local transcript history. It is ordinary Go
// state, not a Wails service; the renderer API and transcript producers share
// it through constructor injection.
type Store struct {
	mu      sync.Mutex
	buffer  historyBuffer
	enabled bool
	nextID  uint64
	copier  insertion.Platform
	closed  bool
}

func NewStore(enabled bool, copier insertion.Platform) *Store {
	return &Store{enabled: enabled, copier: copier}
}

func (s *Store) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = enabled
	if !enabled {
		s.buffer.clear()
	}
}

func (s *Store) Entries() []HistoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buffer.newestFirst()
}

// Begin retains a raw transcript before optional post-processing starts. A
// later processing failure can therefore never erase a successful transcript.
func (s *Store) Begin(text string, outcome HistoryOutcome, processing bool, completedAt time.Time, details HistoryRunDetails) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.enabled || text == "" {
		return 0
	}
	s.nextID++
	status := HistoryProcessingNotRequested
	if processing {
		status = HistoryProcessingPending
	}
	if !s.buffer.add(HistoryEntry{ID: s.nextID, Text: text, RawText: text, CompletedAt: completedAt, Outcome: outcome, ProcessingStatus: status, Details: details}) {
		return 0
	}
	return s.nextID
}

func (s *Store) FinalizeProcessing(id uint64, raw, processed string, status HistoryProcessingStatus, message string, details HistoryRunDetails) {
	if id == 0 {
		return
	}
	if len(message) > 256 {
		message = message[:256]
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer.update(id, func(entry *HistoryEntry) {
		entry.RawText = raw
		entry.ProcessedText = processed
		entry.ProcessingStatus = status
		entry.ProcessingMessage = message
		entry.CompletedAt = time.Now().UTC()
		entry.Details = details
		if status == HistoryProcessingCompleted && processed != "" {
			entry.Text = processed
		} else {
			entry.Text = raw
		}
	})
}

func (s *Store) UpdateDetails(id uint64, details HistoryRunDetails) {
	if id == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer.update(id, func(entry *HistoryEntry) { entry.Details = details })
}

func (s *Store) Finalize(id uint64, outcome HistoryOutcome, details HistoryRunDetails, completedAt time.Time, durationLimitReached bool) {
	if id == 0 {
		return
	}
	FinalizeDetails(&details, completedAt, durationLimitReached)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer.update(id, func(entry *HistoryEntry) {
		entry.Outcome = outcome
		entry.CompletedAt = completedAt
		entry.Details = details
		if entry.ProcessingStatus == HistoryProcessingCompleted && entry.ProcessedText != "" {
			entry.Text = entry.ProcessedText
		} else {
			entry.Text = entry.RawText
		}
	})
}

func (s *Store) Delete(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("application is shutting down")
	}
	if !s.buffer.remove(id) {
		return errors.New("history entry was not found")
	}
	return nil
}

func (s *Store) CopyEntry(id uint64) error { return s.CopyEntryVersion(id, HistoryTextFinal) }

// Text returns one backend-owned transcript version for another native
// capability. It is not exposed through Wails, so renderer code cannot turn
// speech playback into an arbitrary text-to-audio bridge.
func (s *Store) Text(id uint64, version HistoryTextVersion) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", errors.New("application is shutting down")
	}
	for _, entry := range s.buffer.entries {
		if entry.ID != id {
			continue
		}
		var text string
		switch version {
		case HistoryTextFinal:
			text = entry.Text
		case HistoryTextRaw:
			text = entry.RawText
		case HistoryTextProcessed:
			if entry.ProcessingStatus == HistoryProcessingCompleted {
				text = entry.ProcessedText
			}
		default:
			return "", errors.New("history transcript version is invalid")
		}
		if text == "" {
			return "", errors.New("history transcript version is unavailable")
		}
		return text, nil
	}
	return "", errors.New("history entry was not found")
}

func (s *Store) CopyEntryVersion(id uint64, version HistoryTextVersion) error {
	text, err := s.Text(id, version)
	if err != nil {
		return err
	}
	return s.copier.Copy(context.Background(), text)
}

func (s *Store) Clear() {
	s.mu.Lock()
	s.buffer.clear()
	s.mu.Unlock()
}

func (s *Store) Close() {
	s.mu.Lock()
	s.closed = true
	s.buffer.clear()
	s.mu.Unlock()
}
