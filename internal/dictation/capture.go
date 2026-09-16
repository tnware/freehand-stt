package dictation

import (
	"context"
	"errors"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/realtime"
	"github.com/tnware/freehand-stt/internal/settings"
)

type stoppedRecording struct {
	realtime   *realtime.Session
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
	w.profile.VoiceCredential = ""
	w.profile.PostProcessingCredential = ""
}

func (w *stoppedRecording) discard() {
	if w.realtime != nil {
		_ = w.realtime.Wait()
	}
	w.clearCredentials()
	for index := range w.result.PCM {
		w.result.PCM[index] = 0
	}
	if w.segmented != nil {
		_ = w.segmented.wait()
	}
}

func (c *recorder) watchRecording(gen uint64, ctx context.Context, limit time.Duration, interruptions <-chan error, segmented *segmentedRun, live *realtime.Session) {
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
	case <-realtimeFailed(live):
		c.captureInterrupted(gen, errors.New("live transcription connection failed"))
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
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Recording {
		c.mu.Unlock()
		c.transition.Unlock()
		return
	}
	c.cancelWorkLocked()
	segmented := c.segmented
	c.segmented = nil
	live := c.realtime
	c.realtime = nil
	c.mu.Unlock()

	cleanupErr := c.capture.Cancel(context.Background())
	if live != nil {
		_ = live.Wait()
	}
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
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Recording {
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
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Recording {
		c.mu.Unlock()
		c.transition.Unlock()
		return nil, nil
	}
	c.cancelRecordingWatchLocked()
	previous := c.status
	c.status = Status{State: Transcribing, Generation: gen, Message: "Transcribing", CanCancel: true,
		Live: previous.Live, LiveCaptions: previous.LiveCaptions, LiveFinal: previous.LiveFinal, LivePartial: previous.LivePartial}

	s := c.status
	ctx := c.ctx
	segmented := c.segmented
	c.segmented = nil
	live := c.realtime
	c.realtime = nil
	profile, hasProfile := c.runProfiles[gen]
	details := c.runDetails[gen]
	if !previous.StartedAt.IsZero() {
		details.CaptureDurationMilliseconds = time.Since(previous.StartedAt).Milliseconds()
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
		profile.VoiceCredential = ""
		profile.PostProcessingCredential = ""
		c.logger.Error("dictation capture stop failed", "generation", gen, "error_kind", diagnostics.ErrorKind(err))
		c.fail(gen, "Microphone: "+err.Error())
		if live != nil {
			_ = live.Wait()
		}
		return nil, err
	}
	return &stoppedRecording{
		generation: gen,
		context:    ctx,
		result:     res,
		segmented:  segmented,
		realtime:   live,
		profile:    profile,
		hasProfile: hasProfile,
		details:    details,
		limit:      limit || res.LimitReached,
		automatic:  automatic,
	}, nil
}
