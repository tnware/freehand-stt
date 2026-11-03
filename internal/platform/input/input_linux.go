//go:build linux

package input

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/tnware/freehand-stt/internal/insertion"
)

// Linux currently offers explicit Copy only. X11 process/window capture is
// feasible, but this adapter has no qualified target-bound Unicode delivery
// path. Optional accessibility identity could strengthen supported controls;
// missing field identity alone is not a reason to reject app/window delivery.
// Never substitute an unbound paste shortcut for that contract.
type Input struct {
	mu       sync.RWMutex
	copyText func(context.Context, string) error
}

func NewInput(*slog.Logger) *Input { return &Input{} }

func (i *Input) SetClipboardWriter(write func(context.Context, string) error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.copyText = write
}
func (*Input) CaptureTarget() (insertion.Target, error) {
	return insertion.Target{}, insertion.ErrCopyRequired
}
func (*Input) Foreground() (insertion.Target, error) {
	return insertion.Target{}, insertion.ErrCopyRequired
}
func (*Input) InsertUnicode(context.Context, insertion.Target, string) error {
	return insertion.ErrCopyRequired
}
func (i *Input) Copy(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.RLock()
	write := i.copyText
	i.mu.RUnlock()
	if write == nil {
		return errors.Join(insertion.ErrCopyNotPerformed, errors.New("desktop clipboard is unavailable"))
	}
	return write(ctx, text)
}
