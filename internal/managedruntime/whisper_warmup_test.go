package managedruntime

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWhisperWarmupUsesOnlyBoundedSyntheticInput(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/inference" || r.Header.Get("Authorization") != "" {
			t.Error("unexpected warmup route/credentials")
		}
		if err := r.ParseMultipartForm(64 << 10); err != nil {
			t.Error(err)
			return
		}
		defer r.MultipartForm.RemoveAll()
		for key, value := range map[string]string{"language": "en", "response_format": "json", "temperature_inc": "0", "best_of": "1", "beam_size": "1", "duration": "1000", "vad": "false", "prompt": ""} {
			if r.FormValue(key) != value {
				t.Errorf("%s=%q", key, r.FormValue(key))
			}
		}
		f, h, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		data, _ := io.ReadAll(f)
		if h.Filename != "warmup.wav" || len(data) != 44+16000*2 || string(data[:4]) != "RIFF" || binary.LittleEndian.Uint32(data[24:28]) != 16000 || !bytes.Equal(data[44:], make([]byte, 16000*2)) {
			t.Error("warmup is not fixed one-second silent PCM")
		}
		io.WriteString(w, `{"text":"silence can hallucinate; discard this"}`)
	}))
	defer server.Close()
	if err := warmWhisper(t.Context(), server.URL); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("warmup retried", calls)
	}
}

func TestWhisperWarmupRejectsProtocolFailureAndHonorsCancellation(t *testing.T) {
	for _, response := range []string{`{"error":"failure"}`, `{}`, strings.Repeat("x", (64<<10)+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, response) }))
		if err := warmWhisper(t.Context(), server.URL); err == nil {
			t.Error("invalid warmup response accepted")
		}
		server.Close()
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if err := warmWhisper(ctx, server.URL); err == nil {
		t.Fatal("cancelled warmup succeeded")
	}
	if err := warmWhisper(t.Context(), "https://example.com"); err == nil {
		t.Fatal("non-owned remote warmup accepted")
	}
}
