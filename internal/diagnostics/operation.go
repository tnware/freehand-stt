package diagnostics

import (
	"context"
	"log/slog"
)

// Workflow identifies a product operation, not a provider, model or user label.
type Workflow string

const (
	Dictation Workflow = "dictation"
	File      Workflow = "file"
)

type operationKey struct{}
type operation struct {
	workflow   Workflow
	generation uint64
}

// WithOperation carries immutable correlation into a shared processing stage.
// It neither owns cancellation nor changes the injected logger's destination.
func WithOperation(ctx context.Context, workflow Workflow, generation uint64) context.Context {
	if (workflow != Dictation && workflow != File) || generation == 0 {
		return ctx
	}
	return context.WithValue(ctx, operationKey{}, operation{workflow, generation})
}

// OperationLogger retains the component owner while identifying the parent run.
func OperationLogger(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return DiscardLogger()
	}
	if op, ok := ctx.Value(operationKey{}).(operation); ok {
		return logger.With("workflow", string(op.workflow), "generation", op.generation)
	}
	return logger
}
