//go:build darwin && cgo

package input

import (
	"context"
	"testing"

	"github.com/tnware/freehand-stt/internal/insertion"
)

func TestDarwinSegmentRetainsOriginalCaptureUntilRelease(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	target, err := i.CaptureTarget()
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"first", " second"} {
		if err := i.InsertSegment(context.Background(), target, text); err != nil {
			t.Fatal(err)
		}
		if i.state.target != target {
			t.Fatal("segment consumed original target")
		}
	}
	if b.captures != 1 || b.sends != 2 {
		t.Fatal("segment recaptured or failed to dispatch")
	}
	i.ReleaseTarget(target)
	if i.state.target.Valid() {
		t.Fatal("release retained native identity")
	}
	if err := i.InsertSegment(context.Background(), target, "again"); err == nil || b.sends != 2 {
		t.Fatal("released generation reused")
	}
}

func TestDarwinSegmentFailureConsumesOnlyOwnedGeneration(t *testing.T) {
	for _, kind := range []string{"canceled", "partial", "focus", "encoding"} {
		t.Run(kind, func(t *testing.T) {
			i, b := testDarwinInput()
			defer i.Close()
			target, _ := i.CaptureTarget()
			ctx := context.Background()
			text := "segment"
			switch kind {
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "partial":
				b.sendHook = func() bool { return false }
			case "focus":
				b.matches = false
			case "encoding":
				text = "bad\x00text"
			}
			if err := i.InsertSegment(ctx, target, text); err == nil || i.state.target.Valid() {
				t.Fatal("failed segment retained target")
			}
			b.matches = true
			b.sendHook = nil
			newer, err := i.CaptureTarget()
			if err != nil {
				t.Fatal(err)
			}
			i.ReleaseTarget(target)
			if err := i.InsertSegment(context.Background(), target, "stale"); err == nil {
				t.Fatal("stale generation accepted")
			}
			if i.state.target != newer {
				t.Fatal("old session revoked newer capture")
			}
			if err := i.InsertSegment(context.Background(), newer, "new"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestDarwinSessionCloseReleasesNativeCapture(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	target, _ := i.CaptureTarget()
	s, err := (insertion.Policy{Platform: i}).BeginSession(context.Background(), target, insertion.DirectInput)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Deliver(context.Background(), 1, "first"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if i.state.target.Valid() || b.captures != 1 {
		t.Fatal("session failed to end original capture")
	}
}
