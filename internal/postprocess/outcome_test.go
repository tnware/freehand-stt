package postprocess

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/inference"
)

func TestResolveOutcome(t *testing.T) {
	failed := errors.New("cleanup failed")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	deadline, cancelDeadline := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelDeadline()
	for _, test := range []struct {
		name                string
		ctx                 context.Context
		result              string
		err                 error
		wantText            string
		wantErr             error
		cancelled, fallback bool
	}{
		{name: "success preserves exact Unicode", result: "  Café 世界\n", wantText: "  Café 世界\n"},
		{name: "failure discards partial cleanup", result: "partial", err: failed, wantText: "raw 日本語", wantErr: failed, fallback: true},
		{name: "unavailable processor", err: errors.New("post-processing is unavailable"), wantText: "raw 日本語", fallback: true},
		{name: "processor timeout falls back", err: context.DeadlineExceeded, wantText: "raw 日本語", wantErr: context.DeadlineExceeded, fallback: true},
		{name: "processor cancellation without owner cancellation falls back", err: fmt.Errorf("processor: %w", context.Canceled), wantText: "raw 日本語", wantErr: context.Canceled, fallback: true},
		{name: "owner cancellation beats late success", ctx: cancelled, result: "late success", wantText: "raw 日本語", wantErr: context.Canceled, cancelled: true},
		{name: "owner cancellation beats failure", ctx: cancelled, err: failed, wantText: "raw 日本語", wantErr: context.Canceled, cancelled: true},
		{name: "owner deadline beats late success but is not user cancellation", ctx: deadline, result: "late success", wantText: "raw 日本語", wantErr: context.DeadlineExceeded, fallback: true},
		{name: "resolver does not redefine processor output validation", result: "", wantText: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := test.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			metadata := inference.ResponseMetadata{RequestID: "request", EffectiveModel: "cleanup", FinishReason: "stop"}
			started := time.Now().Add(-10 * time.Millisecond)
			got := Resolve(ctx, "raw 日本語", Result{Text: test.result, Metadata: metadata}, test.err, started)
			wantErr := test.wantErr
			if wantErr == nil {
				wantErr = test.err
			}
			if got.Text != test.wantText || !errors.Is(got.Err, wantErr) || got.Cancelled != test.cancelled || got.Fallback() != test.fallback {
				t.Fatalf("outcome = %+v; fallback=%v", got, got.Fallback())
			}
			if !reflect.DeepEqual(got.Metadata, metadata) {
				t.Fatal("response metadata lost")
			}
			if !got.StartedAt.Equal(started) || got.CompletedAt.Before(started) || got.CompletedAt.After(time.Now()) || got.ElapsedMilliseconds != got.CompletedAt.Sub(got.StartedAt).Milliseconds() {
				t.Fatalf("invalid timing: %+v", got)
			}
		})
	}
}
