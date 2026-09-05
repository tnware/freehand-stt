package inference

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestNativeAndVLLMTranscriptionRequests(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.WhisperCPP, compatibility.VLLM} {
		for _, file := range []bool{false, true} {
			t.Run(string(id)+map[bool]string{false: "/microphone", true: "/file"}[file], func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					wantPath := "/proxy/v1/audio/transcriptions"
					if id == compatibility.WhisperCPP {
						wantPath = "/proxy/inference"
					}
					if r.URL.Path != wantPath || r.Method != "POST" {
						t.Errorf("request %s %s", r.Method, r.URL.Path)
					}
					if r.Header.Get("Authorization") != "Bearer private-key" || r.Header.Get("X-Custom") != "yes" {
						t.Error("request headers lost")
					}
					if err := r.ParseMultipartForm(1 << 20); err != nil {
						t.Error(err)
						return
					}
					defer r.MultipartForm.RemoveAll()
					_, hasModel := r.MultipartForm.Value["model"]
					if hasModel != (id != compatibility.WhisperCPP) {
						t.Error("incorrect model selection")
					}
					if r.FormValue("response_format") != "json" || r.FormValue("prompt") != "Acme 日本語" || r.FormValue("temperature") != "0" || r.FormValue("language") != "en" {
						t.Error("wrong form fields")
					}
					if r.FormValue("stream") != "" || r.FormValue("hotwords") != "" {
						t.Error("unqualified form fields")
					}
					f, _, err := r.FormFile("file")
					if err != nil {
						t.Error(err)
						return
					}
					defer f.Close()
					body, _ := io.ReadAll(f)
					if string(body) != "bounded audio fixture" {
						t.Error("upload truncated")
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"text":"Hello 日本語"}`)
				}))
				defer server.Close()
				client := New().WithCompatibility(id).WithTranscriptionOptions(compatibility.TranscriptionOptions{Prompt: "Acme 日本語", TemperatureOverride: true})
				base := server.URL + "/proxy/v1"
				if id == compatibility.WhisperCPP {
					base = server.URL + "/proxy"
				}
				var result TranscriptionResult
				var err error
				audio := "bounded audio fixture"
				if file {
					result, err = client.TranscribeFile(context.Background(), base, "retained-model", "en", "private-key", map[string]string{"X-Custom": "yes"}, "test.wav", int64(len(audio)), strings.NewReader(audio), false, FileTranscriptionCallbacks{})
				} else {
					result, err = client.Transcribe(context.Background(), base, "retained-model", "en", "private-key", map[string]string{"X-Custom": "yes"}, []byte(audio))
				}
				if err != nil || result.Text != "Hello 日本語" || calls != 1 {
					t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
				}
			})
		}
	}
}

func vllmEvent(content, finish string) string {
	choice := map[string]any{"delta": map[string]any{"content": content}}
	if finish != "" {
		choice["finish_reason"] = finish
	}
	body, _ := json.Marshal(map[string]any{"object": "transcription.chunk", "id": "trsc-test", "model": "speech-model", "choices": []any{choice}})
	return "data: " + string(body) + "\n\n"
}

func TestVLLMStreamCompletionAndFailure(t *testing.T) {
	first := vllmEvent("Hello", "stop")
	second := vllmEvent(" 日本語", "")
	last := vllmEvent(".", "stop")
	usage := "data: {\"object\":\"transcription.chunk\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":4,\"total_tokens\":14}}\n\n"
	done := "data: [DONE]\n\n"
	for _, tc := range []struct {
		name, body, want string
		fail             bool
	}{
		{"multiple audio chunks", first + second + last + done, "Hello 日本語.", false},
		{"final usage", first + usage + done, "Hello", false},
		{"missing done", first, "Hello", true},
		{"done before last finish", first + second + done, "Hello 日本語", true},
		{"done alone", done, "", true},
		{"truncated chunk", vllmEvent("Partial", "length") + done, "Partial", true},
		{"aborted chunk", vllmEvent("Partial", "abort") + done, "Partial", true},
		{"provider error", first + "data: {\"error\":{\"message\":\"private-key\"}}\n\n" + done, "Hello", true},
		{"malformed", first + "data: broken\n\n", "Hello", true},
		{"wrong dialect", "data: {\"type\":\"transcript.text.done\",\"text\":\"wrong\"}\n\n" + done, "", true},
		{"after done", first + done + second, "Hello", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var delta strings.Builder
			result, err := readVLLMTranscriptionSSE(strings.NewReader(tc.body), "private-key", func(s string) { delta.WriteString(s) })
			if (err != nil) != tc.fail || result.Text != tc.want {
				t.Fatalf("text=%q err=%v", result.Text, err)
			}
			if err != nil && strings.Contains(err.Error(), "private-key") {
				t.Fatal("reflected credential")
			}
		})
	}
}

func TestVLLMStreamCredentialAcrossChunks(t *testing.T) {
	var published strings.Builder
	result, err := readVLLMTranscriptionSSE(strings.NewReader(vllmEvent("safe pri", "")+vllmEvent("vate-key", "stop")+"data: [DONE]\n\n"), "private-key", func(s string) { published.WriteString(s) })
	if err == nil || result.Text != "" || strings.Contains(published.String(), "private-key") {
		t.Fatalf("credential reflection: %+v %v", result, err)
	}
}

func TestVLLMStreamThroughFileTransport(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
				return
			}
			defer r.MultipartForm.RemoveAll()
			if r.FormValue("stream") != "true" {
				t.Error("missing streaming request")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, vllmEvent("Accepted", ""))
			w.(http.Flusher).Flush()
			if cancelled {
				<-r.Context().Done()
				return
			}
			io.WriteString(w, vllmEvent(".", "stop")+"data: [DONE]\n\n")
		}))
		result, err := New().WithCompatibility(compatibility.VLLM).TranscribeFile(ctx, server.URL+"/v1", "speech", "", "", nil, "fixture.wav", 3, strings.NewReader("wav"), true, FileTranscriptionCallbacks{Delta: func(string) {
			if cancelled {
				cancel()
			}
		}})
		server.Close()
		if calls != 1 || (err != nil) != cancelled {
			t.Fatalf("calls=%d result=%+v err=%v", calls, result, err)
		}
		if !cancelled && result.Text != "Accepted." {
			t.Fatalf("text=%q", result.Text)
		}
	}
}
