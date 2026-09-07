package history

import (
	"errors"
	"sync"
)

// DetailsChangedEvent carries no transcript data. Renderers fetch the current
// selection from the history owner after opening, closing, or removing entries.
const DetailsChangedEvent = "history:details-changed"

type DetailsWindow struct {
	Open    func()
	Hide    func()
	Changed func()
}

// Service is the renderer boundary for transcript history. The Store owns the
// data and is also injected directly into producers such as dictation and file
// transcription; those packages do not call back through this Wails API.
type Service struct {
	store         *Store
	detailsMu     sync.Mutex
	detailsID     uint64
	detailsWindow DetailsWindow
}

func NewService(store *Store, window DetailsWindow) *Service {
	return &Service{store: store, detailsWindow: window}
}

func (s *Service) ServiceShutdown() error {
	s.store.Close()
	return nil
}

func (s *Service) TranscriptHistory() []HistoryEntry { return s.store.Entries() }

// CopyHistoryEntry copies the delivered version of one history entry.
func (s *Service) CopyHistoryEntry(id uint64) error {
	return s.store.CopyEntry(id)
}

// CopyHistoryEntryVersion copies a specific raw or processed history version.
func (s *Service) CopyHistoryEntryVersion(id uint64, version HistoryTextVersion) error {
	return s.store.CopyEntryVersion(id, version)
}

// DeleteHistoryEntry deletes one transcript from bounded in-memory history.
func (s *Service) DeleteHistoryEntry(id uint64) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	s.detailsChanged()
	return nil
}

// ClearHistory deletes all transcripts from bounded in-memory history.
func (s *Service) ClearHistory() {
	s.store.Clear()
	s.detailsChanged()
}

// OpenDetails accepts only a retained, completed run. The window selection owns
// an ID, never a second retained copy of transcript content or metadata.
func (s *Service) OpenDetails(id uint64) error {
	s.detailsMu.Lock()
	defer s.detailsMu.Unlock()
	if s.store.completedEntry(id) == nil {
		return errors.New("transcription details are no longer available")
	}
	if s.detailsWindow.Open == nil {
		return errors.New("transcription details window is unavailable")
	}
	s.detailsID = id
	s.detailsWindow.Open()
	s.detailsChanged()
	return nil
}

// CurrentDetails also recovers the selection after a renderer reload. Deleted,
// disabled, evicted, and shutdown history is never recovered from a stale copy.
func (s *Service) CurrentDetails() *HistoryEntry {
	s.detailsMu.Lock()
	defer s.detailsMu.Unlock()
	return s.store.completedEntry(s.detailsID)
}

func (s *Service) CloseDetails() {
	s.detailsMu.Lock()
	defer s.detailsMu.Unlock()
	s.detailsID = 0
	if s.detailsWindow.Hide != nil {
		s.detailsWindow.Hide()
	}
	s.detailsChanged()
}

func (s *Service) detailsChanged() {
	if s.detailsWindow.Changed != nil {
		s.detailsWindow.Changed()
	}
}
