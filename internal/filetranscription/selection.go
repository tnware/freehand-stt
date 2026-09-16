package filetranscription

import (
	"errors"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/diagnostics"
)

// ChooseAudioFile opens the native file picker in Go and retains the selected
// path as a backend-only capability. The renderer receives only bounded file
// metadata and cannot grant an arbitrary filesystem path to itself.
func (s *Service) ChooseAudioFile() (result FileTranscriptionStatus, err error) {
	started := time.Now()
	outcome := "selected"
	s.log().Info("audio file selection started")
	defer func() {
		if err != nil {
			outcome = "failed"
		}
		attrs := []any{"duration_ms", time.Since(started).Milliseconds(), "outcome", outcome}
		switch {
		case err != nil:
			attrs = append(attrs, "error_kind", diagnostics.ErrorKind(err))
			s.log().Warn("audio file selection failed", attrs...)
		case outcome == "cancelled":
			s.log().Info("audio file selection cancelled", attrs...)
		default:
			s.log().Info("audio file selection completed", attrs...)
		}
	}()
	if s.closed.Load() {
		return FileTranscriptionStatus{}, errors.New("application is shutting down")
	}
	s.fileMu.Lock()
	if s.fileChoosing {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("the audio file picker is already open")
	}
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("cancel the active file transcription before choosing another file")
	}
	s.fileChoosing = true
	picker := s.pickAudioFile
	s.fileMu.Unlock()
	defer func() {
		s.fileMu.Lock()
		s.fileChoosing = false
		s.fileMu.Unlock()
	}()

	if picker == nil {
		return FileTranscriptionStatus{}, errors.New("the native audio file picker is unavailable")
	}
	path, err := picker()
	if err != nil {
		return FileTranscriptionStatus{}, errors.New("the native audio file picker could not be opened")
	}
	if strings.TrimSpace(path) == "" {
		outcome = "cancelled"
		return s.CurrentFileTranscription(), nil
	}
	return s.selectAudioFile(path)
}

// selectAudioFile converts a native-picker result into an app-owned selection.
// It is deliberately unexported so Wails cannot bind a renderer-controlled
// path argument. Tests use it to exercise the post-picker validation boundary.
func (s *Service) selectAudioFile(path string) (FileTranscriptionStatus, error) {
	selection, err := inspectAudioFile(path)
	if err != nil {
		return FileTranscriptionStatus{}, err
	}

	cfg := s.current()
	s.fileMu.Lock()
	if s.closed.Load() {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("application is shutting down")
	}
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("cancel the active file transcription before choosing another file")
	}
	s.fileGeneration++
	s.fileSelection = selection
	s.fileStatus = FileTranscriptionStatus{
		Generation: s.fileGeneration,
		Phase:      FileTranscriptionSelected,
		FileName:   selection.name,
		FileSize:   selection.size,
		CanStart:   true,
	}
	s.resetFileTranscriptLocked("")
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	return status, nil
}

// ClearAudioFile releases the current backend-owned audio-file selection.
func (s *Service) ClearAudioFile() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	cfg := s.current()
	s.fileMu.Lock()
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return errors.New("cancel the active file transcription before clearing it")
	}
	if s.fileChoosing {
		s.fileMu.Unlock()
		return errors.New("finish choosing an audio file before clearing it")
	}
	s.fileGeneration++
	s.fileSelection = nil
	s.fileStatus = FileTranscriptionStatus{Generation: s.fileGeneration, Phase: FileTranscriptionEmpty}
	s.resetFileTranscriptLocked("")
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	return nil
}
