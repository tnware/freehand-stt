package managedruntime

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type recoveryAdapter struct {
	workerAdapter
	inspectError atomic.Bool
	entered      chan struct{}
	resume       chan struct{}
}

func (a *recoveryAdapter) Inspect(ctx context.Context) (string, []Model, error) {
	if a.inspectError.Load() {
		if a.entered != nil {
			close(a.entered)
			<-a.resume
		}
		return "", nil, errors.New("private catalog failure")
	}
	return a.workerAdapter.Inspect(ctx)
}
func catalogComplete(t *testing.T, s *worker) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		s.mu.Lock()
		busy := s.busy
		s.mu.Unlock()
		if !busy {
			return
		}
		select {
		case <-deadline:
			t.Fatal("catalog did not complete")
		case <-time.After(time.Millisecond):
		}
	}
}
func TestCatalogFailurePreservesLiveEndpointAndRecovery(t *testing.T) {
	s := runningConcurrencyWorker(t)
	original, err := resolveWorker(s)
	if err != nil {
		t.Fatal(err)
	}
	a := &recoveryAdapter{workerAdapter: workerAdapter{installed: true, downloaded: true}}
	s.adapter = a
	a.inspectError.Store(true)
	if err := s.RefreshCatalog(); err != nil {
		t.Fatal(err)
	}
	catalogComplete(t, s)
	st := s.GetStatus()
	got, err := resolveWorker(s)
	if st.State != "running" || st.Error == "" || st.Error == "private catalog failure" || err != nil || got != original {
		t.Fatalf("catalog failure broke independently live process: state=%+v endpoint=%+v err=%v", st, got, err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "running")
	got, err = resolveWorker(s)
	if err != nil || got != original {
		t.Fatalf("restart retained process without endpoint: %+v %v", got, err)
	}
	a.inspectError.Store(false)
	if err := s.RefreshCatalog(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "running")
	if st := s.GetStatus(); st.Error != "" {
		t.Fatalf("recovery kept stale error: %+v", st)
	}
}
func TestCatalogFailureCannotResurrectExitedProcess(t *testing.T) {
	s := runningConcurrencyWorker(t)
	a := &recoveryAdapter{entered: make(chan struct{}), resume: make(chan struct{})}
	a.inspectError.Store(true)
	s.adapter = a
	s.mu.Lock()
	proc := s.process
	s.mu.Unlock()
	if err := s.RefreshCatalog(); err != nil {
		t.Fatal(err)
	}
	awaitConcurrency(t, a.entered)
	proc.Kill()
	deadline := time.After(2 * time.Second)
	for {
		s.mu.Lock()
		gone := s.process == nil
		s.mu.Unlock()
		if gone {
			break
		}
		select {
		case <-deadline:
			close(a.resume)
			t.Fatal("process monitor did not clear endpoint")
		case <-time.After(time.Millisecond):
		}
	}
	close(a.resume)
	catalogComplete(t, s)
	if _, err := resolveWorker(s); err == nil || s.GetStatus().State == "running" {
		t.Fatal("catalog failure resurrected dead endpoint")
	}
}
