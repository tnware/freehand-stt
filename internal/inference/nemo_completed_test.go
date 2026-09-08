package inference

import (
	"context"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNemoCompletedRequestsPreserveNativeText(t *testing.T) {
	for _, file := range []bool{false, true} {
		t.Run(map[bool]string{false: "voice", true: "file"}[file], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/audio/transcriptions" {
					t.Error("wrong route")
				}
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				if r.FormValue("verbatim") != "true" || r.FormValue("automatic_punctuation") != "true" || r.FormValue("language") != "fr-FR" || r.FormValue("response_format") != "json" {
					t.Error("wrong recognition options")
				}
				if r.FormValue("stream") != "" || r.FormValue("prompt") != "" || r.FormValue("model") != "" {
					t.Error("unsupported options sent")
				}
				f, _, err := r.FormFile("file")
				if err != nil {
					t.Error(err)
					return
				}
				defer f.Close()
				audio, _ := io.ReadAll(f)
				if string(audio) != "fixture audio" {
					t.Error("upload truncated")
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"text":"Bonjour. <fr-FR>"}`)
			}))
			defer server.Close()
			client := New().WithCompatibility(compatibility.NeMoSpeechV1).WithModelProfile(modelprofile.Nemotron35)
			var result TranscriptionResult
			var err error
			if file {
				result, err = client.TranscribeFile(context.Background(), server.URL+"/v1", "selected-alias", "fr-FR", "", nil, "fixture.wav", 13, strings.NewReader("fixture audio"), false, FileTranscriptionCallbacks{})
			} else {
				result, err = client.Transcribe(context.Background(), server.URL+"/v1", "selected-alias", "fr-FR", "", nil, []byte("fixture audio"))
			}
			if err != nil || result.Text != "Bonjour." {
				t.Fatalf("completed transcription: %v %q", err, result.Text)
			}
		})
	}
}
