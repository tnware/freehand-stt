package history

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/postprocess"
)

func TestFinalizeProcessingOutcome(t *testing.T) {
	for _, test := range []struct {
		name      string
		err       error
		cancelled bool
		status    HistoryProcessingStatus
		kind      string
	}{
		{"completed", nil, false, HistoryProcessingCompleted, ""},
		{"failed", errors.New(strings.Repeat("failure ", 50)), false, HistoryProcessingFailed, "processing"},
		{"timeout", &inference.Error{Kind: "timeout", Message: "request timed out"}, false, HistoryProcessingFailed, "timeout"},
		{"cancelled", context.Canceled, true, HistoryProcessingCancelled, "cancelled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, retention := range []string{"enabled", "disabled", "absent", "deleted", "cleared", "over-budget"} {
				t.Run(retention, func(t *testing.T) {
					cfg := config.Default().PostProcessing
					cfg.Enabled = true
					raw := "raw 日本語"
					details := HistoryRunDetails{Source: HistorySourceVoice, Route: "/audio/transcriptions", Processing: NewProcessingDetails(cfg, raw)}
					store := NewStore(retention != "disabled", nil)
					if retention == "absent" {
						store = nil
					}
					id := uint64(0)
					if store != nil {
						id = store.Begin(raw, HistoryInserted, true, time.Now().UTC(), details)
						if retention == "deleted" {
							if err := store.Delete(id); err != nil {
								t.Fatal(err)
							}
						}
						if retention == "cleared" {
							store.SetEnabled(false)
						}
					}
					processed := "Café 世界"
					if retention == "over-budget" {
						processed = strings.Repeat("x", MaxHistoryBytes)
					}
					text := processed
					if test.err != nil {
						text = raw
					}
					started := time.Now().UTC().Add(-time.Second)
					outcome := postprocess.Outcome{Text: text, Err: test.err, Cancelled: test.cancelled, StartedAt: started, CompletedAt: started.Add(time.Second), ElapsedMilliseconds: 1000, Metadata: inference.ResponseMetadata{RequestID: "request"}}
					got := store.FinalizeProcessing(id, raw, outcome, details)
					want := details
					want.Processing.StartedAt = outcome.StartedAt
					want.Processing.CompletedAt = outcome.CompletedAt
					want.Processing.ElapsedMilliseconds = 1000
					want.Processing.Response = NewResponseDetails(outcome.Metadata)
					want.Processing.Status = test.status
					want.Processing.ErrorKind = test.kind
					if test.err == nil {
						want.Processing.ProcessedCharacters = utf8.RuneCountInString(processed)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("run details = %+v; want %+v", got, want)
					}
					if retention == "absent" {
						return
					}
					entries := store.Entries()
					if retention != "enabled" && retention != "over-budget" {
						if len(entries) != 0 {
							t.Fatal("processing resurrected an unretained entry")
						}
						return
					}
					if len(entries) != 1 {
						t.Fatalf("entries = %d", len(entries))
					}
					entry := entries[0]
					if entry.RawText != raw || entry.Outcome != HistoryInserted {
						t.Fatal("processing changed raw text or delivery outcome")
					}
					if retention == "over-budget" && test.err == nil {
						if entry.Text != raw || entry.ProcessedText != "" || entry.Details.Processing.ErrorKind != historyBudgetErrorKind || outcome.Text != processed || got.Processing.Status != HistoryProcessingCompleted {
							t.Fatal("history budget leaked into delivery policy")
						}
						return
					}
					wantProcessed, wantMessage := processed, ""
					if test.err != nil {
						wantProcessed = ""
						wantMessage = test.err.Error()
						if len(wantMessage) > 256 {
							wantMessage = wantMessage[:256]
						}
					}
					if entry.Text != text || entry.ProcessedText != wantProcessed || entry.ProcessingStatus != test.status || entry.ProcessingMessage != wantMessage || !reflect.DeepEqual(entry.Details, want) {
						t.Fatalf("history did not project outcome: %+v", entry)
					}
				})
			}
		})
	}
}
