package dictation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/settings"
)

type admissionPlatform struct {
	completionPlatform
	targetCalls atomic.Int32
	copyCalls   atomic.Int32
}

func (p *admissionPlatform) CaptureTarget() (insertion.Target, error) {
	p.targetCalls.Add(1)
	return validTarget(), nil
}

func (p *admissionPlatform) Copy(context.Context, string) error {
	p.copyCalls.Add(1)
	return nil
}

func TestAdmissionFailurePreservesCompletedResultAndCopyRecovery(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		for _, pending := range []bool{false, true} {
			for _, boundary := range []string{"profile", "settings"} {
				t.Run(fmt.Sprintf("%s/pending=%t/%s", mode, pending, boundary), func(t *testing.T) {
					var requests atomic.Int32
					client := &inference.Client{HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						requests.Add(1)
						return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"text":"Recoverable transcript"}`))}, nil
					})}}
					service, _ := newCompletionService("http://127.0.0.1/v1", client)
					platform := &admissionPlatform{}
					service.recorder.targetPlatform = platform
					service.recorder.policy.Platform = platform
					cfg := service.settings.Current()
					cfg.AutoInsert = !pending
					service.settings = func() config.Settings { return cfg }
					service.recorder.profiles = func() (settings.RequestProfile, error) { return settings.RequestProfile{Settings: cfg}, nil }
					goodSettings, goodProfiles := service.settings, service.recorder.profiles
					events := &admissionEvents{}
					completed := make(chan struct{})
					var once sync.Once
					service.recorder.changed = func(status Status) {
						events.changed(status)
						if status.Transcript != "" {
							once.Do(func() { close(completed) })
						}
					}
					cancel := startCompletionService(t, service)
					defer cancel()
					defer service.ServiceShutdown()
					if err := service.StartRecording(mode); err != nil {
						t.Fatal(err)
					}
					if err := service.StopRecording(); err != nil {
						t.Fatal(err)
					}
					awaitBoundary(t, completed)
					previous := service.CurrentStatus()
					if previous.Transcript != "Recoverable transcript" || previous.CanCopy != pending {
						t.Fatalf("bad completed fixture: %+v", previous)
					}
					insertion := platform.insertion()
					if boundary == "profile" {
						service.recorder.profiles = func() (settings.RequestProfile, error) {
							return settings.RequestProfile{}, settings.ErrManagedUnavailable
						}
					} else {
						service.settings = func() config.Settings { return config.Default() }
					}
					for range 2 {
						before := len(events.snapshot())
						if err := service.StartRecording(mode); err == nil {
							t.Fatal("bad admission succeeded")
						}
						status := service.CurrentStatus()
						if status.State != Failed || !status.StartRejected || status.Transcript != previous.Transcript || status.Generation != previous.Generation || status.CanCopy != previous.CanCopy || status.CanCancel || len(events.snapshot()) != before+1 {
							t.Fatalf("rejection damaged result or duplicate publication: %+v", status)
						}
						if platform.targetCalls.Load() != 1 || platform.copyCalls.Load() != 0 || platform.insertion() != insertion || requests.Load() != 1 {
							t.Fatal("rejection captured a target, copied, inserted, or invoked inference")
						}
					}
					if text, err := PlaybackTranscript(service, previous.Generation); err != nil || text != previous.Transcript {
						t.Fatalf("result playback capability lost: %q %v", text, err)
					}
					if err := service.CopyCurrent(previous.Generation); err != nil {
						t.Fatal(err)
					}
					if pending {
						if err := service.CopyPending(); err != nil {
							t.Fatal(err)
						}
						if status := service.CurrentStatus(); status.CanCopy || status.Transcript != previous.Transcript || status.Generation != previous.Generation {
							t.Fatal("explicit pending copy lost the result")
						}
					}
					service.settings, service.recorder.profiles = goodSettings, goodProfiles
					if err := service.StartRecording(mode); err != nil {
						t.Fatal(err)
					}
					if status := service.CurrentStatus(); status.State != Recording || status.StartRejected || status.Generation <= previous.Generation || status.Transcript != "" || status.CanCopy {
						t.Fatalf("successful retry did not replace result: %+v", status)
					}
					if err := service.CopyCurrent(previous.Generation); err == nil {
						t.Fatal("old generation still copyable after actual start")
					}
					if err := service.Cancel(); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestProfileAdmissionFailureIsFencedByCancellationAndShutdown(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		for _, fence := range []string{"cancel", "shutdown", "root-context", "source-cancelled"} {
			t.Run(string(mode)+"/"+fence, func(t *testing.T) {
				service, _ := newCompletionService("http://127.0.0.1/v1")
				capture := &countingStartCapture{}
				platform := &admissionPlatform{}
				service.recorder.capture = capture
				service.recorder.targetPlatform = platform
				events := &admissionEvents{}
				service.recorder.changed = events.changed
				cancelRoot := startCompletionService(t, service)
				defer cancelRoot()
				defer service.ServiceShutdown()
				entered, release := make(chan struct{}), make(chan struct{})
				cause := error(settings.ErrManagedUnavailable)
				if fence == "source-cancelled" {
					cause = fmt.Errorf("profile: %w", context.Canceled)
				}
				service.recorder.profiles = func() (settings.RequestProfile, error) {
					close(entered)
					<-release
					return settings.RequestProfile{}, cause
				}
				done := make(chan error, 1)
				go func() { done <- service.StartRecording(mode) }()
				awaitBoundary(t, entered)
				var cancelDone chan error
				switch fence {
				case "cancel":
					cancelDone = make(chan error, 1)
					go func() { cancelDone <- service.Cancel() }()
					waitForCancelEpoch(t, &testRecorder{recorder: service.recorder}, 1)
				case "shutdown":
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					_ = service.shutdown(ctx)
				case "root-context":
					cancelRoot()
				}
				close(release)
				if err := <-done; !errors.Is(err, cause) {
					t.Fatalf("original admission error changed: %v", err)
				}
				if cancelDone != nil {
					if err := <-cancelDone; err != nil {
						t.Fatal(err)
					}
				}
				if fence == "shutdown" {
					if err := service.ServiceShutdown(); err != nil {
						t.Fatal(err)
					}
				}
				if status := service.CurrentStatus(); status.State != Idle || len(events.snapshot()) != 0 || capture.starts.Load() != 0 || platform.targetCalls.Load() != 0 || platform.copyCalls.Load() != 0 || platform.insertion() != "" {
					t.Fatalf("fenced admission published or started work: status=%+v events=%+v", status, events.snapshot())
				}
			})
		}
	}
}
