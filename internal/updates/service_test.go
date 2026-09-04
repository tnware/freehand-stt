package updates

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

type checkerFake struct {
	mu          sync.Mutex
	release     *updater.Release
	err         error
	checks      int
	interactive int
}

func (f *checkerFake) Check(context.Context) (*updater.Release, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checks++
	return f.release, f.err
}

func (f *checkerFake) CheckAndInstall(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.interactive++
	return f.err
}

func (f *checkerFake) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.checks, f.interactive
}

func TestAutomaticCheckPublishesAvailableRelease(t *testing.T) {
	checker := &checkerFake{release: &updater.Release{Version: "0.1.0-alpha.2"}}
	statuses := make(chan Status, 8)
	service := NewService("0.1.0-alpha.1", true, false, func(status Status) { statuses <- status }, nil,
		WithSchedule(time.Millisecond, time.Hour))
	Configure(service, checker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := service.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer service.ServiceShutdown()

	deadline := time.After(time.Second)
	for {
		select {
		case status := <-statuses:
			if status.State == StateAvailable {
				if status.LatestVersion != "0.1.0-alpha.2" || status.LastCheckedAt == "" {
					t.Fatalf("available status = %#v", status)
				}
				deadline := time.Now().Add(time.Second)
				for {
					_, interactive := checker.counts()
					if interactive == 1 {
						return
					}
					if time.Now().After(deadline) {
						t.Fatal("available update did not open the Wails updater")
					}
					time.Sleep(time.Millisecond)
				}
			}
		case <-deadline:
			t.Fatal("automatic update result was not published")
		}
	}
}

func TestDisabledServiceDoesNotPollUntilEnabled(t *testing.T) {
	checker := &checkerFake{}
	service := NewService("0.1.0-alpha.1", false, false, nil, nil,
		WithSchedule(5*time.Millisecond, time.Hour))
	Configure(service, checker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := service.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer service.ServiceShutdown()
	time.Sleep(20 * time.Millisecond)
	if checks, _ := checker.counts(); checks != 0 {
		t.Fatalf("disabled service made %d checks", checks)
	}
	ApplyEnabled(service, true)
	time.Sleep(20 * time.Millisecond)
	if checks, _ := checker.counts(); checks != 1 {
		t.Fatalf("enabled service made %d checks, want 1", checks)
	}
}

func TestAutomaticCheckStaysQuietWhenCurrent(t *testing.T) {
	checker := &checkerFake{}
	service := NewService("0.1.0-alpha.1", true, false, nil, nil)
	Configure(service, checker)
	service.backgroundCheck(context.Background())

	checks, interactive := checker.counts()
	if checks != 1 || interactive != 0 {
		t.Fatalf("checks/interactive = %d/%d, want 1/0", checks, interactive)
	}
	if state := service.Current().State; state != StateCurrent {
		t.Fatalf("state = %q, want current", state)
	}
}

func TestInteractiveCheckIsUnavailableInDevelopment(t *testing.T) {
	service := NewService("0.1.0-alpha.1", true, true, nil, nil)
	Configure(service, &checkerFake{})
	if err := service.CheckForUpdates(); err == nil {
		t.Fatal("development check unexpectedly started")
	}
	if state := service.Current().State; state != StateDevelopment {
		t.Fatalf("state = %q, want development", state)
	}
}

func TestShutdownRejectsNewInteractiveWork(t *testing.T) {
	service := NewService("0.1.0-alpha.1", true, false, nil, nil)
	Configure(service, &checkerFake{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := service.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	if err := service.CheckForUpdates(); err == nil {
		t.Fatal("interactive check started after shutdown")
	}
}

func TestBackgroundFailureIsBounded(t *testing.T) {
	checker := &checkerFake{err: errors.New("GET https://secret.example/path?token=value failed")}
	service := NewService("0.1.0-alpha.1", true, false, nil, nil)
	Configure(service, checker)
	service.backgroundCheck(context.Background())
	status := service.Current()
	if status.State != StateError || status.ErrorKind == "" {
		t.Fatalf("failure status = %#v", status)
	}
}

func TestDisabledPreferenceWinsOverAnInFlightBackgroundResult(t *testing.T) {
	service := NewService("0.1.0-alpha.1", true, false, nil, nil)
	ApplyEnabled(service, false)
	service.finishBackground(&updater.Release{Version: "0.1.0-alpha.2"}, nil)
	status := service.Current()
	if status.State != StateDisabled || status.LatestVersion != "" {
		t.Fatalf("disabled status was replaced by late result: %#v", status)
	}
}
