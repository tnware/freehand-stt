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
	state       updater.State
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

func (f *checkerFake) State() updater.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state != "" {
		return f.state
	}
	if f.release != nil {
		return updater.StateReady
	}
	return updater.StateUpToDate
}

func TestManualUpdateReportsActualUpdaterOutcome(t *testing.T) {
	for _, tc := range []struct {
		name     string
		upstream updater.State
		want     State
	}{
		{"staged", updater.StateReady, StateAvailable},
		{"current", updater.StateUpToDate, StateCurrent},
		{"no outcome", updater.StateIdle, StateIdle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service := NewService("0.1.0", false, false, nil, nil)
				Configure(service, &checkerFake{state: tc.upstream})
				if err := service.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
					t.Fatal(err)
				}
				defer service.ServiceShutdown()
				if err := service.CheckForUpdates(); err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				if status := service.Current(); status.State != tc.want || status.Enabled {
					t.Fatalf("manual outcome = %#v, want %s with automatic checks still off", status, tc.want)
				}
			})
		})
	}
}

func TestAutomaticFailureRetriesThenResumesDailySchedule(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		checker := &checkerFake{err: errors.New("offline")}
		service := NewService("0.1.0", true, false, nil, nil)
		Configure(service, checker)
		if err := service.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
		defer service.ServiceShutdown()
		synctest.Wait()
		synctest.Sleep(initialCheckDelay)
		if service.Current().State != StateError {
			t.Fatal("failed check was not reported")
		}
		checker.mu.Lock()
		checker.err = nil
		checker.mu.Unlock()
		synctest.Sleep(failedCheckRetry - time.Nanosecond)
		if checks, _ := checker.counts(); checks != 1 {
			t.Fatalf("early retry: %d", checks)
		}
		synctest.Sleep(time.Nanosecond)
		if checks, _ := checker.counts(); checks != 2 || service.Current().State != StateCurrent {
			t.Fatalf("retry did not recover: %d", checks)
		}
		synctest.Sleep(checkInterval - time.Nanosecond)
		if checks, _ := checker.counts(); checks != 2 {
			t.Fatalf("successful check retried early: %d", checks)
		}
		synctest.Sleep(time.Nanosecond)
		if checks, _ := checker.counts(); checks != 3 {
			t.Fatalf("daily schedule did not resume: %d", checks)
		}
	})
}

type blockedPresentation struct {
	checkerFake
	releaseCall chan struct{}
	cancelled   chan struct{}
}

func (f *blockedPresentation) CheckAndInstall(ctx context.Context) error {
	<-ctx.Done()
	close(f.cancelled)
	<-f.releaseCall // Model a native presentation waiting for the shutdown thread.
	return nil
}

func TestShutdownBoundsBlockedPresentationAndSuppressesLateOutcome(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		checker := &blockedPresentation{releaseCall: make(chan struct{}), cancelled: make(chan struct{})}
		publications := 0
		service := NewService("0.1.0", false, false, func(Status) { publications++ }, nil)
		Configure(service, checker)
		if err := service.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
		if err := service.CheckForUpdates(); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		before := service.Current()
		started := time.Now()
		if err := service.ServiceShutdown(); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown error = %v", err)
		}
		if elapsed := time.Since(started); elapsed != shutdownTimeout {
			t.Fatalf("shutdown took %v", elapsed)
		}
		<-checker.cancelled
		if err := service.ServiceShutdown(); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("repeated shutdown = %v", err)
		}
		if elapsed := time.Since(started); elapsed != shutdownTimeout {
			t.Fatalf("repeated shutdown extended budget: %v", elapsed)
		}
		close(checker.releaseCall)
		synctest.Wait()
		if service.Current() != before || publications != 1 {
			t.Fatalf("late completion changed status: %#v, events %d", service.Current(), publications)
		}
		if err := service.ServiceShutdown(); err != nil {
			t.Fatal(err)
		}
		if err := service.CheckForUpdates(); err == nil {
			t.Fatal("accepted work after shutdown")
		}
	})
}

func TestAutomaticMetadataCheckHasDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service := NewService("0.1.0", true, false, nil, nil)
		Configure(service, &waitingChecker{result: make(chan *updater.Release)})
		if err := service.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
		defer service.ServiceShutdown()
		synctest.Wait()
		synctest.Sleep(initialCheckDelay + metadataTimeout)
		if status := service.Current(); status.State != StateError || status.ErrorKind != "timeout" {
			t.Fatalf("deadline status = %#v", status)
		}
	})
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
