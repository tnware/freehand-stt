package history

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
)

func historyEntry(id uint64, text string) HistoryEntry {
	return HistoryEntry{ID: id, Text: text, CompletedAt: time.Unix(int64(id), 0), Outcome: HistoryInserted}
}

func TestHistoryEvictsOldestEntriesByCount(t *testing.T) {
	var history historyBuffer
	for i := uint64(1); i <= MaxHistoryEntries+1; i++ {
		history.add(historyEntry(i, fmt.Sprintf("entry-%d", i)))
	}

	entries := history.newestFirst()
	if len(entries) != MaxHistoryEntries {
		t.Fatalf("entry count = %d, want %d", len(entries), MaxHistoryEntries)
	}
	if entries[0].ID != MaxHistoryEntries+1 || entries[len(entries)-1].ID != 2 {
		t.Fatalf("unexpected retained IDs: newest=%d oldest=%d", entries[0].ID, entries[len(entries)-1].ID)
	}
}

func TestHistoryEvictsOldestEntriesByTotalBytes(t *testing.T) {
	var history historyBuffer
	chunk := strings.Repeat("x", MaxHistoryBytes/2)
	history.add(historyEntry(1, chunk))
	history.add(historyEntry(2, chunk))
	history.add(historyEntry(3, "newest"))

	entries := history.newestFirst()
	if len(entries) != 2 || entries[0].ID != 3 || entries[1].ID != 2 {
		t.Fatalf("retained entries = %#v, want IDs 3 and 2", entries)
	}
	if history.bytes > MaxHistoryBytes {
		t.Fatalf("history bytes = %d, limit = %d", history.bytes, MaxHistoryBytes)
	}
}

func TestHistoryMutationKeepsBoundedRawFallbackWhenProcessedTextIsOversized(t *testing.T) {
	var history historyBuffer
	raw := strings.Repeat("r", MaxHistoryBytes/2)
	history.add(historyEntry(1, raw))
	if !history.update(1, func(entry *HistoryEntry) {
		entry.Text = strings.Repeat("p", MaxHistoryBytes)
		entry.ProcessedText = entry.Text
		entry.ProcessingStatus = HistoryProcessingCompleted
		entry.Details.Processing.Status = HistoryProcessingCompleted
	}) {
		t.Fatal("history update did not find the entry")
	}

	entries := history.newestFirst()
	if len(entries) != 1 {
		t.Fatalf("history = %#v", entries)
	}
	entry := entries[0]
	if entry.Text != raw || entry.RawText != raw || entry.ProcessedText != "" {
		t.Fatal("oversized processed text did not fall back to the retained raw transcript")
	}
	if entry.ProcessingStatus != HistoryProcessingFailed || entry.ProcessingMessage != historyBudgetMessage {
		t.Fatalf("processing fallback = %#v", entry)
	}
	if entry.Details.Processing.Status != HistoryProcessingFailed || entry.Details.Processing.ErrorKind != historyBudgetErrorKind {
		t.Fatalf("processing details = %#v", entry.Details.Processing)
	}
	assertHistoryBudget(t, &history)

	// Final run details arrive after processing. They must not accidentally
	// resurrect a completed status for text that was intentionally omitted.
	history.update(1, func(entry *HistoryEntry) {
		entry.Details.Processing.Status = HistoryProcessingCompleted
		entry.Details.Processing.ErrorKind = ""
	})
	entry = history.newestFirst()[0]
	if entry.ProcessingStatus != HistoryProcessingFailed || entry.Details.Processing.ErrorKind != historyBudgetErrorKind {
		t.Fatalf("later details erased the budget fallback: %#v", entry)
	}
	assertHistoryBudget(t, &history)
}

func TestHistoryMutationRemovesEntryWhenBoundedRawTextCannotFit(t *testing.T) {
	var history historyBuffer
	history.add(historyEntry(1, "small"))
	history.update(1, func(entry *HistoryEntry) {
		entry.Text = strings.Repeat("x", MaxHistoryBytes+1)
		entry.RawText = entry.Text
	})
	if entries := history.newestFirst(); len(entries) != 0 {
		t.Fatalf("oversized raw entry was retained: %#v", entries)
	}
	assertHistoryBudget(t, &history)
}

func TestHistoryOldestEntryMutationUsesTheSameEvictionAccounting(t *testing.T) {
	var history historyBuffer
	chunk := strings.Repeat("x", MaxHistoryBytes/4)
	history.add(historyEntry(1, chunk))
	history.add(historyEntry(2, chunk))
	history.add(historyEntry(3, chunk))
	history.update(1, func(entry *HistoryEntry) {
		entry.Text = strings.Repeat("o", MaxHistoryBytes*3/4)
		entry.RawText = entry.Text
	})
	entries := history.newestFirst()
	if len(entries) != 2 || entries[0].ID != 3 || entries[1].ID != 2 {
		t.Fatalf("retained entries = %#v, want newer IDs 3 and 2", entries)
	}
	assertHistoryBudget(t, &history)
}

func assertHistoryBudget(t *testing.T, history *historyBuffer) {
	t.Helper()
	if len(history.entries) > MaxHistoryEntries || history.bytes > MaxHistoryBytes || history.bytes < 0 {
		t.Fatalf("history invariant: entries=%d bytes=%d", len(history.entries), history.bytes)
	}
	want := 0
	for _, entry := range history.entries {
		want += historyEntryBytes(entry)
	}
	if history.bytes != want {
		t.Fatalf("history bytes = %d, recomputed = %d", history.bytes, want)
	}
}

func TestHistoryCountsUnicodeCharacters(t *testing.T) {
	var history historyBuffer
	history.add(historyEntry(1, "hé🙂"))
	if got := history.newestFirst()[0].CharacterCount; got != 3 {
		t.Fatalf("character count = %d, want 3", got)
	}
}

func TestHistoryDetailsAreBoundedAndCopied(t *testing.T) {
	segments := make([]HistorySegmentDetails, MaxHistorySegments+1)
	for i := range segments {
		segments[i] = HistorySegmentDetails{Number: i + 1, Boundary: "silence"}
	}
	segments[1].Boundary = strings.Repeat("boundary", 32)
	entry := historyEntry(1, "kept")
	entry.Details = HistoryRunDetails{
		Source:   HistorySourceVoice,
		Segments: segments,
	}
	var history historyBuffer
	history.add(entry)

	segments[0].Boundary = "mutated outside"
	first := history.newestFirst()[0]
	if len(first.Details.Segments) != MaxHistorySegments || !first.Details.SegmentsTruncated {
		t.Fatalf("segments=%d truncated=%v", len(first.Details.Segments), first.Details.SegmentsTruncated)
	}
	if first.Details.Segments[0].Boundary != "silence" {
		t.Fatal("stored details shared the caller's segment slice")
	}
	if len(first.Details.Segments[1].Boundary) != 64 {
		t.Fatalf("segment boundary length = %d, want 64", len(first.Details.Segments[1].Boundary))
	}

	first.Details.Segments[0].Boundary = "mutated result"
	if got := history.newestFirst()[0].Details.Segments[0].Boundary; got != "silence" {
		t.Fatalf("history result mutated retained details: %q", got)
	}
	if history.bytes <= len(entry.Text) {
		t.Fatal("history memory budget did not include run metadata")
	}
}

func TestHistoryResponseMetadataIsOptionalBoundedAndCopied(t *testing.T) {
	created := int64(1788200000)
	inputTokens := int64(120)
	cost := 0.0012
	metadata := inference.ResponseMetadata{
		RequestID:          strings.Repeat("r", 300),
		EffectiveModel:     "returned-model",
		DetectedLanguages:  []string{"en", "es"},
		CreatedAtUnix:      &created,
		ServerAudioSeconds: floatHistoryPointer(12.5),
		Usage: inference.Usage{
			InputTokens:  &inputTokens,
			ReportedCost: &cost,
		},
		RequestCount:     2,
		UsageReportCount: 1,
		CostReportCount:  1,
	}
	details := NewResponseDetails(metadata)
	entry := historyEntry(1, "kept")
	entry.Details.Transcription = details
	var history historyBuffer
	history.add(entry)

	metadata.DetectedLanguages[0] = "mutated"
	created = 0
	inputTokens = 0
	cost = 0
	retained := history.newestFirst()[0].Details.Transcription
	if retained == nil || len(retained.RequestID) != 256 || retained.EffectiveModel != "returned-model" {
		t.Fatalf("response details = %#v", retained)
	}
	if retained.DetectedLanguages[0] != "en" || retained.CreatedAtUnix == nil || *retained.CreatedAtUnix != 1788200000 {
		t.Fatalf("copied identity metadata = %#v", retained)
	}
	if retained.Usage.InputTokens == nil || *retained.Usage.InputTokens != 120 || retained.Usage.ReportedCost == nil || *retained.Usage.ReportedCost != 0.0012 {
		t.Fatalf("copied usage = %#v", retained.Usage)
	}
	if retained.RequestCount != 2 || retained.UsageReportCount != 1 || retained.CostReportCount != 1 {
		t.Fatalf("report coverage = %#v", retained)
	}

	retained.DetectedLanguages[0] = "result mutation"
	*retained.Usage.InputTokens = 1
	again := history.newestFirst()[0].Details.Transcription
	if again.DetectedLanguages[0] != "en" || *again.Usage.InputTokens != 120 {
		t.Fatalf("returned metadata mutated history: %#v", again)
	}

	if empty := NewResponseDetails(inference.ResponseMetadata{RequestCount: 1}); empty != nil {
		t.Fatalf("request count alone made metadata visible: %#v", empty)
	}
}

func floatHistoryPointer(value float64) *float64 {
	return &value
}

func TestHistoryRemovesOneEntryAndReleasesItsBytes(t *testing.T) {
	var history historyBuffer
	history.add(historyEntry(1, "oldest"))
	history.add(historyEntry(2, "remove me"))
	history.add(historyEntry(3, "newest"))

	if !history.remove(2) {
		t.Fatal("remove returned false for an existing entry")
	}
	entries := history.newestFirst()
	if len(entries) != 2 || entries[0].ID != 3 || entries[1].ID != 1 {
		t.Fatalf("retained entries = %#v, want IDs 3 and 1", entries)
	}
	if want := historyEntryBytes(entries[0]) + historyEntryBytes(entries[1]); history.bytes != want {
		t.Fatalf("history bytes = %d, want %d", history.bytes, want)
	}
	if history.remove(2) {
		t.Fatal("remove returned true for a missing entry")
	}
}

type copyPlatform struct{ copies int }

func (*copyPlatform) CaptureTarget() (insertion.Target, error) { return insertion.Target{}, nil }
func (*copyPlatform) Foreground() (insertion.Target, error)    { return insertion.Target{}, nil }
func (*copyPlatform) InsertUnicode(context.Context, insertion.Target, string) error {
	return nil
}
func (p *copyPlatform) Copy(context.Context, string) error { p.copies++; return nil }

func TestStoreDisableAndCloseClearMemory(t *testing.T) {
	store := NewStore(true, &copyPlatform{})
	store.Begin("first", HistoryInserted, false, time.Now(), HistoryRunDetails{})
	store.SetEnabled(false)
	if entries := store.Entries(); len(entries) != 0 {
		t.Fatalf("history survived disable: %#v", entries)
	}
	store.SetEnabled(true)
	store.Begin("second", HistoryCopyRequired, false, time.Now(), HistoryRunDetails{})
	store.Close()
	if entries := store.Entries(); len(entries) != 0 {
		t.Fatalf("history survived close: %#v", entries)
	}
}

func TestStoreStaysEmptyUntilEnabled(t *testing.T) {
	store := NewStore(false, &copyPlatform{})
	store.Begin("not retained", HistoryInserted, false, time.Now(), HistoryRunDetails{})
	if entries := store.Entries(); len(entries) != 0 {
		t.Fatalf("disabled history retained text: %#v", entries)
	}
}

func TestStoreCopiesOnlyOnExplicitAction(t *testing.T) {
	platform := &copyPlatform{}
	store := NewStore(true, platform)
	id := store.Begin("copy me", HistoryInserted, false, time.Now(), HistoryRunDetails{})
	if platform.copies != 0 {
		t.Fatal("history copied without an explicit action")
	}
	if err := store.CopyEntry(id); err != nil {
		t.Fatal(err)
	}
	if platform.copies != 1 {
		t.Fatalf("clipboard copies = %d, want 1", platform.copies)
	}
}

func TestStoreDeletesOnlySelectedEntry(t *testing.T) {
	store := NewStore(true, &copyPlatform{})
	store.Begin("keep", HistoryInserted, false, time.Now(), HistoryRunDetails{})
	removed := store.Begin("remove", HistoryInserted, false, time.Now(), HistoryRunDetails{})
	if err := store.Delete(removed); err != nil {
		t.Fatal(err)
	}
	remaining := store.Entries()
	if len(remaining) != 1 || remaining[0].Text != "keep" {
		t.Fatalf("history = %#v, want only the unselected entry", remaining)
	}
	if err := store.Delete(removed); err == nil {
		t.Fatal("deleting a missing entry succeeded")
	}
}
