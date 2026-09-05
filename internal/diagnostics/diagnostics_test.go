package diagnostics

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"testing"
)

type classifiedTestError struct {
	kind    string
	message string
}

func (e classifiedTestError) Error() string          { return e.message }
func (e classifiedTestError) DiagnosticKind() string { return e.kind }

func TestApplicationLogLevelKeepsBridgePayloadTracingOff(t *testing.T) {
	if ApplicationLogLevel != slog.LevelInfo {
		t.Fatalf("ApplicationLogLevel = %v, want info", ApplicationLogLevel)
	}
}

func TestErrorKindReturnsBoundedCategories(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", want: ""},
		{name: "cancelled", err: context.Canceled, want: "cancelled"},
		{name: "deadline", err: context.DeadlineExceeded, want: "timeout"},
		{name: "incomplete response", err: classifiedTestError{kind: "incomplete_response", message: "private partial text"}, want: "incomplete_response"},
		{name: "classified", err: classifiedTestError{kind: "response_too_large", message: "secret provider body"}, want: "response_too_large"},
		{name: "unknown classification", err: classifiedTestError{kind: "secret provider body", message: "secret provider body"}, want: "operation"},
		{name: "dns", err: &net.DNSError{Err: "secret resolver detail", Name: "private.example"}, want: "dns"},
		{name: "permission", err: os.ErrPermission, want: "permission"},
		{name: "not found", err: os.ErrNotExist, want: "not_found"},
		{name: "generic", err: errors.New("secret path and transcript"), want: "operation"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ErrorKind(test.err); got != test.want {
				t.Fatalf("ErrorKind() = %q, want %q", got, test.want)
			}
		})
	}
}
