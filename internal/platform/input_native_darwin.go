//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -mmacosx-version-min=13.0 -fblocks
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework Carbon
#include "input_darwin.h"
*/
import "C"

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/insertion"
)

// NewInput allocates an inert owner. It does not inspect focus, prompt for
// permissions, or touch the clipboard. Copies share lifecycle through state.
func NewInput(_ *slog.Logger) Input {
	return newDarwinInput(&nativeDarwinInput{ptr: C.fh_input_create()})
}

type nativeDarwinInput struct{ ptr *C.fh_input_owner }

func (b *nativeDarwinInput) capture() (uint32, uint64, error) {
	var pid C.uint32_t
	var started C.uint64_t
	if C.fh_input_capture(b.ptr, &pid, &started) == 0 {
		return 0, 0, insertion.ErrCopyRequired
	}
	return uint32(pid), uint64(started), nil
}
func (b *nativeDarwinInput) valid(pid uint32, started uint64) bool {
	return C.fh_input_validate(b.ptr, C.uint32_t(pid), C.uint64_t(started)) != 0
}
func (b *nativeDarwinInput) modifiersReleased() bool { return C.fh_input_modifiers_released() != 0 }
func (b *nativeDarwinInput) send(units []uint16) bool {
	if len(units) == 0 {
		return false
	}
	return C.fh_input_send(b.ptr, (*C.uint16_t)(unsafe.Pointer(&units[0])), C.size_t(len(units))) != 0
}
func (b *nativeDarwinInput) copyText(ctx context.Context, stop <-chan struct{}, text string) error {
	if err := ctx.Err(); err != nil {
		return errors.Join(insertion.ErrCopyNotPerformed, err)
	}
	bytes := append([]byte(text), 0)
	request := C.fh_input_copy_begin((*C.char)(unsafe.Pointer(&bytes[0])), C.size_t(len(text)))
	if request == nil {
		return insertion.ErrCopyNotPerformed
	}
	defer C.fh_input_copy_release(request)
	return waitDarwinCopy(ctx, stop,
		func() int { return int(C.fh_input_copy_status(request)) },
		func() int { return int(C.fh_input_copy_cancel(request)) })
}

// No waiter goroutine: cancelled callers cannot accumulate blocked workers.
// Cancellation revokes queued work atomically; running AppKit work owns its
// request until completion, independent of the Input's lifetime.
func waitDarwinCopy(ctx context.Context, stop <-chan struct{}, status, cancel func() int) error {
	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	poll := time.NewTicker(time.Millisecond)
	defer poll.Stop()
	outcome := func(state int, cause error) error {
		switch state {
		case int(C.FH_COPY_SUCCEEDED):
			return nil
		case int(C.FH_COPY_FAILED):
			return insertion.ErrCopyFailed
		case int(C.FH_COPY_CANCELLED):
			return errors.Join(insertion.ErrCopyNotPerformed, cause)
		default:
			return errors.Join(insertion.ErrCopyAmbiguous, cause)
		}
	}
	for {
		state := status()
		if state != int(C.FH_COPY_QUEUED) && state != int(C.FH_COPY_RUNNING) {
			return outcome(state, nil)
		}
		select {
		case <-ctx.Done():
			return outcome(cancel(), ctx.Err())
		case <-stop:
			return outcome(cancel(), nil)
		case <-timer.C:
			return outcome(cancel(), context.DeadlineExceeded)
		case <-poll.C:
		}
	}
}
func (b *nativeDarwinInput) resetTarget() { C.fh_input_reset_target(b.ptr) }
func (b *nativeDarwinInput) close() {
	if b.ptr != nil {
		C.fh_input_destroy(b.ptr)
		b.ptr = nil
	}
}
