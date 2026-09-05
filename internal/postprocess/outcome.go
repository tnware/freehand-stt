package postprocess

import (
	"context"
	"errors"
	"time"

	"github.com/tnware/freehand-stt/internal/inference"
)

// Outcome is the delivery decision for one completed cleanup attempt. It is
// operation-local, not retained state or a renderer DTO. Text is the processed
// result on success and the original raw text on failure. Cancelled forbids
// delivery; keeping raw text available does not authorize inserting or copying it.
type Outcome struct {
	Text                string
	Err                 error
	Cancelled           bool
	StartedAt           time.Time
	CompletedAt         time.Time
	ElapsedMilliseconds int64
	Metadata            inference.ResponseMetadata
}

// Fallback distinguishes a recoverable cleanup failure from user cancellation.
// A processor timeout is a failure, not cancellation of the owning workflow.
func (o Outcome) Fallback() bool { return o.Err != nil && !o.Cancelled }

// Resolve gives the owning operation's context precedence over a late processor
// response, then selects processed text or raw fallback. The caller still owns
// admission, processor invocation, generation checks and delivery. Empty-output
// validation remains with Processor, alongside the response contract.
func Resolve(ctx context.Context, raw string, result Result, err error, started time.Time) Outcome {
	ctxErr := ctx.Err()
	if ctxErr != nil {
		err = ctxErr
	}
	completed := time.Now()
	outcome := Outcome{
		Text:                result.Text,
		Err:                 err,
		Cancelled:           errors.Is(ctxErr, context.Canceled),
		StartedAt:           started.UTC(),
		CompletedAt:         completed.UTC(),
		ElapsedMilliseconds: completed.Sub(started).Milliseconds(),
		Metadata:            result.Metadata,
	}
	if err != nil {
		outcome.Text = raw
	}
	return outcome
}
