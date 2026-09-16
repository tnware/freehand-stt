package tts

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
)

type Player interface {
	Load([]byte, uint32, uint32) error
	Play() error
	Pause() error
	// Rewind stops and resets the retained session without starting playback.
	Rewind() error
	// SeekTo stops and positions retained PCM without starting playback.
	SeekTo(int64) error
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
