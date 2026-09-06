package tts

import (
	"context"
	"errors"
	"io"
	"os"
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
