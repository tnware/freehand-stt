package filetranscription

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
)

const (
	maxFileTranscriptBytes = 8 << 20
	fileProgressInterval   = 100 * time.Millisecond
	DeltaEvent             = "file-transcription:delta"
)

type transcriptProcessor interface {
	ProcessWithCredential(context.Context, config.PostProcessingSettings, string, string) (postprocess.Result, error)
}

type Service struct {
	settings                 settings.Source
	profiles                 settings.ProfileSource
	client                   *inference.Client
	processor                transcriptProcessor
	history                  *history.Store
	input                    insertion.Platform
	pickAudioFile            func() (string, error)
	fileChanged              func(FileTranscriptionStatus)
	fileDelta                func(FileTranscriptionDelta)
	activity                 *activity.Coordinator
	logger                   *slog.Logger
	fileMu                   sync.Mutex
	fileStatus               FileTranscriptionStatus
	fileSelection            *audioFileSelection
	fileChoosing             bool
	fileGeneration           uint64
	fileCancel               context.CancelFunc
	fileDone                 chan struct{}
	fileLastPublish          time.Time
	fileTranscript           strings.Builder
	fileTranscriptRevision   uint64
	fileTranscriptLimitHit   bool
	fileStreamingUnsupported map[fileStreamingCapabilityKey]string
	lifecycleMu              sync.RWMutex
	rootContext              context.Context
	rootCancel               context.CancelFunc
	workers                  sync.WaitGroup
	closed                   atomic.Bool
	shutdownOnce             sync.Once
	shutdownDone             chan struct{}
}

func NewService(source settings.Source, profiles settings.ProfileSource, client *inference.Client, processor transcriptProcessor, transcripts *history.Store, input insertion.Platform, pickAudioFile func() (string, error), changed func(FileTranscriptionStatus), delta func(FileTranscriptionDelta), admission *activity.Coordinator, logger *slog.Logger) *Service {
	if admission == nil {
		admission = activity.New(activity.Sources{})
	}
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &Service{shutdownDone: make(chan struct{}), settings: source, profiles: profiles, client: client, processor: processor, history: transcripts, input: input, pickAudioFile: pickAudioFile, fileChanged: changed, fileDelta: delta, activity: admission, logger: logger.With("component", "file-transcription")}
}

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return diagnostics.DiscardLogger()
}

func (s *Service) current() config.Settings {
	if s.settings == nil {
		return config.Default()
	}
	return s.settings.Current()
}

func (s *Service) captureRequestProfile() (settings.RequestProfile, error) {
	if s.profiles == nil {
		return settings.RequestProfile{}, errors.New("transcription profile is unavailable")
	}
	return s.profiles.Capture()
}

func Active(service *Service) bool {
	return service != nil && service.fileTranscriptionActive()
}
