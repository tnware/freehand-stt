package inference

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestTranscriptionControlsAcrossRequestModes(t *testing.T) {
	for _, mode := range []string{"microphone", "file", "stream"} {
		for _, variant := range []string{"defaults", "generic-zero", "speaches"} {
			t.Run(mode+"/"+variant, func(t *testing.T) {
				id := compatibility.Generic
				options := compatibility.TranscriptionOptions{Temperature: 0.7}
				want := map[string]string{"model": "selected", "language": "en", "response_format": "json"}
				if variant != "defaults" {
					options.Prompt = "Project context: 日本語\nFreehand"
					options.TemperatureOverride = true
					options.Temperature = 0
					want["prompt"], want["temperature"] = options.Prompt, "0"
				}
				if variant == "speaches" {
					id = compatibility.Speaches
					options.Hotwords, options.Temperature = "Speaches, 日本語", 0.3
					want["hotwords"], want["temperature"] = options.Hotwords, "0.3"
				}
				if mode == "stream" {
					want["stream"] = "true"
				}
				audio := "RIFF fixture audio"
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					raw, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					if r.ContentLength != int64(len(raw)) {
						t.Errorf("content length %d != bytes %d", r.ContentLength, len(raw))
					}
					r.Body = io.NopCloser(bytes.NewReader(raw))
					if err := r.ParseMultipartForm(1 << 20); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					defer r.MultipartForm.RemoveAll()
					if len(r.MultipartForm.Value) != len(want) {
						t.Errorf("unexpected fields: %v", r.MultipartForm.Value)
					}
					for field, expected := range want {
						values := r.MultipartForm.Value[field]
						if len(values) != 1 || values[0] != expected {
							t.Errorf("field %s = %v", field, values)
						}
					}
					file, _, err := r.FormFile("file")
					if err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					defer file.Close()
					data, _ := io.ReadAll(file)
					if string(data) != audio {
						t.Error("audio changed")
					}
					if mode == "stream" {
						w.Header().Set("Content-Type", "text/event-stream")
						io.WriteString(w, "data: {\"type\":\"transcript.text.delta\",\"delta\":\"result\"}\n\ndata: {\"type\":\"transcript.text.done\",\"text\":\"result\"}\n\n")
					} else {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, `{"text":"result"}`)
					}
				}))
				defer server.Close()
				base := &Client{HTTP: server.Client()}
				client := base.WithTranscriptionOptions(options).WithCompatibility(id)
				// Both builders copy by value, and editing the draft cannot mutate a job.
				options.Prompt, options.Hotwords, options.Temperature = "new draft", "new terms", 1
				if base.transcriptionOptions != (compatibility.TranscriptionOptions{}) {
					t.Fatal("shared client changed")
				}
				var result TranscriptionResult
				var err error
				if mode == "microphone" {
					result, err = client.Transcribe(context.Background(), server.URL, "selected", "en", "", nil, []byte(audio))
				} else {
					result, err = client.TranscribeFile(context.Background(), server.URL, "selected", "en", "", nil, "sample.wav", int64(len(audio)), strings.NewReader(audio), mode == "stream", FileTranscriptionCallbacks{})
				}
				if err != nil || result.Text != "result" || calls != 1 {
					t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
				}
			})
		}
	}
}

type forbiddenAudioReader struct{ t *testing.T }

func (r forbiddenAudioReader) Read([]byte) (int, error) {
	r.t.Error("invalid controls consumed audio")
	return 0, io.EOF
}

func TestInvalidTranscriptionControlsNeverSendOrReadAudio(t *testing.T) {
	for _, options := range []compatibility.TranscriptionOptions{
		{Hotwords: "private-hotwords"}, {Prompt: strings.Repeat("x", 8193)},
		{TemperatureOverride: true, Temperature: -1}, {TemperatureOverride: true, Temperature: math.NaN()},
	} {
		client := (&Client{HTTP: &http.Client{Transport: optionsTransport(func(*http.Request) (*http.Response, error) {
			t.Error("invalid controls made a request")
			return nil, io.EOF
		})}}).WithTranscriptionOptions(options)
		_, err := client.Transcribe(context.Background(), "http://unused", "model", "", "", nil, nil)
		if err == nil || strings.Contains(err.Error(), "private-hotwords") {
			t.Fatalf("invalid controls error: %v", err)
		}
		_, err = client.TranscribeFile(context.Background(), "http://unused", "model", "", "", nil, "audio.wav", 10, forbiddenAudioReader{t}, true, FileTranscriptionCallbacks{})
		if err == nil {
			t.Fatal("file accepted invalid controls")
		}
	}
}

type optionsTransport func(*http.Request) (*http.Response, error)

func (f optionsTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
