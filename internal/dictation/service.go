package dictation

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service owns the complete live-dictation capability. recorder is an
// unexported implementation detail in this same package, not a shared runtime
// or a second service layer.
type Service struct {
	recorder    *recorder
	settings    settings.Source
	fileActive  func() bool
	activity    *sync.Mutex
	closed      atomic.Bool
	workerMu    sync.Mutex
	completion  chan func()
	workerDone  chan struct{}
	beforeStart func()
}

func NewService(capture audio.Capture, input insertion.Platform, client *inference.Client, processor *postprocess.Processor, source settings.Source, profiles settings.ProfileSource, transcripts *history.Store, fileActive func() bool, activity *sync.Mutex, changed func(Status), logger *slog.Logger) *Service {
	if activity == nil {
		activity = &sync.Mutex{}
	}
	if transcripts == nil {
		transcripts = history.NewStore(source.Current().HistoryEnabled, input)
	}
	service := &Service{
		recorder:   newRecorder(capture, input, client, processor, source, profiles, transcripts, changed, logger),
		settings:   source,
		fileActive: fileActive,
		activity:   activity,
	}
	service.recorder.scheduleCompletion = service.scheduleCompletion
	return service
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closed.Store(false)
	s.recorder.setRootContext(ctx)
	s.workerMu.Lock()
	if s.completion != nil {
		s.workerMu.Unlock()
		return nil
	}
	s.completion = make(chan func(), 1)
	queue := s.completion
	s.workerDone = make(chan struct{})
	done := s.workerDone
	s.workerMu.Unlock()
	go func() {
		defer close(done)
		for complete := range queue {
			complete()
		}
	}()
	return nil
}

func (s *Service) ServiceShutdown() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	closeErr := s.recorder.close()
	s.workerMu.Lock()
	if s.completion != nil {
		close(s.completion)
		s.completion = nil
	}
	done := s.workerDone
	s.workerDone = nil
	s.workerMu.Unlock()
	if done == nil {
		return closeErr
	}
	select {
	case <-done:
		return closeErr
	case <-time.After(5 * time.Second):
		return errors.Join(closeErr, errors.New("dictation completion shutdown exceeded the service deadline"))
	}
}

func (s *Service) scheduleCompletion(complete func()) bool {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.closed.Load() || s.completion == nil {
		return false
	}
	select {
	case s.completion <- complete:
		return true
	default:
		return false
	}
}

// StartRecording begins microphone dictation with an explicit control mode.
// Hold-to-talk recordings end on key release and do not inherit toggle mode's
// automatic silence boundary.
func (s *Service) StartRecording(mode RecordingMode) error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.activity.Lock()
	defer s.activity.Unlock()
	if s.settings == nil || !s.settings.Current().SetupCompleted {
		return errors.New("complete setup before starting a recording")
	}
	if s.fileActive != nil && s.fileActive() {
		return errors.New("an audio file is being transcribed")
	}
	if s.beforeStart != nil {
		s.beforeStart()
	}
	return s.recorder.start(mode)
}

func (s *Service) StopRecording() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	return s.recorder.stopCurrent()
}

func (s *Service) Cancel() error { return s.recorder.cancelRecording() }

func (s *Service) CopyPending() error { return s.recorder.copyPending() }

func (s *Service) CurrentStatus() Status { return s.recorder.currentStatus() }

// Snapshot is for backend collaborators and the composition root. Keeping it a
// package function avoids adding an internal-only method to the Wails surface.
func Snapshot(service *Service) Status {
	if service == nil || service.recorder == nil {
		return Status{State: Idle}
	}
	return service.recorder.currentStatus()
}

// SetBeforeRecording installs a backend-only pre-capture hook without adding
// another renderer-visible method to the Wails service.
func SetBeforeRecording(service *Service, hook func()) {
	if service != nil {
		service.beforeStart = hook
	}
}
