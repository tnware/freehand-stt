package tts

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Phase string
type Source string

const (
	Idle       Phase = "idle"
	Generating Phase = "generating"
	Playing    Phase = "playing"
	Paused     Phase = "paused"
	Completed  Phase = "completed"
	Cancelled  Phase = "cancelled"
	Failed     Phase = "failed"
)

const (
	SourceHistory Source = "history"
	SourceFile    Source = "audio-file"
	SourcePreview Source = "preview"
	SourceCompose Source = "compose"
)

type Status struct {
	Generation           uint64                     `json:"generation"`
	Phase                Phase                      `json:"phase"`
	Source               Source                     `json:"source,omitempty"`
	HistoryID            uint64                     `json:"historyID,omitempty"`
	HistoryVersion       history.HistoryTextVersion `json:"historyVersion,omitempty"`
	PositionMilliseconds int64                      `json:"positionMilliseconds"`
	DurationMilliseconds int64                      `json:"durationMilliseconds"`
	Message              string                     `json:"message,omitempty"`
	ErrorKind            string                     `json:"errorKind,omitempty"`
	CanPause             bool                       `json:"canPause"`
	CanResume            bool                       `json:"canResume"`
	CanRestart           bool                       `json:"canRestart"`
	CanStop              bool                       `json:"canStop"`
	CanSave              bool                       `json:"canSave"`
	CanClear             bool                       `json:"canClear"`
}

type Player interface {
	Load([]byte, uint32, uint32) error
	Play() error
	Pause() error
	Restart() error
	Position() (int64, int64, bool)
	OutputName() string
	Save(string) error
	Stop() error
	Unload() error
	Close() error
}

type SpeechClient interface {
	SynthesizeSpeech(context.Context, string, string, inference.SpeechRequest) ([]byte, error)
}

type Service struct {
	control       sync.Mutex
	mu            sync.Mutex
	profiles      settings.TextToSpeechProfileSource
	client        SpeechClient
	player        Player
	history       *history.Store
	fileText      func() (string, error)
	saveFile      func() (string, error)
	captureActive func() bool
	changed       func(Status)
	logger        *slog.Logger
	rootContext   context.Context
	rootCancel    context.CancelFunc
	operation     context.CancelFunc
	status        Status
	generation    uint64
	workers       sync.WaitGroup
	closed        atomic.Bool
}

func NewService(profiles settings.TextToSpeechProfileSource, client SpeechClient, player Player, transcripts *history.Store, fileText func() (string, error), saveFile func() (string, error), captureActive func() bool, changed func(Status), logger *slog.Logger) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &Service{profiles: profiles, client: client, player: player, history: transcripts, fileText: fileText, saveFile: saveFile, captureActive: captureActive, changed: changed, logger: logger.With("component", "tts"), status: Status{Phase: Idle}}
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	s.rootContext, s.rootCancel = context.WithCancel(ctx)
	s.closed.Store(false)
	s.mu.Unlock()
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.control.Lock()
	if !s.closed.CompareAndSwap(false, true) {
		s.control.Unlock()
		return nil
	}
	s.mu.Lock()
	if s.operation != nil {
		s.operation()
		s.operation = nil
	}
	if s.rootCancel != nil {
		s.rootCancel()
		s.rootCancel = nil
	}
	s.mu.Unlock()
	_ = s.player.Stop()
	_ = s.player.Unload()
	s.control.Unlock()
	s.workers.Wait()
	return s.player.Close()
}

func (s *Service) CurrentStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) PlayHistoryEntry(id uint64, version history.HistoryTextVersion) error {
	if s.history == nil {
		return errors.New("transcript history is unavailable")
	}
	text, err := s.history.Text(id, version)
	if err != nil {
		return err
	}
	return s.start(text, SourceHistory, id, version)
}

func (s *Service) PlayFileTranscript() error {
	if s.fileText == nil {
		return errors.New("audio file transcript playback is unavailable")
	}
	text, err := s.fileText()
	if err != nil {
		return err
	}
	return s.start(text, SourceFile, 0, history.HistoryTextFinal)
}

func (s *Service) PreviewVoice() error {
	return s.start("This is Freehand's speech playback preview.", SourcePreview, 0, history.HistoryTextFinal)
}

// SpeakText generates speech for bounded user-authored text from the
// first-class Text to speech workspace. It never writes to transcript history.
func (s *Service) SpeakText(text string) error {
	return s.start(text, SourceCompose, 0, history.HistoryTextFinal)
}

func (s *Service) start(text string, source Source, historyID uint64, version history.HistoryTextVersion) error {
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	if s.captureActive != nil && s.captureActive() {
		return errors.New("finish the active transcription before starting speech playback")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("there is no transcript to read")
	}
	if utf8.RuneCountInString(text) > config.MaxTTSInputCharacters || len(text) > config.MaxTTSInputBytes {
		return errors.New("speech input is too long")
	}
	profile, err := s.profiles.Capture()
	if err != nil {
		return err
	}
	if err := config.ValidateTextToSpeech(profile.Settings, true); err != nil {
		return err
	}
	s.mu.Lock()
	if s.operation != nil {
		s.operation()
	}
	_ = s.player.Stop()
	_ = s.player.Unload()
	s.generation++
	generation := s.generation
	root := s.rootContext
	if root == nil {
		root = context.Background()
	}
	ctx, cancel := context.WithCancel(root)
	s.operation = cancel
	s.status = Status{Generation: generation, Phase: Generating, Source: source, HistoryID: historyID, HistoryVersion: version, Message: "Generating speech", CanStop: true}
	status := s.status
	s.mu.Unlock()
	s.publish(status)
	s.logger.Info("speech generation started", "generation", generation, "source", source, "input_characters", utf8.RuneCountInString(text), "timeout_seconds", profile.Settings.TimeoutSeconds)
	s.workers.Add(1)
	go s.generate(ctx, generation, profile, text)
	return nil
}

func (s *Service) generate(ctx context.Context, generation uint64, profile settings.TextToSpeechProfile, text string) {
	defer s.workers.Done()
	started := time.Now()
	requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(profile.Settings.TimeoutSeconds)*time.Second)
	audioBytes, err := s.client.SynthesizeSpeech(requestCtx, profile.Settings.BaseURL, profile.Credential, inference.SpeechRequest{Model: profile.Settings.Model, Voice: profile.Settings.Voice, Input: text, Speed: profile.Settings.Speed})
	requestCancel()
	profile.Credential = ""
	text = ""
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.finish(generation, Cancelled, "Speech playback stopped", "")
			return
		}
		s.logger.Warn("speech generation failed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		s.finish(generation, Failed, err.Error(), diagnostics.ErrorKind(err))
		return
	}
	if ctx.Err() != nil || !s.isCurrent(generation) {
		clear(audioBytes)
		return
	}
	pcm, err := decodeWAV(audioBytes)
	clear(audioBytes)
	pcmBytes := len(pcm.Data)
	if err == nil {
		s.control.Lock()
		if s.closed.Load() || ctx.Err() != nil || !s.isCurrent(generation) {
			s.control.Unlock()
			clear(pcm.Data)
			return
		}
		err = s.player.Load(pcm.Data, pcm.SampleRate, pcm.Channels)
		clear(pcm.Data)
		if err == nil {
			err = s.player.Play()
		}
		s.control.Unlock()
	}
	if err != nil {
		_ = s.player.Unload()
		s.logger.Warn("speech playback failed", "generation", generation, "error_kind", diagnostics.ErrorKind(err))
		s.finish(generation, Failed, err.Error(), diagnostics.ErrorKind(err))
		return
	}
	position, duration, _ := s.player.Position()
	s.update(generation, Status{Generation: generation, Phase: Playing, Source: s.source(generation), HistoryID: s.historyID(generation), HistoryVersion: s.historyVersion(generation), PositionMilliseconds: position, DurationMilliseconds: duration, Message: "Playing transcript", CanPause: true, CanRestart: true, CanStop: true, CanSave: true, CanClear: true})
	s.logger.Info("speech playback started", "generation", generation, "generation_ms", time.Since(started).Milliseconds(), "audio_ms", duration, "sample_rate", pcm.SampleRate, "channels", pcm.Channels, "pcm_bytes", pcmBytes, "output_device", s.player.OutputName())
	s.monitor(ctx, generation)
}

func (s *Service) monitor(ctx context.Context, generation uint64) {
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			position, duration, done := s.player.Position()
			if done {
				s.finish(generation, Completed, "Playback complete", "")
				return
			}
			s.mu.Lock()
			if s.status.Generation == generation {
				s.status.PositionMilliseconds = position
				s.status.DurationMilliseconds = duration
				status := s.status
				s.mu.Unlock()
				s.publish(status)
			} else {
				s.mu.Unlock()
				return
			}
		}
	}
}

func (s *Service) Pause() error {
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	generation := s.status.Generation
	valid := s.status.Phase == Playing
	s.mu.Unlock()
	if !valid {
		return errors.New("speech playback is not playing")
	}
	if err := s.player.Pause(); err != nil {
		return err
	}
	position, duration, _ := s.player.Position()
	s.mu.Lock()
	if s.status.Generation == generation {
		s.status.Phase = Paused
		s.status.PositionMilliseconds = position
		s.status.DurationMilliseconds = duration
		s.status.Message = "Playback paused"
		s.status.CanPause = false
		s.status.CanResume = true
		status := s.status
		s.mu.Unlock()
		s.publish(status)
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) Resume() error {
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	generation := s.status.Generation
	valid := s.status.Phase == Paused
	s.mu.Unlock()
	if !valid {
		return errors.New("speech playback is not paused")
	}
	if err := s.player.Play(); err != nil {
		return err
	}
	s.mu.Lock()
	if s.status.Generation == generation {
		s.status.Phase = Playing
		s.status.Message = "Playing transcript"
		s.status.CanPause = true
		s.status.CanResume = false
		status := s.status
		s.mu.Unlock()
		s.publish(status)
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) Restart() error {
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	generation := s.status.Generation
	valid := s.status.CanRestart
	if !valid {
		s.mu.Unlock()
		return errors.New("speech playback cannot be restarted")
	}
	root := s.rootContext
	if root == nil {
		root = context.Background()
	}
	if s.operation != nil {
		s.operation()
	}
	ctx, cancel := context.WithCancel(root)
	s.operation = cancel
	s.mu.Unlock()
	if err := s.player.Restart(); err != nil {
		cancel()
		return err
	}
	s.mu.Lock()
	if s.status.Generation == generation {
		s.status.Phase = Playing
		s.status.PositionMilliseconds = 0
		s.status.Message = "Playing transcript"
		s.status.CanPause = true
		s.status.CanResume = false
		s.status.CanStop = true
		status := s.status
		s.mu.Unlock()
		s.publish(status)
		s.workers.Add(1)
		go func() { defer s.workers.Done(); s.monitor(ctx, generation) }()
		return nil
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) Stop() error {
	s.control.Lock()
	defer s.control.Unlock()
	s.mu.Lock()
	if s.operation != nil {
		s.operation()
		s.operation = nil
	}
	active := s.status.Phase == Generating || s.status.Phase == Playing || s.status.Phase == Paused || s.status.Phase == Completed
	generation := s.status.Generation
	s.mu.Unlock()
	if err := s.player.Stop(); err != nil {
		return err
	}
	_ = s.player.Unload()
	if active {
		s.finish(generation, Cancelled, "Speech playback stopped", "")
	}
	return nil
}

// SaveAudio writes the retained playback session to a user-selected WAV file.
// The native dialog owns path selection and cancellation; audio remains inside
// Go for the entire operation.
func (s *Service) SaveAudio() (bool, error) {
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return false, errors.New("application is shutting down")
	}
	s.mu.Lock()
	canSave := s.status.CanSave
	generation := s.status.Generation
	s.mu.Unlock()
	if !canSave {
		return false, errors.New("there is no generated speech to save")
	}
	if s.saveFile == nil {
		return false, errors.New("speech audio saving is unavailable")
	}
	path, err := s.saveFile()
	if err != nil {
		return false, errors.New("speech audio save dialog could not be opened")
	}
	if path == "" {
		return false, nil
	}
	if err := s.player.Save(path); err != nil {
		s.logger.Warn("speech audio save failed", "generation", generation, "error_kind", diagnostics.ErrorKind(err))
		return false, errors.New("generated speech could not be saved")
	}
	s.logger.Info("speech audio saved", "generation", generation, "outcome", "saved")
	return true, nil
}

// ClearAudio explicitly releases the retained PCM session and returns the
// player to idle. Completed audio otherwise remains available for Restart.
func (s *Service) ClearAudio() error {
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	if s.operation != nil {
		s.operation()
		s.operation = nil
	}
	generation := s.status.Generation
	canClear := s.status.CanClear
	s.mu.Unlock()
	if !canClear {
		return errors.New("there is no generated speech to clear")
	}
	if err := s.player.Stop(); err != nil {
		return err
	}
	if err := s.player.Unload(); err != nil {
		return err
	}
	s.mu.Lock()
	s.status = Status{Generation: generation, Phase: Idle}
	status := s.status
	s.mu.Unlock()
	s.publish(status)
	s.logger.Info("speech audio cleared", "generation", generation, "outcome", "released")
	return nil
}

// StopPlayback is the backend collaboration used before microphone capture.
func StopPlayback(service *Service) {
	if service != nil {
		_ = service.Stop()
	}
}

func (s *Service) source(generation uint64) Source {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Generation == generation {
		return s.status.Source
	}
	return ""
}
func (s *Service) historyID(generation uint64) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Generation == generation {
		return s.status.HistoryID
	}
	return 0
}
func (s *Service) historyVersion(generation uint64) history.HistoryTextVersion {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Generation == generation {
		return s.status.HistoryVersion
	}
	return ""
}

func (s *Service) update(generation uint64, status Status) {
	s.mu.Lock()
	if s.status.Generation != generation {
		s.mu.Unlock()
		return
	}
	s.status = status
	s.mu.Unlock()
	s.publish(status)
}

func (s *Service) isCurrent(generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status.Generation == generation
}

func (s *Service) finish(generation uint64, phase Phase, message, errorKind string) {
	if phase == Completed {
		_ = s.player.Pause()
	}
	position, duration, _ := s.player.Position()
	s.mu.Lock()
	if s.status.Generation != generation {
		s.mu.Unlock()
		return
	}
	terminal := s.status.Phase == Completed || s.status.Phase == Cancelled || s.status.Phase == Failed
	if terminal && phase != Cancelled {
		s.mu.Unlock()
		return
	}
	s.operation = nil
	s.status.Phase, s.status.Message, s.status.ErrorKind = phase, message, errorKind
	s.status.PositionMilliseconds, s.status.DurationMilliseconds = position, duration
	s.status.CanPause, s.status.CanResume, s.status.CanStop = false, false, false
	s.status.CanRestart = phase == Completed
	s.status.CanSave = phase == Completed
	s.status.CanClear = phase == Completed
	status := s.status
	s.mu.Unlock()
	s.publish(status)
	if phase == Completed {
		s.logger.Info("speech playback completed", "generation", generation, "audio_ms", duration, "outcome", "played")
	}
}

func (s *Service) publish(status Status) {
	if s.changed != nil && !s.closed.Load() {
		s.changed(status)
	}
}
