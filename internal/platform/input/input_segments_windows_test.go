//go:build windows

package input

import (
	"context"
	"testing"

	"github.com/tnware/freehand-stt/internal/insertion"
)

func TestWindowsSegmentUsesOriginalTargetAndExistingGuards(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	current := target
	insertionForeground = func() (insertion.Target, error) { return current, nil }
	dispatches := 0
	dispatchUnicodeEvents = func(events []inputEvent) uintptr { dispatches++; return uintptr(len(events)) }
	i := Input{}
	if err := i.InsertSegment(context.Background(), target, "first"); err != nil {
		t.Fatal(err)
	}
	if err := i.InsertSegment(context.Background(), target, " second"); err != nil {
		t.Fatal(err)
	}
	current.FocusHWND++
	if err := i.InsertSegment(context.Background(), target, "wrong field"); err == nil || dispatches != 2 {
		t.Fatal("segment bypassed focus guard")
	}
	current = target
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := i.InsertSegment(ctx, target, "canceled"); err == nil || dispatches != 2 {
		t.Fatal("segment bypassed cancellation")
	}
	dispatchUnicodeEvents = func(events []inputEvent) uintptr { return uintptr(len(events) - 1) }
	if err := i.InsertSegment(context.Background(), target, "partial"); err == nil {
		t.Fatal("partial segment accepted")
	}
}
