package filetranscription

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
)

type processingTransport func(*http.Request) (*http.Response, error)

func (f processingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type processingInput struct {
	inserts, copies int
	text            string
}

func (*processingInput) CaptureTarget() (insertion.Target, error) { return insertion.Target{}, nil }
func (*processingInput) Foreground() (insertion.Target, error)    { return insertion.Target{}, nil }
func (p *processingInput) InsertUnicode(context.Context, insertion.Target, string) error {
	p.inserts++
	return nil
}
func (p *processingInput) Copy(_ context.Context, text string) error {
	p.copies++
	p.text = text
	return nil
}

func TestFileProcessingOutcomes(t *testing.T) {
	for _, mode := range []string{"raw", "success", "unavailable", "http-error", "empty", "timeout"} {
		for _, retention := range []string{"enabled", "disabled", "absent"} {
			t.Run(mode+"/history-"+retention, func(t *testing.T) {
				cfg := config.Default()
				cfg.BaseURL = "https://stt.example/v1"
				cfg.Model = "speech"
				cfg.AuthenticationMode = config.AuthenticationModeNone
				cfg.SetupCompleted = true
				cfg.PostProcessing.Enabled = mode != "raw"
				cfg.PostProcessing.BaseURL = "https://cleanup.example/v1"
				cfg.PostProcessing.Model = "cleanup"
				transcripts := history.NewStore(retention == "enabled", nil)
				if retention == "absent" {
					transcripts = nil
				}
				calls := 0
				client := inference.New()
				client.HTTP.Transport = processingTransport(func(r *http.Request) (*http.Response, error) {
					if r.Body != nil {
						_, _ = io.Copy(io.Discard, r.Body)
						_ = r.Body.Close()
					}
					body, status := `{"text":"raw transcript"}`, http.StatusOK
					switch r.URL.Path {
					case "/v1/audio/transcriptions":
					case "/v1/chat/completions":
						calls++
						if r.Header.Get("Authorization") != "Bearer [REDACTED]" {
							t.Error("operation credential was not forwarded")
						}
						if retention == "enabled" {
							entries := transcripts.Entries()
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
						}
					default:
						t.Errorf("unexpected request path %q", r.URL.Path)
					}
					return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				})
				var processor transcriptProcessor
				if mode != "unavailable" {
					processor = postprocess.New(client, nil)
				}
				input := &processingInput{}
				service := NewService(settings.Source(func() config.Settings { return cfg }), settings.ProfileSource(func() (settings.RequestProfile, error) {
					return settings.RequestProfile{Settings: cfg, PostProcessingCredential: "[REDACTED]"}, nil
				}), client, processor, transcripts, input, nil, nil, nil, nil, nil)
				t.Cleanup(func() { _ = service.ServiceShutdown() })
				path := filepath.Join(t.TempDir(), "recording.wav")
				if err := os.WriteFile(path, []byte("RIFF audio"), 0o600); err != nil {
					t.Fatal(err)
				}
				if _, err := service.selectAudioFile(path); err != nil {
					t.Fatal(err)
				}
				if err := service.StartFileTranscription(false); err != nil {
					t.Fatal(err)
				}
				done := make(chan struct{})
				go func() { service.workers.Wait(); close(done) }()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Fatal("file transcription did not finish")
				}
				wantText, wantStatus := "raw transcript", history.HistoryProcessingFailed
				switch mode {
				case "success":
					wantText, wantStatus = "Cleaned 日本語", history.HistoryProcessingCompleted
				case "raw":
					wantStatus = history.HistoryProcessingNotRequested
				}
				status := service.CurrentFileTranscription()
				if status.Phase != FileTranscriptionCompleted || status.Transcript != wantText || !status.CanCopy || !status.CanStart || status.CanCancel {
					t.Fatalf("status = %+v", status)
				}
				if mode == "timeout" && status.Message != "Transcription complete; post-processing timed out, using raw text" {
					t.Fatal("timeout fallback notice lost")
				}
				if input.copies != 0 || input.inserts != 0 {
					t.Fatal("file transcription delivered without explicit copy")
				}
				if err := service.CopyFileTranscript(); err != nil {
					t.Fatal(err)
				}
				if input.copies != 1 || input.text != wantText || input.inserts != 0 {
					t.Fatalf("explicit copy = %+v", input)
				}
				wantCalls := 1
				if mode == "raw" || mode == "unavailable" {
					wantCalls = 0
				}
				if calls != wantCalls {
					t.Fatalf("processing calls = %d, want %d", calls, wantCalls)
				}
				if transcripts == nil {
					return
				}
				entries := transcripts.Entries()
				if retention == "disabled" {
					if len(entries) != 0 {
						t.Fatal("history retained while disabled")
					}
					return
				}
				if len(entries) != 1 {
					t.Fatalf("history entries = %d", len(entries))
				}
				entry := entries[0]
				if entry.Text != wantText || entry.RawText != "raw transcript" || entry.ProcessingStatus != wantStatus || entry.Details.Processing.Status != wantStatus || entry.Outcome != history.HistoryTranscribed {
					t.Fatalf("history = %+v", entry)
				}
				if mode == "success" && (entry.ProcessedText != wantText || entry.Details.Processing.Response == nil || entry.Details.Processing.Response.ResponseID != "cleanup-response") {
					t.Fatal("successful cleanup text/metadata lost")
				}
				if wantStatus != history.HistoryProcessingCompleted && entry.ProcessedText != "" {
					t.Fatal("failed cleanup retained processed text")
				}
			})
		}
	}
}
