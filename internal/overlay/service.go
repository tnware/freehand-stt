// Package overlay owns Freehand's passive native status surface. It translates
// dictation state into presentation-only native state and exposes only the
// renderer-safe preview controls to Wails.
package overlay

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/dictation"
	nativeoverlay "github.com/tnware/freehand-stt/internal/platform/overlay"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const previewStepDuration = 1500 * time.Millisecond
const outcomeDisplayDuration = 5 * time.Second

type statusOverlay interface {
	SetLevelSource(nativeoverlay.LevelSource)
	Configure(nativeoverlay.OverlayOptions) error
	Update(nativeoverlay.OverlayStatus) error
	Close() error
}

// Service is shared by Wails and the application publishers. Its mutex guards
// the native surface and preview state; all HWND work remains serialized by
// the platform implementation's dedicated message-loop thread.
type Service struct {
	mu          sync.Mutex
	settings    config.Settings
	status      dictation.Status
	run         runPresentation
	overlay     statusOverlay
	newOverlay  func() (statusOverlay, error)
	newLevels   func() nativeoverlay.LevelSource
	logger      *slog.Logger
	rootContext context.Context
	started     bool
	closed      bool

	outcomeTimer     *time.Timer
	outcomeVersion   uint64
	outcomeDismissed bool

	preview        bool
	previewRequest PreviewRequest
	previewKind    nativeoverlay.OverlayKind
	previewVersion uint64
	previewCancel  context.CancelFunc
	previewWG      sync.WaitGroup
}

func NewService(settings config.Settings, newLevels func() nativeoverlay.LevelSource, logger *slog.Logger) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &Service{
		settings:  settings,
		status:    dictation.Status{State: dictation.Idle},
		newLevels: newLevels,
		newOverlay: func() (statusOverlay, error) {
			return nativeoverlay.NewStatusOverlay()
		},
		logger:      logger.With("component", "overlay"),
		rootContext: context.Background(),
	}
}

// Start is called from ApplicationStarted so the native overlay never becomes
// the first Windows message loop created by the process.
func Start(service *Service) {
	if service != nil {
		service.start()
	}
}

// ApplySettings is an ordinary Go entry point, not an exported service method,
// so application wiring does not widen the renderer binding surface.
func ApplySettings(service *Service, settings config.Settings) {
	if service != nil {
		service.applySettings(settings)
	}
}

// ApplyStatus translates the authoritative dictation snapshot. Preview never
// becomes a second state machine and is preempted by any real operation.
func ApplyStatus(service *Service, status dictation.Status) {
	if service != nil {
		service.applyStatus(status)
	}
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	s.rootContext = ctx
	s.closed = false
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.started = false
	s.stopOutcomeTimerLocked()
	s.stopPreviewLocked()
	overlay := s.overlay
	s.overlay = nil
	s.mu.Unlock()
	s.previewWG.Wait()
	return s.closeOverlay(overlay)
}

func (s *Service) start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.started {
		return
	}
	s.started = true
	if !s.settings.OverlayEnabled {
		return
	}
	if _, err := s.ensureOverlayLocked(); err != nil {
		s.logger.Warn("native status overlay start failed", "error_kind", diagnostics.ErrorKind(err))
		return
	}
	s.configureLocked(optionsFromPreferences(s.settings.OverlayPreferences()))
	s.updateLocked(s.presentationLocked(s.status))
}

func (s *Service) applySettings(settings config.Settings) {
	s.mu.Lock()
	s.settings = settings
	if !s.started || s.closed || s.preview {
		s.mu.Unlock()
		return
	}
	if !settings.OverlayEnabled {
		overlay := s.overlay
		s.overlay = nil
		s.mu.Unlock()
		_ = s.closeOverlay(overlay)
		return
	}
	if _, err := s.ensureOverlayLocked(); err != nil {
		s.logger.Warn("native status overlay start failed", "error_kind", diagnostics.ErrorKind(err))
		s.mu.Unlock()
		return
	}
	s.configureLocked(optionsFromPreferences(settings.OverlayPreferences()))
	s.updateLocked(s.presentationLocked(s.status))
	s.mu.Unlock()
}

func (s *Service) applyStatus(status dictation.Status) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.status = status
	s.scheduleOutcomeDismissalLocked(status)
	s.trackRunLocked(status)
	preempted := false
	if status.State != dictation.Idle && s.preview {
		s.stopPreviewLocked()
		preempted = true
		s.logger.Info("native status overlay preview preempted", "reason", "dictation_started")
	}
	if !s.started || s.closed {
		s.mu.Unlock()
		return
	}
	if !s.settings.OverlayEnabled {
		overlay := statusOverlay(nil)
		if preempted {
			overlay = s.overlay
			s.overlay = nil
		}
		s.mu.Unlock()
		_ = s.closeOverlay(overlay)
		return
	}
	if _, err := s.ensureOverlayLocked(); err != nil {
		s.logger.Warn("native status overlay start failed", "error_kind", diagnostics.ErrorKind(err))
		s.mu.Unlock()
		return
	}
	s.configureLocked(optionsFromPreferences(s.settings.OverlayPreferences()))
	s.updateLocked(s.presentationLocked(status))
	s.mu.Unlock()
}

func (s *Service) ensureOverlayLocked() (statusOverlay, error) {
	if s.overlay != nil {
		return s.overlay, nil
	}
	if s.newOverlay == nil {
		return nil, errors.New("native status overlay factory is unavailable")
	}
	overlay, err := s.newOverlay()
	if err != nil {
		return nil, err
	}
	if s.newLevels != nil {
		overlay.SetLevelSource(s.newLevels())
	}
	s.overlay = overlay
	s.logger.Info("native status overlay started")
	return overlay, nil
}

func (s *Service) configureLocked(options nativeoverlay.OverlayOptions) {
	if s.overlay == nil {
		return
	}
	if err := s.overlay.Configure(options); err != nil {
		s.logger.Warn("native status overlay configuration failed", "error_kind", diagnostics.ErrorKind(err))
	}
}

func (s *Service) updateLocked(status nativeoverlay.OverlayStatus) {
	if s.overlay == nil {
		return
	}
	if err := s.overlay.Update(status); err != nil {
		s.logger.Warn("native status overlay update failed", "error_kind", diagnostics.ErrorKind(err))
	}
}

func (s *Service) closeOverlay(overlay statusOverlay) error {
	if overlay == nil {
		return nil
	}
	if err := overlay.Close(); err != nil {
		s.logger.Warn("native status overlay shutdown failed", "error_kind", diagnostics.ErrorKind(err))
		return err
	}
	s.logger.Info("native status overlay stopped")
	return nil
}

// The timer owns presentation only: it never clears dictation or copy recovery.
// Version fencing handles callbacks already queued when Stop returns false.
func (s *Service) stopOutcomeTimerLocked() {
	s.outcomeVersion++
	if s.outcomeTimer != nil {
		s.outcomeTimer.Stop()
		s.outcomeTimer = nil
	}
}

func (s *Service) scheduleOutcomeDismissalLocked(status dictation.Status) {
	s.stopOutcomeTimerLocked()
	s.outcomeDismissed = false
	if status.State != dictation.Failed || s.rootContext.Err() != nil {
		return
	}
	version := s.outcomeVersion
	s.outcomeTimer = time.AfterFunc(outcomeDisplayDuration, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.rootContext.Err() != nil || version != s.outcomeVersion {
			return
		}
		s.outcomeTimer = nil
		s.outcomeDismissed = true
		if s.started && s.settings.OverlayEnabled && !s.preview {
			s.updateLocked(s.presentationLocked(s.status))
		}
	})
}
