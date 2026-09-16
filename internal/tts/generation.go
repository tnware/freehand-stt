package tts

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
)

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
