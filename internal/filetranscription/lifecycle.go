package filetranscription

import (
	"context"
	"errors"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (s *Service) operationContext() (context.Context, context.CancelFunc) {
	s.lifecycleMu.RLock()
	root := s.rootContext
	s.lifecycleMu.RUnlock()
	if root == nil {
		root = context.Background()
	}
	ctx, cancel := context.WithCancel(root)
	if s.closed.Load() {
		cancel()
	}
	return ctx, cancel
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.lifecycleMu.Lock()
	if s.closed.Load() {
		s.lifecycleMu.Unlock()
		return errors.New("file transcription service is closed")
	}
	s.rootContext, s.rootCancel = context.WithCancel(ctx)
	s.lifecycleMu.Unlock()
	return nil
}

func (s *Service) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.shutdown(ctx)
}

func (s *Service) shutdown(ctx context.Context) error {
	s.shutdownOnce.Do(func() {
		s.closed.Store(true)
		s.activity.Close()
		s.lifecycleMu.Lock()
		if s.rootCancel != nil {
			s.rootCancel()
			s.rootCancel = nil
		}
		s.lifecycleMu.Unlock()
		go func() {
			// The owner's state lock is part of the budget too. It serializes worker
			// admission, so Wait cannot overtake a successful workers.Add.
			s.fileMu.Lock()
			if s.fileCancel != nil {
				s.fileCancel()
				s.fileCancel = nil
			}
			s.fileGeneration++
			s.fileStatus = FileTranscriptionStatus{Generation: s.fileGeneration, Phase: FileTranscriptionEmpty}
			s.fileSelection = nil
			s.resetFileTranscriptLocked("")
			s.fileMu.Unlock()
			s.workers.Wait()
			close(s.shutdownDone)
		}()
	})
	select {
	case <-s.shutdownDone:
		return nil
	case <-ctx.Done():
		return errors.Join(errors.New("file transcription shutdown exceeded the service deadline"), ctx.Err())
	}
}
