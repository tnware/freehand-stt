package filetranscription

import (
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
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
)

func TestFileStreamingFailureNeverAutomaticallyResubmits(t *testing.T) {
	const delta = "data: {\"type\":\"transcript.text.delta\",\"delta\":\"kept partial\"}\n\n"
	for _, tc := range []struct {
		name, body, contentType, partial string
		status                           int
		unsupported, completed           bool
	}{
		{"parameter rejected", `{"error":{"message":"stream is not supported"}}`, "application/json", "", 422, true, false},
		{"unrecognized successful SSE", "data: {\"choices\":[{\"delta\":{\"content\":\"already inferred\"}}]}\n\n", "text/event-stream", "", 200, true, false},
		{"malformed first event", "data: not-json\n\n", "text/event-stream", "", 200, true, false},
		{"malformed after partial", delta + "data: not-json\n\n", "text/event-stream", "kept partial", 200, true, false},
		{"typed EOF after partial", delta, "text/event-stream", "kept partial", 200, false, false},
		{"empty SSE", "", "text/event-stream", "", 200, false, false},
		{"completed JSON uses original response", `{"text":"completed once"}`, "application/json", "completed once", 200, true, true},
		{"buffered typed EOF", `{"text":"data: {\"type\":\"transcript.text.delta\",\"delta\":\"kept partial\"}\n\n"}`, "application/json", "kept partial", 200, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.BaseURL = "https://stt.example/v1"
			cfg.Model = "speech"
			cfg.AuthenticationMode = config.AuthenticationModeNone
			cfg.SetupCompleted = true
			cfg.PostProcessing.Enabled = !tc.completed
			cfg.PostProcessing.BaseURL = "https://cleanup.example/v1"
			cfg.PostProcessing.Model = "cleanup"
			calls := 0
			requestedStreams := []string{}
			client := inference.New()
			client.HTTP.Transport = processingTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/v1/audio/transcriptions" {
					t.Error("failed partial transcript invoked cleanup")
					_, _ = io.Copy(io.Discard, r.Body)
				} else {
					if err := r.ParseMultipartForm(1 << 20); err != nil {
						t.Error(err)
					}
					if r.MultipartForm != nil {
						defer r.MultipartForm.RemoveAll()
					}
					requestedStreams = append(requestedStreams, r.FormValue("stream"))
				}
				body, status, contentType := tc.body, tc.status, tc.contentType
				if calls > 1 {
					body, status, contentType = `{"text":"explicit retry result"}`, 200, "application/json"
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			transcripts := history.NewStore(true, nil)
			input := &processingInput{}
			service := NewService(settings.Source(func() config.Settings { return cfg }), settings.ProfileSource(func() (settings.RequestProfile, error) {
				return settings.RequestProfile{Settings: cfg}, nil
			}), client, postprocess.New(client, nil), transcripts, input, nil, nil, nil, nil, nil)
			t.Cleanup(func() { _ = service.ServiceShutdown() })
			path := filepath.Join(t.TempDir(), "recording.wav")
			if err := os.WriteFile(path, []byte("RIFF audio"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := service.selectAudioFile(path); err != nil {
				t.Fatal(err)
			}
			wait := func() {
				t.Helper()
				done := make(chan struct{})
				go func() { service.workers.Wait(); close(done) }()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Fatal("transcription did not finish")
				}
			}
			if err := service.StartFileTranscription(true); err != nil {
				t.Fatal(err)
			}
			wait()
			status := service.CurrentFileTranscription()
			wantPhase := FileTranscriptionFailed
			if tc.completed {
				wantPhase = FileTranscriptionCompleted
			}
			if calls != 1 || len(requestedStreams) != 1 || requestedStreams[0] != "true" {
				t.Fatalf("unexpected requests: %d %v", calls, requestedStreams)
			}
			if status.Phase != wantPhase || status.Transcript != tc.partial || status.StreamingUnavailable != tc.unsupported || !status.CanStart || status.CanCancel || status.CanCopy != (tc.partial != "") {
				t.Fatalf("status = %+v", status)
			}
			if input.copies != 0 || input.inserts != 0 {
				t.Fatal("transcript delivered without explicit copy")
			}
			entries := transcripts.Entries()
			if tc.partial != "" {
				if len(entries) != 1 || entries[0].RawText != tc.partial {
					t.Fatalf("partial history = %+v", entries)
				}
				if !tc.completed && entries[0].Outcome != history.HistoryFailed {
					t.Fatalf("partial marked successful: %+v", entries[0])
				}
				if err := service.CopyFileTranscript(); err != nil {
					t.Fatal(err)
				}
				if input.text != tc.partial {
					t.Fatalf("copied %q", input.text)
				}
			} else if len(entries) != 0 {
				t.Fatal("empty failure created history")
			}
			if tc.unsupported && !tc.completed {
				cfg.PostProcessing.Enabled = false
				if err := service.StartFileTranscription(false); err != nil {
					t.Fatal(err)
				}
				wait()
				status = service.CurrentFileTranscription()
				if calls != 2 || requestedStreams[1] != "" || status.Phase != FileTranscriptionCompleted || status.Transcript != "explicit retry result" {
					t.Fatalf("explicit retry: calls=%d streams=%v status=%+v", calls, requestedStreams, status)
				}
			}
		})
	}
}
