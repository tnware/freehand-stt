package overlay

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type timeoutOverlayFake struct {
	statusOverlayFake
	mu sync.Mutex
}

func (f *timeoutOverlayFake) Update(status platform.OverlayStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.statusOverlayFake.Update(status)
}

func (f *timeoutOverlayFake) snapshot() (platform.OverlayStatus, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.updates[len(f.updates)-1], len(f.updates)
}

func timeoutService(t *testing.T) (*Service, *timeoutOverlayFake) {
	t.Helper()
	fake := &timeoutOverlayFake{}
	service := NewService(config.Default(), nil, nil)
	service.newOverlay = func() (statusOverlay, error) { return fake, nil }
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	Start(service)
	t.Cleanup(func() { _ = service.ServiceShutdown() })
	return service, fake
}

func lastOverlayKind(fake *timeoutOverlayFake) platform.OverlayKind {
	status, _ := fake.snapshot()
	return status.Kind
}

func TestOutcomeOverlayExpiresWithoutClearingResult(t *testing.T) {
	for _, outcome := range []dictation.Status{
		{State: dictation.Failed, Generation: 1, Message: "Transcription failed"},
		{State: dictation.Failed, Generation: 1, CanCopy: true, Transcript: "Retained result", Message: "Transcript ready to copy"},
		{State: dictation.Failed, Generation: 1, CanCopy: true, Transcript: "Retained result", StartRejected: true, Message: "Runtime unavailable"},
	} {
		t.Run(outcome.Message, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service, fake := timeoutService(t)
				ApplyStatus(service, outcome)
				visibleKind := lastOverlayKind(fake)
				if visibleKind == platform.OverlayHidden {
					t.Fatal("outcome was not shown")
				}
				time.Sleep(4 * time.Second)
				synctest.Wait()
				if lastOverlayKind(fake) != visibleKind {
					t.Fatal("outcome disappeared early")
				}
				time.Sleep(time.Second)
				synctest.Wait()
				if lastOverlayKind(fake) != platform.OverlayHidden {
					t.Fatal("outcome did not disappear after five seconds")
				}
				if service.status != outcome {
					t.Fatal("dismissal altered authoritative result or copy capability")
				}
				settings := service.settings
				settings.OverlayEnabled = false
				ApplySettings(service, settings)
				settings.OverlayEnabled = true
				ApplySettings(service, settings)
				if lastOverlayKind(fake) != platform.OverlayHidden {
					t.Fatal("settings replay resurrected dismissed outcome")
				}
				ApplyStatus(service, outcome) // A fresh rejected attempt may retain the same result generation.
				if lastOverlayKind(fake) != visibleKind {
					t.Fatal("fresh attempt did not show feedback again")
				}
				time.Sleep(5 * time.Second)
				synctest.Wait()
				if lastOverlayKind(fake) != platform.OverlayHidden {
					t.Fatal("fresh attempt did not expire")
				}
			})
		})
	}
}

func TestOutcomeTimerCannotHideNewerFeedbackOrActiveWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, fake := timeoutService(t)
		outcome := dictation.Status{State: dictation.Failed, StartRejected: true}
		ApplyStatus(service, outcome)
		time.Sleep(4 * time.Second)
		ApplyStatus(service, outcome)
		time.Sleep(time.Second)
		synctest.Wait()
		if lastOverlayKind(fake) != platform.OverlayFailed {
			t.Fatal("old deadline hid fresh rejection")
		}
		for _, state := range []dictation.State{dictation.Recording, dictation.Transcribing, dictation.PostProcessing} {
			ApplyStatus(service, dictation.Status{State: state, Generation: 2})
			time.Sleep(6 * time.Second)
			synctest.Wait()
			if lastOverlayKind(fake) == platform.OverlayHidden {
				t.Fatalf("timer hid active %s", state)
			}
		}
	})
}

func TestOutcomeExpiryDoesNotInterruptPreviewOrOutliveShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, fake := timeoutService(t)
		ApplyStatus(service, dictation.Status{State: dictation.Failed, CanCopy: true})
		if err := service.StartPreview(PreviewRequest{Preferences: service.settings.OverlayPreferences(), ToggleShortcut: service.settings.ToggleShortcut}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Second)
		synctest.Wait()
		if last, _ := fake.snapshot(); !last.Preview || last.Kind == platform.OverlayHidden {
			t.Fatal("outcome expiry interrupted preview")
		}
		if err := service.StopPreview(); err != nil {
			t.Fatal(err)
		}
		if lastOverlayKind(fake) != platform.OverlayHidden {
			t.Fatal("stopping preview restored expired outcome")
		}
		ApplyStatus(service, dictation.Status{State: dictation.Failed})
		if err := service.ServiceShutdown(); err != nil {
			t.Fatal(err)
		}
		_, updates := fake.snapshot()
		time.Sleep(6 * time.Second)
		synctest.Wait()
		if _, count := fake.snapshot(); count != updates {
			t.Fatal("timer updated native surface after shutdown")
		}
	})
}
