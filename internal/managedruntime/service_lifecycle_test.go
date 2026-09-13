package managedruntime

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestStartupRetainsProcessContext(t *testing.T) {
	a := &serviceAdapter{installed: true, downloaded: true}
	p := Defaults()
	p.Enabled = true
	s := NewService(Options{Preferences: p})
	s.status.Supported = true
	s.adapter = a
	if err := s.startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.ServiceShutdown() })
	waitService(t, s, "running")
	if err := a.startContext.Err(); err != nil {
		t.Fatalf("startup cancelled live server: %v", err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "running")
}

type serviceAdapter struct {
	startContext context.Context
	installed    bool
	downloaded   bool
	pullEntered  chan struct{}
	fail         bool
	process      *ownedProcess
}

func (a *serviceAdapter) Inspect(context.Context) (string, []Model, error) {
	if !a.installed {
		return "", nil, os.ErrNotExist
	}
	m := qualified["nemotron-3.5"]
	m.Installed = a.downloaded
	return "cpu", []Model{m}, nil
}
func (a *serviceAdapter) Install(ctx context.Context, p func(float64)) (string, error) {
	a.installed = true
	p(.5)
	return "cpu", nil
}
func (a *serviceAdapter) Pull(ctx context.Context, id string) error {
	if a.pullEntered != nil {
		close(a.pullEntered)
		<-ctx.Done()
		return ctx.Err()
	}
	a.downloaded = true
	return nil
}
func (a *serviceAdapter) RemoveModel(context.Context, string) error { a.downloaded = false; return nil }
func (a *serviceAdapter) Remove(context.Context) error              { a.installed = false; return nil }
func (a *serviceAdapter) Start(ctx context.Context, id string) (*ownedProcess, Endpoint, error) {
	a.startContext = ctx
	if a.fail {
		return nil, Endpoint{}, errors.New("secret")
	}
	p := &ownedProcess{done: make(chan struct{})}
	p.closeJob = func() { close(p.done) }
	a.process = p
	return p, Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:12345", Model: "authoritative", Profile: qualified[id].Profile}, nil
}
func waitService(t *testing.T, s *Service, state string) {
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
func testService(t *testing.T, a *serviceAdapter) *Service {
	t.Helper()
	s := NewService(Options{Preferences: Defaults()})
	s.status.Supported = true
	s.adapter = a
	if err := s.startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.ServiceShutdown() })
	return s
}
func TestServiceExplicitSetupAndLifecycle(t *testing.T) {
	a := &serviceAdapter{}
	s := testService(t, a)
	waitService(t, s, "not_installed")
	if err := s.Install(); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "installed")
	if a.downloaded {
		t.Fatal("automatic download")
	}
	if err := s.DownloadModel("nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "installed")
	p := Defaults()
	p.Enabled = true
	if err := s.SetPreferences(p); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "running")
	e, err := s.ResolveFor(p)
	if err != nil || e.Model != "authoritative" || !e.Realtime {
		t.Fatalf("endpoint %+v %v", e, err)
	}
	if _, err = s.ResolveFor(Defaults()); err == nil {
		t.Fatal("mixed preferences resolved")
	}
	if err = s.Stop(); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "stopped")
	if err = s.RemoveModel(p.Model); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "installed")
	if err = s.Remove(); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "not_installed")
}
func TestServiceCancellationAndBusy(t *testing.T) {
	a := &serviceAdapter{installed: true, pullEntered: make(chan struct{})}
	s := testService(t, a)
	waitService(t, s, "installed")
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
	waitService(t, s, "installed")
}
func TestServiceUnexpectedExitAndIdleGuard(t *testing.T) {
	a := &serviceAdapter{installed: true, downloaded: true}
	s := testService(t, a)
	waitService(t, s, "installed")
	p := Defaults()
	p.Enabled = true
	if err := s.SetPreferences(p); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "running")
	s.checkIdle = func() error { return errors.New("secret") }
	if s.Stop() == nil {
		t.Fatal("ignored idle guard")
	}
	a.process.kill()
	waitService(t, s, "error")
	if _, err := s.Resolve(); err == nil {
		t.Fatal("dead endpoint resolved")
	}
}
func TestApplyPreferencesReentrantSave(t *testing.T) {
	s := NewService(Options{Preferences: Defaults()})
	s.status.Supported = true
	p := Defaults()
	p.Enabled = true
	s.save = func(got Preferences) error { ApplyPreferences(s, got); _ = s.GetStatus(); return nil }
	done := make(chan error, 1)
	go func() { done <- s.SetPreferences(p) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("reentrant apply deadlock")
	}
	if s.GetPreferences() != p {
		t.Fatal("lost committed preferences")
	}
}
func TestUnsupportedNeverMutates(t *testing.T) {
	s := NewService(Options{Preferences: Defaults()})
	s.status.Supported = false
	for _, f := range []func() error{s.Install, s.Remove, s.Start, s.Stop, s.Cancel, s.RefreshCatalog, func() error { return s.DownloadModel("nemotron-3.5") }, func() error { return s.RemoveModel("nemotron-3.5") }, func() error { return s.SetPreferences(Defaults()) }} {
		if f() == nil {
			t.Fatal("unsupported mutation accepted")
		}
	}
}
