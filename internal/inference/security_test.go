package inference_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
)

const securityKey = "sk-reflection-canary-7f26"
const securityPayload = "private-audio-or-text-canary"
const securityBase = "https://inference.invalid/v1"

type securityTransport func(*http.Request) (*http.Response, error)

func (f securityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Replace only the transport: these tests exercise New's real redirect policy,
// not a test-authored CheckRedirect that could hide a production regression.
func securityClient(t *testing.T, status int, headers http.Header, body string, inspect func(*http.Request)) *inference.Client {
	t.Helper()
	client := inference.New()
	client.HTTP.Transport = securityTransport(func(r *http.Request) (*http.Response, error) {
		if inspect != nil {
			inspect(r)
		}
		if r.Body != nil {
			_, err := io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
			if err != nil {
				return nil, err
			}
		}
		return &http.Response{StatusCode: status, Header: headers.Clone(), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client.HTTP.Timeout = 2 * time.Second
	return client
}

func securityCall(ctx context.Context, client *inference.Client, route, key string, delta func(string)) (any, error) {
	switch route {
	case "stt":
		return client.Transcribe(ctx, securityBase, "model", "", key, nil, []byte(securityPayload))
	case "chat":
		return client.ChatCompletion(ctx, securityBase, "model", key, "instruction", securityPayload)
	case "file", "sse", "buffered":
		return client.TranscribeFile(ctx, securityBase, "model", "", key, nil, "audio.wav", int64(len(securityPayload)), strings.NewReader(securityPayload), route != "file", inference.FileTranscriptionCallbacks{Delta: delta})
	case "tts":
		return client.SynthesizeSpeech(ctx, securityBase, key, inference.SpeechRequest{Model: "model", Voice: "voice", Input: securityPayload, Speed: 1})
	case "models", "health":
		health := ""
		if route == "health" {
			health = "health"
		}
		result := client.TestMetadata(ctx, securityBase, health, key, "model", nil)
		if result.ErrorKind != "" {
			return result, &inference.Error{Kind: result.ErrorKind, Status: result.HTTPStatus}
		}
		return result, nil
	default:
		panic("unknown test route")
	}
}

func TestInferenceRejectsRedirectsWithoutSecondRequest(t *testing.T) {
	for _, route := range []string{"stt", "chat", "file", "sse", "tts", "models", "health"} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			for _, target := range []struct{ name, url string }{
				{"downgrade", "http://inference.invalid:8080/redirected"},
				{"same-origin", "https://inference.invalid/redirected"},
				{"cross-origin", "https://other.invalid/redirected"},
			} {
				t.Run(fmt.Sprintf("%s/%d/%s", route, status, target.name), func(t *testing.T) {
					calls := 0
					client := securityClient(t, status, http.Header{"Location": {target.url + "?reflected=" + securityKey}}, securityKey+securityPayload, func(r *http.Request) {
						calls++
						if calls > 1 {
							t.Error("redirect sent a second request")
							return
						}
						if r.Header.Get("Authorization") != "Bearer "+securityKey {
							t.Error("initial request did not carry canary credential")
						}
						if route != "models" && route != "health" {
							body, err := io.ReadAll(r.Body)
							if err != nil || !strings.Contains(string(body), securityPayload) {
								t.Error("initial request did not carry canary payload")
							}
						}
					})
					result, err := securityCall(context.Background(), client, route, securityKey, nil)
					var failure *inference.Error
					if !errors.As(err, &failure) || failure.Kind != "http" || failure.Status != status {
						t.Fatalf("expected original HTTP rejection, got %v", err)
					}
					if calls != 1 {
						t.Fatalf("requests = %d, want 1", calls)
					}
					assertNoCredential(t, result, securityKey)
					assertNoCredential(t, fmt.Sprintf("%#v", err), securityKey)
					assertNoCredential(t, fmt.Sprintf("%#v", err), securityPayload)
				})
			}
		}
	}
}

func assertNoCredential(t *testing.T, value any, key string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), key) {
		t.Fatal("credential escaped inference boundary")
	}
}

func metadataPayload(value string) map[string]any {
	return map[string]any{
		"text": "useful transcript", "id": value, "request_id": value,
		"model": value, "provider": value, "service_tier": value, "system_fingerprint": value,
		"language": value, "languages": []any{value, map[string]any{"code": value}, "en"},
		"usage":   map[string]any{"type": value, "input_tokens": 12},
		"timings": map[string]any{"prompt_ms": 25}, "duration": 1.5,
		"choices": []any{map[string]any{"message": map[string]any{"content": "useful transcript"}, "finish_reason": value}},
	}
}

func responseForRoute(t *testing.T, route string, payload map[string]any) (http.Header, string) {
	t.Helper()
	headers := http.Header{"Content-Type": {"application/json"}}
	if route == "sse" || route == "buffered" {
		payload["type"] = "transcript.text.done"
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	if route == "sse" || route == "buffered" {
		body = "data: {\"type\":\"transcript.text.delta\",\"delta\":\"useful \"}\n\ndata: " + body + "\n\ndata: [DONE]\n\n"
		if route == "sse" {
			headers.Set("Content-Type", "text/event-stream")
		} else {
			encoded, err = json.Marshal(map[string]string{"text": body})
			if err != nil {
				t.Fatal(err)
			}
			body = string(encoded)
		}
	}
	return headers, body
}

func TestInferenceSanitizesSuccessfulMetadataBeforeHistoryDTO(t *testing.T) {
	for _, route := range []string{"stt", "chat", "file", "sse", "buffered"} {
		for _, reflection := range []struct{ name, key, value string }{
			{"literal", securityKey, "prefix-" + securityKey + "-" + securityKey},
			// The old 512-byte truncation would retain a credential prefix.
			{"straddles-bound", securityKey, strings.Repeat("x", 505) + securityKey},
			{"long-key", strings.Repeat("K", 600), strings.Repeat("K", 600)},
			{"short-key", "~", "prefix~suffix"},
		} {
			t.Run(route+"/"+reflection.name, func(t *testing.T) {
				headers, body := responseForRoute(t, route, metadataPayload(reflection.value))
				headers.Set("X-Request-Id", reflection.value)
				client := securityClient(t, 200, headers, body, nil)
				var deltas strings.Builder
				result, err := securityCall(context.Background(), client, route, reflection.key, func(s string) { deltas.WriteString(s) })
				if err != nil {
					t.Fatal(err)
				}
				var text string
				var metadata inference.ResponseMetadata
				switch value := result.(type) {
				case inference.TranscriptionResult:
					text, metadata = value.Text, value.Metadata
				case inference.ChatCompletionResult:
					text, metadata = value.Text, value.Metadata
				}
				if text != "useful transcript" || metadata.Usage.InputTokens == nil || *metadata.Usage.InputTokens != 12 || metadata.Performance.PromptMilliseconds == nil || *metadata.Performance.PromptMilliseconds != 25 || metadata.RequestCount != 1 || metadata.UsageReportCount != 1 {
					t.Fatal("sanitization lost valid text or benign metrics")
				}
				if metadata.RequestID != "" || metadata.ResponseID != "" || metadata.EffectiveModel != "" || metadata.Provider != "" || metadata.ServiceTier != "" || metadata.SystemFingerprint != "" || metadata.FinishReason != "" || metadata.Usage.Type != "" {
					t.Fatal("reflected metadata was not removed in full")
				}
				if route != "chat" && !reflect.DeepEqual(metadata.DetectedLanguages, []string{"en"}) {
					t.Fatal("language sanitization lost benign entry or retained reflected entry")
				}
				if (route == "sse" || route == "buffered") && deltas.String() != "useful " {
					t.Fatal("metadata sanitization changed progressive text")
				}
				assertNoCredential(t, result, reflection.key)
				store := history.NewStore(true, nil)
				defer store.Close()
				details := history.HistoryRunDetails{Transcription: history.NewResponseDetails(metadata)}
				if route == "chat" {
					details.Transcription = nil
					details.Processing.Response = history.NewResponseDetails(metadata)
				}
				if store.Begin(text, history.HistoryTranscribed, false, time.Unix(1, 0), details) == 0 {
					t.Fatal("history did not retain valid result")
				}
				dto := history.NewService(store).TranscriptHistory()
				if len(dto) != 1 || dto[0].RawText != text {
					t.Fatal("history renderer DTO lost useful transcript")
				}
				assertNoCredential(t, dto, reflection.key)
			})
		}
	}
}

func TestInferenceMetadataHeadersAndBenignControls(t *testing.T) {
	for _, route := range []string{"stt", "chat", "file", "sse", "buffered"} {
		for _, header := range []string{"X-Request-Id", "OpenAI-Request-Id", "Request-Id"} {
			for _, key := range []string{securityKey, ""} {
				t.Run(route+"/"+header+"/authenticated="+fmt.Sprint(key != ""), func(t *testing.T) {
					payload := metadataPayload("benign")
					delete(payload, "request_id")
					headers, body := responseForRoute(t, route, payload)
					headers.Set(header, securityKey)
					result, err := securityCall(context.Background(), securityClient(t, 200, headers, body, nil), route, key, nil)
					if err != nil {
						t.Fatal(err)
					}
					var metadata inference.ResponseMetadata
					switch value := result.(type) {
					case inference.TranscriptionResult:
						metadata = value.Metadata
					case inference.ChatCompletionResult:
						metadata = value.Metadata
					}
					wantID := securityKey
					if key != "" {
						wantID = ""
						assertNoCredential(t, result, key)
					}
					if metadata.RequestID != wantID || metadata.Provider != "benign" || metadata.ResponseID != "benign" || metadata.EffectiveModel != "benign" || metadata.ServiceTier != "benign" || metadata.SystemFingerprint != "benign" || metadata.Usage.Type != "benign" {
						t.Fatal("header policy changed benign metadata")
					}
				})
			}
		}
	}
}

func TestInferenceModelDiscoveryDropsCredentialIDs(t *testing.T) {
	for _, key := range []string{securityKey, ""} {
		t.Run(fmt.Sprint(key != ""), func(t *testing.T) {
			body := `{"data":[{"id":"model"},{"id":"prefix-` + securityKey + `"},{"id":"model"},{"id":"other"}]}`
			client := securityClient(t, 200, http.Header{"X-Request-Id": {securityKey}}, body, nil)
			result := client.TestMetadata(context.Background(), securityBase, "", key, "model", nil)
			want := []string{"model", "other"}
			if key == "" {
				want = []string{"model", "prefix-" + securityKey, "other"}
			} else {
				assertNoCredential(t, result, key)
			}
			if result.ErrorKind != "" || !result.Reachable || result.ModelPresence != "listed" || !reflect.DeepEqual(result.ModelIDs, want) {
				t.Fatal("model discovery did not preserve safe inventory")
			}
		})
	}
}

func TestInferenceStillRejectsCredentialInText(t *testing.T) {
	for _, route := range []string{"stt", "chat", "file", "sse", "buffered"} {
		t.Run(route, func(t *testing.T) {
			payload := metadataPayload("benign")
			payload["text"] = securityKey
			payload["choices"] = []any{map[string]any{"message": map[string]any{"content": securityKey}}}
			headers, body := responseForRoute(t, route, payload)
			result, err := securityCall(context.Background(), securityClient(t, 200, headers, body, nil), route, securityKey, nil)
			var failure *inference.Error
			if !errors.As(err, &failure) || failure.Kind != "credential_reflection" {
				t.Fatalf("expected credential rejection, got %v", err)
			}
			assertNoCredential(t, result, securityKey)
			assertNoCredential(t, fmt.Sprintf("%#v", err), securityKey)
		})
	}
}
