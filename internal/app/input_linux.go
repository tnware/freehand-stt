//go:build linux

package app

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/insertion"
	nativeinput "github.com/tnware/freehand-stt/internal/platform/input"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func (a *App) configureNativeInput(input *nativeinput.Input) {
	input.SetClipboardWriter(func(ctx context.Context, text string) error {
		if a.wails == nil {
			return insertion.ErrCopyNotPerformed
		}
		return dispatchClipboardCopy(ctx, text, application.InvokeAsync, a.wails.Clipboard.SetText)
	})
}

func dispatchClipboardCopy(ctx context.Context, text string, dispatch func(func()), write func(string) bool) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	// 0 queued, 1 executing, 2 cancelled before execution. Never block cancellation
	// behind a native call; once started, an interrupted write has an ambiguous result.
	var phase atomic.Uint32
	done := make(chan error, 1)
	dispatch(func() {
		if !phase.CompareAndSwap(0, 1) {
			done <- insertion.ErrCopyNotPerformed
			return
		}
		if ctx.Err() != nil {
			done <- insertion.ErrCopyNotPerformed
			return
		}
		if !write(text) {
			done <- insertion.ErrCopyFailed
			return
		}
		done <- nil
	})
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if phase.CompareAndSwap(0, 2) {
			return ctx.Err()
		}
		select {
		case err := <-done:
			return err
		default:
			return insertion.ErrCopyAmbiguous
		}
	}
}
