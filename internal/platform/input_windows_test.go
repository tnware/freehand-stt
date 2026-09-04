//go:build windows

package platform

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/insertion"
)

func inputTestTarget() insertion.Target {
	return insertion.Target{HWND: 1, FocusHWND: 2, ThreadID: 3, ProcessID: 4, ProcessCreationTime: 5}
}

func replaceInputHooks(t *testing.T) {
	t.Helper()
	oldForeground := insertionForeground
	oldDispatch := dispatchUnicodeEvents
	t.Cleanup(func() {
		insertionForeground = oldForeground
		dispatchUnicodeEvents = oldDispatch
	})
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
