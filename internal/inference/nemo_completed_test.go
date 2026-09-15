package inference

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestNemoCompletedRequestsPreserveNativeText(t *testing.T) {
	for _, controls := range []compatibility.NeMoOptions{{}, {DisablePunctuation: true, Normalize: true, ProfanityFilter: true, EndpointingMilliseconds: 1200}} {
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
					if r.FormValue("verbatim") != strconv.FormatBool(!controls.Normalize) || r.FormValue("automatic_punctuation") != strconv.FormatBool(!controls.DisablePunctuation) || r.FormValue("profanity_filter") != strconv.FormatBool(controls.ProfanityFilter) || r.FormValue("endpointing_ms") != "" || r.FormValue("language") != "fr-FR" || r.FormValue("response_format") != "verbose_json" {
						t.Error("wrong recognition options")
					}
					if r.FormValue("stream") != "" || r.FormValue("prompt") != "" || r.FormValue("model") != "" {
						t.Error("unsupported options sent")
					}
					var contexts []struct {
						Phrases []string `json:"phrases"`
						Boost   float64  `json:"boost"`
					}
					if err := json.Unmarshal([]byte(r.FormValue("speech_contexts")), &contexts); err != nil || len(contexts) != 1 || strings.Join(contexts[0].Phrases, "\n") != "Freehand\nNew York" || contexts[0].Boost != 2.5 {
						t.Error("wrong speech contexts")
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
				client := New().WithCompatibility(compatibility.NeMoSpeechV1).WithModelProfile(modelprofile.Nemotron35).WithTranscriptionOptions(compatibility.TranscriptionOptions{NeMo: controls, Vocabulary: "Freehand\nNew York", VocabularyBoost: 2.5})
				var result TranscriptionResult
				var err error
				if file {
					result, err = client.TranscribeFile(context.Background(), server.URL+"/v1", "selected-alias", "fr-FR", "", nil, "fixture.wav", 13, strings.NewReader("fixture audio"), false, FileTranscriptionCallbacks{})
				} else {
					result, err = client.Transcribe(context.Background(), server.URL+"/v1", "selected-alias", "fr-FR", "", nil, []byte("fixture audio"))
				}
				if err != nil || result.Text != "Bonjour." || strings.Join(result.Metadata.DetectedLanguages, ",") != "fr-FR" {
					t.Fatalf("completed transcription: %v %q", err, result.Text)
				}
			})
		}
	}

}

func TestNeMoLanguageTagCannotHideCredentialReflection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"text":"Bonjour. <fr-FR>"}`)
	}))
	defer server.Close()
	client := New().WithCompatibility(compatibility.NeMoSpeechV1).WithModelProfile(modelprofile.Nemotron35)
	for _, file := range []bool{false, true} {
		var result TranscriptionResult
		var err error
		if file {
			result, err = client.TranscribeFile(t.Context(), server.URL+"/v1", "selected", "auto", "fr-FR", nil, "fixture.wav", 13, strings.NewReader("fixture audio"), false, FileTranscriptionCallbacks{})
		} else {
			result, err = client.Transcribe(t.Context(), server.URL+"/v1", "selected", "auto", "fr-FR", nil, []byte("fixture audio"))
		}
		var failure *Error
		if !errors.As(err, &failure) || failure.Kind != "credential_reflection" || result.Text != "" || len(result.Metadata.DetectedLanguages) != 0 {
			t.Fatalf("file=%v: credential reflection was retained", file)
		}
	}
}
