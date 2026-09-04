// Package updates owns Freehand's application-update policy. Wails owns the
// provider, download, verification, swap, and update window; this service adds
// the persisted opt-in, quiet no-update checks, renderer status, and lifecycle.
package updates

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

const (
	StatusEvent       = "updates:status"
	initialCheckDelay = 30 * time.Second
	checkInterval     = 24 * time.Hour
)

type State string

const (
	StateDevelopment State = "development"
	StateDisabled    State = "disabled"
	StateIdle        State = "idle"
	StateChecking    State = "checking"
	StateCurrent     State = "current"
	StateAvailable   State = "available"
	StateError       State = "error"
)

// Status is the renderer-safe update summary. Provider errors are classified
// rather than returned verbatim because HTTP errors can contain request URLs.
type Status struct {
	State          State  `json:"state"`
	Enabled        bool   `json:"enabled"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion,omitempty"`
	LastCheckedAt  string `json:"lastCheckedAt,omitempty"`
	ErrorKind      string `json:"errorKind,omitempty"`
}

type Checker interface {
	Check(context.Context) (*updater.Release, error)
	CheckAndInstall(context.Context) error
}

type Option func(*Service)

// WithSchedule is intended for deterministic tests.
func WithSchedule(first, interval time.Duration) Option {
	return func(service *Service) {
		service.initialDelay = first
		service.interval = interval
	}
}

type Service struct {
	mu              sync.RWMutex
	workerMu        sync.Mutex
	checker         Checker
	status          Status
	development     bool
	publish         func(Status)
	logger          *slog.Logger
	initialDelay    time.Duration
	interval        time.Duration
	lifetime        context.Context
	cancel          context.CancelFunc
	wake            chan struct{}
	wait            sync.WaitGroup
	checkInProgress atomic.Bool
	closed          atomic.Bool
}

func NewService(currentVersion string, enabled, development bool, publish func(Status), logger *slog.Logger, options ...Option) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	state := StateIdle
	if development {
		state = StateDevelopment
	} else if !enabled {
		state = StateDisabled
	}
	service := &Service{
		status: Status{
			State:          state,
			Enabled:        enabled,
			CurrentVersion: currentVersion,
		},
		development:  development,
		publish:      publish,
		logger:       logger.With("component", "updates"),
		initialDelay: initialCheckDelay,
		interval:     checkInterval,
		wake:         make(chan struct{}, 1),
	}
	for _, option := range options {
		option(service)
	}
	return service
}

// Configure attaches the Wails updater after application.New has created it.
// It is an ordinary Go composition helper rather than a renderer binding and
// must be called before Wails starts registered services.
func Configure(s *Service, checker Checker) {
	s.mu.Lock()
	s.checker = checker
	s.mu.Unlock()
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if s.development {
		return nil
	}
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.closed.Load() {
		return errors.New("the update service is shutting down")
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return nil
	}
	s.lifetime, s.cancel = context.WithCancel(ctx)
	lifetime := s.lifetime
	s.mu.Unlock()
	s.wait.Add(1)
	go s.run(lifetime)
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.workerMu.Lock()
	s.closed.Store(true)
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.workerMu.Unlock()
	s.wait.Wait()
	return nil
}

// Current returns the last bounded update-check status.
func (s *Service) Current() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// ApplyEnabled updates the runtime polling policy after settings commit. It is
// an ordinary Go composition helper rather than a renderer binding.
func ApplyEnabled(s *Service, enabled bool) {
	s.mu.Lock()
	s.status.Enabled = enabled
	if s.development {
		s.status.State = StateDevelopment
	} else if !enabled {
		s.status.State = StateDisabled
		s.status.ErrorKind = ""
	} else if s.status.State == StateDisabled {
		s.status.State = StateIdle
	}
	status := s.status
	s.mu.Unlock()
	s.publishStatus(status)
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// CheckForUpdates opens Wails' first-party updater window and lets the user
// review, skip, stage, and restart into an update. It starts asynchronously so
// a native update window never blocks the calling WebView binding.
func (s *Service) CheckForUpdates() error {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.closed.Load() {
		return errors.New("the update service is shutting down")
	}
	s.mu.RLock()
	checker := s.checker
	lifetime := s.lifetime
	s.mu.RUnlock()
	if s.development {
		return errors.New("update checks are available in packaged builds")
	}
	if checker == nil || lifetime == nil {
		return errors.New("the update service is not ready")
	}
	if !s.checkInProgress.CompareAndSwap(false, true) {
		return errors.New("an update check is already running")
	}
	s.setChecking()
	s.wait.Add(1)
	go func() {
		defer s.wait.Done()
		defer s.checkInProgress.Store(false)
		started := time.Now()
		s.logger.Info("interactive update check started")
		err := checker.CheckAndInstall(lifetime)
		s.finishInteractive(err)
		if err != nil {
			s.logger.Warn("interactive update check failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
			return
		}
		s.logger.Info("interactive update check completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "presented")
	}()
	return nil
}

func (s *Service) run(ctx context.Context) {
	defer s.wait.Done()
	delay := s.initialDelay
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			delay = s.initialDelay
			timer.Reset(delay)
		case <-timer.C:
			if s.enabled() {
				s.backgroundCheck(ctx)
			}
			delay = s.interval
			timer.Reset(delay)
		}
	}
}

func (s *Service) enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status.Enabled
}

func (s *Service) backgroundCheck(ctx context.Context) {
	if !s.checkInProgress.CompareAndSwap(false, true) {
		return
	}
	defer s.checkInProgress.Store(false)
	s.mu.RLock()
	checker := s.checker
	s.mu.RUnlock()
	if checker == nil {
		return
	}
	s.setChecking()
	started := time.Now()
	s.logger.Info("automatic update check started")
	release, err := checker.Check(ctx)
	s.finishBackground(release, err)
	if err != nil {
		s.logger.Warn("automatic update check failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return
	}
	outcome := "current"
	if release != nil {
		outcome = "available"
	}
	s.logger.Info("automatic update check completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", outcome)
	if release == nil || !s.enabled() {
		return
	}

	// Keep the routine check invisible when Freehand is current, but do not
	// hide a discovered update in About. Wails owns the review, verified
	// download, staging, and restart window. CheckAndInstall performs a fresh
	// provider check so that Wails owns one coherent interactive session.
	presented := time.Now()
	s.logger.Info("automatic update presentation started")
	err = checker.CheckAndInstall(ctx)
	s.finishInteractive(err)
	if err != nil {
		s.logger.Warn("automatic update presentation failed", "duration_ms", time.Since(presented).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return
	}
	s.logger.Info("automatic update presentation completed", "duration_ms", time.Since(presented).Milliseconds(), "outcome", "presented")
}

func (s *Service) setChecking() {
	s.mu.Lock()
	s.status.State = StateChecking
	s.status.ErrorKind = ""
	status := s.status
	s.mu.Unlock()
	s.publishStatus(status)
}

func (s *Service) finishBackground(release *updater.Release, err error) {
	s.mu.Lock()
	if !s.status.Enabled {
		s.status.State = StateDisabled
		s.status.LatestVersion = ""
		s.status.ErrorKind = ""
		status := s.status
		s.mu.Unlock()
		s.publishStatus(status)
		return
	}
	s.status.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	s.status.LatestVersion = ""
	s.status.ErrorKind = ""
	s.status.State = StateCurrent
	if err != nil {
		s.status.State = StateError
		s.status.ErrorKind = diagnostics.ErrorKind(err)
	} else if release != nil {
		s.status.State = StateAvailable
		s.status.LatestVersion = boundedVersion(release.Version)
	}
	status := s.status
	s.mu.Unlock()
	s.publishStatus(status)
}

func (s *Service) finishInteractive(err error) {
	s.mu.Lock()
	s.status.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	s.status.ErrorKind = ""
	if err != nil {
		s.status.State = StateError
		s.status.ErrorKind = diagnostics.ErrorKind(err)
	} else if s.status.State == StateChecking {
		// Detailed outcome remains visible in Wails' updater window. Keep the
		// compact About summary truthful without duplicating that state machine.
		s.status.State = StateCurrent
	}
	status := s.status
	s.mu.Unlock()
	s.publishStatus(status)
}

func (s *Service) publishStatus(status Status) {
	if s.publish != nil {
		s.publish(status)
	}
}

func boundedVersion(version string) string {
	version = strings.TrimSpace(version)
	if len(version) > 64 || strings.ContainsFunc(version, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return ""
	}
	return version
}
