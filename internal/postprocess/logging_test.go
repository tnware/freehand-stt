package postprocess

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
)

type loggingTransport func(*http.Request) (*http.Response, error)

func (f loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProcessingCancellationIsAnInfoOutcome(t *testing.T) {
	var logs bytes.Buffer
	client := inference.New()
	client.HTTP.Transport = loggingTransport(func(r *http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})
	cfg := config.Default().PostProcessing
	cfg.BaseURL, cfg.Model = "https://fixture.invalid/private/path", "private-model"
	processor := New(client, slog.New(slog.NewTextHandler(&logs, nil)))
	_, err := processor.ProcessWithCredential(context.Background(), cfg, "private transcript", "private-key")
	if !errors.Is(err, context.Canceled) {
		// The inference client preserves cancellation as a bounded diagnostic category.
		if err == nil || !strings.Contains(err.Error(), "cancel") {
			t.Fatalf("expected cancellation, got %v", err)
		}
	}
	output := logs.String()
	for _, want := range []string{"transcript post-processing started", "transcript post-processing cancelled", "level=INFO", "outcome=cancelled", "error_kind=cancelled", "duration_ms="} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in logs: %s", want, output)
		}
	}
	for _, forbidden := range []string{"level=WARN", "post-processing failed", "private transcript", "private-key", "private-model", "/private/path"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("unexpected %q in logs", forbidden)
		}
	}
}
