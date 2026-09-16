package filetranscription

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
)

type fileStreamingCapabilityKey struct {
	Profile  compatibility.ID
	Endpoint string
	Model    string
}

const fileStreamingUnavailableNotice = "Streaming is unavailable for this endpoint and model. New requests will use completed transcripts."

func ApplySettings(service *Service, cfg config.Settings) {
	if service != nil {
		service.refreshFileStreamingCapability(cfg)
	}
}

func fileStreamingKey(cfg config.Settings) fileStreamingCapabilityKey {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return fileStreamingCapabilityKey{Profile: compatibility.Effective(cfg.CompatibilityProfile), Endpoint: cfg.BaseURL, Model: cfg.Model}
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return fileStreamingCapabilityKey{Profile: compatibility.Effective(cfg.CompatibilityProfile), Endpoint: parsed.String(), Model: cfg.Model}
}

func connectionServer(requestedURL string) string {
	parsed, err := url.Parse(requestedURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func (s *Service) applyFileStreamingCapabilityLocked(status *FileTranscriptionStatus, cfg config.Settings) {
	reason := ""
	if s.fileStreamingUnsupported != nil {
		reason = s.fileStreamingUnsupported[fileStreamingKey(cfg)]
	}
	contract, err := compatibility.Resolve(cfg.CompatibilityProfile, compatibility.Transcription)
	status.StreamingProfileUnavailable = err != nil || !contract.Capabilities.FileStreaming
	if status.StreamingProfileUnavailable {
		status.StreamingUnavailable = true
		status.StreamingNotice = "This compatibility profile uses completed file transcripts; streaming is unavailable."
		return
	}
	status.StreamingUnavailable = reason != ""
	if reason != "" {
		status.StreamingNotice = fileStreamingUnavailableNotice
	} else {
		status.StreamingNotice = ""
	}
}

func (s *Service) refreshFileStreamingCapability(cfg config.Settings) {
	s.fileMu.Lock()
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return
	}
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
}

func (s *Service) rememberFileStreamingUnsupported(generation uint64, cfg config.Settings, reason, fallbackOutcome string, completedMode bool) {
	s.fileMu.Lock()
	if s.fileStreamingUnsupported == nil {
		s.fileStreamingUnsupported = make(map[fileStreamingCapabilityKey]string)
	}
	s.fileStreamingUnsupported[fileStreamingKey(cfg)] = reason
	if s.fileStatus.Generation == generation {
		s.fileStatus.StreamingUnavailable = true
		s.fileStatus.StreamingNotice = fileStreamingUnavailableNotice
		if completedMode {
			s.fileStatus.Streaming = false
		}
	}
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Warn("audio file streaming unsupported",
		"generation", generation,
		"server", connectionServer(cfg.BaseURL),
		"reason", reason,
		"fallback_outcome", fallbackOutcome,
	)
}

// TryFileStreamingAgain clears the remembered unsupported-stream capability
// for the active endpoint and model so the user may explicitly retry it.
func (s *Service) TryFileStreamingAgain() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	cfg := s.current()
	contract, err := compatibility.Resolve(cfg.CompatibilityProfile, compatibility.Transcription)
	if err != nil || !contract.Capabilities.FileStreaming {
		return errors.New("file streaming is unavailable for this compatibility profile")
	}
	s.fileMu.Lock()
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return errors.New("wait for the active file transcription to finish")
	}
	delete(s.fileStreamingUnsupported, fileStreamingKey(cfg))
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Info("audio file streaming capability reset", "server", connectionServer(cfg.BaseURL))
	return nil
}

func (s *Service) updateFileUpload(generation uint64, sent int64, force bool) {
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || s.fileStatus.Phase != FileTranscriptionUploading {
		s.fileMu.Unlock()
		return
	}
	s.fileStatus.BytesUploaded = min(sent, s.fileStatus.FileSize)
	now := time.Now()
	if !force && now.Sub(s.fileLastPublish) < fileProgressInterval && sent < s.fileStatus.FileSize {
		s.fileMu.Unlock()
		return
	}
	s.fileLastPublish = now
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
}

func (s *Service) fileUploadComplete(generation uint64, stream bool) {
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || s.fileStatus.Phase != FileTranscriptionUploading {
		s.fileMu.Unlock()
		return
	}
	s.fileStatus.BytesUploaded = s.fileStatus.FileSize
	if stream {
		s.fileStatus.Phase = FileTranscriptionStreaming
		s.fileStatus.Message = "Waiting for transcript"
	} else {
		s.fileStatus.Phase = FileTranscriptionProcessing
		s.fileStatus.Message = "Transcribing audio"
	}
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	responseMode := "completed"
	if stream {
		responseMode = "stream"
	}
	s.log().Info("audio file upload completed", "generation", generation, "response_mode", responseMode)
}

func (s *Service) appendFileDelta(generation uint64, delta string) {
	if delta == "" {
		return
	}
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || (s.fileStatus.Phase != FileTranscriptionStreaming && !(s.fileStatus.Phase == FileTranscriptionProcessing && s.fileStatus.Streaming)) {
		s.fileMu.Unlock()
		return
	}
	if s.fileTranscript.Len()+len(delta) > maxFileTranscriptBytes {
		alreadyReported := s.fileTranscriptLimitHit
		s.fileTranscriptLimitHit = true
		s.fileStatus.Message = "Transcript reached the 8 MiB safety limit; preserving partial text"
		status := s.snapshotFileStatusLocked()
		changed := s.fileChanged
		s.fileMu.Unlock()
		if !alreadyReported {
			s.publishFileStatus(changed, status)
		}
		return
	}
	_, _ = s.fileTranscript.WriteString(delta)
	s.fileTranscriptRevision++
	s.fileStatus.TranscriptRevision = s.fileTranscriptRevision
	s.fileStatus.Message = "Receiving transcript"
	changed := s.fileDelta
	payload := FileTranscriptionDelta{Generation: generation, Revision: s.fileTranscriptRevision, Text: delta}
	s.fileMu.Unlock()
	s.publishFileDelta(changed, payload)
}

func (s *Service) fileStreamBuffered(generation uint64) {
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || s.fileStatus.Phase != FileTranscriptionStreaming {
		s.fileMu.Unlock()
		return
	}
	s.fileStatus.Phase = FileTranscriptionProcessing
	s.fileStatus.Buffered = true
	s.fileStatus.Message = "Server buffered the streamed response"
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Info("audio file stream buffered by server", "generation", generation)
}
