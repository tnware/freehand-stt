package managedruntime

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"os"
	"runtime"
	"sync"
	"time"
)

type Options struct {
	Directory       string
	Preferences     Preferences
	SavePreferences func(Preferences) error
	Changed         func(Status)
	Logger          *slog.Logger
	CheckIdle       func() error
}
type runtimeAdapter interface {
	Inspect(context.Context) (string, []Model, error)
	Install(context.Context, func(float64)) (string, error)
	Pull(context.Context, string) error
	RemoveModel(context.Context, string) error
	Remove(context.Context) error
	Start(context.Context, string) (*ownedProcess, Endpoint, error)
}
type Service struct {
	mu                   sync.Mutex
	directory            string
	preferences          Preferences
	save                 func(Preferences) error
	changed              func(Status)
	logger               *slog.Logger
	checkIdle            func() error
	status               Status
	endpoint             Endpoint
	busy, closed, saving bool
	pending              *Preferences
	generation           uint64
	adapter              runtimeAdapter
	ctx                  context.Context
	cancel               context.CancelFunc
	operationCancel      context.CancelFunc
	processCancel        context.CancelFunc
	process              *ownedProcess
	wg                   sync.WaitGroup
}

var errBusy = errors.New("Wait for the current managed speech operation to finish.")
var errUnsupported = errors.New("Managed speech is supported only on Windows x64.")
var errNotReady = errors.New("Managed speech is not ready. Start the selected installed model or disable managed speech.")

func NewService(o Options) *Service {
	s := &Service{directory: o.Directory, preferences: o.Preferences, save: o.SavePreferences, changed: o.Changed, logger: o.Logger, checkIdle: o.CheckIdle, adapter: newAdapter(o.Directory)}
	s.status = Status{Supported: runtime.GOOS == "windows" && runtime.GOARCH == "amd64", State: "not_installed", Version: Version, Progress: -1, Models: []Model{}}
	if err := Validate(o.Preferences); err != nil {
		s.status.State = "error"
		s.status.Error = err.Error()
	}
	return s
}
func (s *Service) GetStatus() Status { s.mu.Lock(); defer s.mu.Unlock(); return s.snapshotLocked() }
func (s *Service) snapshotLocked() Status {
	st := s.status
	st.Enabled = s.preferences.Enabled
	st.SelectedModel = s.preferences.Model
	st.Realtime = s.preferences.Realtime
	st.Models = append([]Model{}, st.Models...)
	return st
}
func (s *Service) notify() {
	if s.changed != nil {
		s.changed(s.GetStatus())
	}
}
func (s *Service) GetPreferences() Preferences {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.preferences
}
func (s *Service) resolveLocked() (Endpoint, error) {
	if !s.preferences.Enabled {
		return Endpoint{}, nil
	}
	// Mutation reserves busy before entering the activity gate. A start that
	// already captured an endpoint holds that gate until its active publication;
	// later starts must not cross the guard-to-transition gap.
	if !s.status.Supported || s.closed || s.busy || s.status.State != "running" || !s.endpoint.Enabled {
		return Endpoint{}, errNotReady
	}
	return s.endpoint, nil
}
func (s *Service) Resolve() (Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolveLocked()
}
func (s *Service) ResolveFor(p Preferences) (Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p != s.preferences {
		return Endpoint{}, errNotReady
	}
	return s.resolveLocked()
}
func (s *Service) admitLocked() error {
	if !s.status.Supported {
		return errUnsupported
	}
	if s.closed {
		return errors.New("Managed speech is shutting down.")
	}
	if s.busy {
		return errBusy
	}
	return nil
}
func (s *Service) idle() error {
	if s.checkIdle != nil && s.checkIdle() != nil {
		return errors.New("Finish active speech work before changing the managed runtime.")
	}
	return nil
}

// ApplyPreferences accepts already-committed settings. During the save callback it
// stages the latest committed snapshot; SetPreferences completes the handshake.
// It never persists or calls an external callback while holding the runtime lock.
func ApplyPreferences(s *Service, p Preferences) {
	if s == nil || Validate(p) != nil {
		return
	}
	s.mu.Lock()
	if s.closed || !s.status.Supported {
		s.mu.Unlock()
		return
	}
	if s.saving {
		q := p
		s.pending = &q
		s.mu.Unlock()
		return
	}
	kill := s.applyLocked(p)
	s.mu.Unlock()
	if kill != nil {
		kill.kill()
	}
	s.notify()
}
func (s *Service) applyLocked(p Preferences) *ownedProcess {
	old := s.preferences
	s.preferences = p
	if old == p {
		return nil
	}
	s.generation++
	if old.Model != p.Model || !p.Enabled {
		if s.operationCancel != nil {
			s.operationCancel()
		}
		kill := s.process
		s.process = nil
		s.endpoint = Endpoint{}
		if s.processCancel != nil {
			s.processCancel()
			s.processCancel = nil
		}
		if kill != nil {
			s.status.State = "stopped"
		}
		return kill
	}
	s.endpoint.Realtime = p.Realtime
	return nil
}
func (s *Service) SetPreferences(p Preferences) error {
	if err := Validate(p); err != nil {
		return err
	}
	s.mu.Lock()
	if err := s.admitLocked(); err != nil {
		s.mu.Unlock()
		return err
	}
	old := s.preferences
	s.busy = true
	s.saving = true
	s.mu.Unlock()
	var err error
	if old != p {
		err = s.idle()
	}
	if err == nil && s.save != nil {
		err = s.save(p)
	}
	s.mu.Lock()
	var kill *ownedProcess
	if !s.closed {
		if s.pending != nil {
			kill = s.applyLocked(*s.pending)
		} else if err == nil {
			kill = s.applyLocked(p)
		}
	}
	s.pending = nil
	s.saving = false
	s.busy = false
	s.mu.Unlock()
	if kill != nil {
		kill.kill()
	}
	s.notify()
	if err != nil {
		return errors.New("Could not change managed speech preferences. Finish active speech work and try again.")
	}
	return nil
}

// run reserves admission before calling guards, so UI tasks cannot overlap.
func (s *Service) run(phase, state string, guard bool, work func(context.Context) error) error {
	s.mu.Lock()
	if err := s.admitLocked(); err != nil {
		s.mu.Unlock()
		return err
	}
	if s.ctx == nil {
		s.mu.Unlock()
		return errors.New("Managed speech has not started yet.")
	}
	if s.process != nil && (phase == "install" || phase == "download") {
		s.mu.Unlock()
		return errors.New("Stop managed speech before changing runtime files.")
	}
	s.busy = true
	s.mu.Unlock()
	if guard {
		if err := s.idle(); err != nil {
			s.mu.Lock()
			s.busy = false
			s.mu.Unlock()
			return err
		}
	}
	s.mu.Lock()
	if s.closed {
		s.busy = false
		s.mu.Unlock()
		return errNotReady
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.operationCancel = cancel
	s.status.Phase = phase
	s.status.Error = ""
	s.status.Progress = -1
	if state != "" {
		s.status.State = state
	}
	s.wg.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wg.Done()
		started := time.Now()
		if s.logger != nil {
			s.logger.Info("managed operation started", "operation", phase)
		}
		s.notify()
		err := work(ctx)
		s.mu.Lock()
		keep := (phase == "start" || phase == "startup") && err == nil && s.process != nil
		if !keep {
			cancel()
		}
		s.operationCancel = nil
		s.busy = false
		s.status.Phase = ""
		s.status.Progress = -1
		kind := "none"
		if err != nil {
			if errors.Is(err, context.Canceled) {
				kind = "cancelled"
				if s.process == nil {
					if s.status.Backend == "" {
						s.status.State = "not_installed"
					} else {
						s.status.State = "installed"
					}
				}
			} else {
				kind = "runtime"
				s.status.Error = "Managed speech could not complete the operation. Check setup and try again."
				// Metadata failure does not invalidate a separately supervised live
				// process. Its monitor still clears the endpoint on process exit.
				if phase != "catalog" || s.process == nil || s.status.State != "running" || !s.endpoint.Enabled || s.endpoint.BaseURL == "" || s.endpoint.Model == "" {
					s.status.State = "error"
					s.endpoint = Endpoint{}
				}
			}
		}
		if s.closed {
			s.status.State = "stopped"
			s.endpoint = Endpoint{}
		}
		s.mu.Unlock()
		if s.logger != nil {
			s.logger.Info("managed operation finished", "operation", phase, "error_kind", kind, "duration_ms", time.Since(started).Milliseconds())
		}
		s.notify()
	}()
	return nil
}
func (s *Service) inspect(ctx context.Context) error {
	backend, models, err := s.adapter.Inspect(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if errors.Is(err, os.ErrNotExist) {
		s.status.Backend = ""
		s.status.Models = []Model{}
		if s.process == nil {
			s.status.State = "not_installed"
		}
		return nil
	}
	if err != nil {
		return err
	}
	s.status.Backend = backend
	s.status.Models = append([]Model{}, models...)
	if s.process == nil {
		s.status.State = "installed"
	} else {
		s.status.State = "running"
	}
	return nil
}
func (s *Service) Install() error {
	return s.run("install", "installing", false, func(ctx context.Context) error {
		s.mu.Lock()
		running := s.process != nil
		s.mu.Unlock()
		if running {
			return errBusy
		}
		_, err := s.adapter.Install(ctx, func(p float64) {
			if math.IsNaN(p) || math.IsInf(p, 0) {
				p = -1
			}
			if p > 1 {
				p = 1
			}
			if p < 0 {
				p = -1
			}
			s.mu.Lock()
			s.status.Progress = p
			s.mu.Unlock()
			s.notify()
		})
		if err != nil {
			return err
		}
		return s.inspect(ctx)
	})
}
func (s *Service) RefreshCatalog() error { return s.run("catalog", "", false, s.inspect) }
func (s *Service) DownloadModel(id string) error {
	if _, ok := qualified[id]; !ok {
		return errors.New("Choose a supported managed speech model.")
	}
	return s.run("download", "installing", false, func(ctx context.Context) error {
		if err := s.adapter.Pull(ctx, id); err != nil {
			return err
		}
		return s.inspect(ctx)
	})
}
func (s *Service) RemoveModel(id string) error {
	if _, ok := qualified[id]; !ok {
		return errors.New("Choose a supported managed speech model.")
	}
	return s.run("remove_model", "", true, func(ctx context.Context) error {
		s.mu.Lock()
		selected := s.preferences.Model == id
		s.mu.Unlock()
		if selected {
			if err := s.stopProcess(ctx); err != nil {
				return err
			}
		}
		if err := s.adapter.RemoveModel(ctx, id); err != nil {
			return err
		}
		return s.inspect(ctx)
	})
}
func (s *Service) Remove() error {
	return s.run("remove", "stopping", true, func(ctx context.Context) error {
		if err := s.stopProcess(ctx); err != nil {
			return err
		}
		if err := s.adapter.Remove(ctx); err != nil {
			return err
		}
		return s.inspect(ctx)
	})
}
func (s *Service) startProcess(ctx context.Context) error {
	s.mu.Lock()
	p := s.preferences
	generation := s.generation
	running := s.process != nil
	installed := false
	for _, m := range s.status.Models {
		if m.ID == p.Model && m.Installed {
			installed = true
		}
	}
	s.mu.Unlock()
	if running {
		s.mu.Lock()
		s.status.State = "running"
		s.mu.Unlock()
		return nil
	}
	if !p.Enabled || !installed {
		return errNotReady
	}
	proc, endpoint, err := s.adapter.Start(ctx, p.Model)
	if err != nil {
		return err
	}
	if proc == nil || !endpoint.Enabled || endpoint.BaseURL == "" || endpoint.Model == "" {
		if proc != nil {
			proc.kill()
		}
		return errNotReady
	}
	s.mu.Lock()
	if s.closed || ctx.Err() != nil || generation != s.generation {
		s.mu.Unlock()
		proc.kill()
		return context.Canceled
	}
	endpoint.Realtime = p.Realtime
	s.process = proc
	s.endpoint = endpoint
	s.processCancel = s.operationCancel
	s.status.State = "running"
	s.wg.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wg.Done()
		select {
		case <-proc.done:
		case <-s.ctx.Done():
			proc.kill()
			<-proc.done
		}
		s.mu.Lock()
		if s.process != proc {
			s.mu.Unlock()
			return
		}
		s.process = nil
		s.endpoint = Endpoint{}
		if s.processCancel != nil {
			s.processCancel()
			s.processCancel = nil
		}
		if !s.closed {
			s.status.State = "error"
			s.status.Error = "Managed speech stopped unexpectedly. Start it again when ready."
		}
		s.mu.Unlock()
		s.notify()
	}()
	return nil
}
func (s *Service) Start() error { return s.run("start", "starting", true, s.startProcess) }
func (s *Service) stopProcess(ctx context.Context) error {
	s.mu.Lock()
	p := s.process
	s.process = nil
	s.endpoint = Endpoint{}
	if s.processCancel != nil {
		s.processCancel()
		s.processCancel = nil
	}
	s.mu.Unlock()
	if p != nil {
		p.kill()
		select {
		case <-p.done:
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(4 * time.Second):
			return errors.New("Managed speech did not stop in time.")
		}
	}
	s.mu.Lock()
	if s.status.Backend == "" {
		s.status.State = "not_installed"
	} else {
		s.status.State = "stopped"
	}
	s.mu.Unlock()
	return nil
}
func (s *Service) Stop() error { return s.run("stop", "stopping", true, s.stopProcess) }
func (s *Service) Cancel() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.status.Supported {
		return errUnsupported
	}
	if s.closed {
		return errNotReady
	}
	if s.saving {
		return errBusy
	}
	if s.operationCancel != nil {
		s.operationCancel()
	}
	return nil
}
func (s *Service) startup(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errNotReady
	}
	if s.ctx != nil {
		s.mu.Unlock()
		return nil
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	supported := s.status.Supported
	s.mu.Unlock()
	if !supported {
		return nil
	}
	return s.run("startup", "", false, func(ctx context.Context) error {
		if err := s.inspect(ctx); err != nil {
			return err
		}
		s.mu.Lock()
		ready := s.preferences.Enabled
		installed := false
		for _, m := range s.status.Models {
			if m.ID == s.preferences.Model && m.Installed {
				installed = true
			}
		}
		s.mu.Unlock()
		if ready && installed {
			return s.startProcess(ctx)
		}
		return nil
	})
}

// ServiceShutdown cancels children and waits at most eight seconds. It bypasses
// active-work admission because the application is already shutting down.
func (s *Service) ServiceShutdown() error {
	s.mu.Lock()
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
	if s.operationCancel != nil {
		s.operationCancel()
	}
	p := s.process
	s.process = nil
	s.endpoint = Endpoint{}
	s.status.State = "stopped"
	s.mu.Unlock()
	if p != nil {
		p.kill()
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-time.After(8 * time.Second):
		return errors.New("Managed speech shutdown exceeded its deadline.")
	}
}
