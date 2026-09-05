package inference

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatCompletionRejectsReportedLengthLimit(t *testing.T) {
	for _, key := range []string{"", "secret", "length"} {
		t.Run("key-"+key, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				_, _ = io.Copy(io.Discard, r.Body)
				_, _ = io.WriteString(w, `{"id":"limited","choices":[{"message":{"content":"private incomplete cleanup"},"finish_reason":"length"}],"usage":{"completion_tokens":8}}`)
			}))
			defer server.Close()
			result, err := New().WithCleanupOptions(compatibility.CleanupOptions{LimitOutputTokens: true, MaxOutputTokens: 8}).ChatCompletion(context.Background(), server.URL+"/v1", "generic-cleanup", key, "instruction", "raw transcript")
			var failure *Error
			if !errors.As(err, &failure) || failure.Kind != "incomplete_response" || result.Text != "" || calls != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
			}
			if strings.Contains(err.Error(), "private incomplete cleanup") {
				t.Fatal("partial cleanup leaked in error")
			}
			if result.Metadata.ResponseID != "limited" || result.Metadata.RequestCount != 1 || result.Metadata.Usage.OutputTokens == nil || *result.Metadata.Usage.OutputTokens != 8 {
				t.Fatalf("safe diagnostic metadata lost: %+v", result.Metadata)
			}
			wantReason := "length"
			if key == "length" {
				wantReason = ""
			}
			if result.Metadata.FinishReason != wantReason {
				t.Fatal("finish reason redaction changed")
			}
		})
	}
}

func TestChatCompletionDoesNotInferTruncationFromMissingOrOtherFinishReasons(t *testing.T) {
	for _, reason := range []any{nil, "stop", "provider-specific", ""} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "Complete cleanup."}, "finish_reason": reason}}})
		}))
		result, err := New().WithCleanupOptions(compatibility.CleanupOptions{LimitOutputTokens: true, MaxOutputTokens: 8}).ChatCompletion(context.Background(), server.URL, "generic-cleanup", "", "instruction", "raw")
		server.Close()
		if err != nil || result.Text != "Complete cleanup." {
			t.Fatalf("finish reason %v: result=%+v error=%v", reason, result, err)
		}
	}
}
