// Package realtime owns versioned speech transports. It never captures audio,
// retains history, runs cleanup, or inserts text into another application.
package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

const MaxTranscriptBytes = 256 * 1024
const finalizeTimeout = 30 * time.Second

// Update is a complete presentation snapshot. Committed turns precede the
// current provisional turn. A final replaces that turn's provisional text.
type Update struct {
	Final   string
	Partial string
	Turn    uint64
}
type Result struct {
	Text              string
	Language          string
	AudioMilliseconds int64
	Err               error
}
type wireEvent struct {
	Type       string `json:"type"`
	Delta      string `json:"delta"`
	Transcript string `json:"transcript"`
	Session    struct {
		Model string `json:"model"`
	} `json:"session"`
}

type Session struct {
	committed atomic.Bool
	conn      *websocket.Conn
	ctx       context.Context
	cancel    context.CancelFunc
	Pipe      *audio.FramePipe
	Failed    chan struct{}
	Done      chan struct{}
	once      sync.Once
	result    Result
}

// Open admits the exact selected contract and completes configuration before
// microphone capture starts. Credentials travel only in the upgrade header.
func Open(parent context.Context, cfg config.VoiceTranscriptionSettings, key string, publish func(Update)) (*Session, error) {
	if !cfg.Realtime {
		return nil, errors.New("realtime transcription is not enabled")
	}
	if err := config.ValidateVoiceTranscription(cfg); err != nil {
		return nil, err
	}
	if cfg.AuthenticationMode == config.AuthenticationModeAPIKey && key == "" {
		return nil, errors.New("realtime credential is not configured")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" {
		return nil, errors.New("invalid realtime endpoint")
	}
	u.Path = path.Join(u.Path, "realtime")
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	headers := http.Header{}
	for name, value := range cfg.Headers {
		headers.Set(name, value)
	}
	if cfg.AuthenticationMode == config.AuthenticationModeAPIKey {
		headers.Set("Authorization", "Bearer "+key)
	}
	ctx, cancel := context.WithCancel(parent)
	setup, stopSetup := context.WithTimeout(ctx, 10*time.Second)
	defer stopSetup()
	// Never follow an upgrade redirect carrying a credential to another peer.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	conn, response, err := websocket.Dial(setup, u.String(), &websocket.DialOptions{HTTPClient: client, HTTPHeader: headers})
	headers.Del("Authorization")
	key = ""
	if err != nil {
		cancel()
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, errors.New("could not connect to the realtime server")
	}
	conn.SetReadLimit(MaxTranscriptBytes + 4096)
	fail := func() (*Session, error) {
		cancel()
		_ = conn.CloseNow()
		return nil, errors.New("realtime server configuration was not accepted")
	}
	created, err := readEvent(setup, conn)
	if err != nil || created.Type != "session.created" {
		return fail()
	}
	if cfg.Model != "" && created.Session.Model != cfg.Model {
		cancel()
		_ = conn.CloseNow()
		return nil, errors.New("the realtime server loaded a different model; refresh its model selection")
	}
	phrases := []string{}
	for _, line := range strings.Split(cfg.Options.Vocabulary, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			phrases = append(phrases, line)
		}
	}
	options := map[string]any{"sample_rate": audio.SampleRate, "language": cfg.Language, "automatic_punctuation": true, "verbatim": true, "speaker_diarization": false}
	if len(phrases) > 0 {
		options["speech_contexts"] = []any{map[string]any{"phrases": phrases, "boost": cfg.Options.Boost}}
	}
	message, _ := json.Marshal(map[string]any{"type": "session.update", "session": options})
	if conn.Write(setup, websocket.MessageText, message) != nil {
		return fail()
	}
	updated, err := readEvent(setup, conn)
	if err != nil || updated.Type != "session.updated" {
		return fail()
	}
	s := &Session{conn: conn, ctx: ctx, cancel: cancel, Pipe: audio.NewFramePipe(), Failed: make(chan struct{}), Done: make(chan struct{})}
	go s.run(publish)
	return s, nil
}

func readEvent(ctx context.Context, conn *websocket.Conn) (wireEvent, error) {
	kind, message, err := conn.Read(ctx)
	if err != nil {
		return wireEvent{}, errors.New("realtime connection ended before completion")
	}
	var event wireEvent
	if kind != websocket.MessageText || json.Unmarshal(message, &event) != nil {
		return event, errors.New("invalid realtime response")
	}
	return event, nil
}

// AbortBeforeCapture closes the producer only when capture has not started.
func (s *Session) AbortBeforeCapture() { s.cancel(); s.Pipe.Close(); <-s.Done }
func (s *Session) Wait() Result        { <-s.Done; return s.result }

func (s *Session) run(publish func(Update)) {
	defer close(s.Done)
	defer s.cancel()
	defer s.conn.CloseNow()
	readDone := make(chan Result, 1)
	go func() {
		result := s.read(publish)
		if result.Err != nil {
			s.once.Do(func() { close(s.Failed) })
			s.cancel()
		}
		readDone <- result
	}()
	bytes := int64(0)
	var writeErr error
	// Always drain after cancellation until the native producer closes. This
	// releases pooled audio and prevents a full pipe blocking capture shutdown.
	for frame := range s.Pipe.Frames() {
		if s.ctx.Err() == nil && writeErr == nil {
			writeCtx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
			if err := s.conn.Write(writeCtx, websocket.MessageBinary, frame); err != nil {
				writeErr = errors.New("realtime audio could not be sent")
				s.once.Do(func() { close(s.Failed) })
				s.cancel()
			} else {
				bytes += int64(len(frame))
			}
			cancel()
		}
		s.Pipe.Release(frame)
	}
	timer := time.AfterFunc(finalizeTimeout, s.cancel)
	defer timer.Stop()
	if s.ctx.Err() == nil && writeErr == nil {
		s.committed.Store(true)
		if err := s.conn.Write(s.ctx, websocket.MessageText, []byte(`{"type":"input_audio_buffer.commit"}`)); err != nil {
			writeErr = errors.New("realtime completion could not be requested")
			s.cancel()
		}
	}
	s.result = <-readDone
	if writeErr != nil {
		s.result.Err = writeErr
	}
	s.result.AudioMilliseconds = bytes * 1000 / (audio.SampleRate * 2)
}

func (s *Session) read(publish func(Update)) Result {
	update := Update{Turn: 1}
	language := ""
	for {
		event, err := readEvent(s.ctx, s.conn)
		if err != nil {
			return Result{Err: err}
		}
		switch event.Type {
		case "conversation.item.input_audio_transcription.delta":
			if len(update.Final)+len(update.Partial)+len(event.Delta) > MaxTranscriptBytes {
				return Result{Err: errors.New("live transcript exceeded its size limit")}
			}
			update.Partial += event.Delta
			if event.Delta != "" && publish != nil {
				publish(update)
			}
		case "conversation.item.input_audio_transcription.completed":
			text, detected := modelprofile.StripNemotronLanguageTag(event.Transcript)
			if detected != "" {
				language = detected
			}
			if len(update.Final)+len(text)+1 > MaxTranscriptBytes || update.Turn > 1024 {
				return Result{Err: errors.New("live transcript exceeded its size limit")}
			}
			update.Final = strings.TrimSpace(update.Final + " " + text)
			update.Partial = ""
			update.Turn++
			if publish != nil {
				publish(update)
			}
		case "input_audio_buffer.committed":
			if !s.committed.Load() || update.Turn == 1 || update.Partial != "" {
				return Result{Err: errors.New("realtime completion was missing its final transcript")}
			}
			return Result{Text: update.Final, Language: language}
		case "error", "input_audio_buffer.cleared":
			return Result{Err: errors.New("realtime server could not finish this recording")}
		}
	}
}
