package inference

import (
	"context"
	"errors"
	"fmt"
	"net"
)

type Error struct {
	Kind    string
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("%s (%d): %s", e.Kind, e.Status, e.Message)
	}
	return e.Kind + ": " + e.Message
}

// DiagnosticKind exposes only the bounded client classification to logs.
func (e *Error) DiagnosticKind() string { return e.Kind }

func requestFailure(err error, ctx context.Context, operation string) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return &Error{Kind: "cancelled", Message: operation + " cancelled"}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return &Error{Kind: "timeout", Message: operation + " timed out"}
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return &Error{Kind: "timeout", Message: operation + " timed out"}
	}
	return &Error{Kind: "network", Message: operation + " failed"}
}

// FileStreamUnsupportedError carries only client-classified capability
// evidence. Provider response bodies remain behind the network boundary.
// PartialText is populated only when accepted SSE transcript events preceded
// an incompatible event, so callers can preserve that result without retrying.
type FileStreamUnsupportedError struct {
	Reason      string
	PartialText string
}

func (e *FileStreamUnsupportedError) Error() string {
	return "stream_unsupported: endpoint does not support streamed file transcripts"
}

// DiagnosticKind exposes only the bounded client classification to logs.
func (e *FileStreamUnsupportedError) DiagnosticKind() string { return "stream_unsupported" }
