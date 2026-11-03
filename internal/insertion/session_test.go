package insertion

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type segmentFake struct {
	fake
	releases []Target
	sent     []string
	failure  error
	send     func(context.Context) error
}

func (f *segmentFake) InsertSegment(ctx context.Context, t Target, text string) error {
	f.insertTarget = t
	f.sent = append(f.sent, text)
	if f.send != nil {
		return f.send(ctx)
	}
	return f.failure
}
func (f *segmentFake) ReleaseTarget(t Target) { f.releases = append(f.releases, t) }
func sessionTarget() Target {
	return Target{HWND: 1, FocusHWND: 2, ThreadID: 3, ProcessID: 4, ProcessCreationTime: 5}
}
func beginTestSession(t *testing.T, f *segmentFake) *Session {
	t.Helper()
	s, err := (Policy{f}).BeginSession(context.Background(), f.target, DirectInput)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}
func TestSessionRetainsOnlyOriginalTargetForNewSegments(t *testing.T) {
	f := &segmentFake{fake: fake{target: sessionTarget()}}
	s := beginTestSession(t, f)
	for n, text := range []string{"first", " second", " third"} {
		if err := s.Deliver(context.Background(), uint64(n+1), text); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.sent) != 3 || f.insertTarget != sessionTarget() || len(f.releases) != 0 || f.inserts != 0 || f.copies != 0 {
		t.Fatalf("unexpected dispatch state: %+v", f)
	}
	s.Close()
	s.Close()
	if len(f.releases) != 1 || f.releases[0] != sessionTarget() {
		t.Fatal("original target not released exactly once")
	}
	if s.Deliver(context.Background(), 4, "again") == nil || len(f.sent) != 3 {
		t.Fatal("closed session replayed")
	}
}
func TestSessionRejectionPermanentlyCloses(t *testing.T) {
	for _, kind := range []string{"duplicate", "out-of-order", "zero", "focus", "partial", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			f := &segmentFake{fake: fake{target: sessionTarget()}}
			s := beginTestSession(t, f)
			if err := s.Deliver(context.Background(), 2, "first"); err != nil {
				t.Fatal(err)
			}
			id := uint64(3)
			ctx := context.Background()
			switch kind {
			case "duplicate":
				id = 2
			case "out-of-order":
				id = 1
			case "zero":
				id = 0
			case "focus":
				f.target.FocusHWND++
			case "partial":
				f.failure = errors.New("partial dispatch")
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if err := s.Deliver(ctx, id, "second"); err == nil {
				t.Fatal("unsafe delivery accepted")
			}
			count := len(f.sent)
			f.target = sessionTarget()
			f.failure = nil
			if err := s.Deliver(context.Background(), 4, "retry"); err == nil || len(f.sent) != count || len(f.releases) != 1 || f.copies != 0 {
				t.Fatal("failed session remained usable")
			}
		})
	}
}
func TestSessionCloseCancelsInFlightSend(t *testing.T) {
	entered := make(chan struct{})
	f := &segmentFake{fake: fake{target: sessionTarget()}, send: func(ctx context.Context) error { close(entered); <-ctx.Done(); return ctx.Err() }}
	s := beginTestSession(t, f)
	done := make(chan error, 1)
	go func() { done <- s.Deliver(context.Background(), 1, "partial") }()
	<-entered
	s.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled send succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("send did not cancel")
	}
	if len(f.releases) != 1 {
		t.Fatal("close must release once")
	}
}
func TestSessionRunCancellationReleasesWithoutAnotherSegment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var released sync.WaitGroup
	released.Add(1)
	f := &releasingPlatform{segmentFake: segmentFake{fake: fake{target: sessionTarget()}}, released: &released}
	s, err := (Policy{f}).BeginSession(ctx, f.target, DirectInput)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	released.Wait()
	s.Close()
	if len(f.releases) != 1 {
		t.Fatal("cancellation did not release exactly once")
	}
}

type releasingPlatform struct {
	segmentFake
	released *sync.WaitGroup
}

func (f *releasingPlatform) ReleaseTarget(t Target) {
	f.segmentFake.ReleaseTarget(t)
	f.released.Done()
}
func TestSessionUnsupportedPreservesOneShotFallback(t *testing.T) {
	f := &fake{target: sessionTarget()}
	if _, err := (Policy{f}).BeginSession(context.Background(), f.target, DirectInput); !errors.Is(err, ErrSessionUnsupported) {
		t.Fatal(err)
	}
	if err := (Policy{f}).Deliver(context.Background(), f.target, "one-shot", DirectInput); err != nil || f.inserts != 1 {
		t.Fatal("one-shot fallback broken", err)
	}
}
func TestSessionRejectsNonDirectMode(t *testing.T) {
	for _, mode := range []Mode{ManualCopy, ClipboardPaste, Mode("unknown")} {
		f := &segmentFake{fake: fake{target: sessionTarget()}}
		if _, err := (Policy{f}).BeginSession(context.Background(), f.target, mode); err == nil || len(f.releases) != 1 || len(f.sent) != 0 || f.copies != 0 {
			t.Fatal("unsafe mode admitted", mode)
		}
	}
}
