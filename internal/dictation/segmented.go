package dictation

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
)

const pendingSegmentCapacity = 3

type segmentedResult struct {
	text              string
	chunks            int
	audioMilliseconds int64
	segments          []history.HistorySegmentDetails
	segmentsTruncated bool
	metadata          inference.ResponseMetadata
	err               error
}

type audioSegment struct {
	pcm      []byte
	index    int
	boundary string
}

// segmentedRun owns all transient audio between the realtime callback and the
// ordered HTTP worker. ready is closed instead of sending one result so both
// the recording watcher and the normal stop path may observe completion.
type segmentedRun struct {
	pipe     *audio.FramePipe
	cancel   context.CancelFunc
	ready    chan struct{}
	autoStop chan struct{}
	autoOnce sync.Once
	result   segmentedResult
}

type segmentProgressFunc func(segment int, phase SegmentPhase)
type vadProgressFunc func(active bool)
type autoStopProgressFunc func(armed, countdown bool, remainingMilliseconds int)

func newSegmentedRun(parent context.Context, cfg config.Settings, key string, client *inference.Client, detector audio.VoiceDetector, logger *slog.Logger, progress segmentProgressFunc, vadProgress vadProgressFunc, autoProgress autoStopProgressFunc) (*segmentedRun, error) {
	targetSeconds := 0
	if cfg.SilenceSplitting {
		targetSeconds = cfg.SegmentSeconds
	}
	autoStopSilenceMilliseconds := 0
	autoStopMinimumSpeechMilliseconds := 0
	if cfg.AutoStopEnabled {
		autoStopSilenceMilliseconds = cfg.AutoStopSilenceMS
		autoStopMinimumSpeechMilliseconds = cfg.AutoStopMinimumSpeechMS
	}
	segmenter, err := audio.NewSpeechSegmenter(detector, audio.SpeechSegmenterOptions{
		TargetSeconds:                     targetSeconds,
		SegmentSilenceMilliseconds:        cfg.SegmentSilenceMS,
		ActivitySilenceMilliseconds:       cfg.VADActivitySilenceMS,
		TrimSilence:                       cfg.SilenceTrimming,
		SpeechPaddingMilliseconds:         cfg.SpeechPaddingMS,
		AutoStopSilenceMilliseconds:       autoStopSilenceMilliseconds,
		AutoStopMinimumSpeechMilliseconds: autoStopMinimumSpeechMilliseconds,
	})
	if err != nil {
		_ = detector.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	run := &segmentedRun{pipe: audio.NewFramePipe(), cancel: cancel, ready: make(chan struct{})}
	if cfg.AutoStopEnabled {
		run.autoStop = make(chan struct{})
	}
	go func() {
		run.result = transcribeSegments(ctx, run.pipe, segmenter, cfg, key, client, logger, progress, vadProgress, autoProgress, func() {
			run.autoOnce.Do(func() { close(run.autoStop) })
		})
		cancel()
		close(run.ready)
	}()
	return run, nil
}

func (r *segmentedRun) wait() segmentedResult {
	<-r.ready
	return r.result
}

func (r *segmentedRun) abortBeforeCapture() segmentedResult {
	r.cancel()
	r.pipe.Close()
	return r.wait()
}

func transcribeSegments(ctx context.Context, pipe *audio.FramePipe, segmenter *audio.Segmenter, cfg config.Settings, key string, client *inference.Client, logger *slog.Logger, progress segmentProgressFunc, vadProgress vadProgressFunc, autoProgress autoStopProgressFunc, autoStop func()) segmentedResult {
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	chunks := make(chan audioSegment, pendingSegmentCapacity)
	segmenting := make(chan error, 1)
	go func() {
		segmenting <- produceSegments(workCtx, pipe, segmenter, chunks, logger, vadProgress, autoProgress, autoStop)
		close(chunks)
	}()

	var texts []string
	defer func() { key = "" }()
	completed := 0
	audioMilliseconds := int64(0)
	segments := make([]history.HistorySegmentDetails, 0)
	segmentsTruncated := false
	var metadata inference.ResponseMetadata
	for segment := range chunks {
		segmentAudioMilliseconds := int64(pcmMilliseconds(segment.pcm))
		audioMilliseconds += segmentAudioMilliseconds
		started := time.Now()
		logger.Info("dictation segment transcription started",
			"segment", segment.index,
			"audio_ms", pcmMilliseconds(segment.pcm),
			"boundary", segment.boundary,
			"timeout_seconds", cfg.TranscriptionTimeoutSeconds,
		)
		if progress != nil {
			progress(segment.index, SegmentTranscribing)
		}
		transcription, err := transcribePCM(workCtx, client, cfg, key, segment.pcm)
		requestMilliseconds := time.Since(started).Milliseconds()
		if err != nil {
			logger.Error("dictation segment transcription failed", "segment", segment.index, "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
			cancel()
			drainAndZero(chunks)
			return segmentedResult{err: err}
		}
		text := transcription.Text
		metadata.Add(transcription.Metadata)
		completed++
		if len(segments) < history.MaxHistorySegments {
			segments = append(segments, history.HistorySegmentDetails{
				Number:              segment.index,
				AudioMilliseconds:   segmentAudioMilliseconds,
				Boundary:            segment.boundary,
				RequestMilliseconds: requestMilliseconds,
				CharacterCount:      utf8.RuneCountInString(text),
			})
		} else {
			segmentsTruncated = true
		}
		logger.Info("dictation segment transcription completed",
			"segment", segment.index,
			"duration_ms", requestMilliseconds,
			"characters", utf8.RuneCountInString(text),
		)
		if progress != nil {
			progress(segment.index, SegmentCompleted)
		}
		if text != "" {
			texts = append(texts, text)
		}
	}
	if err := <-segmenting; err != nil {
		return segmentedResult{err: err}
	}
	return segmentedResult{text: strings.Join(texts, " "), chunks: completed, audioMilliseconds: audioMilliseconds, segments: segments, segmentsTruncated: segmentsTruncated, metadata: metadata}
}

func produceSegments(ctx context.Context, pipe *audio.FramePipe, segmenter *audio.Segmenter, chunks chan<- audioSegment, logger *slog.Logger, vadProgress vadProgressFunc, autoProgress autoStopProgressFunc, autoStop func()) error {
	defer segmenter.Close()
	index := 0
	lastVADKnown := false
	lastVADActive := false
	lastAutoState := -1
	for {
		select {
		case <-ctx.Done():
			go drainFrames(pipe)
			return ctx.Err()
		case frame, ok := <-pipe.Frames():
			if !ok {
				chunk := segmenter.Flush()
				if len(chunk) == 0 {
					return nil
				}
				index++
				segment := audioSegment{pcm: chunk, index: index, boundary: "recording_stopped"}
				logSegmentReady(logger, segment)
				select {
				case chunks <- segment:
					return nil
				case <-ctx.Done():
					zero(segment.pcm)
					return ctx.Err()
				}
			}
			chunk, err := segmenter.Push(frame)
			pipe.Release(frame)
			if err != nil {
				go drainFrames(pipe)
				return err
			}
			if active, known := segmenter.VoiceActivity(); known && (!lastVADKnown || active != lastVADActive) {
				lastVADKnown = true
				lastVADActive = active
				if vadProgress != nil {
					vadProgress(active)
				}
			}
			if enabled, armed, countdown, remaining := segmenter.AutoStopStatus(); enabled {
				autoState := 0
				if armed {
					autoState = 1
				}
				if countdown {
					autoState = 2
				}
				if autoState != lastAutoState {
					lastAutoState = autoState
					if autoProgress != nil {
						autoProgress(armed, countdown, remaining)
					}
				}
				if segmenter.AutoStopReady() && autoStop != nil {
					autoStop()
				}
			}
			if len(chunk) == 0 {
				continue
			}
			index++
			boundary := "silence"
			if len(chunk) >= audio.MaxSegmentSeconds*audio.SampleRate*2 {
				boundary = "hard_limit"
			}
			segment := audioSegment{pcm: chunk, index: index, boundary: boundary}
			logSegmentReady(logger, segment)
			select {
			case chunks <- segment:
			case <-ctx.Done():
				zero(segment.pcm)
				go drainFrames(pipe)
				return ctx.Err()
			}
		}
	}
}

func transcribePCM(ctx context.Context, client *inference.Client, cfg config.Settings, key string, pcm []byte) (inference.TranscriptionResult, error) {
	wav, err := audio.WAV(pcm)
	zero(pcm)
	if err != nil {
		return inference.TranscriptionResult{}, err
	}
	defer zero(wav)
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TranscriptionTimeoutSeconds)*time.Second)
	defer cancel()
	return client.Transcribe(requestCtx, cfg.BaseURL, cfg.Model, cfg.Language, key, cfg.Headers, wav)
}

func drainFrames(pipe *audio.FramePipe) {
	for frame := range pipe.Frames() {
		pipe.Release(frame)
	}
}

func drainAndZero(chunks <-chan audioSegment) {
	for segment := range chunks {
		zero(segment.pcm)
	}
}

func logSegmentReady(logger *slog.Logger, segment audioSegment) {
	logger.Info("dictation segment ready",
		"segment", segment.index,
		"audio_ms", pcmMilliseconds(segment.pcm),
		"boundary", segment.boundary,
	)
}

func pcmMilliseconds(pcm []byte) int {
	return len(pcm) * 1000 / (audio.SampleRate * 2)
}

func zero(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
