package updates

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
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
	synctest.Test(t, func(t *testing.T) {
		checker := &checkerFake{release: &updater.Release{Version: "0.1.0-alpha.2"}}
		var statuses []Status
		service := NewService("0.1.0-alpha.1", true, false, func(status Status) { statuses = append(statuses, status) }, nil)
		Configure(service, checker)
		if err := service.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
		defer service.ServiceShutdown()

		synctest.Wait()
		synctest.Sleep(initialCheckDelay - time.Nanosecond)
		if checks, _ := checker.counts(); checks != 0 {
			t.Fatalf("automatic check ran before its initial delay: %d", checks)
		}
		synctest.Sleep(time.Nanosecond)
		if checks, interactive := checker.counts(); checks != 1 || interactive != 1 {
			t.Fatalf("checks/interactive = %d/%d, want 1/1", checks, interactive)
		}
		for _, status := range statuses {
			if status.State == StateAvailable && status.LatestVersion == "0.1.0-alpha.2" && status.LastCheckedAt != "" {
				return
			}
		}
		t.Fatalf("available release was not published: %#v", statuses)
	})
}

func TestDisabledServiceDoesNotPollUntilEnabled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		checker := &checkerFake{}
		service := NewService("0.1.0-alpha.1", false, false, nil, nil)
		Configure(service, checker)
		if err := service.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
		defer service.ServiceShutdown()
		synctest.Wait()
		synctest.Sleep(initialCheckDelay + 2*checkInterval)
		if checks, _ := checker.counts(); checks != 0 {
			t.Fatalf("disabled service made %d checks", checks)
		}
		ApplyEnabled(service, true)
		synctest.Wait()
		synctest.Sleep(initialCheckDelay)
		if checks, _ := checker.counts(); checks != 1 {
			t.Fatalf("enabled service made %d checks, want 1", checks)
		}
		synctest.Sleep(checkInterval)
		if checks, _ := checker.counts(); checks != 2 {
			t.Fatalf("enabled service did not repeat the check: %d", checks)
		}
		ApplyEnabled(service, false)
		synctest.Wait()
		synctest.Sleep(initialCheckDelay + 2*checkInterval)
		if checks, _ := checker.counts(); checks != 2 {
			t.Fatalf("disabled service kept polling: %d", checks)
		}
	})
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
	var published Status
	service := NewService("0.1.0-alpha.1", true, false, func(status Status) { published = status }, nil)
	Configure(service, checker)
	service.backgroundCheck(t.Context())
	status := service.Current()
	if status.State != StateError || status.ErrorKind != "operation" || status.LatestVersion != "" {
		t.Fatalf("failure status = %#v", status)
	}
	if published != status {
		t.Fatalf("renderer event = %#v, want bounded current status %#v", published, status)
	}
}

func TestDisabledPreferenceWinsOverAnInFlightBackgroundResult(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		checker := &waitingChecker{result: make(chan *updater.Release, 1)}
		service := NewService("0.1.0-alpha.1", true, false, nil, nil)
		Configure(service, checker)
		if err := service.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
		defer service.ServiceShutdown()
		synctest.Wait()
		synctest.Sleep(initialCheckDelay)
		if state := service.Current().State; state != StateChecking {
			t.Fatalf("state before result = %q, want checking", state)
		}

		ApplyEnabled(service, false)
		checker.result <- &updater.Release{Version: "0.1.0-alpha.2"}
		synctest.Wait()
		status := service.Current()
		if status.State != StateDisabled || status.LatestVersion != "" {
			t.Fatalf("disabled status was replaced by late result: %#v", status)
		}
		if _, interactive := checker.counts(); interactive != 0 {
			t.Fatal("late result opened an updater after updates were disabled")
		}
	})
}

type waitingChecker struct {
	checkerFake
	result chan *updater.Release
}

func (f *waitingChecker) Check(ctx context.Context) (*updater.Release, error) {
	select {
	case release := <-f.result:
		return release, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
