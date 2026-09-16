package filetranscription

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
)

// StartFileTranscription uploads the selected file using completed or streamed
// response mode while keeping the file path and bytes out of the renderer.
func (s *Service) StartFileTranscription(stream bool) error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	release, err := s.activity.BeginFileTranscription()
	if err != nil {
		return err
	}
	defer release()
	s.fileMu.Lock()
	if s.closed.Load() {
		s.fileMu.Unlock()
		return errors.New("application is shutting down")
	}
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return errors.New("an audio file is already being transcribed")
	}
	if s.fileChoosing {
		s.fileMu.Unlock()
		return errors.New("finish choosing an audio file before starting transcription")
	}
	selection := s.fileSelection
	s.fileMu.Unlock()
	if selection == nil {
		return errors.New("choose an audio file first")
	}

	profile, err := s.captureRequestProfile()
	if err != nil {
		return err
	}
	defer func() {
		profile.STTCredential = ""
		profile.PostProcessingCredential = ""
	}()
	cfg := profile.Settings
	file, info, err := selection.open()
	if err != nil {
		return err
	}
	key := profile.STTCredential
	processingKey := profile.PostProcessingCredential
	ctx, cancel := s.operationContext()
	done := make(chan struct{})

	s.fileMu.Lock()
	if s.closed.Load() || s.fileSelection != selection || filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		cancel()
		_ = file.Close()
		key = ""
		processingKey = ""
		return errors.New("the selected audio file changed before transcription started")
	}
	s.fileGeneration++
	generation := s.fileGeneration
	streamingUnavailable := false
	if s.fileStreamingUnsupported != nil {
		_, streamingUnavailable = s.fileStreamingUnsupported[fileStreamingKey(cfg)]
	}
	contract, contractErr := compatibility.Resolve(cfg.CompatibilityProfile, compatibility.Transcription)
	effectiveStream := stream && !streamingUnavailable && contractErr == nil && contract.Capabilities.FileStreaming
	s.fileCancel = cancel
	s.fileDone = done
	s.fileLastPublish = time.Time{}
	s.workers.Add(1)
	s.fileStatus = FileTranscriptionStatus{
		Generation: generation,
		Phase:      FileTranscriptionUploading,
		FileName:   info.Name(),
		FileSize:   info.Size(),
		Streaming:  effectiveStream,
		Message:    "Uploading audio",
		CanCancel:  true,
	}
	s.resetFileTranscriptLocked("")
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Info("audio file transcription started", "generation", generation, "server", connectionServer(cfg.BaseURL), "bytes", info.Size(), "streaming_requested", stream, "streaming_effective", effectiveStream, "timeout_seconds", cfg.FileTranscriptionTimeoutSeconds, "post_processing_timeout_seconds", cfg.PostProcessing.TimeoutSeconds)

	go func() {
		defer s.workers.Done()
		s.runFileTranscription(ctx, generation, file, info.Size(), key, processingKey, cfg, effectiveStream, done, profile.PostProcessingUnavailable)
	}()
	return nil
}

func (s *Service) runFileTranscription(ctx context.Context, generation uint64, file *os.File, size int64, key, processingKey string, cfg config.Settings, stream bool, done chan struct{}, processingUnavailable error) {
	started := time.Now()
	startedAt := started.UTC()
	responseMode := history.HistoryResponseCompleted
	if stream {
		responseMode = history.HistoryResponseStreamed
	}
	details := history.HistoryRunDetails{
		Source:                history.HistorySourceAudioFile,
		StartedAt:             startedAt,
		Server:                history.SanitizedServer(cfg.BaseURL),
		Route:                 "/audio/transcriptions",
		AuthenticationMode:    string(cfg.AuthenticationMode),
		Model:                 cfg.Model,
		Language:              cfg.Language,
		ResponseMode:          responseMode,
		InsertionMode:         insertion.ManualCopy,
		FileName:              filepath.Base(file.Name()),
		FileSize:              size,
		RequestTimeoutSeconds: cfg.FileTranscriptionTimeoutSeconds,
		Processing:            history.NewProcessingDetails(cfg.PostProcessing, ""),
	}
	var uploadMilliseconds atomic.Int64
	defer func() {
		s.fileMu.Lock()
		if s.fileDone == done {
			s.fileDone = nil
		}
		s.fileMu.Unlock()
		close(done)
	}()
	defer file.Close()
	defer func() {
		key = ""
		processingKey = ""
	}()

	transcribe := func(streaming bool) (inference.TranscriptionResult, string, error) {
		unsupportedReason := ""
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(cfg.FileTranscriptionTimeoutSeconds)*time.Second)
		defer requestCancel()
		result, err := s.client.WithCompatibility(cfg.CompatibilityProfile).WithModelProfile(cfg.ModelProfile).WithTranscriptionOptions(cfg.TranscriptionOptions.Inference()).TranscribeFile(requestCtx, cfg.BaseURL, cfg.Model, cfg.Language, key, cfg.Headers, filepath.Base(file.Name()), size, io.LimitReader(file, size), streaming, inference.FileTranscriptionCallbacks{
			UploadProgress: func(sent, _ int64) { s.updateFileUpload(generation, sent, false) },
			UploadComplete: func() {
				uploadMilliseconds.Store(time.Since(started).Milliseconds())
				s.fileUploadComplete(generation, streaming)
			},
			Delta:             func(delta string) { s.appendFileDelta(generation, delta) },
			StreamBuffered:    func() { s.fileStreamBuffered(generation) },
			StreamUnsupported: func(reason string) { unsupportedReason = reason },
		})
		return result, unsupportedReason, err
	}

	transcription, unsupportedReason, err := transcribe(stream)
	text := transcription.Text
	s.fileMu.Lock()
	partialText := ""
	transcriptLimitHit := s.fileStatus.Generation == generation && s.fileTranscriptLimitHit
	if s.fileStatus.Generation == generation && (transcriptLimitHit || diagnostics.ErrorKind(err) == "response_too_large") {
		partialText = s.fileTranscript.String()
	}
	s.fileMu.Unlock()
	if transcriptLimitHit {
		text = partialText
		err = &inference.Error{Kind: "response_too_large", Message: "transcript exceeded Freehand's 8 MiB safety limit; partial text was preserved"}
		s.log().Warn("audio file transcript safety limit reached", "generation", generation, "limit_bytes", maxFileTranscriptBytes, "partial_bytes", len(partialText))
	} else if partialText != "" && diagnostics.ErrorKind(err) == "response_too_large" {
		// The SSE reader bounds the complete wire response as well as the text
		// accumulator. Preserve already accepted deltas when framing overhead
		// reaches that limit first.
		text = partialText
	}
	if stream && unsupportedReason != "" && err == nil {
		details.ResponseMode = history.HistoryResponseCompleted
		details.StreamFallbackReason = unsupportedReason
		s.rememberFileStreamingUnsupported(generation, cfg, unsupportedReason, "completed_response_used", true)
	}
	var unsupported *inference.FileStreamUnsupportedError
	if stream && errors.As(err, &unsupported) {
		details.StreamFallbackReason = unsupported.Reason
		if unsupported.PartialText != "" {
			text = unsupported.PartialText
			s.rememberFileStreamingUnsupported(generation, cfg, unsupported.Reason, "partial_result_preserved", false)
		} else {
			// An unreadable stream may already represent completed inference.
			// Remember the capability, but require an explicit new user action
			// before submitting the file again, even after parameter rejection.
			s.rememberFileStreamingUnsupported(generation, cfg, unsupported.Reason, "explicit_retry_required", true)
		}
	}
	details.Transcription = history.NewResponseDetails(transcription.Metadata)
	totalTranscriptionMilliseconds := time.Since(started).Milliseconds()
	details.UploadMilliseconds = uploadMilliseconds.Load()
	details.TranscriptionMilliseconds = max(0, totalTranscriptionMilliseconds-details.UploadMilliseconds)
	if s.closed.Load() {
		s.log().Info("audio file transcription cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
		return
	}
	s.fileMu.Lock()
	details.Buffered = s.fileStatus.Generation == generation && s.fileStatus.Buffered
	if text != "" && err != nil && s.fileStatus.Generation == generation {
		s.replaceFileTranscriptLocked(text)
	}
	s.fileMu.Unlock()
	details.Processing = history.NewProcessingDetails(cfg.PostProcessing, text)

	processingFallback := false
	processingFallbackKind := ""
	historyID := uint64(0)
	if text != "" && s.history != nil {
		historyID = s.history.Begin(text, history.HistoryTranscribed, details.Processing.Requested, time.Now().UTC(), details)
	}
	if err == nil && text != "" && cfg.PostProcessing.Enabled {
		s.fileMu.Lock()
		if s.fileStatus.Generation == generation {
			s.fileStatus.Phase = FileTranscriptionProcessing
			s.replaceFileTranscriptLocked(text)
			s.fileStatus.Message = "Post-processing transcript"
			status := s.snapshotFileStatusLocked()
			changed := s.fileChanged
			s.fileMu.Unlock()
			s.publishFileStatus(changed, status)
		} else {
			s.fileMu.Unlock()
		}

		processingStarted := time.Now()
		var processingResult postprocess.Result
		var processingErr error
		var detectedLanguages []string
		if details.Transcription != nil {
			detectedLanguages = details.Transcription.DetectedLanguages
		}
		processingErr = processingUnavailable
		if processingErr == nil {
			processingErr = postprocess.ValidateLanguage(cfg.PostProcessing, cfg.Language, detectedLanguages)
		}
		if processingErr == nil {
			if s.processor == nil {
				processingErr = errors.New("post-processing is unavailable")
			} else {
				processingResult, processingErr = s.processor.ProcessWithCredential(diagnostics.WithOperation(ctx, diagnostics.File, generation), cfg.PostProcessing, text, processingKey)
			}
		}
		processing := postprocess.Resolve(ctx, text, processingResult, processingErr, processingStarted)
		details = s.history.FinalizeProcessing(historyID, text, processing, details)
		// Cancellation may arrive while history finalizes a failed attempt.
		// Keep the final fallback-admission check with the live workflow owner.
		if ctxErr := ctx.Err(); processing.Err != nil && errors.Is(ctxErr, context.Canceled) {
			err = ctxErr
		}
		processingFallback = processing.Fallback() && err == nil
		if processingFallback {
			processingFallbackKind = diagnostics.ErrorKind(processing.Err)
			s.log().Warn("audio file post-processing fell back to raw transcript", "generation", generation, "error_kind", processingFallbackKind)
		}
		text = processing.Text
	}

	s.fileMu.Lock()
	if s.closed.Load() || s.fileStatus.Generation != generation {
		s.fileMu.Unlock()
		s.log().Info("audio file transcription cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
		return
	}
	s.fileCancel = nil
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.fileStatus.Phase = FileTranscriptionSelected
			s.fileStatus.Message = "Transcription cancelled"
			s.replaceFileTranscriptLocked("")
			s.fileStatus.CanStart = true
			s.fileStatus.CanCancel = false
			s.fileStatus.CanCopy = false
		} else {
			s.fileStatus.Phase = FileTranscriptionFailed
			if text != "" {
				if diagnostics.ErrorKind(err) == "response_too_large" {
					s.fileStatus.Message = "Transcript reached the 8 MiB safety limit; partial text preserved"
				} else {
					s.fileStatus.Message = "Stream ended after a partial transcript"
				}
			} else {
				s.fileStatus.Message = err.Error()
			}
			s.fileStatus.CanStart = true
			s.fileStatus.CanCancel = false
			s.fileStatus.CanCopy = s.fileTranscript.Len() != 0
		}
	} else {
		s.fileStatus.Phase = FileTranscriptionCompleted
		s.fileStatus.BytesUploaded = s.fileStatus.FileSize
		s.replaceFileTranscriptLocked(text)
		if processingFallback {
			s.fileStatus.Message = "Transcription complete; post-processing failed, using raw text"
			if processingFallbackKind == "unsupported_language" {
				s.fileStatus.Message = "Transcription complete; S1-mini skipped (English only), using raw text"
			} else if processingFallbackKind == "timeout" {
				s.fileStatus.Message = "Transcription complete; post-processing timed out, using raw text"
			} else if processingFallbackKind == "incomplete_response" {
				s.fileStatus.Message = "Transcription complete; post-processing reached the output limit, using raw text"
			}
		} else if text == "" {
			s.fileStatus.Message = "No speech detected"
		} else {
			s.fileStatus.Message = "Transcription complete"
		}
		s.fileStatus.CanStart = true
		s.fileStatus.CanCancel = false
		s.fileStatus.CanCopy = text != ""
	}
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	if historyID != 0 && s.history != nil {
		outcome := history.HistoryTranscribed
		if err != nil {
			details.ErrorKind = history.ErrorKind(err)
			details.Processing.DeliveredCharacters = utf8.RuneCountInString(text)
			outcome = history.HistoryFailed
			if errors.Is(ctx.Err(), context.Canceled) {
				outcome = history.HistoryCancelled
			}
		} else {
			details.Processing.DeliveredCharacters = utf8.RuneCountInString(text)
		}
		s.history.Finalize(historyID, outcome, details, time.Now().UTC(), false)
	}

	s.publishFileStatus(changed, status)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.log().Info("audio file transcription cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
		} else {
			s.log().Error("audio file transcription failed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		}
		return
	}
	s.log().Info("audio file transcription completed", "generation", generation, "characters", utf8.RuneCountInString(text), "duration_ms", time.Since(started).Milliseconds(), "outcome", "transcribed", "response_mode", details.ResponseMode, "stream_fallback_reason", details.StreamFallbackReason)
}

// CancelFileTranscription requests cancellation of active stored-audio work.
func (s *Service) CancelFileTranscription() error {
	s.fileMu.Lock()
	if !filePhaseActive(s.fileStatus.Phase) || s.fileCancel == nil {
		s.fileMu.Unlock()
		return nil
	}
	s.fileStatus.Phase = FileTranscriptionCancelling
	s.fileStatus.Message = "Cancelling transcription"
	s.fileStatus.CanCancel = false
	cancel := s.fileCancel
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	cancel()
	return nil
}
