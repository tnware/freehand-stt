package dictation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/inference"
)

type shutdownCapture struct {
	completionCapture
	entered, release chan struct{}
	block            string
	closes           atomic.Int32
}

func (c *shutdownCapture) Cancel(context.Context) error {
	if c.block == "Cancel" {
		close(c.entered)
		<-c.release
	}
	return nil
}
func (c *shutdownCapture) Close() error {
	if c.block == "Close" {
		close(c.entered)
		<-c.release
	}
	c.closes.Add(1)
	return nil
}
func (c *shutdownCapture) Start(ctx context.Context, _ string, _ int) (<-chan error, error) {
	if c.block == "Start" {
		close(c.entered)
		<-c.release
	}
	return make(chan error), nil
}
func awaitBoundary(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("boundary not reached")
	}
}
func awaitDictationShutdown(t *testing.T, done <-chan error, want error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Fatalf("shutdown = %v, want %v", err, want)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown ignored its wait deadline")
	}
}

func TestShutdownDeadlineIncludesCaptureTeardown(t *testing.T) {
	for _, stage := range []string{"Cancel", "Close"} {
		t.Run(stage, func(t *testing.T) {
			service, _ := newCompletionService("https://fixture.invalid/v1")
			capture := &shutdownCapture{entered: make(chan struct{}), release: make(chan struct{}), block: stage}
			service.recorder.capture = capture
			defer startCompletionService(t, service)()
			if err := service.StartRecording(RecordingToggle); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- service.shutdown(ctx) }()
			awaitBoundary(t, capture.entered)
			awaitBoundary(t, service.recorder.rootContext.Done())
			cancel()
			awaitDictationShutdown(t, done, context.Canceled)
			if err := service.StartRecording(RecordingToggle); err == nil {
				t.Fatal("closed service admitted capture")
			}
			if capture.closes.Load() != 0 {
				t.Fatal("capture freed before in-flight call finished")
			}
			close(capture.release)
			awaitBoundary(t, service.shutdownDone)
			if capture.closes.Load() != 1 {
				t.Fatal("capture not released")
			}
			if err := service.ServiceShutdown(); err != nil {
				t.Fatal(err)
			}
			if capture.closes.Load() != 1 {
				t.Fatal("capture closed twice")
			}
		})
	}
}

func TestShutdownCancelsWhileCaptureStartHoldsTransition(t *testing.T) {
	service, _ := newCompletionService("https://fixture.invalid/v1")
	capture := &shutdownCapture{entered: make(chan struct{}), release: make(chan struct{}), block: "Start"}
	service.recorder.capture = capture
	defer startCompletionService(t, service)()
	started := make(chan error, 1)
	go func() { started <- service.StartRecording(RecordingToggle) }()
	awaitBoundary(t, capture.entered)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitBoundary(t, service.recorder.rootContext.Done())
	cancel()
	awaitDictationShutdown(t, done, context.Canceled)
	close(capture.release)
	if err := <-started; err == nil {
		t.Fatal("late microphone startup succeeded")
	}
	awaitBoundary(t, service.shutdownDone)
	if capture.closes.Load() != 1 {
		t.Fatal("late capture was not released")
	}
}

func TestShutdownFencesTranscriptionThatReturnsAfterCancellation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	client := inference.New()
	client.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		close(entered)
		<-release
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"text":"late transcript"}`)), Request: r}, nil
	})
	service, platform := newCompletionService("https://fixture.invalid/v1", client)
	var publications atomic.Int32
	service.recorder.changed = func(Status) { publications.Add(1) }
	defer startCompletionService(t, service)()
	if err := service.StartRecording(RecordingToggle); err != nil {
		t.Fatal(err)
	}
	if err := service.StopRecording(); err != nil {
		t.Fatal(err)
	}
	awaitBoundary(t, entered)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitBoundary(t, service.recorder.rootContext.Done())
	count := publications.Load()
	cancel()
	awaitDictationShutdown(t, done, context.Canceled)
	close(release)
	awaitBoundary(t, service.shutdownDone)
	if platform.insertion() != "" || publications.Load() != count {
		t.Fatal("late transcription was inserted or published")
	}
	if service.CurrentStatus().Transcript != "" {
		t.Fatal("closed dictation retained a current result")
	}
}

// Keep the production ServiceShutdown wrapper covered in addition to the
// deterministic cancellation controls above.
func TestServiceShutdownBudgetIncludesNativeClose(t *testing.T) {
	service, _ := newCompletionService("https://fixture.invalid/v1")
	capture := &shutdownCapture{entered: make(chan struct{}), release: make(chan struct{}), block: "Close"}
	service.recorder.capture = capture
	defer startCompletionService(t, service)()
	done := make(chan error, 1)
	go func() { done <- service.ServiceShutdown() }()
	awaitBoundary(t, capture.entered)
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown = %v", err)
		}
	case <-time.After(6 * time.Second):
		close(capture.release)
		t.Fatal("native Close escaped the service budget")
	}
	close(capture.release)
	awaitBoundary(t, service.shutdownDone)
}

var _ audio.Capture = (*shutdownCapture)(nil)

type failingShutdownCapture struct {
	completionCapture
	cancelErr, closeErr error
}

func (c *failingShutdownCapture) Cancel(context.Context) error { return c.cancelErr }
func (c *failingShutdownCapture) Close() error                 { return c.closeErr }
func TestShutdownPreservesCaptureCleanupErrors(t *testing.T) {
	service, _ := newCompletionService("https://fixture.invalid/v1")
	cancelErr, closeErr := errors.New("cancel failed"), errors.New("close failed")
	service.recorder.capture = &failingShutdownCapture{cancelErr: cancelErr, closeErr: closeErr}
	defer startCompletionService(t, service)()
	if err := service.StartRecording(RecordingToggle); err != nil {
		t.Fatal(err)
	}
	err := service.ServiceShutdown()
	if !errors.Is(err, cancelErr) || !errors.Is(err, closeErr) {
		t.Fatalf("lost native cleanup error: %v", err)
	}
}
