package dictation

import (
	"context"
	"errors"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/realtime"
	"github.com/tnware/freehand-stt/internal/settings"
)

// Start begins a toggle-controlled recording. Native hold-to-talk starts use
// StartWithMode so releasing the key remains their sole automatic boundary.
func (c *recorder) start(mode RecordingMode) error {
	return c.startWithMode(mode)
}

type startAttempt struct {
	epoch      uint64
	generation uint64
}

// Fence the whole request, including service validation and activity admission.
func (c *recorder) prepareStart() startAttempt {
	c.mu.Lock()
	defer c.mu.Unlock()
	return startAttempt{epoch: c.cancelEpoch, generation: c.status.Generation}
}

func (c *recorder) startWithMode(mode RecordingMode) error {
	return c.startWithAttempt(mode, c.prepareStart())
}

func (c *recorder) startWithAttempt(mode RecordingMode, attempt startAttempt) error {
	if mode != RecordingToggle && mode != RecordingHold {
		return errors.New("recording mode is invalid")
	}
	startRequested := time.Now()
	c.transition.Lock()
	defer c.transition.Unlock()
	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return errors.New("application is shutting down")
	}
	if c.status.State != Idle && c.status.State != Failed {
		c.mu.Unlock()
		return errors.New("dictation is already active")
	}
	if c.cancelEpoch != attempt.epoch || c.status.Generation != attempt.generation || c.rootContext.Err() != nil {
		c.mu.Unlock()
		return context.Canceled
	}
	c.mu.Unlock()
	profile, err := c.captureRequestProfile()
	if err != nil {
		message := "Cannot start recording. Check Voice settings, connection, and credentials."
		if errors.Is(err, settings.ErrManagedUnavailable) {
			message = "Managed runtime is unavailable. Start or repair the selected runtime, then try again."
		}
		c.rejectStartLocked(attempt, err, message)
		return err
	}
	defer func() {
		profile.STTCredential = ""
		profile.VoiceCredential = ""
		profile.PostProcessingCredential = ""
	}()
	target, captureErr := c.targetPlatform.CaptureTarget()
	// Capture failures deliberately produce an invalid target. Recording may
	// continue, but final text can only be copied by an explicit user action.
	if captureErr != nil || !target.Valid() {
		target = insertion.Target{}
		captureErr = insertion.CopyRequired(captureErr)
	}
	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return errors.New("application is shutting down")
	}
	if c.status.State != Idle && c.status.State != Failed {
		c.mu.Unlock()
		return errors.New("dictation is already active")
	}
	if c.cancelEpoch != attempt.epoch || c.status.Generation != attempt.generation || c.rootContext.Err() != nil {
		c.mu.Unlock()
		return context.Canceled
	}
	c.cancelWorkLocked()
	c.pending = ""
	cfg := profile.Settings
	if cfg.VoiceTranscription.Realtime {
		cfg.VADEnabled = false
		cfg.SilenceTrimming = false
		cfg.AutoStopEnabled = false
		cfg.SilenceSplitting = false
	}
	autoStopActive := cfg.AutoStopEnabled && mode == RecordingToggle
	runCfg := cfg
	runCfg.AutoStopEnabled = autoStopActive
	c.generation++
	gen := c.generation
	c.ctx, c.cancel = context.WithCancel(c.rootContext)
	recordingCtx, recordingCancel := context.WithCancel(c.ctx)
	c.recordingCancel = recordingCancel
	startedAt := startRequested.UTC()
	c.status = Status{State: Recording, RecordingMode: mode, Generation: gen, StartedAt: startedAt, Message: "Recording", CanCancel: true}
	if autoStopActive {
		c.status.AutoStopState = AutoStopWaiting
		c.status.AutoStopDurationMilliseconds = cfg.AutoStopSilenceMS
	}
	c.targets[gen] = capturedTarget{target: target, rejection: captureErr}
	c.runProfiles[gen] = profile
	c.runDetails[gen] = history.HistoryRunDetails{
		Source:                            history.HistorySourceVoice,
		RecordingMode:                     string(mode),
		StartedAt:                         startedAt,
		Server:                            history.SanitizedServer(cfg.BaseURL),
		Route:                             "/audio/transcriptions",
		AuthenticationMode:                string(cfg.AuthenticationMode),
		Model:                             cfg.Model,
		Language:                          cfg.Language,
		RequestTimeoutSeconds:             cfg.TranscriptionTimeoutSeconds,
		ResponseMode:                      history.HistoryResponseCompleted,
		InsertionMode:                     insertionMode(cfg.AutoInsert),
		Microphone:                        "System default",
		VADEnabled:                        cfg.VADEnabled,
		VADMode:                           string(cfg.VADMode),
		VADActivitySilenceMilliseconds:    cfg.VADActivitySilenceMS,
		SilenceTrimming:                   cfg.SilenceTrimming,
		SpeechPaddingMilliseconds:         cfg.SpeechPaddingMS,
		AutoStopEnabled:                   cfg.AutoStopEnabled,
		AutoStopActive:                    autoStopActive,
		AutoStopSilenceMilliseconds:       cfg.AutoStopSilenceMS,
		AutoStopMinimumSpeechMilliseconds: cfg.AutoStopMinimumSpeechMS,
		SilenceSplitting:                  cfg.SilenceSplitting,
		Processing: history.HistoryProcessingDetails{
			Requested:      cfg.PostProcessing.Enabled,
			Status:         history.HistoryProcessingNotRequested,
			TimeoutSeconds: cfg.PostProcessing.TimeoutSeconds,
		},
	}
	if cfg.VoiceTranscription.Realtime {
		c.status.Live = true
		c.status.LiveCaptions = cfg.VoiceTranscription.Captions
		details := c.runDetails[gen]
		details.Server = history.SanitizedServer(cfg.VoiceTranscription.BaseURL)
		details.Route = "/realtime"
		details.AuthenticationMode = string(cfg.VoiceTranscription.AuthenticationMode)
		details.Model = cfg.VoiceTranscription.Model
		details.Language = cfg.VoiceTranscription.Language
		details.RequestTimeoutSeconds = 30
		c.runDetails[gen] = details
	}
	c.mu.Unlock()
	c.logger.Info("dictation recording requested",
		"generation", gen,
		"recording_mode", mode,
		"insertion_mode", insertionMode(cfg.AutoInsert),
		"vad_enabled", cfg.VADEnabled,
		"vad_mode", cfg.VADMode,
		"vad_activity_silence_ms", cfg.VADActivitySilenceMS,
		"silence_trimming", cfg.SilenceTrimming,
		"speech_padding_ms", cfg.SpeechPaddingMS,
		"auto_stop_configured", cfg.AutoStopEnabled,
		"auto_stop_active", autoStopActive,
		"auto_stop_silence_ms", cfg.AutoStopSilenceMS,
		"auto_stop_minimum_speech_ms", cfg.AutoStopMinimumSpeechMS,
		"silence_splitting", cfg.SilenceSplitting,
		"maximum_seconds", cfg.MaxDurationSeconds,
		"transcription_timeout_seconds", cfg.TranscriptionTimeoutSeconds,
		"post_processing_timeout_seconds", cfg.PostProcessing.TimeoutSeconds,
		"segment_target_seconds", cfg.SegmentSeconds,
		"segment_silence_ms", cfg.SegmentSilenceMS,
	)
	var interruptions <-chan error
	var segmented *segmentedRun
	var live *realtime.Session
	var e error
	captureStarted := time.Now()
	if cfg.VoiceTranscription.Realtime {
		streamCapture, ok := c.capture.(audio.StreamCapture)
		if !ok {
			e = errors.New("streaming capture is unavailable on this platform")
		} else {
			live, e = realtime.Open(c.ctx, cfg.VoiceTranscription, profile.VoiceCredential, func(update realtime.Update) { c.publishLive(gen, update) })
			if e == nil {
				c.mu.Lock()
				c.realtime = live
				c.mu.Unlock()
				interruptions, e = streamCapture.StartStream(c.ctx, cfg.MicrophoneID, cfg.MaxDurationSeconds, live.Pipe)
			}
		}
	} else if cfg.VADEnabled && (cfg.SilenceTrimming || cfg.AutoStopEnabled || cfg.SilenceSplitting) {
		streamCapture, ok := c.capture.(audio.StreamCapture)
		if !ok {
			e = errors.New("voice activity detection is unavailable on this platform")
		} else {
			var detector audio.VoiceDetector
			detector, e = c.newDetector(cfg.VADMode)
			if e == nil {
				segmented, e = newSegmentedRun(c.ctx, runCfg, profile.STTCredential, c.client, detector, c.logger.With("generation", gen), func(segment int, phase SegmentPhase) {
					c.publishSegmentProgress(gen, segment, phase)
				}, func(active bool) {
					c.publishVADState(gen, active)
				}, func(armed, countdown bool, remainingMilliseconds int) {
					c.publishAutoStopState(gen, armed, countdown, remainingMilliseconds)
				})
			}
			if e == nil {
				c.mu.Lock()
				c.segmented = segmented
				c.mu.Unlock()
				interruptions, e = streamCapture.StartStream(c.ctx, cfg.MicrophoneID, cfg.MaxDurationSeconds, segmented.pipe)
			}
		}
	} else {
		interruptions, e = c.capture.Start(c.ctx, cfg.MicrophoneID, cfg.MaxDurationSeconds)
	}
	if e != nil {
		if live != nil {
			live.AbortBeforeCapture()
			c.mu.Lock()
			c.realtime = nil
			c.mu.Unlock()
		}
		if segmented != nil {
			_ = segmented.abortBeforeCapture()
			c.mu.Lock()
			if c.segmented == segmented {
				c.segmented = nil
			}
			c.mu.Unlock()
		}
		c.fail(gen, "Microphone: "+e.Error())
		c.logger.Error("dictation capture failed to start", "generation", gen, "duration_ms", time.Since(startRequested).Milliseconds(), "error_kind", diagnostics.ErrorKind(e))
		return e
	}
	if c.closed.Load() || c.ctx.Err() != nil {
		return errors.New("recording cancelled during microphone startup")
	}
	readyAt := time.Now().UTC()
	c.mu.Lock()
	if c.status.Generation == gen && c.status.State == Recording {
		c.status.StartedAt = readyAt
		details := c.runDetails[gen]
		details.StartedAt = readyAt
		if namer, ok := c.capture.(audio.DeviceNamer); ok {
			if microphone := boundedLabel(namer.DeviceName(), 256); microphone != "" {
				details.Microphone = microphone
			}
		}
		c.runDetails[gen] = details
	}
	s := c.status
	c.mu.Unlock()
	c.logger.Info("microphone ready to record",
		"generation", gen,
		"startup_ms", time.Since(startRequested).Milliseconds(),
		"capture_start_ms", time.Since(captureStarted).Milliseconds(),
	)
	c.publish(s)
	go c.watchRecording(gen, recordingCtx, time.Duration(cfg.MaxDurationSeconds)*time.Second, interruptions, segmented, live)
	return nil
}

func (c *recorder) rejectStart(attempt startAttempt, err error, message string) error {
	c.transition.Lock()
	defer c.transition.Unlock()
	c.rejectStartLocked(attempt, err, message)
	return err
}

// rejectStartLocked publishes admission feedback, not a failed recording run.
// The caller holds transition; cancellation can still signal through mu.
// Keep the previous result and its copy capability/generation intact.
// Messages are fixed application text, never external error strings.
func (c *recorder) rejectStartLocked(attempt startAttempt, err error, message string) {
	c.mu.Lock()
	if c.closed.Load() || c.rootContext.Err() != nil || c.cancelEpoch != attempt.epoch || c.status.Generation != attempt.generation ||
		(c.status.State != Idle && c.status.State != Failed) || errors.Is(err, context.Canceled) {
		c.mu.Unlock()
		return
	}
	c.status.State = Failed
	c.status.StartRejected = true
	c.status.Message = message
	status := c.status
	c.mu.Unlock()
	c.publish(status)
}
