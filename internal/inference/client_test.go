package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTranscribeShape(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("bad request %s", r.URL.Path)
		}
		if e := r.ParseMultipartForm(1 << 20); e != nil {
			t.Fatal(e)
		}
		if r.FormValue("model") != "speech/stt" || r.FormValue("language") != "en" || r.FormValue("response_format") != "json" {
			t.Error("fields")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("microphone transcription must explicitly request completed JSON")
		}
		f, _, _ := r.FormFile("file")
		b, _ := io.ReadAll(f)
		if !strings.HasPrefix(string(b), "RIFF") {
			t.Error("wav")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "  héllo\nworld  "})
	}))
	defer s.Close()
	c := New()
	got, e := c.Transcribe(context.Background(), s.URL+"/v1", "speech/stt", "en", "secret", nil, append([]byte("RIFF"), make([]byte, 40)...))
	if e != nil || got.Text != "héllo\nworld" {
		t.Fatalf("%q %v", got.Text, e)
	}
}
func TestErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{{"non2xx", "no", 500}, {"malformed", "{", 200}, {"oversized", strings.Repeat("x", maxResponse+1), 200}} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer s.Close()
			_, e := New().Transcribe(context.Background(), s.URL, "m", "", "", nil, []byte("RIFF"))
			if e == nil {
				t.Fatal("expected error")
			}
		})
	}
}
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := New().Transcribe(ctx, "https://example.invalid", "m", "", "", nil, []byte("RIFF"))
	if e == nil {
		t.Fatal("expected cancellation")
	}
}

func TestTranscribeFileCompletedResponseAndUploadProgress(t *testing.T) {
	var uploaded int64
	var total int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("unexpected request path=%q authorization=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		received, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if int64(len(received)) != r.ContentLength {
			t.Fatalf("multipart length = %d, declared %d", len(received), r.ContentLength)
		}
		r.Body = io.NopCloser(bytes.NewReader(received))
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("model") != "speech/stt" || r.FormValue("language") != "en" || r.FormValue("response_format") != "json" || r.FormValue("stream") != "" {
			t.Fatalf("unexpected fields: %#v", r.MultipartForm.Value)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		body, _ := io.ReadAll(file)
		if header.Filename != "meeting.mp3" || string(body) != "stored-audio" {
			t.Fatalf("file = %q %q", header.Filename, body)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": " completed transcript "})
	}))
	defer server.Close()

	text, err := New().TranscribeFile(context.Background(), server.URL+"/v1", "speech/stt", "en", "secret", nil, "meeting.mp3", int64(len("stored-audio")), strings.NewReader("stored-audio"), false, FileTranscriptionCallbacks{
		UploadProgress: func(sent, size int64) { uploaded, total = sent, size },
	})
	if err != nil || text.Text != "completed transcript" {
		t.Fatalf("text=%q err=%v", text.Text, err)
	}
	if uploaded != int64(len("stored-audio")) || total != uploaded {
		t.Fatalf("progress=%d/%d", uploaded, total)
	}
}

func TestTranscribeFileCurrentOpenAIStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("accept=%q", r.Header.Get("Accept"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("stream") != "true" || r.FormValue("response_format") != "json" {
			t.Fatalf("unexpected fields: %#v", r.MultipartForm.Value)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: transcript.text.delta\ndata: {\"type\":\"transcript.text.delta\",\"delta\":\"Hello \"}\n\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"transcript.text.delta\",\"delta\":\"world\"}\n\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"transcript.text.done\",\"text\":\"Hello world.\"}\n\n")
	}))
	defer server.Close()
	var deltas strings.Builder
	text, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.wav", 4, strings.NewReader("RIFF"), true, FileTranscriptionCallbacks{Delta: func(delta string) { deltas.WriteString(delta) }})
	if err != nil || text.Text != "Hello world." || deltas.String() != "Hello world" {
		t.Fatalf("text=%q deltas=%q err=%v", text.Text, deltas.String(), err)
	}
}

func TestTranscribeFileOlderSpeachesStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		_, _ = io.WriteString(w, "data: {\"text\":\"first segment\"}\n\n")
		_, _ = io.WriteString(w, "data: {\"text\":\"second segment\"}\n\n")
		_, _ = io.WriteString(w, "data: {\"text\":\", with punctuation\"}\n\n")
	}))
	defer server.Close()
	text, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.ogg", 5, strings.NewReader("audio"), true, FileTranscriptionCallbacks{})
	if err != nil || text.Text != "first segment second segment, with punctuation" {
		t.Fatalf("text=%q err=%v", text.Text, err)
	}
}

func TestTranscribeFileAcceptsJSONWhenStreamingIsIgnored(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"text":"fallback result"}`)
	}))
	defer server.Close()
	buffered := false
	unsupported := ""
	text, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.m4a", 5, strings.NewReader("audio"), true, FileTranscriptionCallbacks{
		StreamBuffered:    func() { buffered = true },
		StreamUnsupported: func(reason string) { unsupported = reason },
	})
	if err != nil || text.Text != "fallback result" || !buffered || unsupported != "completed_json" {
		t.Fatalf("text=%q buffered=%v unsupported=%q err=%v", text.Text, buffered, unsupported, err)
	}
}

func TestTranscribeFileClassifiesExplicitStreamParameterRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"error":{"message":"stream is not supported"}}`)
	}))
	defer server.Close()

	_, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.wav", 4, strings.NewReader("RIFF"), true, FileTranscriptionCallbacks{})
	var unsupported *FileStreamUnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Reason != "stream_parameter_rejected" || unsupported.PartialText != "" {
		t.Fatalf("error = %#v", err)
	}
}

func TestTranscribeFileDoesNotClassifyTransientFailuresAsUnsupported(t *testing.T) {
	for _, status := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusRequestEntityTooLarge,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"error":{"message":"stream is not supported"}}`)
			}))
			defer server.Close()

			_, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.wav", 4, strings.NewReader("RIFF"), true, FileTranscriptionCallbacks{})
			var unsupported *FileStreamUnsupportedError
			if errors.As(err, &unsupported) {
				t.Fatalf("status %d was classified as unsupported: %v", status, err)
			}
		})
	}
}

func TestTranscribeFileIncompatibleSSEPreservesAcceptedPartialText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"transcript.text.delta\",\"delta\":\"kept partial\"}\n\n")
		_, _ = io.WriteString(w, "data: not-json\n\n")
	}))
	defer server.Close()

	_, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.wav", 4, strings.NewReader("RIFF"), true, FileTranscriptionCallbacks{})
	var unsupported *FileStreamUnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Reason != "invalid_sse_event" || unsupported.PartialText != "kept partial" {
		t.Fatalf("error = %#v", err)
	}
}

func TestTranscribeFileNormalizesBufferedSpeachesSSEInJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"text": "data: {\"text\":\"first segment\"}\n\ndata: {\"text\":\"second segment\"}\n\n",
		})
	}))
	defer server.Close()
	var deltas strings.Builder
	text, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.mp3", 5, strings.NewReader("audio"), true, FileTranscriptionCallbacks{Delta: func(delta string) { deltas.WriteString(delta) }})
	if err != nil || text.Text != "first segment second segment" || deltas.String() != text.Text {
		t.Fatalf("text=%q deltas=%q err=%v", text.Text, deltas.String(), err)
	}
}

func TestTranscribeFileReportsEarlyProviderSizeRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	}))
	defer server.Close()
	audio := strings.Repeat("a", 512<<10)
	_, err := New().TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "audio.mp3", int64(len(audio)), strings.NewReader(audio), false, FileTranscriptionCallbacks{})
	var requestErr *Error
	if !errors.As(err, &requestErr) || requestErr.Status != http.StatusRequestEntityTooLarge || !strings.Contains(requestErr.Message, "server upload limit") {
		t.Fatalf("error=%v", err)
	}
}

func TestTranscribeFileCancellationAfterUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	uploaded := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := New().TranscribeFile(ctx, server.URL, "m", "", "", nil, "audio.wav", 4, strings.NewReader("RIFF"), false, FileTranscriptionCallbacks{UploadComplete: func() { close(uploaded) }})
		result <- err
	}()
	select {
	case <-uploaded:
	case <-time.After(5 * time.Second):
		t.Fatal("upload did not complete")
	}
	cancel()
	select {
	case err := <-result:
		var requestErr *Error
		if !errors.As(err, &requestErr) || requestErr.Kind != "cancelled" {
			t.Fatalf("error=%v", err)
		}
		var unsupported *FileStreamUnsupportedError
		if errors.As(err, &unsupported) {
			t.Fatalf("cancellation was classified as unsupported: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled request did not return")
	}
}

func TestTranscribeFileTimeoutIsNotClassifiedAsUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := New().TranscribeFile(ctx, server.URL, "m", "", "", nil, "audio.wav", 4, strings.NewReader("RIFF"), true, FileTranscriptionCallbacks{})
	var unsupported *FileStreamUnsupportedError
	if err == nil || errors.As(err, &unsupported) {
		t.Fatalf("timeout classification = %#v", err)
	}
	var requestErr *Error
	if !errors.As(err, &requestErr) || requestErr.Kind != "timeout" {
		t.Fatalf("timeout error = %#v", err)
	}
}

func TestTranscribeDoesNotReflectCredentialFromErrorResponse(t *testing.T) {
	const secret = "reviewer-credential-canary"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("rejected " + r.Header.Get("Authorization")))
	}))
	defer s.Close()

	_, err := New().Transcribe(context.Background(), s.URL, "m", "", secret, nil, []byte("RIFF"))
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(strings.ToLower(err.Error()), "bearer") {
		t.Fatalf("credential reflected in error: %v", err)
	}
}

func TestTranscribeRejectsCredentialReflectedAsSuccessfulText(t *testing.T) {
	const secret = "successful-response-credential-canary"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "prefix " + secret + " suffix"})
	}))
	defer s.Close()

	text, err := New().Transcribe(context.Background(), s.URL, "m", "", secret, nil, []byte("RIFF"))
	if err == nil {
		t.Fatal("expected reflected credential to be rejected")
	}
	if text.Text != "" || strings.Contains(err.Error(), secret) {
		t.Fatalf("credential escaped rejection boundary: text=%q err=%v", text.Text, err)
	}
}

func TestMetadataReturnsBoundedDistinctModelsAndConfiguredPresence(t *testing.T) {
	models := make([]map[string]string, 0, MaxDiscoveredModels+4)
	for i := 0; i < MaxDiscoveredModels+2; i++ {
		models = append(models, map[string]string{"id": fmt.Sprintf("model-%03d", i)})
	}
	models = append(models, map[string]string{"id": "model-000"})
	models = append(models, map[string]string{"id": strings.Repeat("x", MaxDiscoveredModelIDBytes+1)})

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get("X-Test") != "yes" {
			t.Errorf("unexpected metadata request: path=%s authorization=%q custom=%q", r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("X-Test"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": models})
	}))
	defer s.Close()

	result := New().TestMetadata(context.Background(), s.URL+"/v1", "", "secret", "model-201", map[string]string{"X-Test": "yes"})
	if !result.Reachable || result.Probe != "models" || result.RequestedURL != s.URL+"/v1/models" || result.HTTPStatus != http.StatusOK || result.ErrorKind != "" {
		t.Fatalf("metadata result = %#v", result)
	}
	if result.ModelPresence != "listed" {
		t.Fatalf("model presence = %q, want listed", result.ModelPresence)
	}
	if len(result.ModelIDs) != MaxDiscoveredModels {
		t.Fatalf("returned model count = %d, want %d", len(result.ModelIDs), MaxDiscoveredModels)
	}
	if result.ModelIDs[0] != "model-000" || result.ModelIDs[len(result.ModelIDs)-1] != "model-199" {
		t.Fatalf("unexpected bounded model inventory: first=%q last=%q", result.ModelIDs[0], result.ModelIDs[len(result.ModelIDs)-1])
	}
}

func TestMetadataHealthProbeAndUnavailableModelInventory(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			t.Errorf("path = %q, want /ready", r.URL.Path)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer s.Close()

	result := New().TestMetadata(context.Background(), s.URL, "/ready", "", "speech/stt", nil)
	if !result.Reachable || result.Probe != "health" || result.ModelPresence != "unavailable" || result.ErrorKind != "" || len(result.ModelIDs) != 0 {
		t.Fatalf("health result = %#v", result)
	}
}

func TestMetadataReportsStructuredFailures(t *testing.T) {
	t.Run("http status", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer s.Close()
		result := New().TestMetadata(context.Background(), s.URL, "", "", "speech/stt", nil)
		if !result.Reachable || result.HTTPStatus != http.StatusUnauthorized || result.ErrorKind != "http" {
			t.Fatalf("HTTP result = %#v", result)
		}
	})

	t.Run("oversized response", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", maxResponse+1)))
		}))
		defer s.Close()
		result := New().TestMetadata(context.Background(), s.URL, "", "", "speech/stt", nil)
		if !result.Reachable || result.ErrorKind != "response_too_large" {
			t.Fatalf("oversized result = %#v", result)
		}
	})

	t.Run("malformed model list remains reachable", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		}))
		defer s.Close()
		result := New().TestMetadata(context.Background(), s.URL, "", "", "speech/stt", nil)
		if !result.Reachable || result.ErrorKind != "response" || result.ModelPresence != "unavailable" {
			t.Fatalf("malformed-list result = %#v", result)
		}
	})
}

func TestMetadataClassifiesNetworkFailures(t *testing.T) {
	if got := metadataNetworkErrorKind(context.DeadlineExceeded); got != "timeout" {
		t.Fatalf("deadline kind = %q", got)
	}
	if got := metadataNetworkErrorKind(&net.DNSError{Err: "no such host", Name: "missing.test"}); got != "dns" {
		t.Fatalf("DNS kind = %q", got)
	}
	if got := metadataNetworkErrorKind(&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}); got != "network" {
		t.Fatalf("network kind = %q", got)
	}
	s := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer s.Close()
	result := New().TestMetadata(context.Background(), s.URL, "", "", "speech/stt", nil)
	if result.ErrorKind != "tls" || result.Reachable {
		t.Fatalf("TLS result = %#v", result)
	}
}
