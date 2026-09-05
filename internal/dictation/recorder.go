package dictation

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/tnware/freehand-stt/internal/webrtcvad"
)

type State string

type SegmentPhase string

type VADState string

type AutoStopState string

type RecordingMode string

const (
	Idle           State = "idle"
	Recording      State = "recording"
	Transcribing   State = "transcribing"
	PostProcessing State = "post-processing"
	Ready          State = "ready-to-insert"
	Cancelling     State = "cancelling"
	Failed         State = "failed"
)

const (
	SegmentTranscribing SegmentPhase = "transcribing"
	SegmentCompleted    SegmentPhase = "completed"
)

const (
	VADSpeech  VADState = "speech"
	VADSilence VADState = "silence"
)

const (
	AutoStopWaiting   AutoStopState = "waiting-for-speech"
	AutoStopListening AutoStopState = "listening"
	AutoStopCountdown AutoStopState = "countdown"
)

const (
	RecordingToggle RecordingMode = "toggle"
	RecordingHold   RecordingMode = "hold"
)

type Status struct {
	State                        State         `json:"state"`
	RecordingMode                RecordingMode `json:"recordingMode,omitempty"`
	Generation                   uint64        `json:"generation"`
	StartedAt                    time.Time     `json:"startedAt,omitempty"`
	Message                      string        `json:"message,omitempty"`
	CanCancel                    bool          `json:"canCancel"`
	CanCopy                      bool          `json:"canCopy"`
	SegmentNumber                int           `json:"segmentNumber,omitempty"`
	SegmentPhase                 SegmentPhase  `json:"segmentPhase,omitempty"`
	VADState                     VADState      `json:"vadState,omitempty"`
	AutoStopState                AutoStopState `json:"autoStopState,omitempty"`
	AutoStopDeadline             time.Time     `json:"autoStopDeadline,omitempty"`
	AutoStopDurationMilliseconds int           `json:"autoStopDurationMilliseconds,omitempty"`
}
type recorder struct {
	mu                 sync.Mutex
	transition         sync.Mutex
	status             Status
	generation         uint64
	ctx                context.Context
	rootContext        context.Context
	cancel             context.CancelFunc
	recordingCancel    context.CancelFunc
	capture            audio.Capture
	targetPlatform     insertion.Platform
	policy             insertion.Policy
	client             *inference.Client
	processor          *postprocess.Processor
	settings           settings.Source
	profiles           settings.ProfileSource
	history            *history.Store
	changed            func(Status)
	closed             bool
	targets            map[uint64]insertion.Target
	pending            string
	runProfiles        map[uint64]settings.RequestProfile
	runDetails         map[uint64]history.HistoryRunDetails
	segmented          *segmentedRun
	scheduleCompletion func(func()) bool
	newDetector        func(config.VADMode) (audio.VoiceDetector, error)
	logger             *slog.Logger
}

type stoppedRecording struct {
	generation uint64
	context    context.Context
	result     audio.Result
	segmented  *segmentedRun
	profile    settings.RequestProfile
	hasProfile bool
	details    history.HistoryRunDetails
	limit      bool
	automatic  bool
}

func (w *stoppedRecording) clearCredentials() {
	w.profile.STTCredential = ""
	w.profile.PostProcessingCredential = ""
}

func (w *stoppedRecording) discard() {
	w.clearCredentials()
	for index := range w.result.PCM {
		w.result.PCM[index] = 0
	}
	if w.segmented != nil {
		_ = w.segmented.wait()
	}
}

func newRecorder(cap audio.Capture, p insertion.Platform, client *inference.Client, processor *postprocess.Processor, source settings.Source, profiles settings.ProfileSource, store *history.Store, changed func(Status), logger *slog.Logger) *recorder {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &recorder{status: Status{State: Idle}, rootContext: context.Background(), capture: cap, targetPlatform: p, policy: insertion.Policy{Platform: p}, client: client, processor: processor, settings: source, profiles: profiles, history: store, changed: changed, targets: make(map[uint64]insertion.Target), runProfiles: make(map[uint64]settings.RequestProfile), runDetails: make(map[uint64]history.HistoryRunDetails), logger: logger, newDetector: func(mode config.VADMode) (audio.VoiceDetector, error) {
		nativeMode := webrtcvad.ModeAggressive
		switch mode {
		case config.VADModeQuality:
			nativeMode = webrtcvad.ModeQuality
		case config.VADModeLowBitrate:
			nativeMode = webrtcvad.ModeLowBitrate
		case config.VADModeAggressive:
			nativeMode = webrtcvad.ModeAggressive
		case config.VADModeVeryAggressive:
			nativeMode = webrtcvad.ModeVeryAggressive
		default:
			return nil, errors.New("voice activity detection mode is invalid")
		}
		return webrtcvad.New(audio.SampleRate, nativeMode)
	}}
}

func (c *recorder) setRootContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	if !c.closed {
		c.rootContext = ctx
	}
	c.mu.Unlock()
}

func (c *recorder) captureRequestProfile() (settings.RequestProfile, error) {
	c.mu.Lock()
	source := c.profiles
	c.mu.Unlock()
	if source != nil {
		return source.Capture()
	}
	if c.settings == nil {
		return settings.RequestProfile{}, errors.New("transcription profile is unavailable")
	}
	return settings.RequestProfile{Settings: c.settings.Current()}, nil
}
func (c *recorder) snapshotLocked() Status { return c.status }
func (c *recorder) cancelRecordingWatchLocked() {
	if c.recordingCancel != nil {
		c.recordingCancel()
		c.recordingCancel = nil
	}
}
func (c *recorder) cancelWorkLocked() {
	c.cancelRecordingWatchLocked()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}
func (c *recorder) publish(s Status) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return
	}
	if c.changed != nil {
		c.changed(s)
	}
}

func (c *recorder) currentStatus() Status { c.mu.Lock(); defer c.mu.Unlock(); return c.status }

// Start begins a toggle-controlled recording. Native hold-to-talk starts use
// StartWithMode so releasing the key remains their sole automatic boundary.
func (c *recorder) start(mode RecordingMode) error {
	return c.startWithMode(mode)
}

func (c *recorder) startWithMode(mode RecordingMode) error {
	if mode != RecordingToggle && mode != RecordingHold {
		return errors.New("recording mode is invalid")
	}
	startRequested := time.Now()
	c.transition.Lock()
	defer c.transition.Unlock()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("application is shutting down")
	}
	if c.status.State != Idle && c.status.State != Failed {
		c.mu.Unlock()
		return errors.New("dictation is already active")
	}
	c.mu.Unlock()
	profile, err := c.captureRequestProfile()
	if err != nil {
		return err
	}
	defer func() {
		profile.STTCredential = ""
		profile.PostProcessingCredential = ""
	}()
	target, _ := c.targetPlatform.CaptureTarget()
	// Capture failures deliberately produce an invalid target. Recording may
	// continue, but final text can only be copied by an explicit user action.
	if !target.Valid() {
		target = insertion.Target{}
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("application is shutting down")
	}
	if c.status.State != Idle && c.status.State != Failed {
		c.mu.Unlock()
		return errors.New("dictation is already active")
	}
	c.cancelWorkLocked()
	c.pending = ""
	cfg := profile.Settings
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
	c.targets[gen] = target
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
	var e error
	captureStarted := time.Now()
	if cfg.VADEnabled && (cfg.SilenceTrimming || cfg.AutoStopEnabled || cfg.SilenceSplitting) {
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
	go c.watchRecording(gen, recordingCtx, time.Duration(cfg.MaxDurationSeconds)*time.Second, interruptions, segmented)
	return nil
}

func (c *recorder) publishVADState(gen uint64, active bool) {
	state := VADSilence
	if active {
		state = VADSpeech
	}
	c.mu.Lock()
	if c.status.Generation != gen || c.status.State != Recording || c.status.VADState == state {
		c.mu.Unlock()
		return
	}
	c.status.VADState = state
	s := c.status
	c.mu.Unlock()
	c.publish(s)
}

func (c *recorder) publishAutoStopState(gen uint64, armed, countdown bool, remainingMilliseconds int) {
	state := AutoStopWaiting
	deadline := time.Time{}
	if armed {
		state = AutoStopListening
	}
	if countdown {
		state = AutoStopCountdown
		deadline = time.Now().UTC().Add(time.Duration(remainingMilliseconds) * time.Millisecond)
	}
	c.mu.Lock()
	if c.status.Generation != gen || c.status.State != Recording ||
		(c.status.AutoStopState == state && c.status.AutoStopDeadline.Equal(deadline)) {
		c.mu.Unlock()
		return
	}
	c.status.AutoStopState = state
	c.status.AutoStopDeadline = deadline
	s := c.status
	c.mu.Unlock()
	c.publish(s)
}

func (c *recorder) publishSegmentProgress(gen uint64, segment int, phase SegmentPhase) {
	c.mu.Lock()
	if c.status.Generation != gen || c.status.State != Recording {
		c.mu.Unlock()
		return
	}
	c.status.SegmentNumber = segment
	c.status.SegmentPhase = phase
	s := c.status
	c.mu.Unlock()
	c.publish(s)
}

func (c *recorder) watchRecording(gen uint64, ctx context.Context, limit time.Duration, interruptions <-chan error, segmented *segmentedRun) {
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case <-timer.C:
		_ = c.stop(gen, true, false)
	case <-segmentedAutoStop(segmented):
		c.logger.Info("dictation automatic stop triggered", "generation", gen)
		_ = c.stop(gen, false, true)
	case cause, ok := <-interruptions:
		if !ok {
			cause = audio.ErrDeviceInterrupted
		}
		c.captureInterrupted(gen, cause)
	case <-segmentedReady(segmented):
		result := segmented.wait()
		if result.err == nil {
			result.err = errors.New("silence-aware processor stopped unexpectedly")
		}
		c.captureInterrupted(gen, result.err)
	case <-ctx.Done():
	}
}

func segmentedReady(run *segmentedRun) <-chan struct{} {
	if run == nil {
		return nil
	}
	return run.ready
}

func segmentedAutoStop(run *segmentedRun) <-chan struct{} {
	if run == nil {
		return nil
	}
	return run.autoStop
}

func (c *recorder) captureInterrupted(gen uint64, cause error) {
	c.transition.Lock()
	c.mu.Lock()
	if c.status.Generation != gen || c.status.State != Recording {
		c.mu.Unlock()
		c.transition.Unlock()
		return
	}
	c.cancelWorkLocked()
	segmented := c.segmented
	c.segmented = nil
	c.mu.Unlock()

	cleanupErr := c.capture.Cancel(context.Background())
	if segmented != nil {
		result := segmented.wait()
		if cause == nil || errors.Is(cause, context.Canceled) {
			cause = result.err
		}
	}
	if cause == nil {
		cause = cleanupErr
	}
	if cause == nil {
		cause = audio.ErrDeviceInterrupted
	}

	c.mu.Lock()
	if c.status.Generation != gen || c.status.State != Recording {
		c.mu.Unlock()
		c.transition.Unlock()
		return
	}
	delete(c.targets, gen)
	delete(c.runProfiles, gen)
	delete(c.runDetails, gen)
	c.pending = ""
	c.status = Status{State: Failed, Generation: gen, Message: "Microphone: " + cause.Error()}
	s := c.status
	c.mu.Unlock()
	c.transition.Unlock()
	c.logger.Error("dictation capture interrupted", "generation", gen, "error_kind", diagnostics.ErrorKind(cause))
	c.publish(s)
}

func (c *recorder) stopCurrent() error {
	c.mu.Lock()
	g := c.status.Generation
	c.mu.Unlock()
	return c.stop(g, false, false)
}

func (c *recorder) stop(gen uint64, limit, automatic bool) error {
	work, err := c.stopCapture(gen, limit, automatic)
	if err != nil || work == nil {
		return err
	}
	if c.scheduleCompletion == nil {
		return c.completeStopped(work)
	}
	if c.scheduleCompletion(func() { _ = c.completeStopped(work) }) {
		return nil
	}
	err = errors.New("application is shutting down")
	c.fail(gen, err.Error())
	work.discard()
	return err
}

func (c *recorder) stopCapture(gen uint64, limit, automatic bool) (*stoppedRecording, error) {
	c.transition.Lock()
	c.mu.Lock()
	if c.status.Generation != gen || c.status.State != Recording {
		c.mu.Unlock()
		c.transition.Unlock()
		return nil, nil
	}
	c.cancelRecordingWatchLocked()
	c.status = Status{State: Transcribing, Generation: gen, StartedAt: c.status.StartedAt, Message: "Transcribing", CanCancel: true}
	s := c.status
	ctx := c.ctx
	segmented := c.segmented
	c.segmented = nil
	profile, hasProfile := c.runProfiles[gen]
	details := c.runDetails[gen]
	if !c.status.StartedAt.IsZero() {
		details.CaptureDurationMilliseconds = time.Since(c.status.StartedAt).Milliseconds()
	}
	details.AutoStopped = automatic
	delete(c.runProfiles, gen)
	delete(c.runDetails, gen)
	c.mu.Unlock()
	res, err := c.capture.Stop(context.Background())
	c.transition.Unlock()
	c.publish(s)
	if err != nil {
		profile.STTCredential = ""
		profile.PostProcessingCredential = ""
		c.logger.Error("dictation capture stop failed", "generation", gen, "error_kind", diagnostics.ErrorKind(err))
		c.fail(gen, "Microphone: "+err.Error())
		return nil, err
	}
	return &stoppedRecording{
		generation: gen,
		context:    ctx,
		result:     res,
		segmented:  segmented,
		profile:    profile,
		hasProfile: hasProfile,
		details:    details,
		limit:      limit || res.LimitReached,
		automatic:  automatic,
	}, nil
}

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
	if segmented != nil {
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
		transcription, transcriptionErr := c.client.Transcribe(requestCtx, cfg.BaseURL, cfg.Model, cfg.Language, profile.STTCredential, cfg.Headers, wav)
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
	if c.status.Generation != gen || c.status.State != Transcribing {
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
		if processor == nil {
			processingErr = errors.New("post-processing is unavailable")
		} else {
			processingResult, processingErr = processor.ProcessWithCredential(ctx, cfg.PostProcessing, text, profile.PostProcessingCredential)
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
	if c.status.Generation != gen || c.status.State != completionState {
		c.mu.Unlock()
		return nil
	}
	c.status.State = Ready
	c.status.Message = "Ready to insert"
	s = c.status
	c.mu.Unlock()
	c.publish(s)
	c.mu.Lock()
	if c.status.Generation != gen || c.status.State != Ready || ctx.Err() != nil {
		c.mu.Unlock()
		return nil
	}
	target := c.targets[gen]
	delete(c.targets, gen)
	deliveryMode := insertionMode(cfg.AutoInsert)
	e = c.policy.Deliver(ctx, target, text, deliveryMode)
	outcome := history.HistoryInserted
	if errors.Is(e, insertion.ErrCopyRequired) {
		c.pending = text
		message := "Transcript ready—copy required"
		if deliveryMode == insertion.ManualCopy {
			message = "Transcript ready to copy"
			e = nil
		}
		c.status = Status{State: Failed, Generation: gen, Message: message, CanCopy: true}
		outcome = history.HistoryCopyRequired
	} else if e != nil {
		c.pending = text
		c.status = Status{State: Failed, Generation: gen, Message: "Transcript ready—copy required", CanCopy: true}
		outcome = history.HistoryFailed
	} else if processingFallback {
		message := "Post-processing failed; raw transcript used"
		if processingFallbackKind == "timeout" {
			message = "Post-processing timed out; raw transcript used"
		}
		c.status = Status{State: Idle, Generation: gen, Message: message}
	} else if automatic {
		c.status = Status{State: Idle, Generation: gen, Message: "Silence detected; dictation completed"}
	} else if limit {
		c.status = Status{State: Idle, Generation: gen, Message: "Maximum duration reached; dictation completed"}
	} else {
		c.status = Status{State: Idle, Generation: gen}
	}
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

func (c *recorder) cancelRecording() error {
	c.transition.Lock()
	defer c.transition.Unlock()
	c.mu.Lock()
	if c.status.State == Idle {
		c.cancelWorkLocked()
		c.mu.Unlock()
		return nil
	}
	activeGeneration := c.status.Generation
	startedAt := c.status.StartedAt
	c.generation++
	clear(c.targets)
	clear(c.runProfiles)
	clear(c.runDetails)
	c.pending = ""
	c.cancelWorkLocked()
	segmented := c.segmented
	c.segmented = nil
	c.status = Status{State: Cancelling, Generation: c.generation, Message: "Cancelling"}
	s := c.status
	c.mu.Unlock()
	c.logger.Info("dictation cancellation requested", "generation", activeGeneration)
	c.publish(s)
	_ = c.capture.Cancel(context.Background())
	if segmented != nil {
		_ = segmented.wait()
	}
	c.mu.Lock()
	c.status = Status{State: Idle, Generation: c.generation}
	s = c.status
	c.mu.Unlock()
	c.publish(s)
	durationMilliseconds := int64(0)
	if !startedAt.IsZero() {
		durationMilliseconds = time.Since(startedAt).Milliseconds()
	}
	c.logger.Info("dictation cancelled", "generation", activeGeneration, "duration_ms", durationMilliseconds, "outcome", "cancelled")
	return nil
}

func (c *recorder) copyPending() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("application is shutting down")
	}
	text := c.pending
	gen := c.status.Generation
	if text == "" || !c.status.CanCopy {
		c.mu.Unlock()
		return errors.New("no transcript is waiting to be copied")
	}
	if e := c.targetPlatform.Copy(context.Background(), text); e != nil {
		c.mu.Unlock()
		return e
	}
	if c.status.Generation == gen && c.pending == text {
		c.pending = ""
		c.status = Status{State: Idle, Generation: gen, Message: "Transcript copied"}
		s := c.status
		c.mu.Unlock()
		c.publish(s)
		return nil
	}
	c.mu.Unlock()
	return errors.New("pending transcript changed before copy completed")
}

func (c *recorder) fail(gen uint64, msg string) {
	c.mu.Lock()
	if c.status.Generation == gen {
		c.cancelWorkLocked()
		delete(c.targets, gen)
		delete(c.runProfiles, gen)
		delete(c.runDetails, gen)
		c.status = Status{State: Failed, Generation: gen, Message: msg}
		s := c.status
		c.mu.Unlock()
		c.publish(s)
		return
	}
	c.mu.Unlock()
}
func (c *recorder) close() error {
	c.mu.Lock()
	c.closed = true
	c.pending = ""
	clear(c.targets)
	clear(c.runProfiles)
	clear(c.runDetails)
	c.mu.Unlock()
	_ = c.cancelRecording()
	return c.capture.Close()
}
