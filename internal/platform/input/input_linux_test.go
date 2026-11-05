//go:build linux

package input

import (
	"context"
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/insertion"
)

func TestLinuxDeliveryRequiresExplicitCopy(t *testing.T) {
	i := NewInput(nil)
	calls := 0
	i.SetClipboardWriter(func(context.Context, string) error { calls++; return nil })
	target, err := i.CaptureTarget()
	if target.Valid() || !errors.Is(err, insertion.ErrCopyRequired) {
		t.Fatal("Linux must not claim a retained target", target, err)
	}
	if err := i.InsertUnicode(context.Background(), target, "private text"); !errors.Is(err, insertion.ErrCopyRequired) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("automatic delivery touched clipboard")
	}
	if err := i.Copy(context.Background(), "explicit copy"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("explicit copy was not delegated")
	}
}
func TestLinuxCancelledCopyDoesNotWrite(t *testing.T) {
	i := NewInput(nil)
	i.SetClipboardWriter(func(context.Context, string) error { t.Fatal("cancelled copy wrote clipboard"); return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := i.Copy(ctx, "private text"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestLinuxCopyWithoutDesktopFailsClosed(t *testing.T) {
	if err := NewInput(nil).Copy(context.Background(), "private text"); !errors.Is(err, insertion.ErrCopyNotPerformed) {
		t.Fatal(err)
	}
}
