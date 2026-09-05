package inference

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestCleanupOptionsRequestContract(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.Generic, compatibility.LlamaCPP, compatibility.VLLM} {
		for _, variant := range []string{"defaults", "retained-disabled", "limit", "reasoning", "both"} {
			if id == compatibility.Generic && (variant == "reasoning" || variant == "both") {
				continue
			}
			t.Run(string(id)+"/"+variant, func(t *testing.T) {
				options := compatibility.CleanupOptions{}
				if variant == "retained-disabled" {
					options.MaxOutputTokens = 1234
				}
				if variant == "limit" || variant == "both" {
					options.LimitOutputTokens = true
					options.MaxOutputTokens = 4096
				}
				if variant == "reasoning" || variant == "both" {
					options.DisableReasoning = true
				}
				captured := options
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					wantFields := 4
					if captured.LimitOutputTokens {
						wantFields++
					}
					if captured.DisableReasoning {
						wantFields++
					}
					if len(body) != wantFields || string(body["model"]) != `"chosen"` || string(body["temperature"]) != "0" || string(body["stream"]) != "false" {
						t.Errorf("unexpected chat fields: %v", body)
					}
					if string(body["messages"]) != `[{"role":"system","content":"fixed instruction"},{"role":"user","content":"raw 日本語"}]` {
						t.Error("prompt contract changed")
					}
					if captured.LimitOutputTokens {
						if string(body["max_tokens"]) != "4096" {
							t.Error("missing output-token limit")
						}
					} else if _, found := body["max_tokens"]; found {
						t.Error("disabled output limit sent")
					}
					if captured.DisableReasoning {
						if string(body["reasoning_effort"]) != `"none"` {
							t.Error("missing reasoning override")
						}
					} else if _, found := body["reasoning_effort"]; found {
						t.Error("unsolicited reasoning override")
					}
					io.WriteString(w, `{"choices":[{"message":{"content":"cleaned"},"finish_reason":"stop"}]}`)
				}))
				defer server.Close()
				base := &Client{HTTP: server.Client()}
				client := base.WithCleanupOptions(options).WithCompatibility(id)
				options.MaxOutputTokens = 8
				options.DisableReasoning = !options.DisableReasoning
				if base.cleanupOptions != (compatibility.CleanupOptions{}) {
					t.Fatal("mutated shared client")
				}
				result, err := client.ChatCompletion(context.Background(), server.URL, "chosen", "", "fixed instruction", "raw 日本語")
				if err != nil || result.Text != "cleaned" || calls != 1 {
					t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
				}
			})
		}
	}
}

func TestInvalidCleanupOptionsNeverMakeRequest(t *testing.T) {
	for _, options := range []compatibility.CleanupOptions{
		{DisableReasoning: true}, {LimitOutputTokens: true}, {MaxOutputTokens: -1}, {MaxOutputTokens: 65537},
	} {
		client := (&Client{HTTP: &http.Client{Transport: optionsTransport(func(*http.Request) (*http.Response, error) {
			t.Error("invalid options made a request")
			return nil, io.EOF
		})}}).WithCleanupOptions(options)
		_, err := client.ChatCompletion(context.Background(), "http://unused", "chosen", "", "instruction", "raw")
		var failure *Error
		if !errors.As(err, &failure) || failure.Kind != "invalid_settings" {
			t.Fatalf("error = %v", err)
		}
	}
}

func TestCleanupOptionRejectionIsNotRetried(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(400)
		io.WriteString(w, `{"error":"unsupported reasoning_effort"}`)
	}))
	defer server.Close()
	client := New().WithCompatibility(compatibility.LlamaCPP).WithCleanupOptions(compatibility.CleanupOptions{DisableReasoning: true})
	result, err := client.ChatCompletion(context.Background(), server.URL, "chosen", "", "instruction", "raw")
	if err == nil || calls != 1 || result.Text != "" {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
	}
}
