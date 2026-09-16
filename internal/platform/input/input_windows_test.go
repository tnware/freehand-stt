//go:build windows

package input

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/tnware/freehand-stt/internal/insertion"
)

func inputTestTarget() insertion.Target {
	return insertion.Target{HWND: 1, FocusHWND: 2, ThreadID: 3, ProcessID: 4, ProcessCreationTime: 5}
}

func replaceInputHooks(t *testing.T) {
	t.Helper()
	oldForeground := insertionForeground
	oldDispatch := dispatchUnicodeEvents
	oldModifiers := insertionModifiersReleased
	insertionModifiersReleased = func() bool { return true }
	t.Cleanup(func() {
		insertionForeground = oldForeground
		dispatchUnicodeEvents = oldDispatch
		insertionModifiersReleased = oldModifiers
	})
}

func TestInsertUnicodeWaitsForModifiersBeforeEachBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		replaceInputHooks(t)
		target := inputTestTarget()
		insertionForeground = func() (insertion.Target, error) { return target, nil }
		blockedUntil := time.Now().Add(30 * time.Millisecond)
		insertionModifiersReleased = func() bool { return !time.Now().Before(blockedUntil) }
		dispatches := 0
		dispatchUnicodeEvents = func(events []inputEvent) uintptr {
			if time.Now().Before(blockedUntil) {
				t.Fatal("dispatched with physical modifiers held")
			}
			dispatches++
			blockedUntil = time.Now().Add(30 * time.Millisecond)
			return uintptr(len(events))
		}
		started := time.Now()
		if err := (Input{}).InsertUnicode(t.Context(), target, strings.Repeat("x", unicodeInputFastPathUnits+1)); err != nil {
			t.Fatal(err)
		}
		if dispatches != 3 || time.Since(started) != 90*time.Millisecond {
			t.Fatalf("dispatches=%d, waited=%s", dispatches, time.Since(started))
		}
	})
}

func TestInsertUnicodeModifierWaitRejectsWithoutDispatch(t *testing.T) {
	for _, test := range []struct {
		name      string
		interrupt func(context.CancelFunc, *insertion.Target)
		want      error
	}{
		{name: "timeout", want: insertion.ErrCopyRequired},
		{name: "cancel", interrupt: func(cancel context.CancelFunc, _ *insertion.Target) { cancel() }, want: context.Canceled},
		{name: "focus", interrupt: func(_ context.CancelFunc, target *insertion.Target) { target.HWND++ }, want: insertion.ErrCopyRequired},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				replaceInputHooks(t)
				target := inputTestTarget()
				current := target
				insertionForeground = func() (insertion.Target, error) { return current, nil }
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				insertionModifiersReleased = func() bool {
					if test.interrupt != nil {
						test.interrupt(cancel, &current)
					}
					return false
				}
				dispatchUnicodeEvents = func([]inputEvent) uintptr { t.Fatal("unexpected dispatch"); return 0 }
				if err := (Input{}).InsertUnicode(ctx, target, "private text"); !errors.Is(err, test.want) {
					t.Fatalf("error=%v, want %v", err, test.want)
				}
			})
		})
	}
}

func TestInsertUnicodeUsesSingleDispatchForOrdinaryTranscript(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	foregroundChecks := 0
	insertionForeground = func() (insertion.Target, error) {
		foregroundChecks++
		return target, nil
	}
	dispatches := 0
	eventCount := 0
	dispatchUnicodeEvents = func(events []inputEvent) uintptr {
		dispatches++
		eventCount = len(events)
		return uintptr(len(events))
	}

	text := strings.Repeat("x", unicodeInputFastPathUnits)
	if err := (Input{}).InsertUnicode(context.Background(), target, text); err != nil {
		t.Fatal(err)
	}
	if dispatches != 1 || eventCount != unicodeInputFastPathUnits*2 || foregroundChecks != 1 {
		t.Fatalf("dispatches=%d events=%d foreground checks=%d", dispatches, eventCount, foregroundChecks)
	}
}

func TestInsertUnicodeUsesLargerFocusCheckedBatchesForLongTranscript(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	foregroundChecks := 0
	insertionForeground = func() (insertion.Target, error) {
		foregroundChecks++
		return target, nil
	}
	var batchSizes []int
	dispatchUnicodeEvents = func(events []inputEvent) uintptr {
		batchSizes = append(batchSizes, len(events))
		return uintptr(len(events))
	}

	text := strings.Repeat("x", unicodeInputFastPathUnits+1)
	if err := (Input{}).InsertUnicode(context.Background(), target, text); err != nil {
		t.Fatal(err)
	}
	wantSizes := []int{unicodeInputLongBatchUnits * 2, unicodeInputLongBatchUnits * 2, 2}
	if len(batchSizes) != len(wantSizes) {
		t.Fatalf("batch sizes = %v, want %v", batchSizes, wantSizes)
	}
	for index, want := range wantSizes {
		if batchSizes[index] != want {
			t.Fatalf("batch sizes = %v, want %v", batchSizes, wantSizes)
		}
	}
	if foregroundChecks != len(wantSizes) {
		t.Fatalf("foreground checks = %d, want %d", foregroundChecks, len(wantSizes))
	}
}

func TestInsertUnicodeStopsBeforeBatchWhenFocusChanges(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	checks := 0
	insertionForeground = func() (insertion.Target, error) {
		checks++
		if checks == 1 {
			return target, nil
		}
		changed := target
		changed.HWND++
		return changed, nil
	}
	dispatches := 0
	dispatchUnicodeEvents = func(events []inputEvent) uintptr {
		dispatches++
		return uintptr(len(events))
	}
	err := (Input{}).InsertUnicode(context.Background(), target, strings.Repeat("x", unicodeInputFastPathUnits+1))
	if !errors.Is(err, insertion.ErrCopyRequired) || dispatches != 1 {
		t.Fatalf("error = %v, dispatches = %d", err, dispatches)
	}
}

func TestInsertUnicodeStopsBeforeNextBatchWhenCancelled(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	insertionForeground = func() (insertion.Target, error) { return target, nil }
	ctx, cancel := context.WithCancel(context.Background())
	dispatches := 0
	dispatchUnicodeEvents = func(events []inputEvent) uintptr {
		dispatches++
		cancel()
		return uintptr(len(events))
	}

	err := (Input{}).InsertUnicode(ctx, target, strings.Repeat("x", unicodeInputFastPathUnits+1))
	if !errors.Is(err, context.Canceled) || dispatches != 1 {
		t.Fatalf("error = %v, dispatches = %d", err, dispatches)
	}
}

func TestInsertUnicodeKeepsSurrogatePairInOneBatch(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	insertionForeground = func() (insertion.Target, error) { return target, nil }
	var batchSizes []int
	dispatchUnicodeEvents = func(events []inputEvent) uintptr {
		batchSizes = append(batchSizes, len(events))
		return uintptr(len(events))
	}
	text := strings.Repeat("x", unicodeInputLongBatchUnits-1) + "😀" + strings.Repeat("y", unicodeInputFastPathUnits)
	if err := (Input{}).InsertUnicode(context.Background(), target, text); err != nil {
		t.Fatal(err)
	}
	wantSizes := []int{(unicodeInputLongBatchUnits - 1) * 2, unicodeInputLongBatchUnits * 2, unicodeInputLongBatchUnits * 2, 4}
	if len(batchSizes) != len(wantSizes) {
		t.Fatalf("batch sizes = %v, want %v", batchSizes, wantSizes)
	}
	for index, want := range wantSizes {
		if batchSizes[index] != want {
			t.Fatalf("batch sizes = %v, want %v", batchSizes, wantSizes)
		}
	}
}

func TestInsertUnicodeRejectsPartialBatch(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	insertionForeground = func() (insertion.Target, error) { return target, nil }
	dispatchUnicodeEvents = func(events []inputEvent) uintptr { return uintptr(len(events) - 1) }
	var logs bytes.Buffer
	input := NewInput(slog.New(slog.NewTextHandler(&logs, nil)))
	const transcript = "partial private transcript"

	err := input.InsertUnicode(context.Background(), target, transcript)
	if err == nil || errors.Is(err, insertion.ErrCopyRequired) {
		t.Fatalf("partial batch error = %v", err)
	}
	output := logs.String()
	for _, want := range []string{"direct input failed", "batch_count=1", "stage=dispatch", "error_kind=operation"} {
		if !strings.Contains(output, want) {
			t.Fatalf("log %q does not contain %q", output, want)
		}
	}
	if strings.Contains(output, transcript) {
		t.Fatalf("log contains transcript content: %q", output)
	}
}

func TestInsertUnicodeLogsOnlyBoundedDeliveryMetadata(t *testing.T) {
	replaceInputHooks(t)
	target := inputTestTarget()
	insertionForeground = func() (insertion.Target, error) { return target, nil }
	dispatchUnicodeEvents = func(events []inputEvent) uintptr { return uintptr(len(events)) }
	var logs bytes.Buffer
	input := NewInput(slog.New(slog.NewTextHandler(&logs, nil)))
	const transcript = "private transcript content"

	if err := input.InsertUnicode(context.Background(), target, transcript); err != nil {
		t.Fatal(err)
	}
	output := logs.String()
	for _, want := range []string{"direct input completed", "utf16_units=26", "batch_count=1", "strategy=single"} {
		if !strings.Contains(output, want) {
			t.Fatalf("log %q does not contain %q", output, want)
		}
	}
	if strings.Contains(output, transcript) {
		t.Fatalf("log contains transcript content: %q", output)
	}
}
