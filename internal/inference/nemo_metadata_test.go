package inference

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestNeMoMetadataFiltersTaskAndReadsVersionWithoutInference(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer fixture-secret" || r.Header.Get("X-Tenant") != "test" {
			t.Error("metadata request changed authentication or method")
		}
		switch r.URL.Path {
		case "/proxy/v1/models":
			io.WriteString(w, `{"data":[{"id":"asr","capability":"transcription","device":"gpu:0"},{"id":"tts","capability":"speech"},{"id":"translate","capability":"translation"},{"id":"legacy"},{"id":"new","capability":"unqualified"},{"id":"fixture-secret","capability":"transcription"}]}`)
		case "/proxy/health":
			io.WriteString(w, `{"version":"0.1.0","status":"ok"}`)
		default:
			t.Error("unexpected route")
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	r := New().TestProfileMetadata(t.Context(), compatibility.NeMoSpeechV1, compatibility.Transcription, server.URL+"/proxy/v1/", "", "fixture-secret", "tts", map[string]string{"X-Tenant": "test"})
	if r.ErrorKind != "" || r.ServerVersion != "0.1.0" || r.ModelPresence != "not-listed" || !reflect.DeepEqual(r.ModelIDs, []string{"asr", "legacy"}) || len(r.Models) != 5 || r.Models[0].Device != "gpu:0" {
		t.Fatalf("incorrect metadata: %+v", r)
	}
	if !reflect.DeepEqual(paths, []string{"/proxy/v1/models", "/proxy/health"}) {
		t.Fatalf("unexpected probes: %v", paths)
	}
}

func TestNeMoOptionalVersionDoesNotInvalidateModels(t *testing.T) {
	for _, healthBody := range []string{`{"version":"fixture-secret"}`, `<html>proxy error</html>`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/models" {
				io.WriteString(w, `{"data":[{"id":"asr","capability":"transcription"}]}`)
				return
			}
			io.WriteString(w, healthBody)
		}))
		r := New().TestProfileMetadata(t.Context(), compatibility.NeMoSpeechV1, compatibility.Transcription, server.URL+"/v1", "", "fixture-secret", "asr", nil)
		server.Close()
		if r.ErrorKind != "" || r.ModelPresence != "listed" || r.ServerVersion != "" {
			t.Fatalf("optional health metadata spoiled the model result: %+v", r)
		}
	}
}
