package postprocess

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
)

func TestS1MiniUsesDocumentedControlLineWithoutProviderSpecificOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			ChatTemplateKwargs map[string]bool `json:"chat_template_kwargs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Messages) != 2 || body.Messages[0].Content != S1MiniSystemInstruction {
			t.Fatalf("messages = %#v", body.Messages)
		}
		wantUser := "[Styling: formal] [Structure: lists] [Context: email]\nraw words"
		if body.Messages[1].Content != wantUser {
			t.Fatalf("user prompt = %q, want %q", body.Messages[1].Content, wantUser)
		}
		if body.ChatTemplateKwargs != nil {
			t.Fatalf("S1 profile sent provider-specific options: %#v", body.ChatTemplateKwargs)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Clean words."}}]}`))
	}))
	defer server.Close()

	cfg := config.Default().PostProcessing
	cfg.BaseURL = server.URL + "/v1"
	cfg.AllowInsecureHTTP = true
	cfg.Model = "s1-mini"
	cfg.Preset = config.PostProcessingPresetS1Mini
	cfg.Styling = "formal"
	cfg.Structure = "lists"
	cfg.Context = "email"
	processor := New(inference.New(), nil, nil)
	got, err := processor.Process(context.Background(), cfg, "raw words")
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Clean words." {
		t.Fatalf("processed = %q", got.Text)
	}
}

func TestCustomProfileUsesStoredInstructionWithoutS1Controls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			ChatTemplateKwargs map[string]bool `json:"chat_template_kwargs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Messages) != 2 || body.Messages[0].Content != "My cleanup instruction." || body.Messages[1].Content != "raw words" {
			t.Fatalf("messages = %#v", body.Messages)
		}
		if body.ChatTemplateKwargs != nil {
			t.Fatalf("custom profile sent S1 options: %#v", body.ChatTemplateKwargs)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Clean words."}}]}`))
	}))
	defer server.Close()

	cfg := config.Default().PostProcessing
	cfg.BaseURL = server.URL + "/v1"
	cfg.AllowInsecureHTTP = true
	cfg.Model = "ordinary-chat-model"
	cfg.Preset = config.PostProcessingPresetGeneric
	cfg.SystemPrompt = "  My cleanup instruction.  "
	processor := New(inference.New(), nil, nil)
	got, err := processor.Process(context.Background(), cfg, "raw words")
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Clean words." {
		t.Fatalf("processed = %q", got.Text)
	}
}

func TestEmptyOutputFailsForRawFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
	}))
	defer server.Close()

	cfg := config.Default().PostProcessing
	cfg.BaseURL = server.URL + "/v1"
	cfg.AllowInsecureHTTP = true
	cfg.Model = "s1-mini"
	cfg.Preset = config.PostProcessingPresetS1Mini
	processor := New(inference.New(), nil, nil)
	if _, err := processor.Process(context.Background(), cfg, "raw words"); err == nil {
		t.Fatal("empty output was accepted")
	}
}

func TestProcessingLogsLifecycleWithoutRequestContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"private cleaned transcript"}}]}`))
	}))
	defer server.Close()

	var logs bytes.Buffer
	cfg := config.Default().PostProcessing
	cfg.BaseURL = server.URL + "/private/path"
	cfg.AllowInsecureHTTP = true
	cfg.Model = "private-model-id"
	cfg.Preset = config.PostProcessingPresetS1Mini
	processor := New(inference.New(), nil, slog.New(slog.NewTextHandler(&logs, nil)))
	if _, err := processor.ProcessWithCredential(context.Background(), cfg, "private raw transcript", "private-api-key"); err != nil {
		t.Fatal(err)
	}

	output := logs.String()
	for _, marker := range []string{"transcript post-processing started", "transcript post-processing completed", "server=", "duration_ms="} {
		if !strings.Contains(output, marker) {
			t.Fatalf("logs missing %q: %s", marker, output)
		}
	}
	for _, forbidden := range []string{"private raw transcript", "private cleaned transcript", "private-api-key", "private-model-id", "/private/path"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("logs exposed %q: %s", forbidden, output)
		}
	}
}
