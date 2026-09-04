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
	"github.com/tnware/freehand-stt/internal/hotkey"
	"github.com/tnware/freehand-stt/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const previewStepDuration = 1500 * time.Millisecond

type statusOverlay interface {
	SetLevelSource(platform.LevelSource)
	Configure(platform.OverlayOptions) error
	Update(platform.OverlayStatus) error
	Close() error
}

type PreviewRequest struct {
	Preferences    config.OverlayPreferences `json:"preferences"`
	ToggleShortcut string                    `json:"toggleShortcut"`
	HoldShortcut   string                    `json:"holdShortcut,omitempty"`
}

type runPresentation struct {
	generation  uint64
	startedAt   time.Time
	finishedAt  time.Time
	mode        platform.OverlayRecordingMode
	shortcut    string
	checkpoints int
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
	newLevels   func() platform.LevelSource
	logger      *slog.Logger
	rootContext context.Context
	started     bool
	closed      bool

	preview        bool
	previewRequest PreviewRequest
	previewKind    platform.OverlayKind
	previewVersion uint64
	previewCancel  context.CancelFunc
	previewWG      sync.WaitGroup
}

func NewService(settings config.Settings, newLevels func() platform.LevelSource, logger *slog.Logger) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &Service{
		settings:  settings,
		status:    dictation.Status{State: dictation.Idle},
		newLevels: newLevels,
		newOverlay: func() (statusOverlay, error) {
			return platform.NewStatusOverlay()
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
	s.status = status
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

// StartPreview starts or updates the native preview. It accepts only the
// validated presentation subset and normalized shortcut labels; it neither
// saves settings nor affects dictation state.
func (s *Service) StartPreview(request PreviewRequest) error {
	if err := config.ValidateOverlayPreferences(request.Preferences); err != nil {
		return err
	}
	toggle, err := normalizeShortcut(hotkey.ToggleRecording, request.ToggleShortcut)
	if err != nil {
		return errors.New("overlay preview toggle shortcut is invalid")
	}
	hold := ""
	if request.HoldShortcut != "" {
		hold, err = normalizeShortcut(hotkey.HoldToTalk, request.HoldShortcut)
		if err != nil {
			return errors.New("overlay preview hold shortcut is invalid")
		}
	}
	request.ToggleShortcut = toggle
	request.HoldShortcut = hold

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.started {
		return errors.New("native status overlay preview is unavailable")
	}
	if dictationActive(s.status) {
		return errors.New("finish the current dictation before previewing the overlay")
	}
	if _, err := s.ensureOverlayLocked(); err != nil {
		return errors.New("native status overlay preview could not be started")
	}
	s.previewRequest = request
	s.configureLocked(optionsFromPreferences(request.Preferences))
	if s.preview {
		s.updateLocked(s.previewPresentationLocked(s.previewKind))
		return nil
	}

	s.preview = true
	s.previewVersion++
	version := s.previewVersion
	ctx, cancel := context.WithCancel(s.rootContext)
	s.previewCancel = cancel
	s.previewKind = platform.OverlayRecordingSpeech
	s.updateLocked(s.previewPresentationLocked(s.previewKind))
	s.previewWG.Add(1)
	go s.runPreview(ctx, version)
	s.logger.Info("native status overlay preview started")
	return nil
}

func (s *Service) StopPreview() error {
	s.mu.Lock()
	wasPreviewing := s.preview
	s.stopPreviewLocked()
	if !wasPreviewing {
		s.mu.Unlock()
		return nil
	}
	if s.settings.OverlayEnabled {
		s.configureLocked(optionsFromPreferences(s.settings.OverlayPreferences()))
		s.updateLocked(s.presentationLocked(s.status))
		s.mu.Unlock()
	} else {
		overlay := s.overlay
		s.overlay = nil
		s.mu.Unlock()
		if err := s.closeOverlay(overlay); err != nil {
			return err
		}
	}
	s.logger.Info("native status overlay preview stopped")
	return nil
}

func (s *Service) runPreview(ctx context.Context, version uint64) {
	defer s.previewWG.Done()
	kinds := []platform.OverlayKind{
		platform.OverlayRecordingSpeech,
		platform.OverlayRecordingSilence,
		platform.OverlayRecordingCountdown,
		platform.OverlayTranscribing,
		platform.OverlayPostProcessing,
		platform.OverlayReady,
		platform.OverlayCopyRequired,
		platform.OverlayFailed,
	}
	ticker := time.NewTicker(previewStepDuration)
	defer ticker.Stop()
	index := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.preview || s.previewVersion != version || s.closed {
				s.mu.Unlock()
				return
			}
			visible := previewKinds(s.previewRequest.Preferences.Visibility, kinds)
			if len(visible) == 0 {
				s.mu.Unlock()
				continue
			}
			index = (index + 1) % len(visible)
			s.previewKind = visible[index]
			s.updateLocked(s.previewPresentationLocked(s.previewKind))
			s.mu.Unlock()
		}
	}
}

func (s *Service) stopPreviewLocked() {
	if !s.preview {
		return
	}
	s.preview = false
	s.previewVersion++
	if s.previewCancel != nil {
		s.previewCancel()
	}
	s.previewCancel = nil
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

func (s *Service) configureLocked(options platform.OverlayOptions) {
	if s.overlay == nil {
		return
	}
	if err := s.overlay.Configure(options); err != nil {
		s.logger.Warn("native status overlay configuration failed", "error_kind", diagnostics.ErrorKind(err))
	}
}

func (s *Service) updateLocked(status platform.OverlayStatus) {
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

func (s *Service) trackRunLocked(status dictation.Status) {
	if status.State == dictation.Recording && s.run.generation != status.Generation {
		s.run = runPresentation{
			generation: status.Generation,
			startedAt:  status.StartedAt,
			mode:       overlayRecordingMode(status.RecordingMode),
			shortcut:   shortcutForMode(s.settings, status.RecordingMode),
		}
	}
	if status.State == dictation.Recording && status.SegmentNumber > s.run.checkpoints {
		s.run.checkpoints = status.SegmentNumber
	}
	if (status.State == dictation.Ready || status.State == dictation.Failed) && s.run.finishedAt.IsZero() {
		s.run.finishedAt = time.Now()
	}
}

func (s *Service) presentationLocked(status dictation.Status) platform.OverlayStatus {
	presentation := overlayForStatus(status)
	if !visibilityIncludes(s.settings.OverlayVisibility, presentation.Kind) {
		presentation.Kind = platform.OverlayHidden
	}
	if presentation.Generation == 0 {
		presentation.Generation = s.run.generation
	}
	presentation.StartedAt = s.run.startedAt
	presentation.FinishedAt = s.run.finishedAt
	presentation.RecordingMode = s.run.mode
	presentation.Shortcut = s.run.shortcut
	presentation.Checkpoints = s.run.checkpoints
	return presentation
}

func (s *Service) previewPresentationLocked(kind platform.OverlayKind) platform.OverlayStatus {
	now := time.Now()
	status := platform.OverlayStatus{
		Kind:          kind,
		Generation:    ^uint64(0),
		Preview:       true,
		StartedAt:     now.Add(-12 * time.Second),
		RecordingMode: platform.OverlayRecordingToggle,
		Shortcut:      s.previewRequest.ToggleShortcut,
		Checkpoints:   2,
	}
	if kind == platform.OverlayRecordingCountdown {
		status.CountdownDuration = 3 * time.Second
		status.CountdownDeadline = now.Add(2 * time.Second)
	}
	return status
}

func overlayForStatus(status dictation.Status) platform.OverlayStatus {
	kind := platform.OverlayHidden
	switch status.State {
	case dictation.Recording:
		switch {
		case status.AutoStopState == dictation.AutoStopCountdown:
			kind = platform.OverlayRecordingCountdown
		case status.VADState == dictation.VADSpeech:
			kind = platform.OverlayRecordingSpeech
		case status.VADState == dictation.VADSilence:
			kind = platform.OverlayRecordingSilence
		default:
			kind = platform.OverlayRecording
		}
	case dictation.Transcribing:
		kind = platform.OverlayTranscribing
	case dictation.PostProcessing:
		kind = platform.OverlayPostProcessing
	case dictation.Ready:
		kind = platform.OverlayReady
	case dictation.Cancelling:
		kind = platform.OverlayCancelling
	case dictation.Failed:
		if status.CanCopy {
			kind = platform.OverlayCopyRequired
		} else {
			kind = platform.OverlayFailed
		}
	}
	return platform.OverlayStatus{
		Kind:              kind,
		Generation:        status.Generation,
		CountdownDeadline: status.AutoStopDeadline,
		CountdownDuration: time.Duration(status.AutoStopDurationMilliseconds) * time.Millisecond,
	}
}

func optionsFromPreferences(preferences config.OverlayPreferences) platform.OverlayOptions {
	return platform.OverlayOptions{
		Layout:     overlayLayout(preferences.Layout),
		Anchor:     overlayAnchor(preferences.Anchor),
		Motion:     overlayMotion(preferences.Motion),
		Surface:    overlaySurface(preferences.Surface),
		Visualizer: overlayVisualizer(preferences.Visualizer),
		Scale:      float64(preferences.SizePercent) / 100,
		Opacity:    float64(preferences.OpacityPercent) / 100,
		EdgeOffset: int32(preferences.EdgeOffset),
		Glow:       float64(preferences.GlowPercent) / 100,
	}
}

func overlayLayout(value config.OverlayLayout) platform.OverlayLayout {
	switch value {
	case config.OverlayLayoutMinimal:
		return platform.OverlayLayoutMinimal
	case config.OverlayLayoutMeter:
		return platform.OverlayLayoutMeter
	case config.OverlayLayoutDetailed:
		return platform.OverlayLayoutDetailed
	default:
		return platform.OverlayLayoutCapsule
	}
}

func overlayAnchor(value config.OverlayAnchor) platform.OverlayAnchor {
	switch value {
	case config.OverlayAnchorTopLeft:
		return platform.OverlayAnchorTopLeft
	case config.OverlayAnchorTopRight:
		return platform.OverlayAnchorTopRight
	case config.OverlayAnchorBottomLeft:
		return platform.OverlayAnchorBottomLeft
	case config.OverlayAnchorBottomCenter:
		return platform.OverlayAnchorBottomCenter
	case config.OverlayAnchorBottomRight:
		return platform.OverlayAnchorBottomRight
	default:
		return platform.OverlayAnchorTopCenter
	}
}

func overlayMotion(value config.OverlayMotion) platform.OverlayMotion {
	if value == config.OverlayMotionReduced {
		return platform.OverlayMotionReduced
	}
	return platform.OverlayMotionSystem
}

func overlaySurface(value config.OverlaySurface) platform.OverlaySurface {
	switch value {
	case config.OverlaySurfaceSolid:
		return platform.OverlaySurfaceSolid
	case config.OverlaySurfaceMinimal:
		return platform.OverlaySurfaceMinimal
	default:
		return platform.OverlaySurfaceGlass
	}
}

func overlayVisualizer(value config.OverlayVisualizer) platform.OverlayVisualizer {
	switch value {
	case config.OverlayVisualizerPulse:
		return platform.OverlayVisualizerPulse
	case config.OverlayVisualizerEnvelope:
		return platform.OverlayVisualizerEnvelope
	case config.OverlayVisualizerMeter:
		return platform.OverlayVisualizerMeter
	default:
		return platform.OverlayVisualizerBars
	}
}

func overlayRecordingMode(mode dictation.RecordingMode) platform.OverlayRecordingMode {
	if mode == dictation.RecordingHold {
		return platform.OverlayRecordingHold
	}
	return platform.OverlayRecordingToggle
}

func shortcutForMode(settings config.Settings, mode dictation.RecordingMode) string {
	value := settings.ToggleShortcut
	action := hotkey.ToggleRecording
	if mode == dictation.RecordingHold && settings.HoldShortcut != "" {
		value = settings.HoldShortcut
		action = hotkey.HoldToTalk
	}
	normalized, err := normalizeShortcut(action, value)
	if err != nil {
		return ""
	}
	return normalized
}

func normalizeShortcut(action hotkey.ShortcutAction, value string) (string, error) {
	return hotkey.NormalizeFor(action, value)
}

func dictationActive(status dictation.Status) bool {
	return status.State != dictation.Idle && status.State != dictation.Failed
}

func previewKinds(visibility config.OverlayVisibility, kinds []platform.OverlayKind) []platform.OverlayKind {
	result := make([]platform.OverlayKind, 0, len(kinds))
	for _, kind := range kinds {
		if visibilityIncludes(visibility, kind) {
			result = append(result, kind)
		}
	}
	return result
}

func visibilityIncludes(visibility config.OverlayVisibility, kind platform.OverlayKind) bool {
	switch kind {
	case platform.OverlayRecording, platform.OverlayRecordingSpeech, platform.OverlayRecordingSilence, platform.OverlayRecordingCountdown:
		return true
	case platform.OverlayTranscribing, platform.OverlayPostProcessing, platform.OverlayCancelling:
		return visibility != config.OverlayVisibilityRecording
	case platform.OverlayReady, platform.OverlayCopyRequired, platform.OverlayFailed:
		return visibility == config.OverlayVisibilityAll
	default:
		return false
	}
}
