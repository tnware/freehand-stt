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

func TestProfileRequestsPreserveExistingWireContracts(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.Generic, compatibility.Speaches} {
		t.Run(string(id), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer captured-key" {
					t.Error("method/auth changed")
				}
				switch r.URL.Path {
				case "/nested/v1/audio/transcriptions":
					if err := r.ParseMultipartForm(1 << 20); err != nil {
						t.Error(err)
						return
					}
					defer r.MultipartForm.RemoveAll()
					if r.FormValue("model") != "chosen-model" || r.FormValue("language") != "en" || r.FormValue("response_format") != "json" {
						t.Error("transcription fields changed")
					}
					if len(r.MultipartForm.Value) != 3 {
						t.Errorf("unexpected fields: %v", r.MultipartForm.Value)
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"text":"hello"}`)
				case "/nested/v1/audio/speech":
					var request map[string]any
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					if len(request) != 5 || request["model"] != "chosen-model" || request["voice"] != "voice-id" || request["response_format"] != "wav" || request["speed"] != 1.0 || request["input"] != "hello" {
						t.Errorf("speech fields changed: %v", request)
					}
					w.Header().Set("Content-Type", "audio/wav")
					io.WriteString(w, "fixture-audio")
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			client := New().WithCompatibility(id)
			result, err := client.Transcribe(context.Background(), server.URL+"/nested/v1", "chosen-model", "en", "captured-key", nil, []byte("fixture"))
			if err != nil || result.Text != "hello" {
				t.Fatalf("microphone: %#v %v", result, err)
			}
			result, err = client.TranscribeFile(context.Background(), server.URL+"/nested/v1", "chosen-model", "en", "captured-key", nil, "fixture.wav", 7, strings.NewReader("fixture"), false, FileTranscriptionCallbacks{})
			if err != nil || result.Text != "hello" {
				t.Fatalf("file: %#v %v", result, err)
			}
			audio, err := client.SynthesizeSpeech(context.Background(), server.URL+"/nested/v1", "captured-key", SpeechRequest{CompatibilityProfile: id, Model: "chosen-model", Voice: "voice-id", Input: "hello", Speed: 1})
			if err != nil || string(audio) != "fixture-audio" || calls != 3 {
				t.Fatalf("speech/call count: %q %v %d", audio, err, calls)
			}
		})
	}
}

func TestProfileStreamingCompletionContracts(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.Generic, compatibility.Speaches} {
		for _, tc := range []struct {
			name, body, text string
			fail             bool
		}{
			{"typed", "data: {\"type\":\"transcript.text.delta\",\"delta\":\"draft\"}\n\ndata: {\"type\":\"transcript.text.done\",\"text\":\"final\"}\n\n", "final", false},
			{"legacy", "data: {\"text\":\"hello\"}\n\ndata: {\"text\":\"world\"}\n\n", "hello world", false},
			{"incomplete", "data: {\"type\":\"transcript.text.delta\",\"delta\":\"partial\"}\n\ndata: [DONE]\n\n", "partial", true},
			{"empty-final", "data: {\"type\":\"transcript.text.delta\",\"delta\":\"draft\"}\n\ndata: {\"type\":\"transcript.text.done\",\"text\":\"\"}\n\n", "", false},
			{"vllm", "data: {\"choices\":[{\"delta\":{\"content\":\"unhandled\"}}]}\n\ndata: [DONE]\n\n", "", true},
		} {
			t.Run(string(id)+"/"+tc.name, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					io.Copy(io.Discard, r.Body)
					w.Header().Set("Content-Type", "text/event-stream")
					io.WriteString(w, tc.body)
				}))
				defer server.Close()
				result, err := New().WithCompatibility(id).TranscribeFile(context.Background(), server.URL, "m", "", "", nil, "a.wav", 1, strings.NewReader("a"), true, FileTranscriptionCallbacks{})
				if (err != nil) != tc.fail || result.Text != tc.text || calls != 1 {
					t.Fatalf("result=%#v err=%v calls=%d", result, err, calls)
				}
			})
		}
	}
}

func TestLlamaCPPUsesSharedChatAndRejectsTruncation(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.Generic, compatibility.LlamaCPP} {
		for _, finish := range []string{"stop", "length"} {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/v1/chat/completions" {
					t.Error("wrong chat path")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if len(body) != 4 || body["model"] != "s1-mini" || body["stream"] != false || body["temperature"] != 0.0 {
					t.Errorf("unexpected chat contract: %v", body)
				}
				messages := body["messages"].([]any)
				if len(messages) != 2 || messages[0].(map[string]any)["role"] != "system" || messages[1].(map[string]any)["content"] != "raw text" {
					t.Error("prompt roles/content changed")
				}
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "cleaned"}, "finish_reason": finish}}})
			}))
			result, err := New().WithCompatibility(id).ChatCompletion(context.Background(), server.URL+"/v1", "s1-mini", "", "instruction", "raw text")
			server.Close()
			if (err != nil) != (finish == "length") || calls != 1 {
				t.Fatalf("%s/%s err=%v calls=%d", id, finish, err, calls)
			}
			if finish == "length" && result.Text != "" {
				t.Fatal("truncated cleanup escaped")
			}
		}
	}
}

type profileNoNetwork struct{ calls int }

func (p *profileNoNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	p.calls++
	return nil, errors.New("unexpected network request")
}

func TestUnavailableProfilesFailBeforeNetworkOrUpload(t *testing.T) {
	transport := &profileNoNetwork{}
	client := &Client{HTTP: &http.Client{Transport: transport}}
	for _, id := range []compatibility.ID{compatibility.LocalAI, compatibility.ID("unknown")} {
		selected := client.WithCompatibility(id)
		_, micErr := selected.Transcribe(context.Background(), "https://example.invalid", "m", "", "", nil, nil)
		_, fileErr := selected.TranscribeFile(context.Background(), "https://example.invalid", "m", "", "", nil, "a.wav", 1, strings.NewReader("a"), true, FileTranscriptionCallbacks{})
		_, chatErr := selected.ChatCompletion(context.Background(), "https://example.invalid", "m", "", "s", "u")
		_, speechErr := client.SynthesizeSpeech(context.Background(), "https://example.invalid", "", SpeechRequest{CompatibilityProfile: id})
		for _, err := range []error{micErr, fileErr, chatErr, speechErr} {
			var failure *Error
			if !errors.As(err, &failure) || failure.Kind != "invalid_settings" {
				t.Fatalf("unexpected validation: %v", err)
			}
		}
	}
	if transport.calls != 0 || client.profile != "" {
		t.Fatal("selection mutated shared client or contacted server")
	}
}
