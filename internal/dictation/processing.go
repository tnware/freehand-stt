package dictation

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
)

func (c *recorder) completeStopped(work *stoppedRecording) error {
	gen := work.generation
	ctx := work.context
	res := work.result
	segmented := work.segmented
	profile := work.profile
	details := work.details
	limit := work.limit
	automatic := work.automatic
	defer func() {
		profile.STTCredential = ""
		profile.VoiceCredential = ""
		profile.PostProcessingCredential = ""
		work.clearCredentials()
	}()

	var (
		e error
		s Status
	)
	if !work.hasProfile {
		var profileErr error
		profile, profileErr = c.captureRequestProfile()
		if profileErr != nil {
			c.fail(gen, profileErr.Error())
			return profileErr
		}
	}
	cfg := profile.Settings
	text := ""
	if work.realtime != nil {
		result := work.realtime.Wait()
		text, e = result.Text, result.Err
		details.AudioDurationMilliseconds = result.AudioMilliseconds
		cfg.Language = cfg.VoiceTranscription.Language
		languages := result.Languages
		if result.Language != "" {
			languages = append(languages, result.Language)
		}
		details.Transcription = history.NewResponseDetails(inference.ResponseMetadata{DetectedLanguages: languages, RequestCount: 1})
	} else if segmented != nil {
		result := segmented.wait()
		if result.err != nil {
			e = result.err
		} else {
			text = result.text
			details.Transcription = history.NewResponseDetails(result.metadata)
			details.AudioDurationMilliseconds = result.audioMilliseconds
			if cfg.SilenceSplitting {
				details.SegmentCount = result.chunks
				details.Segments = result.segments
				details.SegmentsTruncated = result.segmentsTruncated
			}
			for _, segment := range result.segments {
				details.TranscriptionMilliseconds += segment.RequestMilliseconds
			}
		}
	} else {
		details.AudioDurationMilliseconds = int64(len(res.PCM) * 1000 / (audio.SampleRate * 2))
		wav, wavErr := audio.WAV(res.PCM)
		for i := range res.PCM {
			res.PCM[i] = 0
		}
		if wavErr != nil {
			c.fail(gen, wavErr.Error())
			return wavErr
		}
		defer func() {
			for i := range wav {
				wav[i] = 0
			}
		}()
		transcriptionStarted := time.Now()
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(cfg.TranscriptionTimeoutSeconds)*time.Second)
		transcription, transcriptionErr := c.client.WithCompatibility(cfg.CompatibilityProfile).WithModelProfile(cfg.ModelProfile).WithTranscriptionOptions(cfg.TranscriptionOptions.Inference()).Transcribe(requestCtx, cfg.BaseURL, cfg.Model, cfg.Language, profile.STTCredential, cfg.Headers, wav)
		requestCancel()
		text = transcription.Text
		e = transcriptionErr
		details.Transcription = history.NewResponseDetails(transcription.Metadata)
		details.TranscriptionMilliseconds = time.Since(transcriptionStarted).Milliseconds()
	}
	if e != nil {
		c.logger.Error("dictation transcription failed", "generation", gen, "error_kind", diagnostics.ErrorKind(e))
		c.fail(gen, e.Error())
		return e
	}
	if text == "" {
		c.mu.Lock()
		if c.status.Generation == gen && c.status.State == Transcribing {
			delete(c.targets, gen)
			c.cancelWorkLocked()
			c.status = Status{State: Idle, Generation: gen, Message: "No speech detected"}
			s = c.status
			c.mu.Unlock()
			c.logger.Info("dictation completed without detected speech", "generation", gen)
			c.publish(s)
			return nil
		}
		c.mu.Unlock()
		return nil
	}
	historyID := uint64(0)
	processingFallback := false
	processingFallbackKind := ""
	completionState := Transcribing
	c.mu.Lock()
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Transcribing {
		c.mu.Unlock()
		return nil
	}
	details.Processing = history.NewProcessingDetails(cfg.PostProcessing, text)
	historyID = c.history.Begin(text, history.HistoryInserted, cfg.PostProcessing.Enabled, time.Now().UTC(), details)
	processor := c.processor
	if cfg.PostProcessing.Enabled {
		completionState = PostProcessing
		c.status.State = PostProcessing
		c.status.Message = "Post-processing transcript"
		s = c.status
	}
	c.mu.Unlock()
	if cfg.PostProcessing.Enabled {
		c.publish(s)
		processingStarted := time.Now()
		var processingResult postprocess.Result
		var processingErr error
		var detectedLanguages []string
		if details.Transcription != nil {
			detectedLanguages = details.Transcription.DetectedLanguages
		}
		processingErr = profile.PostProcessingUnavailable
		if processingErr == nil {
			processingErr = postprocess.ValidateLanguage(cfg.PostProcessing, cfg.Language, detectedLanguages)
		}
		if processingErr == nil {
			if processor == nil {
				processingErr = errors.New("post-processing is unavailable")
			} else {
				processingResult, processingErr = processor.ProcessWithCredential(diagnostics.WithOperation(ctx, diagnostics.Dictation, gen), cfg.PostProcessing, text, profile.PostProcessingCredential)
			}
		}
		processing := postprocess.Resolve(ctx, text, processingResult, processingErr, processingStarted)
		details = c.history.FinalizeProcessing(historyID, text, processing, details)
		// History finalization may block. Recheck workflow cancellation before
		// admitting raw fallback; the resolved attempt is only a snapshot.
		if processing.Err != nil && errors.Is(ctx.Err(), context.Canceled) {
			c.history.Finalize(historyID, history.HistoryCancelled, details, time.Now().UTC(), false)
			return nil
		}
		processingFallback = processing.Fallback()
		if processingFallback {
			processingFallbackKind = diagnostics.ErrorKind(processing.Err)
			c.logger.Warn("dictation post-processing fell back to raw transcript", "generation", gen, "error_kind", processingFallbackKind)
		}
		text = processing.Text
		if !processingFallback {
			if text == "" {
				c.mu.Lock()
				if c.status.Generation == gen && c.status.State == PostProcessing {
					delete(c.targets, gen)
					c.cancelWorkLocked()
					c.status = Status{State: Idle, Generation: gen, Message: "Post-processing removed filler or noise"}
					s = c.status
					c.mu.Unlock()
					c.publish(s)
					return nil
				}
				c.mu.Unlock()
				return nil
			}
		}
	}
	c.mu.Lock()
	if c.closed.Load() || c.status.Generation != gen || c.status.State != completionState {
		c.mu.Unlock()
		return nil
	}
	c.status.State = Ready
	c.status.Message = "Ready to insert"
	s = c.status
	c.mu.Unlock()
	c.publish(s)
	c.mu.Lock()
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Ready || ctx.Err() != nil {
		c.mu.Unlock()
		return nil
	}
	target := c.targets[gen]
	delete(c.targets, gen)
	deliveryMode := insertionMode(cfg.AutoInsert)
	e = c.policy.Deliver(ctx, target.target, text, deliveryMode)
	if errors.Is(e, insertion.ErrCopyRequired) && target.rejection != nil {
		e = target.rejection
	}
	outcome := history.HistoryInserted
	if errors.Is(e, insertion.ErrCopyRequired) {
		c.pending = text
		message := insertion.CopyRequiredMessage(e)
		if deliveryMode == insertion.ManualCopy {
			message = "Transcript ready to copy"
			e = nil
		}
		c.status = Status{State: Failed, Generation: gen, Message: message, CanCopy: true}
		outcome = history.HistoryCopyRequired
	} else if e != nil {
		c.pending = text
		c.status = Status{State: Failed, Generation: gen, Message: insertion.CopyRequiredMessage(e), CanCopy: true}
		outcome = history.HistoryFailed
	} else if processingFallback {
		message := "Post-processing failed; raw transcript used"
		if processingFallbackKind == "unsupported_language" {
			message = "S1-mini skipped: English only; raw transcript used"
		} else if processingFallbackKind == "timeout" {
			message = "Post-processing timed out; raw transcript used"
		} else if processingFallbackKind == "incomplete_response" {
			message = "Post-processing reached the output limit; raw transcript used"
		}
		c.status = Status{State: Idle, Generation: gen, Message: message}
	} else if automatic {
		c.status = Status{State: Idle, Generation: gen, Message: "Silence detected; dictation completed"}
	} else if limit {
		c.status = Status{State: Idle, Generation: gen, Message: "Maximum duration reached; dictation completed"}
	} else {
		c.status = Status{State: Idle, Generation: gen}
	}
	c.status.Transcript = text
	completedAt := time.Now().UTC()
	details.Processing.DeliveredCharacters = utf8.RuneCountInString(text)
	if historyID == 0 {
		history.FinalizeDetails(&details, completedAt, limit)
		historyID = c.history.Begin(text, outcome, false, completedAt, details)
	} else {
		c.history.Finalize(historyID, outcome, details, completedAt, limit)
	}
	c.cancelWorkLocked()
	s = c.status
	c.mu.Unlock()
	c.logger.Info("dictation completed",
		"generation", gen,
		"characters", utf8.RuneCountInString(text),
		"outcome", outcome,
		"duration_limit_reached", limit,
		"automatic_stop", automatic,
	)
	c.publish(s)
	return e
}

func insertionMode(autoInsert bool) insertion.Mode {
	if autoInsert {
		return insertion.DirectInput
	}
	return insertion.ManualCopy
}

func boundedLabel(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}
