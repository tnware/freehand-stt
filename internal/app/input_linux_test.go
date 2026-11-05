//go:build linux

package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/insertion"
)

func TestLinuxQueuedCopyCancellationPreventsLaterWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	queued := make(chan func(), 1)
	result := make(chan error, 1)
	go func() {
		result <- dispatchClipboardCopy(ctx, "private", func(f func()) { queued <- f }, func(string) bool { t.Error("cancelled copy wrote clipboard"); return true })
	}()
	callback := <-queued
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	callback()
}
func TestLinuxExecutingCopyCancellationIsBoundedAndAmbiguous(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started, finish, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- dispatchClipboardCopy(ctx, "private", func(f func()) { go f() }, func(string) bool { close(started); <-finish; return true })
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, insertion.ErrCopyAmbiguous) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation waited for native write")
	}
	close(finish)
}
