package dictation

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/insertion"
)

// Authorization waits for the recording context, not a test-controlled release.
// Cancel must signal it before waiting for serialized device cleanup.
type authorizationCapture struct {
	capFake
	entered        chan struct{}
	starting       atomic.Bool
	cleanupOverlap atomic.Bool
	cancels        atomic.Int32
}

func (c *authorizationCapture) Start(ctx context.Context, _ string, _ int) (<-chan error, error) {
	c.starting.Store(true)
	defer c.starting.Store(false)
	close(c.entered)
	<-ctx.Done()
	return nil, ctx.Err()
}

func (c *authorizationCapture) Cancel(context.Context) error {
	if c.starting.Load() {
		c.cleanupOverlap.Store(true)
	}
	c.cancels.Add(1)
	return nil
}

func TestCancelInterruptsCaptureAuthorization(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		t.Run(string(mode), func(t *testing.T) {
			capture := &authorizationCapture{entered: make(chan struct{})}
			platform := &platFake{}
			c := New(capture, platform, nil, nil, settingsFake{}, nil)
			root, release := context.WithCancel(context.Background())
			defer release() // Only a failure safety net, never the successful release.
			c.setRootContext(root)
			startDone := make(chan error, 1)
			go func() { startDone <- c.StartWithMode(mode) }()
			awaitBoundary(t, capture.entered)
			generation := c.Status().Generation
			cancelDone := make(chan error, 1)
			go func() { cancelDone <- c.Cancel() }()
			select {
			case err := <-cancelDone:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("Cancel did not interrupt microphone authorization")
			}
			if err := <-startDone; !errors.Is(err, context.Canceled) {
				t.Fatalf("Start = %v, want context cancellation", err)
			}
			if capture.cleanupOverlap.Load() || capture.cancels.Load() != 1 {
				t.Fatalf("cleanup overlap = %v, cancels = %d", capture.cleanupOverlap.Load(), capture.cancels.Load())
			}
			if status := c.Status(); status.State != Idle || status.Generation <= generation || status.Transcript != "" || status.CanCopy {
				t.Fatalf("cancelled status = %+v", status)
			}
			c.mu.Lock()
			retained := len(c.targets) + len(c.runProfiles) + len(c.runDetails) + len(c.pending)
			c.mu.Unlock()
			if retained != 0 || platform.inserts != 0 || platform.copies != 0 {
				t.Fatal("cancellation retained or delivered recording data")
			}
		})
	}
}

type preparingPlatform struct {
	platFake
	entered, release chan struct{}
	calls            atomic.Int32
}

func (p *preparingPlatform) CaptureTarget() (insertion.Target, error) {
	if p.calls.Add(1) == 1 {
		close(p.entered)
		<-p.release
	}
	return validTarget(), nil
}

type countingStartCapture struct {
	capFake
	starts atomic.Int32
}

func (c *countingStartCapture) Start(context.Context, string, int) (<-chan error, error) {
	c.starts.Add(1)
	return nil, nil
}

func TestCancelFencesStartBeforeRunPublication(t *testing.T) {
	capture := &countingStartCapture{}
	platform := &preparingPlatform{entered: make(chan struct{}), release: make(chan struct{})}
	c := New(capture, platform, nil, nil, settingsFake{}, nil)
	// Observe Cancel's signal while preparation still owns transition, before
	// the new recording context/status have been installed.
	c.ctx, c.cancel = context.WithCancel(context.Background())
	oldContext := c.ctx
	startDone := make(chan error, 1)
	go func() { startDone <- c.Start() }()
	awaitBoundary(t, platform.entered)
	cancelDone := make(chan error, 1)
	go func() { cancelDone <- c.Cancel() }()
	awaitBoundary(t, oldContext.Done())
	close(platform.release)
	if err := <-startDone; !errors.Is(err, context.Canceled) {
		t.Errorf("pre-publication Start = %v, want context cancellation", err)
	}
	if err := <-cancelDone; err != nil {
		t.Fatal(err)
	}
	if capture.starts.Load() != 0 || c.Status().State != Idle {
		t.Fatalf("cancelled preparation opened capture: starts = %d, status = %+v", capture.starts.Load(), c.Status())
	}
	if err := c.StartWithMode(RecordingHold); err != nil {
		t.Fatal(err)
	}
	defer c.Cancel()
	if capture.starts.Load() != 1 || c.Status().RecordingMode != RecordingHold {
		t.Fatal("cancellation poisoned the next recording")
	}
}

func waitForCancelEpoch(t *testing.T, c *testRecorder, want uint64) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		c.mu.Lock()
		epoch := c.cancelEpoch
		c.mu.Unlock()
		if epoch == want {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("cancellation epoch = %d, want %d", epoch, want)
		case <-ticker.C:
		}
	}
}

func TestConcurrentCancellationsCleanUpOnce(t *testing.T) {
	capture := &authorizationCapture{}
	c := New(capture, &platFake{}, nil, nil, settingsFake{}, nil)
	// Hold native ownership while every caller signals the same existing run.
	c.mu.Lock()
	c.generation = 1
	c.status = Status{State: Recording, Generation: 1, CanCancel: true}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()
	c.transition.Lock()
	done := make(chan error, 8)
	for range 8 {
		go func() { done <- c.Cancel() }()
	}
	waitForCancelEpoch(t, c, 8)
	c.transition.Unlock()
	for range 8 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if capture.cancels.Load() != 1 || c.Status().State != Idle {
		t.Fatalf("cancels = %d, status = %+v", capture.cancels.Load(), c.Status())
	}
}

func TestQueuedCancellationDoesNotTouchNewerRun(t *testing.T) {
	capture := &authorizationCapture{}
	c := New(capture, &platFake{}, nil, nil, settingsFake{}, nil)
	c.generation = 1
	c.status = Status{State: Recording, Generation: 1, CanCancel: true}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	oldContext := c.ctx
	c.transition.Lock()
	done := make(chan error, 1)
	go func() { done <- c.Cancel() }()
	awaitBoundary(t, oldContext.Done())
	// Model another canceller completing cleanup and a new Start acquiring
	// transition before this queued caller. Mutex waiters have no FIFO contract.
	c.mu.Lock()
	c.generation++
	generation := c.generation
	c.status = Status{State: Recording, Generation: generation, CanCancel: true}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	newContext := c.ctx
	defer c.cancel()
	c.pending = "new run data"
	c.mu.Unlock()
	c.transition.Unlock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if newContext.Err() != nil || capture.cancels.Load() != 0 || c.Status().Generation != generation || c.Status().State != Recording {
		t.Fatal("queued cancellation was redirected to the newer run")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending != "new run data" {
		t.Fatal("stale cancellation erased newer run data")
	}
}

var _ audio.Capture = (*authorizationCapture)(nil)
