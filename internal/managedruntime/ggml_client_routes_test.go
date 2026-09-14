package managedruntime_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/postprocess"
)

func TestManagedGGMLProductionClientRoutes(t *testing.T) {
	t.Run("llama cleanup", func(t *testing.T) {
		managedruntime.WithGGMLEndpointFixture(t, managedruntime.LlamaCPP, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/chat/completions" || r.Method != "POST" {
				t.Errorf("wrong route %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
				return
			}
			var body struct {
				Model           string
				ReasoningEffort string `json:"reasoning_effort"`
				Temperature     float64
				Messages        []struct{ Content string }
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Model != "s1-mini" || body.ReasoningEffort != "none" || body.Temperature != 0 || len(body.Messages) != 2 || body.Messages[0].Content != postprocess.S1MiniSystemInstruction {
				t.Errorf("wrong S1 request: %+v", body)
			}
			if r.Header.Get("Authorization") != "" {
				t.Error("managed credentials")
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"choices":[{"message":{"content":"Clean words."},"finish_reason":"stop"}]}`))
		}), func(ep managedruntime.Endpoint) {
			cfg := config.Default().PostProcessing
			cfg.ManagedInstanceID = string(managedruntime.LlamaCPP)
			cfg.BaseURL = ep.BaseURL
			cfg.Model = ep.Model
			cfg.AllowInsecureHTTP = true
			cfg.CompatibilityProfile = compatibility.LlamaCPP
			cfg.Preset = config.PostProcessingPresetS1Mini
			got, err := postprocess.New(inference.New(), nil).ProcessWithCredential(t.Context(), cfg, "raw words", "")
			if err != nil || got.Text != "Clean words." {
				t.Fatalf("cleanup: %+v %v", got, err)
			}
		})
	})
	t.Run("whisper completed microphone and files", func(t *testing.T) {
		managedruntime.WithGGMLEndpointFixture(t, managedruntime.WhisperCPP, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/inference" || r.Method != "POST" {
				t.Errorf("wrong route %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
				return
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			if r.FormValue("response_format") != "json" {
				t.Error("not native whisper JSON")
			}
			if r.Header.Get("Authorization") != "" {
				t.Error("managed credentials")
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"text":"fixture transcript"}`))
		}), func(ep managedruntime.Endpoint) {
			c := inference.New().WithCompatibility(compatibility.WhisperCPP).WithModelProfile(modelprofile.Generic)
			got, err := c.Transcribe(t.Context(), ep.BaseURL, ep.Model, "", "", nil, []byte("fixture"))
			if err != nil || got.Text != "fixture transcript" {
				t.Fatalf("microphone: %+v %v", got, err)
			}
			got, err = c.TranscribeFile(t.Context(), ep.BaseURL, ep.Model, "", "", nil, "fixture.wav", 7, bytes.NewBufferString("fixture"), false, inference.FileTranscriptionCallbacks{})
			if err != nil || got.Text != "fixture transcript" {
				t.Fatalf("file: %+v %v", got, err)
			}
		})
	})
}
