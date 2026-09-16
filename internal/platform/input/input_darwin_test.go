//go:build darwin && cgo

package input

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"
	"unicode/utf16"

	"github.com/tnware/freehand-stt/internal/insertion"
)

// A fake is necessary here: deterministic tests must never type into the user's
// foreground app, prompt for permissions, or replace the real clipboard.
type testDarwinInputBackend struct {
	pid                               uint32
	started                           uint64
	matches                           bool
	captures, releases, sends, copies int
	chunks                            [][]uint16
	sendHook                          func() bool
	modifierHook                      func() bool
	validHook                         func() bool
	resets                            int
	copyHook                          func(context.Context) error
	captureErr                        error
	reason                            string
}

func (b *testDarwinInputBackend) capture() (uint32, uint64, error) {
	b.captures++
	return b.pid, b.started, b.captureErr
}
func (b *testDarwinInputBackend) valid(pid uint32, start uint64) bool {
	if b.validHook != nil && !b.validHook() {
		return false
	}
	return b.matches && pid == b.pid && start == b.started
}
func (b *testDarwinInputBackend) modifiersReleased() bool {
	if b.modifierHook != nil {
		return b.modifierHook()
	}
	return true
}
func (b *testDarwinInputBackend) send(u []uint16) bool {
	b.sends++
	b.chunks = append(b.chunks, append([]uint16(nil), u...))
	if b.sendHook != nil {
		return b.sendHook()
	}
	return true
}
func (b *testDarwinInputBackend) copyText(ctx context.Context, _ <-chan struct{}, _ string) error {
	b.copies++
	if b.copyHook != nil {
		return b.copyHook(ctx)
	}
	return nil
}
func (b *testDarwinInputBackend) close() { b.releases++ }
func testDarwinInput() (Input, *testDarwinInputBackend) {
	b := &testDarwinInputBackend{pid: 42, started: 99, matches: true}
	return newDarwinInput(b), b
}
func (b *testDarwinInputBackend) resetTarget() { b.resets++; b.reason = "" }
func (b *testDarwinInputBackend) rejection(stage insertion.Stage) error {
	return insertion.NewRejection(stage, b.reason)
}

func TestDarwinRejectionSurvivesTargetReset(t *testing.T) {
	for _, stage := range []insertion.Stage{insertion.Capture, insertion.Validate, insertion.Send} {
		t.Run(string(stage), func(t *testing.T) {
			i, b := testDarwinInput()
			defer i.Close()
			want := insertion.NewRejection(stage, "value_not_settable")
			var err error
			if stage == insertion.Capture {
				b.captureErr = want
				_, err = i.CaptureTarget()
			} else {
				target, _ := i.CaptureTarget()
				if stage == insertion.Validate {
					b.validHook = func() bool { b.reason = "value_not_settable"; return false }
					_, err = i.Foreground()
				} else {
					b.sendHook = func() bool { b.reason = "value_not_settable"; return false }
					err = i.InsertUnicode(context.Background(), target, "test")
				}
			}
			if !errors.Is(err, insertion.ErrCopyRequired) || insertion.CopyRequiredMessage(err) != insertion.CopyRequiredMessage(want) {
				t.Fatalf("lost %s rejection across cleanup: %v", stage, err)
			}
			if b.reason != "" || i.state.target.Valid() {
				t.Fatal("rejection retained native target")
			}
			b.captureErr = nil
			b.validHook = nil
			b.sendHook = nil
			if _, err := i.CaptureTarget(); err != nil {
				t.Fatal("stale reason", err)
			}
			if _, err := i.Foreground(); err != nil {
				t.Fatal("stale reason", err)
			}
		})
	}
}

func TestDarwinInsertionReleasesTarget(t *testing.T) {
	for _, mode := range []string{"success", "cancel", "invalid", "focus", "stale"} {
		t.Run(mode, func(t *testing.T) {
			i, b := testDarwinInput()
			defer i.Close()
			target, _ := i.CaptureTarget()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			text := "hello"
			switch mode {
			case "cancel":
				cancel()
			case "invalid":
				text = "bad\x00"
			case "focus":
				b.matches = false
			case "stale":
				_, _ = i.CaptureTarget()
			}
			before := b.resets
			_ = i.InsertUnicode(ctx, target, text)
			want := 1
			if mode == "stale" {
				want = 0
			}
			if b.resets-before != want {
				t.Fatalf("native target releases=%d want=%d", b.resets-before, want)
			}
			if mode == "stale" {
				if _, err := i.Foreground(); err != nil {
					t.Fatal("stale operation released newer target")
				}
			}
		})
	}
}

func TestDarwinUnicodeChunksPreserveScalars(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	target, _ := i.CaptureTarget()
	text := strings.Repeat("x", 19) + "😀世界e\u0301\n" + strings.Repeat("🧪", 30)
	if err := i.InsertUnicode(context.Background(), target, text); err != nil {
		t.Fatal(err)
	}
	var all []uint16
	for _, chunk := range b.chunks {
		if len(chunk) == 0 || len(chunk) > 20 {
			t.Fatalf("unbounded Quartz payload: %d", len(chunk))
		}
		if chunk[len(chunk)-1] >= 0xD800 && chunk[len(chunk)-1] <= 0xDBFF {
			t.Fatal("split surrogate pair")
		}
		if chunk[0] >= 0xDC00 && chunk[0] <= 0xDFFF {
			t.Fatal("orphan low surrogate")
		}
		all = append(all, chunk...)
	}
	if got := string(utf16.Decode(all)); got != text {
		t.Fatalf("Unicode corrupted: %q", got)
	}
	if b.copies != 0 {
		t.Fatal("direct insertion modified clipboard")
	}
	// Posting has no delivery acknowledgement; replay must never be automatic.
	if err := i.InsertUnicode(context.Background(), target, text); err == nil {
		t.Fatal("consumed target allowed replay")
	}
}

func TestDarwinInsertionCancellation(t *testing.T) {
	for _, afterFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "between-chunks"}[afterFirst], func(t *testing.T) {
			i, b := testDarwinInput()
			defer i.Close()
			target, _ := i.CaptureTarget()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if afterFirst {
				b.sendHook = func() bool { cancel(); return true }
			} else {
				cancel()
			}
			err := i.InsertUnicode(ctx, target, strings.Repeat("x", 60))
			wantSends := 0
			if afterFirst {
				wantSends = 1
			}
			if !errors.Is(err, context.Canceled) || b.sends != wantSends || b.copies != 0 {
				t.Fatalf("cancel err=%v sends=%d copies=%d", err, b.sends, b.copies)
			}
		})
	}
}

func TestDarwinModifierReleaseIsBoundedAndRevalidated(t *testing.T) {
	for _, mode := range []string{"timeout", "focus-change", "cancel", "close", "release"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				i, b := testDarwinInput()
				defer i.Close()
				target, _ := i.CaptureTarget()
				started := time.Now()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				b.modifierHook = func() bool { return mode != "timeout" && time.Since(started) >= 30*time.Millisecond }
				if mode == "focus-change" {
					b.validHook = func() bool { return time.Since(started) < 30*time.Millisecond }
				}
				if mode == "cancel" || mode == "close" {
					done := make(chan struct{})
					go func() {
						defer close(done)
						time.Sleep(15 * time.Millisecond)
						if mode == "cancel" {
							cancel()
						} else {
							_ = i.Close()
						}
					}()
					defer func() { <-done }()
				}
				err := i.InsertUnicode(ctx, target, "hello")
				if mode == "release" {
					if err != nil || b.sends != 1 {
						t.Fatalf("released: %v sends=%d", err, b.sends)
					}
				} else if err == nil || b.sends != 0 || b.copies != 0 {
					t.Fatalf("unsafe modifier wait %s: %v sends=%d", mode, err, b.sends)
				}
				if time.Since(started) > 300*time.Millisecond {
					t.Fatal("modifier release wait unbounded")
				}
			})
		})
	}
}

func TestDarwinInvalidUnicodeNeverDispatches(t *testing.T) {
	for _, text := range []string{"bad\x00text", "bad\xfftext"} {
		i, b := testDarwinInput()
		target, _ := i.CaptureTarget()
		if err := i.InsertUnicode(context.Background(), target, text); err == nil || b.sends != 0 {
			t.Fatal("invalid input dispatched", err)
		}
		_ = i.Close()
	}
}
func TestDarwinOversizedTextNeverReachesBackend(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	target, _ := i.CaptureTarget()
	text := strings.Repeat("x", (1<<20)+1)
	if err := i.InsertUnicode(context.Background(), target, text); err == nil || b.sends != 0 {
		t.Fatal("oversized insertion dispatched")
	}
	if err := i.Copy(context.Background(), text); err == nil || b.copies != 0 {
		t.Fatal("oversized copy dispatched")
	}
}

func TestDarwinExplicitCopyRejectionIsNotInsertionAdvice(t *testing.T) {
	i, _ := testDarwinInput()
	defer i.Close()
	for _, text := range []string{"bad\x00", "bad\xff"} {
		err := i.Copy(context.Background(), text)
		if !errors.Is(err, insertion.ErrCopyNotPerformed) || errors.Is(err, insertion.ErrCopyRequired) {
			t.Fatalf("misleading explicit-copy error: %v", err)
		}
	}
}

func TestDarwinCopyWaitOutcomes(t *testing.T) {
	// State values are the bridge enum; the native fixture exercises its transitions.
	for _, running := range []bool{false, true} {
		for _, timeout := range []bool{false, true} {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if !timeout {
					go func() { time.Sleep(25 * time.Millisecond); cancel() }()
				}
				state := 0
				if running {
					state = 1
				}
				cancelled := false
				start := time.Now()
				err := waitDarwinCopy(ctx, nil, func() int { return state }, func() int {
					cancelled = true
					if !running {
						state = 4
					}
					return state
				})
				want := insertion.ErrCopyNotPerformed
				if running {
					want = insertion.ErrCopyAmbiguous
				}
				cause := context.Canceled
				if timeout {
					cause = context.DeadlineExceeded
				}
				if !cancelled || !errors.Is(err, want) || !errors.Is(err, cause) || errors.Is(err, insertion.ErrCopyRequired) {
					t.Fatalf("outcome running=%v timeout=%v: %v", running, timeout, err)
				}
				if time.Since(start) > 250*time.Millisecond {
					t.Fatal("caller waiting unbounded")
				}
			})
		}
	}
	for _, state := range []int{2, 3} {
		err := waitDarwinCopy(context.Background(), nil, func() int { return state }, func() int { t.Fatal("cancelled completed copy"); return 0 })
		if state == 2 && err != nil || state == 3 && !errors.Is(err, insertion.ErrCopyFailed) {
			t.Fatal(err)
		}
	}
}

func TestDarwinCopyPropagatesCancellation(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.copyHook = func(got context.Context) error {
		cancel()
		if got.Err() != context.Canceled {
			t.Fatal("native copy did not receive caller cancellation")
		}
		return got.Err()
	}
	if err := i.Copy(ctx, "never"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDarwinCopyAdmissionCancellation(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	i.state.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- i.Copy(ctx, "never") }()
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("copy cancellation blocked on insertion ownership")
	}
	i.state.mu.Unlock()
	if b.copies != 0 {
		t.Fatal("cancelled admission copied")
	}
}

func TestDarwinExplicitCopyAndCancellation(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := i.Copy(ctx, "private"); !errors.Is(err, context.Canceled) || b.copies != 0 {
		t.Fatal("cancelled copy mutated clipboard", err)
	}
	for _, text := range []string{"bad\xff", "bad\x00"} {
		if err := i.Copy(context.Background(), text); err == nil || b.copies != 0 {
			t.Fatal("invalid copy accepted")
		}
	}
	if err := i.Copy(context.Background(), "Unicode 世界 😀"); err != nil || b.copies != 1 || b.sends != 0 {
		t.Fatal("explicit copy did not use clipboard only", err)
	}
	_ = i.Close()
	if err := i.Copy(context.Background(), "never"); err == nil || b.copies != 1 {
		t.Fatal("copy after close")
	}
}
func TestDarwinChangedFocusAndAmbiguousSendNeverRetry(t *testing.T) {
	for _, mode := range []string{"focus", "pid-reuse", "dispatch"} {
		t.Run(mode, func(t *testing.T) {
			i, b := testDarwinInput()
			defer i.Close()
			target, _ := i.CaptureTarget()
			b.sendHook = func() bool {
				if mode == "focus" {
					b.matches = false
				}
				if mode == "pid-reuse" {
					b.started++
				}
				return mode != "dispatch"
			}
			if err := i.InsertUnicode(context.Background(), target, strings.Repeat("x", 60)); err == nil || b.sends != 1 || b.copies != 0 {
				t.Fatal("continued after ambiguity", err)
			}
			b.matches = true
			b.started = target.ProcessCreationTime
			b.sendHook = nil
			if err := i.InsertUnicode(context.Background(), target, "retry"); err == nil || b.sends != 1 {
				t.Fatal("ambiguous target replayed")
			}
		})
	}
}

func TestDarwinNativeOwnerCloseIsInert(t *testing.T) {
	// Native allocation/release only: never captures the real user's target,
	// requests permissions, dispatches events, or writes the clipboard.
	i := NewInput(nil)
	if _, err := i.Foreground(); err == nil {
		t.Fatal("uncaptured native owner accepted target")
	}
	if err := i.InsertUnicode(context.Background(), insertion.Target{}, "never"); err == nil {
		t.Fatal("uncaptured native owner dispatched")
	}
	if err := i.Close(); err != nil {
		t.Fatal(err)
	}
	if err := i.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := i.CaptureTarget(); err == nil {
		t.Fatal("closed native owner captured")
	}
}
func TestDarwinOwnProcessCaptureRejected(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	b.pid = uint32(os.Getpid())
	if target, err := i.CaptureTarget(); err == nil || target.Valid() {
		t.Fatal("own process accepted")
	}
}
func TestDarwinCaptureOwnerAndGeneration(t *testing.T) {
	i, b := testDarwinInput()
	defer i.Close()
	first, err := i.CaptureTarget()
	if err != nil || !first.Valid() || first.DarwinToken == 0 || first.HWND != 0 || first.ThreadID != 0 {
		t.Fatalf("native identity: %+v %v", first, err)
	}
	current, err := i.Foreground()
	if err != nil || current != first {
		t.Fatal("matching focus changed identity", err)
	}
	second, err := i.CaptureTarget()
	if err != nil || second == first {
		t.Fatal("recapture must revoke previous generation")
	}
	if err := i.InsertUnicode(context.Background(), first, "never"); !errors.Is(err, insertion.ErrCopyRequired) || b.sends != 0 {
		t.Fatal("stale capture inserted", err)
	}
	other, _ := testDarwinInput()
	defer other.Close()
	foreign, _ := other.CaptureTarget()
	if foreign == second {
		t.Fatal("different owners share an identity")
	}
	if err := i.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := i.CaptureTarget(); err == nil {
		t.Fatal("closed input accepted capture")
	}
	if b.releases != 1 {
		t.Fatal("close must release native owner exactly once")
	}
}
