package connection

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/inference"
)

type ConnectionProbe string

const (
	ConnectionProbeHealth ConnectionProbe = "health"
	ConnectionProbeModels ConnectionProbe = "models"
)

type ModelPresence string

const (
	ModelPresenceUnavailable ModelPresence = "unavailable"
	ModelPresenceListed      ModelPresence = "listed"
	ModelPresenceNotListed   ModelPresence = "not-listed"
)

type ConnectionErrorKind string

const (
	ConnectionErrorCredentialMissing     ConnectionErrorKind = "credential_missing"
	ConnectionErrorCredentialUnavailable ConnectionErrorKind = "credential_unavailable"
	ConnectionErrorRuntimeUnavailable    ConnectionErrorKind = "runtime_unavailable"
	ConnectionErrorDNS                   ConnectionErrorKind = "dns"
	ConnectionErrorTLS                   ConnectionErrorKind = "tls"
	ConnectionErrorHTTP                  ConnectionErrorKind = "http"
	ConnectionErrorResponseTooLarge      ConnectionErrorKind = "response_too_large"
	ConnectionErrorResponse              ConnectionErrorKind = "response"
	ConnectionErrorInvalidURL            ConnectionErrorKind = "invalid_url"
	ConnectionErrorInvalidSettings       ConnectionErrorKind = "invalid_settings"
	ConnectionErrorTimeout               ConnectionErrorKind = "timeout"
	ConnectionErrorNetwork               ConnectionErrorKind = "network"
)

type ConnectionResult struct {
	Models              []inference.ModelMetadata `json:"models,omitempty"`
	ServerVersion       string                    `json:"serverVersion,omitempty"`
	Checks              []Check                   `json:"checks,omitempty"`
	Reachable           bool                      `json:"reachable"`
	Probe               ConnectionProbe           `json:"probe"`
	RequestedURL        string                    `json:"requestedURL"`
	HTTPStatus          int                       `json:"httpStatus"`
	LatencyMilliseconds int64                     `json:"latencyMilliseconds"`
	ErrorKind           ConnectionErrorKind       `json:"errorKind"`
	CheckedAt           time.Time                 `json:"checkedAt"`
	ModelPresence       ModelPresence             `json:"modelPresence"`
	ModelIDs            []string                  `json:"modelIDs"`
}

// metadataLogOutcome uses the probe's bounded result taxonomy. Capture ctxErr
// before the operation's deferred cancel: cleanup must not turn a failure into
// cancellation, and cancellation after a successful result must not hide it.
// This affects diagnostics only, not the renderer's existing result contract.
func metadataLogOutcome(errorKind string, ctxErr error) (slog.Level, string, string) {
	if errorKind == "" {
		return slog.LevelInfo, "completed", ""
	}
	if errors.Is(ctxErr, context.Canceled) {
		return slog.LevelInfo, "cancelled", diagnostics.ErrorKind(ctxErr)
	}
	switch errorKind {
	case "invalid_settings", "invalid_url", "credential_missing", "unsupported":
		return slog.LevelWarn, "failed", errorKind
	default:
		return slog.LevelError, "failed", errorKind
	}
}

func safeConnectionValidationError(err error) string {
	message := err.Error()
	switch {
	case strings.HasPrefix(message, "base URL"),
		strings.HasPrefix(message, "post-processing base URL"),
		strings.HasPrefix(message, "speech playback base URL"),
		strings.HasPrefix(message, "model "),
		strings.HasPrefix(message, "post-processing model "),
		strings.HasPrefix(message, "speech playback model "),
		strings.HasPrefix(message, "language "),
		strings.HasPrefix(message, "health path"),
		strings.HasPrefix(message, "maximum duration"),
		strings.HasPrefix(message, "voice activity detection"),
		strings.HasPrefix(message, "silence splitting"),
		strings.HasPrefix(message, "segment target"),
		strings.HasPrefix(message, "segment silence"),
		strings.HasPrefix(message, "microphone identifier"):
		return message
	case strings.Contains(message, "shortcut"):
		return "one or more shortcuts are invalid"
	case strings.Contains(message, "header"):
		return "one or more custom headers are invalid"
	default:
		return "settings validation failed"
	}
}

func connectionServer(requestedURL string) string {
	parsed, err := url.Parse(requestedURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
