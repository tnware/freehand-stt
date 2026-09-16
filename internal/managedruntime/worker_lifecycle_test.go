package managedruntime

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestStartupRetainsProcessContext(t *testing.T) {
	a := &workerAdapter{installed: true, downloaded: true}
	s := newTestWorker(t, true)
	s.status.Supported = true
	s.adapter = a
	if err := s.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.ServiceShutdown() })
	waitWorker(t, s, "running")
	if err := a.startContext.Err(); err != nil {
		t.Fatalf("startup cancelled live server: %v", err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "running")
}

type workerAdapter struct {
	startContext context.Context
	installed    bool
	downloaded   bool
	pullEntered  chan struct{}
	fail         bool
	process      processHandle
}

func (a *workerAdapter) Inspect(context.Context) (string, []Model, error) {
	if !a.installed {
		return "", nil, os.ErrNotExist
	}
	m := qualified["nemotron-3.5"]
	m.Installed = a.downloaded
	return "cpu", []Model{m}, nil
}
func (a *workerAdapter) Install(ctx context.Context, p func(float64)) (string, error) {
	a.installed = true
	p(.5)
	return "cpu", nil
}
func (a *workerAdapter) Pull(ctx context.Context, id string, progress func(AcquisitionProgress)) error {
	if a.pullEntered != nil {
		close(a.pullEntered)
		<-ctx.Done()
		return ctx.Err()
	}
	a.downloaded = true
	return nil
}
func (a *workerAdapter) RemoveModel(context.Context, string) error { a.downloaded = false; return nil }
func (a *workerAdapter) Remove(context.Context) error              { a.installed = false; return nil }
func (a *workerAdapter) Start(ctx context.Context, id string) (processHandle, Endpoint, error) {
	a.startContext = ctx
	if a.fail {
		return nil, Endpoint{}, errors.New("secret")
	}
	p := &fakeProcess{done: make(chan struct{})}
	p.closeJob = func() { close(p.done) }
	a.process = p
	return p, Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:12345", Model: "authoritative", Profile: qualified[id].Profile}, nil
}
func waitWorker(t *testing.T, s *worker, state string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		st := s.GetStatus()
		s.mu.Lock()
		busy := s.busy
		s.mu.Unlock()
		if st.State == state && !busy {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("wanted %s got %+v", state, st)
		case <-time.After(time.Millisecond):
		}
	}
}
func testWorker(t *testing.T, a *workerAdapter) *worker {
	t.Helper()
	s := newTestWorker(t, false)
	s.status.Supported = true
	s.adapter = a
	if err := s.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.ServiceShutdown() })
	return s
}
func TestWorkerExplicitSetupAndLifecycle(t *testing.T) {
	a := &workerAdapter{}
	s := testWorker(t, a)
	waitWorker(t, s, "not_installed")
	if err := s.Install(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "installed")
	if a.downloaded {
		t.Fatal("automatic download")
	}
	if err := s.DownloadModel("nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "installed")
	p := workerConfig{Model: "nemotron-3.5", Enabled: true, Realtime: true}
	configureWorker(s, p)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "running")
	e, err := resolveWorker(s)
	if err != nil || e.Model != "authoritative" || !e.Realtime {
		t.Fatalf("endpoint %+v %v", e, err)
	}
	if err = s.Stop(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "stopped")
	if err = s.RemoveModel(p.Model); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "installed")
	if err = s.Remove(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "not_installed")
}
func TestWorkerCancellationAndBusy(t *testing.T) {
	a := &workerAdapter{installed: true, pullEntered: make(chan struct{})}
	s := testWorker(t, a)
	waitWorker(t, s, "installed")
	if err := s.DownloadModel("nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	<-a.pullEntered
	if s.RefreshCatalog() == nil {
		t.Fatal("accepted concurrent operation")
	}
	if err := s.Cancel(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "installed")
}
func TestWorkerUnexpectedExitAndIdleGuard(t *testing.T) {
	a := &workerAdapter{installed: true, downloaded: true}
	s := testWorker(t, a)
	waitWorker(t, s, "installed")
	p := workerConfig{Model: "nemotron-3.5", Enabled: true, Realtime: true}
	configureWorker(s, p)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitWorker(t, s, "running")
	s.checkIdle = func() error { return errors.New("secret") }
	if s.Stop() == nil {
		t.Fatal("ignored idle guard")
	}
	a.process.Kill()
	waitWorker(t, s, "error")
	if _, err := resolveWorker(s); err == nil {
		t.Fatal("dead endpoint resolved")
	}
}
func TestUnsupportedNeverMutates(t *testing.T) {
	s := newTestWorker(t, false)
	s.status.Supported = false
	for _, f := range []func() error{s.Install, s.Remove, s.Start, s.Stop, s.Cancel, s.RefreshCatalog, func() error { return s.DownloadModel("nemotron-3.5") }, func() error { return s.RemoveModel("nemotron-3.5") }} {
		if f() == nil {
			t.Fatal("unsupported mutation accepted")
		}
	}
}
