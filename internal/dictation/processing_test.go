package dictation

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
)

type processingTransport func(*http.Request) (*http.Response, error)

func (f processingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Exercise the real recorder -> Processor -> HTTP client path. The transport
// can return decoded success after cancellation without relying on socket timing.
func TestRecorderProcessingOutcomes(t *testing.T) {
	for _, mode := range []string{"raw", "success", "unavailable", "http-error", "empty", "timeout", "cancel-late-success", "replacement-late-success"} {
		for _, retain := range []bool{true, false} {
			name := mode + "/history-off"
			if retain {
				name = mode + "/history-on"
			}
			t.Run(name, func(t *testing.T) {
				cfg := config.Default()
				cfg.BaseURL = "https://stt.example/v1"
				cfg.Model = "speech"
				cfg.AuthenticationMode = config.AuthenticationModeNone
				cfg.HistoryEnabled = retain
				cfg.AutoInsert = true
				cfg.VADEnabled = false
				cfg.SilenceSplitting = false
				cfg.AutoStopEnabled = false
				cfg.PostProcessing.Enabled = mode != "raw"
				cfg.PostProcessing.BaseURL = "https://cleanup.example/v1"
				cfg.PostProcessing.Model = "cleanup"
				var recorder *testRecorder
				calls := 0
				client := inference.New()
				client.HTTP.Transport = processingTransport(func(r *http.Request) (*http.Response, error) {
					body, status := `{"text":"raw transcript"}`, http.StatusOK
					switch r.URL.Path {
					case "/v1/audio/transcriptions":
					case "/v1/chat/completions":
						calls++
						if r.Header.Get("Authorization") != "Bearer [REDACTED]" {
							t.Error("operation credential was not forwarded")
						}
						if retain {
							entries := recorder.History()
							if len(entries) != 1 || entries[0].RawText != "raw transcript" || entries[0].ProcessingStatus != history.HistoryProcessingPending {
								t.Error("raw transcript was not retained before cleanup")
							}
						}
						body = `{"id":"cleanup-response","choices":[{"message":{"content":"Cleaned 日本語"},"finish_reason":"stop"}]}`
						switch mode {
						case "http-error":
							status, body = http.StatusBadGateway, `{}`
						case "empty":
							body = `{"choices":[{"message":{"content":" "}}]}`
						case "timeout":
							return nil, context.DeadlineExceeded
						case "cancel-late-success", "replacement-late-success":
							if err := recorder.Cancel(); err != nil {
								t.Error(err)
							}
							if mode == "replacement-late-success" {
								if err := recorder.Start(); err != nil {
									t.Error(err)
								}
							}
						}
					default:
						t.Errorf("unexpected request path %q", r.URL.Path)
					}
					return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				})
				platform := &platFake{}
				recorder = New(capFake{}, platform, client, nil, staticSettings{value: cfg}, nil)
				t.Cleanup(func() { _ = recorder.Cancel() })
				recorder.profiles = settings.ProfileSource(func() (settings.RequestProfile, error) {
					return settings.RequestProfile{Settings: cfg, PostProcessingCredential: "[REDACTED]"}, nil
				})
				if mode != "unavailable" {
					recorder.SetPostProcessor(postprocess.New(client, nil, nil))
				}
				if err := recorder.Start(); err != nil {
					t.Fatal(err)
				}
				generation := recorder.Status().Generation
				if err := recorder.Stop(); err != nil {
					t.Fatal(err)
				}
				cancelled := strings.Contains(mode, "late-success")
				wantText, wantStatus := "raw transcript", history.HistoryProcessingFailed
				switch mode {
				case "success":
					wantText, wantStatus = "Cleaned 日本語", history.HistoryProcessingCompleted
				case "raw":
					wantStatus = history.HistoryProcessingNotRequested
				}
				if cancelled {
					wantStatus = history.HistoryProcessingCancelled
					if platform.inserts != 0 || platform.copies != 0 || recorder.Status().CanCopy {
						t.Fatal("cancelled cleanup delivered text")
					}
					if mode == "replacement-late-success" && (recorder.Status().State != Recording || recorder.Status().Generation <= generation) {
						t.Fatal("late completion changed replacement recording")
					}
				} else if platform.inserts != 1 || platform.lastInsert != wantText || platform.copies != 0 {
					t.Fatalf("delivery = %+v", platform)
				}
				if mode == "timeout" && recorder.Status().Message != "Post-processing timed out; raw transcript used" {
					t.Fatal("timeout fallback notice lost")
				}
				wantCalls := 1
				if mode == "raw" || mode == "unavailable" {
					wantCalls = 0
				}
				if calls != wantCalls {
					t.Fatalf("processing calls = %d, want %d", calls, wantCalls)
				}
				entries := recorder.History()
				if !retain {
					if len(entries) != 0 {
						t.Fatal("history retained while disabled")
					}
					return
				}
				if len(entries) != 1 {
					t.Fatalf("history entries = %d", len(entries))
				}
				entry := entries[0]
				if entry.Text != wantText || entry.RawText != "raw transcript" || entry.ProcessingStatus != wantStatus || entry.Details.Processing.Status != wantStatus {
					t.Fatalf("history = %+v", entry)
				}
				if cancelled && entry.Outcome != history.HistoryCancelled {
					t.Fatal("cancelled run was finalized as delivered")
				}
				if !cancelled && entry.Outcome != history.HistoryInserted {
					t.Fatal("delivery outcome lost")
				}
				if mode == "success" && (entry.ProcessedText != wantText || entry.Details.Processing.Response == nil || entry.Details.Processing.Response.ResponseID != "cleanup-response") {
					t.Fatal("successful cleanup text/metadata lost")
				}
				if wantStatus != history.HistoryProcessingCompleted && entry.ProcessedText != "" {
					t.Fatal("failed cleanup retained processed text")
				}
				if mode == "replacement-late-success" {
					if err := recorder.Cancel(); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
