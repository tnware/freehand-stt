package input

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const ShortcutCaptureProgressEvent = "shortcut:capture-progress"

type ShortcutCaptureOutcome string

const (
	ShortcutCaptured  ShortcutCaptureOutcome = "captured"
	ShortcutCancelled ShortcutCaptureOutcome = "cancelled"
	ShortcutRejected  ShortcutCaptureOutcome = "rejected"
)

type ShortcutCaptureRequest struct {
	Action      hotkey.ShortcutAction      `json:"action"`
	Assignments hotkey.ShortcutAssignments `json:"assignments"`
}

type ShortcutCaptureResult struct {
	Outcome           ShortcutCaptureOutcome       `json:"outcome"`
	Shortcut          string                       `json:"shortcut,omitempty"`
	Changed           bool                         `json:"changed"`
	RejectionKind     hotkey.ShortcutRejectionKind `json:"rejectionKind,omitempty"`
	Message           string                       `json:"message,omitempty"`
	ConflictingAction hotkey.ShortcutAction        `json:"conflictingAction,omitempty"`
}

type ShortcutCaptureProgress struct {
	Action   hotkey.ShortcutAction `json:"action"`
	Shortcut string                `json:"shortcut"`
}

func init() {
	application.RegisterEvent[ShortcutCaptureProgress](ShortcutCaptureProgressEvent)
}

type ShortcutCapturer interface {
	Capture(context.Context, hotkey.ShortcutPolicy, func(hotkey.Chord)) (hotkey.Chord, bool, error)
	Cancel()
	Close() error
}

type ShortcutCaptureGuard interface {
	Suspend() error
	Resume() error
}

type Service struct {
	capture         audio.Capture
	shortcutCapture ShortcutCapturer
	shortcutGuard   ShortcutCaptureGuard
	activity        *activity.Coordinator
	settings        settings.Source
	shortcutChanged func(ShortcutCaptureProgress)
	logger          *slog.Logger
	captureMu       sync.Mutex
	lifecycleMu     sync.RWMutex
	rootContext     context.Context
	rootCancel      context.CancelFunc
	workers         sync.WaitGroup
	closed          atomic.Bool
}

func NewService(capture audio.Capture, shortcutCapture ShortcutCapturer, shortcutGuard ShortcutCaptureGuard, admission *activity.Coordinator, source settings.Source, shortcutChanged func(ShortcutCaptureProgress), logger *slog.Logger) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &Service{capture: capture, shortcutCapture: shortcutCapture, shortcutGuard: shortcutGuard, activity: admission, settings: source, shortcutChanged: shortcutChanged, logger: logger.With("component", "input")}
}

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return diagnostics.DiscardLogger()
}

func (s *Service) operationContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	s.lifecycleMu.RLock()
	root := s.rootContext
	s.lifecycleMu.RUnlock()
	if root == nil {
		root = context.Background()
	}
	ctx, cancel := context.WithTimeout(root, timeout)
	if s.closed.Load() {
		cancel()
	}
	return ctx, cancel
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closed.Store(false)
	s.lifecycleMu.Lock()
	s.rootContext, s.rootCancel = context.WithCancel(ctx)
	root := s.rootContext
	s.lifecycleMu.Unlock()
	preparable, ok := s.capture.(audio.PreparableCapture)
	if !ok {
		return nil
	}
	microphoneID := ""
	if s.settings != nil {
		microphoneID = s.settings.Current().MicrophoneID
	}
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		started := time.Now()
		s.log().Info("microphone preparation started")
		if err := preparable.Prepare(root, microphoneID); err != nil {
			if root.Err() != nil || s.closed.Load() {
				s.log().Info("microphone preparation cancelled", "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled")
			} else {
				s.log().Warn("microphone preparation failed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "failed", "error_kind", diagnostics.ErrorKind(err))
			}
			return
		}
		s.log().Info("microphone preparation completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "prepared")
	}()
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.closed.Store(true)
	s.activity.Close()
	s.lifecycleMu.Lock()
	cancel := s.rootCancel
	s.rootCancel = nil
	s.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if s.shortcutCapture != nil {
		s.shortcutCapture.Cancel()
	}
	var closeErr error
	if s.shortcutCapture != nil {
		closeErr = s.shortcutCapture.Close()
	}
	done := make(chan struct{})
	go func() { s.workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		return errors.Join(closeErr, errors.New("microphone preparation shutdown exceeded the service deadline"))
	}
	return closeErr
}

// Native input bindings. These expose inventory and capture capabilities but
// keep device access and keyboard-hook ownership in Go.

// ListMicrophones returns the current bounded capture-device inventory.
func (s *Service) ListMicrophones() ([]audio.Device, error) {
	if s.closed.Load() {
		return nil, errors.New("application is shutting down")
	}
	ctx, cancel := s.operationContext(5 * time.Second)
	defer cancel()
	return s.capture.List(ctx)
}

// ShortcutPolicies returns the native action matrix used by validation,
// capture, persistence, and registration.
func (s *Service) ShortcutPolicies() []hotkey.ShortcutPolicy { return hotkey.Policies() }

func shortcutRejection(err error) (ShortcutCaptureResult, bool) {
	rejection, ok := hotkey.RejectionDetails(err)
	if !ok {
		return ShortcutCaptureResult{}, false
	}
	return ShortcutCaptureResult{
		Outcome:           ShortcutRejected,
		RejectionKind:     rejection.Kind,
		Message:           rejection.Error(),
		ConflictingAction: rejection.ConflictingAction,
	}, true
}

func unavailable(message string) ShortcutCaptureResult {
	return ShortcutCaptureResult{
		Outcome:       ShortcutRejected,
		RejectionKind: hotkey.RejectionUnavailable,
		Message:       message,
	}
}

// CaptureShortcut records one action-specific chord. Expected user-input
// rejections resolve as structured results; native/lifecycle failures reject
// the binding promise as errors.
func (s *Service) CaptureShortcut(request ShortcutCaptureRequest) (result ShortcutCaptureResult, err error) {
	policy, knownAction := hotkey.PolicyFor(request.Action)
	if !knownAction {
		return ShortcutCaptureResult{
			Outcome:       ShortcutRejected,
			RejectionKind: hotkey.RejectionInvalidAction,
			Message:       "Shortcut action is invalid.",
		}, nil
	}
	started := time.Now()
	s.log().Info("shortcut capture started", "action", request.Action)
	defer func() {
		outcome := string(result.Outcome)
		if err != nil {
			outcome = "failed"
		}
		attrs := []any{"action", request.Action, "duration_ms", time.Since(started).Milliseconds(), "outcome", outcome}
		switch {
		case err != nil:
			attrs = append(attrs, "error_kind", diagnostics.ErrorKind(err))
			s.log().Warn("shortcut capture failed", attrs...)
		case result.Outcome == ShortcutCancelled:
			s.log().Info("shortcut capture cancelled", attrs...)
		case result.Outcome == ShortcutRejected:
			attrs = append(attrs, "rejection_kind", result.RejectionKind)
			s.log().Info("shortcut capture rejected", attrs...)
		default:
			s.log().Info("shortcut capture completed", attrs...)
		}
	}()
	if s.closed.Load() {
		return ShortcutCaptureResult{}, errors.New("application is shutting down")
	}
	if !s.captureMu.TryLock() {
		return unavailable("Another shortcut capture is already in progress."), nil
	}
	defer s.captureMu.Unlock()
	if err := s.activity.CheckShortcutCapture(); err != nil {
		if errors.Is(err, activity.ErrClosed) {
			return ShortcutCaptureResult{}, err
		}
		return unavailable(err.Error()), nil
	}
	if s.shortcutCapture == nil {
		return unavailable("Shortcut capture is unavailable in this build."), nil
	}
	if s.shortcutGuard == nil {
		return unavailable("Shortcut capture is not ready yet."), nil
	}
	if err := s.shortcutGuard.Suspend(); err != nil {
		return ShortcutCaptureResult{}, fmt.Errorf("shortcut capture could not start: %w", err)
	}
	ctx, cancel := s.operationContext(20 * time.Second)
	defer cancel()
	progress := func(chord hotkey.Chord) {
		if s.shortcutChanged == nil {
			return
		}
		s.shortcutChanged(ShortcutCaptureProgress{Action: request.Action, Shortcut: chord.String()})
	}
	chord, canceled, err := s.shortcutCapture.Capture(ctx, policy, progress)
	resumeErr := s.shortcutGuard.Resume()
	if err != nil {
		if rejected, ok := shortcutRejection(err); ok && resumeErr == nil {
			return rejected, nil
		}
		return ShortcutCaptureResult{}, fmt.Errorf("shortcut capture failed: %w", errors.Join(err, resumeErr))
	}
	if resumeErr != nil {
		return ShortcutCaptureResult{}, fmt.Errorf("shortcut capture finished but working shortcuts were not restored: %w", resumeErr)
	}
	if canceled {
		return ShortcutCaptureResult{Outcome: ShortcutCancelled, Message: "Capture cancelled. The current shortcut was kept."}, nil
	}
	shortcut := chord.String()
	if shortcut == "" {
		return ShortcutCaptureResult{}, errors.New("shortcut capture returned an unsupported chord")
	}
	if hotkey.AssignmentMatches(request.Action, chord, request.Assignments) {
		return ShortcutCaptureResult{
			Outcome:  ShortcutCaptured,
			Shortcut: shortcut,
			Message:  "This shortcut is already entered.",
		}, nil
	}
	if conflict, found := hotkey.FindConflict(request.Action, chord, request.Assignments); found {
		return ShortcutCaptureResult{
			Outcome:           ShortcutRejected,
			RejectionKind:     hotkey.RejectionDuplicate,
			Message:           fmt.Sprintf("%s already uses %s. Choose a different shortcut.", hotkey.ActionLabel(conflict), shortcut),
			ConflictingAction: conflict,
		}, nil
	}
	return ShortcutCaptureResult{
		Outcome:  ShortcutCaptured,
		Shortcut: shortcut,
		Changed:  true,
		Message:  "Captured. Save changes to let Windows register this shortcut.",
	}, nil
}

// CancelShortcutCapture cancels the active native shortcut capture, if any.
func (s *Service) CancelShortcutCapture() {
	if s.shortcutCapture != nil {
		s.shortcutCapture.Cancel()
	}
}
