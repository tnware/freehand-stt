package filetranscription

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
)

func TestProviderFileWorkflowSelectionAndPartialRecovery(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.WhisperCPP, compatibility.VLLM} {
		t.Run(string(id), func(t *testing.T) {
			cfg := config.Default()
			cfg.BaseURL = "https://speech.example"
			cfg.AuthenticationMode = config.AuthenticationModeNone
			cfg.SetupCompleted = true
			cfg.CompatibilityProfile = id
			cfg.Model = ""
			if id == compatibility.VLLM {
				cfg.Model = "speech"
				cfg.PostProcessing.Enabled = true
				cfg.PostProcessing.Model = "cleanup"
			}
			calls := 0
			client := inference.New()
			client.HTTP.Transport = processingTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					return nil, err
				}
				defer r.MultipartForm.RemoveAll()
				contentType, body := "application/json", `{"text":"Completed once"}`
				if id == compatibility.WhisperCPP {
					if r.URL.Path != "/inference" || r.FormValue("stream") != "" || r.FormValue("model") != "" {
						t.Error("native file did not select completed mode")
					}
				} else {
					if r.URL.Path != "/audio/transcriptions" || r.FormValue("stream") != "true" {
						t.Error("unexpected vLLM request or automatic cleanup")
					}
					contentType = "text/event-stream"
					body = "data: {\"object\":\"transcription.chunk\",\"choices\":[{\"delta\":{\"content\":\"Accepted partial\"},\"finish_reason\":\"stop\"}]}\n\n"
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			input := &processingInput{}
			transcripts := history.NewStore(true, nil)
			service := NewService(settings.Source(func() config.Settings { return cfg }), settings.ProfileSource(func() (settings.RequestProfile, error) { return settings.RequestProfile{Settings: cfg}, nil }), client, postprocess.New(client, nil), transcripts, input, nil, nil, nil, nil, nil)
			t.Cleanup(func() { service.ServiceShutdown() })
			path := filepath.Join(t.TempDir(), "sample.wav")
			if err := os.WriteFile(path, []byte("RIFF audio"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := service.selectAudioFile(path); err != nil {
				t.Fatal(err)
			}
			if err := service.StartFileTranscription(true); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			go func() { service.workers.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("file job did not finish")
			}
			status := service.CurrentFileTranscription()
			if calls != 1 || input.inserts != 0 || input.copies != 0 {
				t.Fatalf("unexpected request/delivery counts %d %+v", calls, input)
			}
			if id == compatibility.WhisperCPP {
				if status.Phase != FileTranscriptionCompleted || status.Streaming || !status.StreamingProfileUnavailable {
					t.Fatalf("native status=%+v", status)
				}
				if service.TryFileStreamingAgain() == nil {
					t.Fatal("native profile permitted streaming retry")
				}
			} else {
				if status.Phase != FileTranscriptionFailed || status.Transcript != "Accepted partial" || !status.CanCopy || status.StreamingUnavailable {
					t.Fatalf("partial status=%+v", status)
				}
				entries := transcripts.Entries()
				if len(entries) != 1 || entries[0].Outcome != history.HistoryFailed {
					t.Fatal("partial history marked successful")
				}
			}
		})
	}
}
