package tts

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/tnware/freehand-stt/internal/diagnostics"
)

// Native file selection authorizes this destination. Cancellation is checked
// before opening and between writes. A blocked OS call cannot be interrupted
// portably; the service's shutdown deadline still bounds its wait for export.
func writeAudioFile(ctx context.Context, path string, wav []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	err = writeAudio(ctx, file, wav)
	return errors.Join(err, file.Close())
}

func writeAudio(ctx context.Context, out io.Writer, wav []byte) error {
	const chunkSize = 64 * 1024
	for len(wav) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk := wav[:min(len(wav), chunkSize)]
		n, err := out.Write(chunk)
		if err != nil {
			return err
		}
		if n != len(chunk) {
			return io.ErrShortWrite
		}
		wav = wav[n:]
	}
	return ctx.Err()
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
