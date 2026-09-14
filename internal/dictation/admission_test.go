package dictation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/settings"
)

type admissionEvents struct {
	mu     sync.Mutex
	values []Status
}

func (e *admissionEvents) changed(status Status) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.values = append(e.values, status)
}

func (e *admissionEvents) snapshot() []Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Status(nil), e.values...)
}

func TestProfileAdmissionFailurePublishesOnceAndRecovers(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		for _, managed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/managed=%t", mode, managed), func(t *testing.T) {
				service, platform := newCompletionService("http://127.0.0.1/v1")
				capture := &countingStartCapture{}
				service.recorder.capture = capture
				events := &admissionEvents{}
				service.recorder.changed = events.changed
				var logs bytes.Buffer
				service.recorder.logger = slog.New(slog.NewTextHandler(&logs, nil))
				cause := errors.New("https://secret.example/private?token=api-secret\n" + strings.Repeat("secret", 1000))
				if managed {
					cause = fmt.Errorf("%w: %v", settings.ErrManagedUnavailable, cause)
				}
				originalProfiles := service.recorder.profiles
				service.recorder.profiles = func() (settings.RequestProfile, error) { return settings.RequestProfile{}, cause }
				cancel := startCompletionService(t, service)
				defer cancel()
				defer service.ServiceShutdown()
				for attempt := 1; attempt <= 2; attempt++ {
					if err := service.StartRecording(mode); err != cause {
						t.Fatal("start did not preserve the original API error")
					}
					status := service.CurrentStatus()
					published := events.snapshot()
					if status.State != Failed || !status.StartRejected || len(published) != attempt || published[attempt-1] != status {
						t.Fatalf("rejected start did not publish exactly once: status=%+v events=%+v", status, published)
					}
					if status.Message == "" || len(status.Message) > 256 || strings.Contains(status.Message+logs.String(), "secret") || strings.Contains(status.Message, "\n") {
						t.Fatal("admission feedback missing, unbounded, or contains private error details")
					}
					if managed && !strings.Contains(strings.ToLower(status.Message), "managed") {
						t.Fatalf("managed failure lacks recovery guidance: %q", status.Message)
					}
					if status.Generation != 0 || status.CanCancel || status.CanCopy || Active(service) || capture.starts.Load() != 0 || platform.insertion() != "" {
						t.Fatalf("failed admission started work: %+v", status)
					}
					if err := service.StopRecording(); err != nil {
						t.Fatal(err)
					}
					if len(events.snapshot()) != attempt {
						t.Fatal("hold release/stop duplicated failure")
					}
				}
				service.recorder.profiles = originalProfiles
				if err := service.StartRecording(mode); err != nil {
					t.Fatal(err)
				}
				if status := service.CurrentStatus(); status.State != Recording || status.RecordingMode != mode || status.Generation != 1 || capture.starts.Load() != 1 {
					t.Fatalf("retry did not recover: %+v", status)
				}
				if err := service.Cancel(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestServiceAdmissionFailurePublishesWithoutStartingWork(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		for _, admission := range []string{"setup", "settings", "file-active", "playback"} {
			t.Run(string(mode)+"/"+admission, func(t *testing.T) {
				service, platform := newCompletionService("http://127.0.0.1/v1")
				capture := &countingStartCapture{}
				service.recorder.capture = capture
				events := &admissionEvents{}
				service.recorder.changed = events.changed
				cancel := startCompletionService(t, service)
				defer cancel()
				defer service.ServiceShutdown()
				profileCalls := 0
				service.recorder.profiles = func() (settings.RequestProfile, error) {
					profileCalls++
					return settings.RequestProfile{}, errors.New("profile must not be captured")
				}
				switch admission {
				case "setup":
					service.settings = nil
				case "settings":
					service.settings = func() config.Settings { return config.Default() }
				case "file-active":
					service.activity = activity.New(activity.Sources{FileActive: func() bool { return true }})
				case "playback":
					service.activity = activity.New(activity.Sources{StopPlayback: func() error { return errors.New("private player details") }})
				}
				if err := service.StartRecording(mode); err == nil {
					t.Fatal("admission unexpectedly succeeded")
				}
				status := service.CurrentStatus()
				if got := events.snapshot(); len(got) != 1 || got[0] != status || status.State != Failed || status.Message == "" || len(status.Message) > 256 {
					t.Fatalf("missing bounded admission feedback: status=%+v events=%+v", status, got)
				}
				if profileCalls != 0 || capture.starts.Load() != 0 || platform.insertion() != "" || Active(service) {
					t.Fatal("admission failure authorized work")
				}
			})
		}
	}
}

func TestCancelFencesServiceAdmissionBeforeRecorderStart(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		service, _ := newCompletionService("http://127.0.0.1/v1")
		capture := &countingStartCapture{}
		service.recorder.capture = capture
		cancel := startCompletionService(t, service)
		defer cancel()
		defer service.ServiceShutdown()
		entered, release := make(chan struct{}), make(chan struct{})
		service.activity = activity.New(activity.Sources{StopPlayback: func() error {
			close(entered)
			<-release
			return nil
		}})
		done := make(chan error, 1)
		go func() { done <- service.StartRecording(mode) }()
		awaitBoundary(t, entered)
		if err := service.Cancel(); err != nil {
			t.Fatal(err)
		}
		close(release)
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled admission returned %v", err)
		}
		if capture.starts.Load() != 0 || service.CurrentStatus().State != Idle {
			t.Fatal("cancelled admission began recording")
		}
	}
}
