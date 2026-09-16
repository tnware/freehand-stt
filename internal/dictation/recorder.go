package dictation

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/realtime"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/tnware/freehand-stt/internal/webrtcvad"
)

type recorder struct {
	realtime           *realtime.Session
	mu                 sync.Mutex
	transition         sync.Mutex
	status             Status
	generation         uint64
	cancelEpoch        uint64 // Fences starts still preparing their run context.
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
	closed             atomic.Bool
	targets            map[uint64]capturedTarget
	pending            string
	runProfiles        map[uint64]settings.RequestProfile
	runDetails         map[uint64]history.HistoryRunDetails
	segmented          *segmentedRun
	scheduleCompletion func(func()) bool
	newDetector        func(config.VADMode) (audio.VoiceDetector, error)
	logger             *slog.Logger
}

// The bounded capture reason has exactly the target's generation and lifetime.
// Deleting a run's target on any terminal path also discards its diagnostic.
type capturedTarget struct {
	target    insertion.Target
	rejection error
}

func newRecorder(cap audio.Capture, p insertion.Platform, client *inference.Client, processor *postprocess.Processor, source settings.Source, profiles settings.ProfileSource, store *history.Store, changed func(Status), logger *slog.Logger) *recorder {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &recorder{status: Status{State: Idle}, rootContext: context.Background(), capture: cap, targetPlatform: p, policy: insertion.Policy{Platform: p}, client: client, processor: processor, settings: source, profiles: profiles, history: store, changed: changed, targets: make(map[uint64]capturedTarget), runProfiles: make(map[uint64]settings.RequestProfile), runDetails: make(map[uint64]history.HistoryRunDetails), logger: logger, newDetector: func(mode config.VADMode) (audio.VoiceDetector, error) {
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
	if !c.closed.Load() {
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
func (c *recorder) cancelRecording() error {
	// Signal and fence this run before waiting for startup/stop to relinquish
	// device ownership. In particular, microphone authorization needs this
	// context cancellation to return from Start.
	c.mu.Lock()
	c.cancelEpoch++
	activeGeneration := c.status.Generation
	startedAt := c.status.StartedAt
	c.cancelWorkLocked()
	if c.status.State != Idle && c.status.State != Cancelling {
		c.generation++
		clear(c.targets)
		clear(c.runProfiles)
		clear(c.runDetails)
		c.pending = ""
		c.status = Status{State: Cancelling, Generation: c.generation, Message: "Cancelling"}
	} else if c.closed.Load() && c.status.State == Idle {
		c.status = Status{State: Idle, Generation: c.generation}
	}
	s := c.status
	c.mu.Unlock()

	c.transition.Lock()
	defer c.transition.Unlock()
	c.mu.Lock()
	// Another canceller may already have cleaned up, and a subsequent start
	// may own the device. Never redirect an old cancellation to that run.
	if s.State != Cancelling || c.status.Generation != s.Generation || c.status.State != Cancelling {
		c.mu.Unlock()
		return nil
	}
	// Startup can attach these workers after cancellation is signalled. Take
	// them only after acquiring transition, without overlapping native calls.
	segmented := c.segmented
	c.segmented = nil
	live := c.realtime
	c.realtime = nil
	c.mu.Unlock()
	c.logger.Info("dictation cancellation requested", "generation", activeGeneration)
	c.publish(s)
	cancelErr := c.capture.Cancel(context.Background())
	if live != nil {
		_ = live.Wait()
	}
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
	// Interactive cancellation remains a best-effort reset after device loss.
	// Shutdown reports cleanup failures to the lifecycle owner.
	if c.closed.Load() {
		return cancelErr
	}
	return nil
}

func (c *recorder) fail(gen uint64, msg string) {
	c.mu.Lock()
	if !c.closed.Load() && c.status.Generation == gen {
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
	c.closed.Store(true)
	c.mu.Lock()
	c.cancelWorkLocked()
	c.pending = ""
	clear(c.targets)
	clear(c.runProfiles)
	clear(c.runDetails)
	c.mu.Unlock()
	cancelErr := c.cancelRecording()
	return errors.Join(cancelErr, c.capture.Close())
}
