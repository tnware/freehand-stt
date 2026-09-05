package inference

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestLanguageMappingForMicrophoneAndFiles(t *testing.T) {
	for _, profile := range []compatibility.ID{compatibility.Generic, compatibility.Speaches, compatibility.WhisperCPP, compatibility.VLLM} {
		for _, selected := range []string{"", "auto", "en", "es", "ja", "haw", "yue", "custom-Lang"} {
			for _, file := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/file-%t", profile, selected, file), func(t *testing.T) {
					calls := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						if err := r.ParseMultipartForm(1 << 20); err != nil {
							t.Error(err)
							return
						}
						defer r.MultipartForm.RemoveAll()
						want := selected
						if selected == "auto" && profile != compatibility.WhisperCPP {
							want = ""
						}
						values, exists := r.MultipartForm.Value["language"]
						if want == "" && exists {
							t.Error("server default/detection must omit the language field")
						}
						if want != "" && (len(values) != 1 || values[0] != want) {
							t.Errorf("language = %v, want %q", values, want)
						}
						if r.FormValue("response_format") != "json" {
							t.Error("changed response format")
						}
						if file {
							// ParseMultipartForm reads the exact declared body; a wrong size fails the request.
							if r.ContentLength <= 0 {
								t.Error("file request lost content length")
							}
						}
						io.WriteString(w, `{"text":"Hello"}`)
					}))
					defer server.Close()
					client := New().WithCompatibility(profile)
					var err error
					if file {
						_, err = client.TranscribeFile(context.Background(), server.URL, "model", selected, "", nil, "sample.wav", 4, strings.NewReader("RIFF"), false, FileTranscriptionCallbacks{})
					} else {
						_, err = client.Transcribe(context.Background(), server.URL, "model", selected, "", nil, []byte("RIFF"))
					}
					if err != nil || calls != 1 {
						t.Fatalf("err=%v calls=%d", err, calls)
					}
				})
			}
		}
	}
}

func TestInvalidLanguageDoesNotSendOrReadAudio(t *testing.T) {
	for _, language := range []string{"en\n", "en\x00", strings.Repeat("x", 33), string([]byte{0xff})} {
		client := New()
		var validationErr *Error
		if _, err := client.Transcribe(context.Background(), "http://127.0.0.1:1", "model", language, "", nil, []byte("RIFF")); !errors.As(err, &validationErr) || validationErr.Kind != "invalid_settings" {
			t.Fatalf("invalid microphone language was not rejected as a settings error: %v", err)
		}
		if _, err := client.TranscribeFile(context.Background(), "http://127.0.0.1:1", "model", language, "", nil, "sample.wav", 4, failRead{t}, false, FileTranscriptionCallbacks{}); !errors.As(err, &validationErr) || validationErr.Kind != "invalid_settings" {
			t.Fatalf("invalid file language was not rejected before the request: %v", err)
		}
	}
}

type failRead struct{ t *testing.T }

func (r failRead) Read([]byte) (int, error) {
	r.t.Fatal("read audio before language validation")
	return 0, io.EOF
}
