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

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The entire speech teardown shares this budget, including player locks and export.
const shutdownTimeout = 2 * time.Second

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
	SourceVoice   Source = "voice"
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
	CanSeek              bool                       `json:"canSeek"`
	CanRestart           bool                       `json:"canRestart"`
	CanStop              bool                       `json:"canStop"`
	CanSave              bool                       `json:"canSave"`
	CanClear             bool                       `json:"canClear"`
}

type Player interface {
	Load([]byte, uint32, uint32) error
	Play() error
	Pause() error
	// Rewind stops and resets the retained session without starting playback.
	Rewind() error
	// Seek stops and positions retained PCM without starting playback.
	Seek(int64) error
	Position() (int64, int64, bool)
	OutputName() string
	Snapshot() ([]byte, error)
	Stop() error
	Unload() error
	Close() error
}

type SpeechClient interface {
	SynthesizeSpeech(context.Context, string, string, inference.SpeechRequest) ([]byte, error)
}

// TranscriptSources reads current results from their feature owners, independently of history.
type TranscriptSources struct {
	File  func() (string, error)
	Voice func(uint64) (string, error)
}

type Service struct {
	control      sync.Mutex
	mu           sync.Mutex
	profiles     settings.TextToSpeechProfileSource
	client       SpeechClient
	player       Player
	history      *history.Store
	texts        TranscriptSources
	saveFile     func() (string, error)
	activity     *activity.Coordinator
	changed      func(Status)
	logger       *slog.Logger
	lifecycleMu  sync.Mutex
	rootContext  context.Context
	rootCancel   context.CancelFunc
	operation    context.CancelFunc
	status       Status
	generation   uint64
	workers      sync.WaitGroup
	closed       atomic.Bool
	saving       atomic.Bool
	writeAudio   func(context.Context, string, []byte) error
	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shutdownErr  error
}

func NewService(profiles settings.TextToSpeechProfileSource, client SpeechClient, player Player, transcripts *history.Store, texts *TranscriptSources, saveFile func() (string, error), admission *activity.Coordinator, changed func(Status), logger *slog.Logger) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	if admission == nil {
		admission = activity.New(activity.Sources{})
	}
	sources := TranscriptSources{}
	if texts != nil {
		sources = *texts
	}
	root, cancel := context.WithCancel(context.Background())
	return &Service{rootContext: root, rootCancel: cancel, shutdownDone: make(chan struct{}), writeAudio: writeAudioFile, profiles: profiles, client: client, player: player, history: transcripts, texts: sources, saveFile: saveFile, activity: admission, changed: changed, logger: logger.With("component", "tts"), status: Status{Phase: Idle}}
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.closed.Load() {
		return errors.New("speech service is closed")
	}
	s.rootCancel()
	s.rootContext, s.rootCancel = context.WithCancel(ctx)
	return nil
}

func (s *Service) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.shutdown(ctx)
}

func (s *Service) shutdown(ctx context.Context) error {
	s.shutdownOnce.Do(func() {
		s.closed.Store(true)
		s.activity.Close()
		// Neither native calls nor the player control lock can delay cancellation.
		s.lifecycleMu.Lock()
		s.rootCancel()
		s.lifecycleMu.Unlock()
		go func() {
			s.control.Lock()
			s.mu.Lock()
			s.operation = nil
			s.generation++
			s.status = Status{Generation: s.generation, Phase: Idle}
			s.mu.Unlock()
			// Stay serialized with in-flight native calls, even if the caller times out.
			s.shutdownErr = errors.Join(s.player.Stop(), s.player.Unload(), s.player.Close())
			s.control.Unlock()
			// Acquiring control above also fences all workers.Add calls.
			s.workers.Wait()
			close(s.shutdownDone)
		}()
	})
	select {
	case <-s.shutdownDone:
		return s.shutdownErr
	case <-ctx.Done():
		s.logger.Warn("speech shutdown deadline reached", "error_kind", "timeout")
		return errors.Join(errors.New("speech shutdown exceeded the service deadline"), ctx.Err())
	}
}

func (s *Service) operationContext() (context.Context, context.CancelFunc) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	return context.WithCancel(s.rootContext)
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
	if s.texts.File == nil {
		return errors.New("audio file transcript playback is unavailable")
	}
	text, err := s.texts.File()
	if err != nil {
		return err
	}
	return s.start(text, SourceFile, 0, history.HistoryTextFinal)
}

// PlayVoiceTranscript plays the completed result the user selected, never a newer recording.
func (s *Service) PlayVoiceTranscript(generation uint64) error {
	if s.texts.Voice == nil {
		return errors.New("voice transcript playback is unavailable")
	}
	text, err := s.texts.Voice(generation)
	if err != nil {
		return err
	}
	return s.start(text, SourceVoice, 0, history.HistoryTextFinal)
}

func (s *Service) PreviewVoice(draft *settings.TextToSpeechPreview) error {
	return s.startWithPreview("This is Freehand's speech playback preview.", SourcePreview, 0, history.HistoryTextFinal, draft)
}

// SpeakText generates speech for bounded user-authored text from the
// first-class Text to speech workspace. It never writes to transcript history.
func (s *Service) SpeakText(text string) error {
	return s.start(text, SourceCompose, 0, history.HistoryTextFinal)
}

func (s *Service) start(text string, source Source, historyID uint64, version history.HistoryTextVersion) error {
	return s.startWithPreview(text, source, historyID, version, nil)
}

func (s *Service) startWithPreview(text string, source Source, historyID uint64, version history.HistoryTextVersion, draft *settings.TextToSpeechPreview) error {
	release, err := s.activity.BeginPlayback()
	if err != nil {
		return err
	}
	defer release()
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("there is no transcript to read")
	}
	if utf8.RuneCountInString(text) > config.MaxTTSInputCharacters || len(text) > config.MaxTTSInputBytes {
		return errors.New("speech input is too long")
	}
	profile, err := s.profiles.CapturePreview(draft)
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
	s.mu.Unlock()
	_ = s.player.Stop()
	_ = s.player.Unload()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	s.generation++
	generation := s.generation
	ctx, cancel := s.operationContext()
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
	audioBytes, err := s.client.SynthesizeSpeech(requestCtx, profile.Settings.BaseURL, profile.Credential, inference.SpeechRequest{Options: profile.Settings.Options, ModelProfile: profile.Settings.ModelProfile, CompatibilityProfile: profile.Settings.CompatibilityProfile, Model: profile.Settings.Model, Voice: profile.Settings.Voice, Input: text, Speed: profile.Settings.Speed})
	requestCancel()
	profile.Credential = ""
	text = ""
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.logger.Info("speech generation cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
			s.finish(generation, Cancelled, "Speech playback stopped", "")
			return
		}
		s.logger.Error("speech generation failed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "failed", "stage", "inference", "error_kind", diagnostics.ErrorKind(err))
		s.finish(generation, Failed, err.Error(), diagnostics.ErrorKind(err))
		return
	}
	if ctx.Err() != nil || !s.isCurrent(generation) {
		clear(audioBytes)
		s.logger.Info("speech generation cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
		return
	}
	stage := "decode"
	pcm, err := decodeWAV(audioBytes)
	clear(audioBytes)
	pcmBytes := len(pcm.Data)
	// Player access, admission, and publication are one ordered operation.
	s.control.Lock()
	if s.closed.Load() || ctx.Err() != nil || !s.isCurrent(generation) {
		s.control.Unlock()
		clear(pcm.Data)
		s.logger.Info("speech generation cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
		return
	}
	if err == nil {
		stage = "load"
		err = s.player.Load(pcm.Data, pcm.SampleRate, pcm.Channels)
		if err == nil && !s.closed.Load() && ctx.Err() == nil {
			stage = "play"
			err = s.player.Play()
		}
	}
	clear(pcm.Data)
	if s.closed.Load() || ctx.Err() != nil {
		s.control.Unlock()
		s.logger.Info("speech generation cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
		return
	}
	if err != nil {
		_ = s.player.Unload()
		s.logger.Error("speech generation failed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "failed", "stage", stage, "error_kind", diagnostics.ErrorKind(err))
		s.finishLocked(generation, Failed, err.Error(), diagnostics.ErrorKind(err))
		s.control.Unlock()
		return
	}
	position, duration, _ := s.player.Position()
	s.update(generation, Status{Generation: generation, Phase: Playing, Source: s.source(generation), HistoryID: s.historyID(generation), HistoryVersion: s.historyVersion(generation), PositionMilliseconds: position, DurationMilliseconds: duration, Message: "Playing speech", CanPause: true, CanSeek: duration > 0, CanRestart: true, CanStop: true, CanSave: true, CanClear: true})
	s.logger.Info("speech generation completed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "completed", "audio_ms", duration, "sample_rate", pcm.SampleRate, "channels", pcm.Channels, "pcm_bytes", pcmBytes)
	s.control.Unlock()
	s.monitor(ctx, generation)
}

func (s *Service) monitor(ctx context.Context, generation uint64) {
	started := time.Now()
	// The monitor owns playback diagnostics, including restarted sessions and
	// stale generations that can no longer publish a status transition.
	s.logger.Info("speech playback started", "generation", generation)
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("speech playback cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
			return
		case <-ticker.C:
			s.control.Lock()
			if s.closed.Load() || ctx.Err() != nil || !s.isCurrent(generation) {
				s.control.Unlock()
				s.logger.Info("speech playback cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled", "error_kind", "cancelled")
				return
			}
			position, duration, done := s.player.Position()
			if done {
				s.finishLocked(generation, Completed, "Playback complete", "")
				s.control.Unlock()
				s.logger.Info("speech playback completed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "played")
				return
			}
			s.mu.Lock()
			s.status.PositionMilliseconds = position
			s.status.DurationMilliseconds = duration
			status := s.status
			s.mu.Unlock()
			s.publish(status)
			s.control.Unlock()
		}
	}
}

func (s *Service) Pause() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
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
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
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
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.control.Lock()
	defer s.control.Unlock()
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	s.mu.Lock()
	valid := s.status.CanRestart
	if !valid {
		s.mu.Unlock()
		return errors.New("speech playback cannot be restarted")
	}
	if s.operation != nil {
		s.operation()
	}
	ctx, cancel := s.operationContext()
	s.generation++
	generation := s.generation
	s.status.Generation = generation
	s.operation = cancel
	s.mu.Unlock()
	if err := s.player.Rewind(); err != nil {
		cancel()
		return err
	}
	// Native Stop inside Rewind may have outlived shutdown's wait budget.
	// Play is a separate step and must not be admitted after cancellation.
	if s.closed.Load() {
		cancel()
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		cancel()
		return err
	}
	if err := s.player.Play(); err != nil {
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
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
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
	active := s.status.Phase == Generating || s.status.Phase == Playing || s.status.Phase == Paused || s.status.Phase == Completed
	s.generation++
	generation := s.generation
	s.status.Generation = generation
	s.mu.Unlock()
	if err := s.player.Stop(); err != nil {
		return err
	}
	_ = s.player.Unload()
	if active {
		s.finishLocked(generation, Cancelled, "Speech playback stopped", "")
	}
	return nil
}

// SaveAudio writes the retained playback session to a user-selected WAV file.
// The native dialog owns path selection and cancellation; audio remains inside
// Go for the entire operation.
func (s *Service) SaveAudio() (bool, error) {
	if !s.saving.CompareAndSwap(false, true) {
		return false, errors.New("speech audio save dialog is already open")
	}
	defer s.saving.Store(false)
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
	s.control.Lock()
	if s.closed.Load() || !s.isCurrent(generation) {
		s.control.Unlock()
		return false, errors.New("speech session changed while choosing a save location")
	}
	// Pin this generation's audio while serialized with replacement/clear. Disk
	// I/O owns only the independent WAV, so playback can stop immediately.
	wav, err := s.player.Snapshot()
	if err != nil {
		s.control.Unlock()
		return false, errors.New("generated speech could not be saved")
	}
	ctx, cancel := s.operationContext()
	s.workers.Add(1)
	s.control.Unlock()
	defer s.workers.Done()
	defer cancel()
	defer clear(wav)
	err = s.writeAudio(ctx, path, wav)
	if ctx.Err() != nil || s.closed.Load() {
		return false, errors.New("speech audio save cancelled during shutdown")
	}
	if err != nil {
		s.logger.Warn("speech audio save failed", "generation", generation, "error_kind", diagnostics.ErrorKind(err))
		return false, errors.New("generated speech could not be saved")
	}
	s.logger.Info("speech audio saved", "generation", generation, "outcome", "saved")
	return true, nil
}

// ClearAudio explicitly releases the retained PCM session and returns the
// player to idle. Completed audio otherwise remains available for Restart.
func (s *Service) ClearAudio() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
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
	s.generation++
	generation := s.generation
	s.status = Status{Generation: generation, Phase: Idle}
	status := s.status
	s.mu.Unlock()
	s.publish(status)
	s.logger.Info("speech audio cleared", "generation", generation, "outcome", "released")
	return nil
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
	if s.closed.Load() || s.status.Generation != generation {
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
	s.control.Lock()
	defer s.control.Unlock()
	s.finishLocked(generation, phase, message, errorKind)
}

// Caller owns control; stale work must never touch the shared player.
func (s *Service) finishLocked(generation uint64, phase Phase, message, errorKind string) {
	if s.closed.Load() || !s.isCurrent(generation) {
		return
	}

	if phase == Completed {
		_ = s.player.Pause()
	}
	position, duration, _ := s.player.Position()
	s.mu.Lock()
	if s.closed.Load() || s.status.Generation != generation {
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
	s.status.CanSeek = phase == Completed && duration > 0
	s.status.CanRestart = phase == Completed
	s.status.CanSave = phase == Completed
	s.status.CanClear = phase == Completed
	status := s.status
	s.mu.Unlock()
	s.publish(status)
}

func (s *Service) publish(status Status) {
	if s.changed != nil && !s.closed.Load() {
		s.changed(status)
	}
}
