package inference

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSynthesizeSpeechUsesOpenAICompatibleWAVRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("request = %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body struct {
			Model          string  `json:"model"`
			Input          string  `json:"input"`
			Voice          string  `json:"voice"`
			ResponseFormat string  `json:"response_format"`
			Speed          float64 `json:"speed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "tts-model" || body.Input != "hello" || body.Voice != "voice-id" || body.ResponseFormat != "wav" || body.Speed != 1.25 {
			t.Fatalf("body = %#v", body)
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("RIFFaudio"))
	}))
	defer server.Close()

	client := New()
	result, err := client.SynthesizeSpeech(context.Background(), server.URL+"/v1", "secret", SpeechRequest{Model: "tts-model", Voice: "voice-id", Input: "hello", Speed: 1.25})
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "RIFFaudio" {
		t.Fatalf("result = %q", result)
	}
}

func TestSynthesizeSpeechDoesNotReflectProviderErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("sensitive provider detail"))
	}))
	defer server.Close()

	_, err := New().SynthesizeSpeech(context.Background(), server.URL, "", SpeechRequest{})
	if err == nil || err.Error() != "http (400): speech generation request failed" {
		t.Fatalf("error = %v", err)
	}
}
