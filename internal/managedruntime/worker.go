package managedruntime

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type runtimeAdapter interface {
	Inspect(context.Context) (string, []Model, error)
	Install(context.Context, func(float64)) (string, error)
	Pull(context.Context, string, func(AcquisitionProgress)) error
	RemoveModel(context.Context, string) error
	Remove(context.Context) error
	Start(context.Context, string) (*ownedProcess, Endpoint, error)
}
type worker struct {
	mu                   sync.Mutex
	configuration        workerConfig
	changed              func(Status)
	logger               *slog.Logger
	checkIdle            func() error
	status               Status
	endpoint             Endpoint
	busy, closed, saving bool
	generation           uint64
	provider             provider
	adapter              runtimeAdapter
	ctx                  context.Context
	cancel               context.CancelFunc
	operationCancel      context.CancelFunc
	processCancel        context.CancelFunc
	process              *ownedProcess
	providerProcess      *providerProcess
	wg                   sync.WaitGroup
}

var errBusy = errors.New("Wait for the current managed speech operation to finish.")
var errUnsupported = errors.New("Managed speech is supported only on Windows x64.")
var errNotReady = errors.New("Managed speech is not ready. Start the selected installed model or disable managed speech.")

// Lease identities remain distinct even when a deleted instance is recreated.
var leaseGeneration atomic.Uint64
var operationSequence atomic.Uint64

type workerConfig struct {
	Model                        string
	Enabled, Realtime, AutoStart bool
}

func newWorker(root string, p provider, cfg workerConfig, logger *slog.Logger, idle func() error) *worker {
	d := p.descriptor()
	return &worker{configuration: cfg, provider: p, adapter: p.newAdapter(root), providerProcess: &providerProcess{}, logger: logger, checkIdle: idle, generation: leaseGeneration.Add(1),
		status: Status{Supported: d.Supported, State: "not_installed", Version: d.Version, Progress: -1, Models: []Model{}}}
}
func (s *worker) GetStatus() Status { s.mu.Lock(); defer s.mu.Unlock(); return s.snapshotLocked() }
func (s *worker) snapshotLocked() Status {
	st := s.status
	st.Enabled = s.configuration.Enabled
	st.SelectedModel = s.configuration.Model
	st.Realtime = s.configuration.Realtime
	st.Models = append([]Model{}, st.Models...)
	return st
}
func (s *worker) notify() {
	if s.changed != nil {
		s.changed(s.GetStatus())
	}
}
func (s *worker) resolveLocked() (Endpoint, error) {
	if !s.configuration.Enabled {
		return Endpoint{}, nil
	}
	// Mutation reserves busy before entering the activity gate. A start that
	// already captured an endpoint holds that gate until its active publication;
	// later starts must not cross the guard-to-transition gap.
	if !s.status.Supported || s.closed || s.busy || s.status.State != "running" || !s.endpoint.Enabled {
		return Endpoint{}, errNotReady
	}
	// The monitor may be waiting for this mutex after the child has exited.
	// Never lease its endpoint just because the status event has not caught up.
	if s.process != nil {
		select {
		case <-s.process.done:
			return Endpoint{}, errNotReady
		default:
		}
	}
	return s.endpoint, nil
}
func (s *worker) admitLocked() error {
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
func (s *worker) idle() error {
	if s.checkIdle != nil && s.checkIdle() != nil {
		return errors.New("Finish active speech work before changing the managed runtime.")
	}
	return nil
}

func (s *worker) configureLocked(p workerConfig) *ownedProcess {
	old := s.configuration
	s.configuration = p
	if old == p {
		return nil
	}
	s.generation = leaseGeneration.Add(1)
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

// run reserves admission before calling guards, so UI tasks cannot overlap.
func (s *worker) run(phase, state string, guard bool, work func(context.Context) error) error {
	return s.runOperation(phase, state, guard, "", work)
}
func (s *worker) runOperation(phase, state string, guard bool, model string, work func(context.Context) error) error {
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
	s.status.Acquisition = AcquisitionProgress{}
	s.status.Operation = Operation{ID: operationSequence.Add(1), Kind: phase, Model: model, Outcome: "running"}
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
		s.status.Phase = ""
		s.status.Progress = -1
		kind := "none"
		s.status.Operation.Outcome = "succeeded"
		if err != nil {
			if errors.Is(err, context.Canceled) {
				kind = "cancelled"
				s.status.Operation.Outcome = "cancelled"
				if s.process == nil {
					if s.status.Backend == "" {
						s.status.State = "not_installed"
					} else {
						s.status.State = "installed"
					}
				}
			} else {
				kind = "runtime"
				s.status.Operation.Outcome = "failed"
				s.status.Error = "Managed speech could not complete the operation. Check setup and try again."
				if errors.Is(err, errProviderRunning) {
					s.status.Error = errProviderRunning.Error()
				}
				if errors.Is(err, errCUDAUnavailable) {
					s.status.Error = errCUDAUnavailable.Error()
				}
				// Metadata failure does not invalidate a separately supervised live
				// process. Its monitor still clears the endpoint on process exit.
				if phase != "catalog" || s.process == nil || s.status.State != "running" || !s.endpoint.Enabled || s.endpoint.BaseURL == "" || s.endpoint.Model == "" {
					s.status.State = "error"
					s.endpoint = Endpoint{}
				}
			}
		}
		s.status.Operation.Error = s.status.Error
		if s.closed {
			s.status.State = "stopped"
			s.endpoint = Endpoint{}
		}
		s.mu.Unlock()
		if s.logger != nil {
			s.logger.Info("managed operation finished", "operation", phase, "error_kind", kind, "duration_ms", time.Since(started).Milliseconds())
		}
		// Publish the terminal result before another admission can replace it.
		s.notify()
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
	}()
	return nil
}
func (s *worker) inspect(ctx context.Context) error {
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

type backendInstaller interface {
	InstallBackend(context.Context, string, func(float64)) (string, error)
}

func (s *worker) InstallBackend(backend string) error {
	if backend != "cpu" && backend != "cuda" {
		return errors.New("Choose CPU or NVIDIA CUDA.")
	}
	a, ok := s.adapter.(backendInstaller)
	if !ok {
		return errors.New("This provider does not support explicit backend selection.")
	}
	// Fence a child whose endpoint was cleared but whose Job Object is still
	// draining, and prevent another owner starting until publication finishes.
	owner := s.providerProcess
	if !owner.mu.TryLock() {
		return errProviderRunning
	}
	if owner.process != nil {
		select {
		case <-owner.process.done:
			owner.process = nil
		default:
			owner.mu.Unlock()
			return errProviderRunning
		}
	}
	err := s.install(func(ctx context.Context, progress func(float64)) (string, error) {
		defer owner.mu.Unlock()
		return a.InstallBackend(ctx, backend, progress)
	}, true)
	if err != nil {
		owner.mu.Unlock()
	}
	return err
}
func (s *worker) Install() error {
	return s.install(s.adapter.Install, false)
}
func (s *worker) install(install func(context.Context, func(float64)) (string, error), guard bool) error {
	return s.run("install", "installing", guard, func(ctx context.Context) error {
		s.mu.Lock()
		running := s.process != nil
		s.mu.Unlock()
		if running {
			return errBusy
		}
		_, err := install(ctx, func(p float64) {
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
func (s *worker) RefreshCatalog() error { return s.run("catalog", "", false, s.inspect) }
func (s *worker) DownloadModel(id string) error {
	if !s.supportsModel(id) {
		return errors.New("Choose a supported managed speech model.")
	}
	return s.runOperation("download", "installing", false, id, func(ctx context.Context) error {
		if err := s.adapter.Pull(ctx, id, func(p AcquisitionProgress) { s.acquisitionProgress(ctx, p) }); err != nil {
			return err
		}
		if err := s.inspect(ctx); err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, model := range s.status.Models {
			if model.ID == id && model.Installed {
				return nil
			}
		}
		return errIntegrity
	})
}
func (s *worker) RemoveModel(id string) error {
	if !s.supportsModel(id) {
		return errors.New("Choose a supported managed speech model.")
	}
	return s.run("remove_model", "", true, func(ctx context.Context) error {
		s.mu.Lock()
		selected := s.configuration.Model == id
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
func (s *worker) Remove() error {
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
func (s *worker) startProcess(ctx context.Context) error {
	s.mu.Lock()
	p := s.configuration
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
	proc, endpoint, err := s.providerProcess.start(ctx, s.adapter, p.Model)
	if err != nil {
		if proc != nil {
			s.discardProcess(proc)
		}
		return err
	}
	if proc == nil || !endpoint.Enabled || endpoint.BaseURL == "" || endpoint.Model == "" {
		if proc != nil {
			s.discardProcess(proc)
		}
		return errNotReady
	}
	s.mu.Lock()
	if s.closed || ctx.Err() != nil || generation != s.generation {
		s.mu.Unlock()
		s.discardProcess(proc)
		return context.Canceled
	}
	s.generation = leaseGeneration.Add(1) // Every newly owned endpoint has a distinct lease generation.
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

// Called inside an admitted start operation, whose WaitGroup reservation is
// still held. Rejected launches remain owned until the OS confirms their exit.
func (s *worker) discardProcess(proc *ownedProcess) {
	s.wg.Add(1)
	proc.kill()
	go func() {
		defer s.wg.Done()
		<-proc.done
	}()
}
func (s *worker) Start() error { return s.run("start", "starting", true, s.startProcess) }
func (s *worker) stopProcess(ctx context.Context) error {
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
func (s *worker) Stop() error { return s.run("stop", "stopping", true, s.stopProcess) }
func (s *worker) Cancel() error {
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
func (s *worker) startup(ctx context.Context) error {
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
		ready := s.configuration.Enabled && s.configuration.AutoStart
		installed := false
		for _, m := range s.status.Models {
			if m.ID == s.configuration.Model && m.Installed {
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
func (s *worker) closeNow() {
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
}

func (s *worker) supportsModel(id string) bool {
	for _, model := range s.provider.descriptor().Models {
		if model.ID == id {
			return true
		}
	}
	return false
}
func (s *worker) ServiceShutdown() error {
	s.closeNow()
	return waitWorkers([]*worker{s}, 8*time.Second)
}
func waitWorkers(workers []*worker, timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		for _, w := range workers {
			w.wg.Wait()
		}
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("Managed runtime shutdown exceeded its deadline.")
	}
}
