// Package diagnostics defines the stable, content-free vocabulary used by
// application logs. It deliberately classifies errors without formatting them:
// an error may contain a path, URL, provider body, transcript, or credential.
package diagnostics

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
)

// ApplicationLogLevel keeps Wails bridge argument/result tracing disabled.
// Debug bridge logging is a development-only diagnostic because it can contain
// complete binding request and response payloads.
const ApplicationLogLevel = slog.LevelInfo

var discardLogger = slog.New(slog.DiscardHandler)

// DiscardLogger is the non-emitting fallback for isolated unit construction.
// The composed application always injects a child of the Wails logger instead.
func DiscardLogger() *slog.Logger { return discardLogger }

type classifiedError interface {
	DiagnosticKind() string
}

// ErrorKind returns a bounded category without exposing err.Error().
func ErrorKind(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var classified classifiedError
	if errors.As(err, &classified) {
		if kind := knownErrorKind(classified.DiagnosticKind()); kind != "" {
			return kind
		}
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return "dns"
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		if networkError.Timeout() {
			return "timeout"
		}
		return "network"
	}
	if errors.Is(err, os.ErrPermission) {
		return "permission"
	}
	if errors.Is(err, os.ErrNotExist) {
		return "not_found"
	}
	return "operation"
}

func knownErrorKind(kind string) string {
	switch kind {
	case "cancelled",
		"credential_reflection",
		"http",
		"incomplete_response",
		"invalid_file",
		"malformed_response",
		"network",
		"request",
		"request_too_large",
		"response",
		"response_too_large",
		"stream_unsupported",
		"timeout":
		return kind
	default:
		return ""
	}
}
