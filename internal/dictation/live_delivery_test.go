package dictation

import (
	"context"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/realtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type segmentFixture struct {
	mu                sync.Mutex
	parts             []string
	ones              []string
	sends             chan string
	failure           error
	block             chan struct{}
	copies            int
	foregroundEntered chan struct{}
	foregroundRelease chan struct{}
}

func (*segmentFixture) CaptureTarget() (insertion.Target, error) { return validTarget(), nil }
func (p *segmentFixture) Foreground() (insertion.Target, error) {
	if p.foregroundEntered != nil {
		p.foregroundEntered <- struct{}{}
		<-p.foregroundRelease
	}
	return validTarget(), nil
}
func (p *segmentFixture) InsertUnicode(_ context.Context, _ insertion.Target, text string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ones = append(p.ones, text)
	return nil
}
func (p *segmentFixture) Copy(context.Context, string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.copies++
	return nil
}
func (*segmentFixture) ReleaseTarget(insertion.Target) {}
func (p *segmentFixture) InsertSegment(ctx context.Context, _ insertion.Target, text string) error {
	p.mu.Lock()
	p.parts = append(p.parts, text)
	p.mu.Unlock()
	if p.sends != nil {
		p.sends <- text
	}
	if p.block != nil {
		select {
		case <-p.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return p.failure
}
func finalUpdate(id uint64, text string) realtime.Update {
	return realtime.Update{Final: text, Finalized: true, EventID: "event", Turn: id}
}
func awaitSend(t *testing.T, p *segmentFixture) string {
	t.Helper()
	select {
	case s := <-p.sends:
		return s
	case <-time.After(time.Second):
		t.Fatal("no segment attempt")
		return ""
	}
}

func TestLiveDeliveryFinalsAreIncrementalAndFinishDoesNotReplay(t *testing.T) {
	p := &segmentFixture{sends: make(chan string, 10)}
	d := newLiveDelivery(t.Context(), insertion.Policy{Platform: p}, validTarget(), nil)
	d.offer(realtime.Update{Partial: "wrong"})
	d.offer(finalUpdate(2, "Hello"))
	if got := awaitSend(t, p); got != "Hello" {
		t.Fatal(got)
	}
	d.offer(finalUpdate(3, "Hello world"))
	if got := awaitSend(t, p); got != " world" {
		t.Fatal(got)
	}
	d.offer(finalUpdate(4, "Hello world"))
	got := d.wait()
	if got.err != nil || got.posted != "Hello world" || !got.attempted {
		t.Fatalf("%+v", got)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.parts) != 2 || p.copies != 0 {
		t.Fatalf("parts=%v copies=%d", p.parts, p.copies)
	}
}
func TestLiveDeliveryFailureAndMissingIdentityNeverResume(t *testing.T) {
	for _, mode := range []string{"send", "missing", "prefix"} {
		t.Run(mode, func(t *testing.T) {
			p := &segmentFixture{sends: make(chan string, 10), block: make(chan struct{})}
			if mode == "send" {
				p.failure = insertion.ErrCopyRequired
			}
			d := newLiveDelivery(t.Context(), insertion.Policy{Platform: p}, validTarget(), nil)
			d.offer(finalUpdate(2, "first"))
			awaitSend(t, p)
			if mode == "missing" {
				u := finalUpdate(3, "first second")
				u.EventID = ""
				d.offer(u)
			}
			if mode == "prefix" {
				d.offer(finalUpdate(3, "changed"))
			}
			d.offer(finalUpdate(4, "first second third"))
			close(p.block)
			got := d.wait()
			if got.err == nil || !got.attempted {
				t.Fatalf("%+v", got)
			}
			p.mu.Lock()
			defer p.mu.Unlock()
			if len(p.parts) != 1 || p.copies != 0 {
				t.Fatalf("replayed: %v", p.parts)
			}
		})
	}
}
func TestLiveDeliveryUnknownBeforeAttemptFallsBackPermanently(t *testing.T) {
	p := &segmentFixture{}
	d := newLiveDelivery(t.Context(), insertion.Policy{Platform: p}, validTarget(), nil)
	u := finalUpdate(2, "first")
	u.EventID = ""
	d.offer(u)
	d.offer(finalUpdate(3, "first second"))
	got := d.wait()
	if got.attempted || got.final != "first second" {
		t.Fatalf("%+v", got)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.parts) != 0 {
		t.Fatal(p.parts)
	}
}
func TestLiveDeliveryCancellationStopsBlockedSend(t *testing.T) {
	p := &segmentFixture{sends: make(chan string, 10), block: make(chan struct{})}
	ctx, cancel := context.WithCancel(t.Context())
	d := newLiveDelivery(ctx, insertion.Policy{Platform: p}, validTarget(), nil)
	d.offer(finalUpdate(2, "keep me"))
	awaitSend(t, p)
	cancel()
	got := d.wait()
	if !got.attempted || got.final != "keep me" || got.err == nil {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(liveRecoveryMessage("Cancelled."), "has not been removed") {
		t.Fatal("missing no-undo warning")
	}
}

func TestLiveDeliveryTimeoutPoisonsAndBoundsFinish(t *testing.T) {
	p := &segmentFixture{sends: make(chan string, 10), block: make(chan struct{})}
	d := newLiveDelivery(t.Context(), insertion.Policy{Platform: p}, validTarget(), nil)
	d.offer(finalUpdate(2, "bounded"))
	awaitSend(t, p)
	started := time.Now()
	got := d.wait()
	if got.err == nil || !got.attempted || time.Since(started) > 4*time.Second {
		t.Fatalf("unbounded or unsafe result: %+v", got)
	}
}

func TestLiveDeliveryCoalescesBlockedSuccess(t *testing.T) {
	p := &segmentFixture{sends: make(chan string, 10), block: make(chan struct{})}
	d := newLiveDelivery(t.Context(), insertion.Policy{Platform: p}, validTarget(), nil)
	d.offer(finalUpdate(2, "first"))
	awaitSend(t, p)
	d.offer(finalUpdate(3, "first second"))
	d.offer(finalUpdate(4, "first second third"))
	close(p.block)
	got := d.wait()
	if got.err != nil || got.posted != "first second third" {
		t.Fatalf("%+v", got)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.parts) != 2 || p.parts[0] != "first" || p.parts[1] != " second third" {
		t.Fatalf("not coalesced: %v", p.parts)
	}
}
