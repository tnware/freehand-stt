package connection

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type SavedConnectionSource interface {
	ResolveSavedConnection(string) (savedconnection.Connection, string, error)
}

type Service struct {
	savedConnections SavedConnectionSource
	keys             credential.Store
	processKeys      credential.Store
	ttsKeys          credential.Store
	client           *inference.Client
	logger           *slog.Logger
	lifecycleMu      sync.RWMutex
	rootContext      context.Context
	rootCancel       context.CancelFunc
	closed           atomic.Bool
}

func NewService(keys, processKeys, ttsKeys credential.Store, client *inference.Client, logger *slog.Logger, sources ...SavedConnectionSource) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	s := &Service{keys: keys, processKeys: processKeys, ttsKeys: ttsKeys, client: client, logger: logger.With("component", "connection")}
	if len(sources) > 0 {
		s.savedConnections = sources[0]
	}
	return s
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

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return diagnostics.DiscardLogger()
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closed.Store(false)
	s.lifecycleMu.Lock()
	s.rootContext, s.rootCancel = context.WithCancel(ctx)
	s.lifecycleMu.Unlock()
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.closed.Store(true)
	s.lifecycleMu.Lock()
	cancel := s.rootCancel
	s.rootCancel = nil
	s.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// Endpoint metadata bindings. Each request includes only values used by its
// metadata probe; unrelated settings cannot invalidate the operation.
