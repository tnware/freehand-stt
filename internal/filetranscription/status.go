package filetranscription

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// FileTranscriptionPhase identifies the current stored-audio operation stage.
type FileTranscriptionPhase string

const (
	FileTranscriptionEmpty      FileTranscriptionPhase = "empty"
	FileTranscriptionSelected   FileTranscriptionPhase = "selected"
	FileTranscriptionUploading  FileTranscriptionPhase = "uploading"
	FileTranscriptionProcessing FileTranscriptionPhase = "processing"
	FileTranscriptionStreaming  FileTranscriptionPhase = "streaming"
	FileTranscriptionCompleted  FileTranscriptionPhase = "completed"
	FileTranscriptionFailed     FileTranscriptionPhase = "failed"
	FileTranscriptionCancelling FileTranscriptionPhase = "cancelling"
)

// FileTranscriptionStatus is the bounded renderer snapshot for a Go-owned
// native file selection. It never contains the full path or audio bytes.
type FileTranscriptionStatus struct {
	Generation                  uint64                 `json:"generation"`
	Phase                       FileTranscriptionPhase `json:"phase"`
	FileName                    string                 `json:"fileName,omitempty"`
	FileSize                    int64                  `json:"fileSize,omitempty"`
	BytesUploaded               int64                  `json:"bytesUploaded,omitempty"`
	Streaming                   bool                   `json:"streaming"`
	Buffered                    bool                   `json:"buffered"`
	StreamingProfileUnavailable bool                   `json:"streamingProfileUnavailable"`
	StreamingUnavailable        bool                   `json:"streamingUnavailable"`
	StreamingNotice             string                 `json:"streamingNotice,omitempty"`
	Transcript                  string                 `json:"transcript,omitempty"`
	TranscriptRevision          uint64                 `json:"transcriptRevision"`
	Message                     string                 `json:"message,omitempty"`
	CanStart                    bool                   `json:"canStart"`
	CanCancel                   bool                   `json:"canCancel"`
	CanCopy                     bool                   `json:"canCopy"`
}

// FileTranscriptionDelta is the incremental renderer contract for streamed
// transcript text. The complete status remains available as a binding snapshot
// and is emitted once more when the operation reaches a terminal phase.
type FileTranscriptionDelta struct {
	Generation uint64 `json:"generation"`
	Revision   uint64 `json:"revision"`
	Text       string `json:"text"`
}

func init() {
	application.RegisterEvent[FileTranscriptionDelta](DeltaEvent)
}

// snapshotFileStatusLocked materializes the accumulated transcript only at a
// real status boundary or an explicit snapshot request. Streaming deltas never
// copy the growing transcript into FileTranscriptionStatus.
func (s *Service) snapshotFileStatusLocked() FileTranscriptionStatus {
	status := s.fileStatus
	status.Transcript = s.fileTranscript.String()
	status.TranscriptRevision = s.fileTranscriptRevision
	return status
}

func (s *Service) resetFileTranscriptLocked(text string) {
	s.fileTranscript.Reset()
	_, _ = s.fileTranscript.WriteString(text)
	s.fileTranscriptRevision = 0
	s.fileTranscriptLimitHit = false
	s.fileStatus.Transcript = text
	s.fileStatus.TranscriptRevision = 0
}

func (s *Service) replaceFileTranscriptLocked(text string) {
	s.fileTranscript.Reset()
	_, _ = s.fileTranscript.WriteString(text)
	s.fileTranscriptRevision++
	s.fileStatus.Transcript = text
	s.fileStatus.TranscriptRevision = s.fileTranscriptRevision
}

// CurrentFileTranscription returns the latest stored-audio operation snapshot.
func (s *Service) CurrentFileTranscription() FileTranscriptionStatus {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	status := s.snapshotFileStatusLocked()
	if status.Phase == "" {
		status.Phase = FileTranscriptionEmpty
	}
	return status
}

// PlaybackTranscript is an ordinary Go collaboration boundary. It keeps the
// retained file transcript out of renderer arguments while allowing the TTS
// feature to read the completed result when in-memory history is disabled.
func PlaybackTranscript(service *Service) (string, error) {
	if service == nil {
		return "", errors.New("audio file transcription is unavailable")
	}
	service.fileMu.Lock()
	defer service.fileMu.Unlock()
	if service.fileStatus.Phase != FileTranscriptionCompleted || service.fileTranscript.Len() == 0 {
		return "", errors.New("no completed audio file transcript is available")
	}
	return service.fileTranscript.String(), nil
}

// CopyFileTranscript copies the completed stored-audio transcript.
func (s *Service) CopyFileTranscript() error {
	s.fileMu.Lock()
	text := s.fileTranscript.String()
	canCopy := s.fileStatus.CanCopy
	s.fileMu.Unlock()
	if !canCopy || text == "" {
		return errors.New("no completed file transcript is available")
	}
	return s.input.Copy(context.Background(), text)
}

func (s *Service) fileTranscriptionActive() bool {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	return filePhaseActive(s.fileStatus.Phase)
}

func filePhaseActive(phase FileTranscriptionPhase) bool {
	switch phase {
	case FileTranscriptionUploading, FileTranscriptionProcessing, FileTranscriptionStreaming, FileTranscriptionCancelling:
		return true
	default:
		return false
	}
}

func (s *Service) publishFileStatus(changed func(FileTranscriptionStatus), status FileTranscriptionStatus) {
	if changed != nil && !s.closed.Load() {
		changed(status)
	}
}

func (s *Service) publishFileDelta(changed func(FileTranscriptionDelta), delta FileTranscriptionDelta) {
	if changed != nil && !s.closed.Load() {
		changed(delta)
	}
}
