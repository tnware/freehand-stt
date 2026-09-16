package tts

import (
	"context"
	"errors"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The entire speech teardown shares this budget, including player locks and export.
const shutdownTimeout = 2 * time.Second

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
