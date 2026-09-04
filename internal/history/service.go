package history

// Service is the renderer boundary for transcript history. The Store owns the
// data and is also injected directly into producers such as dictation and file
// transcription; those packages do not call back through this Wails API.
type Service struct{ store *Store }

func NewService(store *Store) *Service { return &Service{store: store} }

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
	return s.store.Delete(id)
}

// ClearHistory deletes all transcripts from bounded in-memory history.
func (s *Service) ClearHistory() { s.store.Clear() }
