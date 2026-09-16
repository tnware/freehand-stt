package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/compatibility"
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
	Languages         []string
	AudioMilliseconds int64
	Err               error
}
type wireEvent struct {
	Type       string   `json:"type"`
	Delta      string   `json:"delta"`
	Transcript string   `json:"transcript"`
	Text       *string  `json:"text"`
	Language   string   `json:"language"`
	Languages  []string `json:"languages"`
	Session    struct {
		Model string `json:"model"`
	} `json:"session"`
}

type Session struct {
	committed    atomic.Bool
	conn         *websocket.Conn
	ctx          context.Context
	cancel       context.CancelFunc
	Pipe         *audio.FramePipe
	Failed       chan struct{}
	Done         chan struct{}
	once         sync.Once
	result       Result
	backend      compatibility.ID
	modelProfile modelprofile.ID
}

// Open admits the exact selected contract and completes configuration before
// microphone capture starts. Credentials travel only in the upgrade header.
func Open(parent context.Context, cfg config.VoiceTranscriptionSettings, key string, publish func(Update)) (*Session, error) {
	if !cfg.Realtime {
		return nil, errors.New("realtime transcription is not enabled")
	}
	if err := config.ValidateVoiceRecording(cfg); err != nil {
		return nil, err
	}
	if cfg.AuthenticationMode == config.AuthenticationModeAPIKey && key == "" {
		return nil, errors.New("realtime credential is not configured")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" {
		return nil, errors.New("invalid realtime endpoint")
	}
	contract, err := compatibility.Resolve(cfg.CompatibilityProfile, compatibility.Realtime)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, contract.Path)
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
	fail := func(err error) (*Session, error) {
		cancel()
		_ = conn.CloseNow()
		return nil, err
	}
	created, err := readEvent(setup, conn)
	if err != nil || created.Type != "session.created" {
		return fail(errors.New("realtime server configuration was not accepted"))
	}
	s := &Session{conn: conn, ctx: ctx, cancel: cancel, Pipe: audio.NewFramePipe(), Failed: make(chan struct{}), Done: make(chan struct{})}
	if cfg.CompatibilityProfile == compatibility.VLLM {
		if err := configureVLLM(setup, conn, cfg.Model); err != nil {
			return fail(err)
		}
		s.backend = compatibility.VLLM
		s.modelProfile = cfg.ModelProfile
	} else if err := configureNeMo(setup, conn, cfg, created); err != nil {
		return fail(err)
	}
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

func (s *Session) writeAudio(ctx context.Context, frame []byte) error {
	if s.backend == compatibility.VLLM {
		return s.writeVLLMAudio(ctx, frame)
	}
	return s.conn.Write(ctx, websocket.MessageBinary, frame)
}

func (s *Session) run(publish func(Update)) {
	defer close(s.Done)
	defer s.cancel()
	defer s.conn.CloseNow()
	readDone := make(chan Result, 1)
	go func() {
		var result Result
		if s.backend == compatibility.VLLM {
			result = s.readVLLM(publish)
		} else {
			result = s.readNeMo(publish)
		}
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
			if err := s.writeAudio(writeCtx, frame); err != nil {
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
		commit := []byte(`{"type":"input_audio_buffer.commit"}`)
		if s.backend == compatibility.VLLM {
			commit = []byte(`{"type":"input_audio_buffer.commit","final":true}`)
		}
		if err := s.conn.Write(s.ctx, websocket.MessageText, commit); err != nil {
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
