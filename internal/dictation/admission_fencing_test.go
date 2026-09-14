package dictation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
)

func TestAdmissionRejectionDoesNotFailAnActiveDictation(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		for _, phase := range []State{Recording, Transcribing} {
			t.Run(string(mode)+"/"+string(phase), func(t *testing.T) {
				requestEntered := make(chan struct{})
				client := &inference.Client{HTTP: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					close(requestEntered)
					<-request.Context().Done()
					return nil, request.Context().Err()
				})}}
				service, platform := newCompletionService("http://127.0.0.1/v1", client)
				events := &admissionEvents{}
				service.recorder.changed = events.changed
				cancel := startCompletionService(t, service)
				defer cancel()
				defer service.ServiceShutdown()
				if err := service.StartRecording(mode); err != nil {
					t.Fatal(err)
				}
				if phase == Transcribing {
					if err := service.StopRecording(); err != nil {
						t.Fatal(err)
					}
					awaitBoundary(t, requestEntered)
				}
				previous, eventCount := service.CurrentStatus(), len(events.snapshot())
				goodSettings := service.settings
				service.settings = func() config.Settings { return config.Default() }
				if err := service.StartRecording(mode); err == nil {
					t.Fatal("invalid settings admitted")
				}
				service.settings = goodSettings
				profileCalls := 0
				service.recorder.profiles = func() (settings.RequestProfile, error) {
					profileCalls++
					return settings.RequestProfile{}, settings.ErrManagedUnavailable
				}
				if err := service.StartRecording(mode); err == nil {
					t.Fatal("duplicate start admitted")
				}
				if status := service.CurrentStatus(); status != previous || len(events.snapshot()) != eventCount || !Active(service) || profileCalls != 0 || platform.insertion() != "" {
					t.Fatalf("rejected start damaged active job: previous=%+v status=%+v", previous, status)
				}
				service.recorder.mu.Lock()
				contextErr := service.recorder.ctx.Err()
				service.recorder.mu.Unlock()
				if contextErr != nil {
					t.Fatal("rejected start cancelled active work")
				}
				if err := service.Cancel(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestDelayedServiceAdmissionFailureDoesNotOverwriteNewerResult(t *testing.T) {
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		t.Run(string(mode), func(t *testing.T) {
			client := &inference.Client{HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"text":"Newer result"}`))}, nil
			})}}
			service, _ := newCompletionService("http://127.0.0.1/v1", client)
			cancel := startCompletionService(t, service)
			defer cancel()
			defer service.ServiceShutdown()
			entered, release, completed := make(chan struct{}), make(chan struct{}), make(chan struct{})
			events := &admissionEvents{}
			service.recorder.changed = func(status Status) {
				events.changed(status)
				if status.Transcript != "" {
					close(completed)
				}
			}
			cfg := service.settings.Current()
			var calls atomic.Int32
			service.settings = func() config.Settings {
				if calls.Add(1) == 1 {
					close(entered)
					<-release
					return config.Default()
				}
				return cfg
			}
			done := make(chan error, 1)
			go func() { done <- service.StartRecording(mode) }()
			awaitBoundary(t, entered)
			if err := service.StartRecording(mode); err != nil {
				t.Fatal(err)
			}
			if err := service.StopRecording(); err != nil {
				t.Fatal(err)
			}
			awaitBoundary(t, completed)
			previous, eventCount := service.CurrentStatus(), len(events.snapshot())
			close(release)
			if err := <-done; err == nil {
				t.Fatal("late invalid settings accepted")
			}
			if status := service.CurrentStatus(); status != previous || len(events.snapshot()) != eventCount || status.Transcript != "Newer result" {
				t.Fatalf("stale rejection overwrote newer result: %+v", status)
			}
		})
	}
}

func TestClosedAdmissionDoesNotPublishFailure(t *testing.T) {
	service, _ := newCompletionService("http://127.0.0.1/v1")
	events := &admissionEvents{}
	service.recorder.changed = events.changed
	cancel := startCompletionService(t, service)
	defer cancel()
	defer service.ServiceShutdown()
	service.activity.Close()
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		if err := service.StartRecording(mode); err == nil {
			t.Fatal("closed admission accepted work")
		}
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		if err := service.StartRecording(mode); err == nil {
			t.Fatal("closed service accepted work")
		}
	}
	if len(events.snapshot()) != 0 {
		t.Fatal("shutdown presented as a failed start")
	}
}

func TestCancelledRootDoesNotAdmitCapture(t *testing.T) {
	service, _ := newCompletionService("http://127.0.0.1/v1")
	capture := &countingStartCapture{}
	platform := &admissionPlatform{}
	service.recorder.capture = capture
	service.recorder.targetPlatform = platform
	cancel := startCompletionService(t, service)
	cancel()
	defer service.ServiceShutdown()
	for _, mode := range []RecordingMode{RecordingToggle, RecordingHold} {
		if err := service.StartRecording(mode); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled root start = %v", err)
		}
	}
	if capture.starts.Load() != 0 || platform.targetCalls.Load() != 0 {
		t.Fatal("cancelled root admitted capture")
	}
}
