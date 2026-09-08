package inference

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestCompletedFamilyRequestFixtures(t *testing.T) {
	for _, fixture := range []struct {
		id                     modelprofile.ID
		backend                compatibility.ID
		language, wireLanguage string
	}{
		{modelprofile.ParakeetTDT, compatibility.NeMoSpeechV1, "auto", ""},
		{modelprofile.CohereTranscribe, compatibility.VLLM, "ja", "ja"},
		{modelprofile.CohereTranscribe, compatibility.VLLM, "auto", ""},
		{modelprofile.VoxtralRealtime, compatibility.VLLM, "auto", ""},
	} {
		t.Run(string(fixture.id)+fixture.language, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/nested/v1/audio/transcriptions" || r.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Error("wrong endpoint or credential")
				}
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				if r.FormValue("language") != fixture.wireLanguage || r.FormValue("prompt") != "" || r.FormValue("hotwords") != "" || r.FormValue("temperature") != "" {
					t.Errorf("wrong fields: %v", r.MultipartForm.Value)
				}
				if fixture.backend == compatibility.NeMoSpeechV1 {
					if r.FormValue("model") != "" || r.FormValue("automatic_punctuation") != "true" || r.FormValue("verbatim") != "true" {
						t.Error("wrong server-loaded contract")
					}
				} else if r.FormValue("model") != "selected-alias" {
					t.Error("model selection lost")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"text":"Hello 世界"}`))
			}))
			defer server.Close()
			c := New().WithCompatibility(fixture.backend).WithModelProfile(fixture.id)
			mic, err := c.Transcribe(t.Context(), server.URL+"/nested/v1", "selected-alias", fixture.language, "fixture-key", nil, []byte("fixture audio"))
			if err != nil || mic.Text != "Hello 世界" {
				t.Fatalf("mic: %#v %v", mic, err)
			}
			file, err := c.TranscribeFile(t.Context(), server.URL+"/nested/v1", "selected-alias", fixture.language, "fixture-key", nil, "fixture.wav", 13, bytes.NewReader([]byte("fixture audio")), false, FileTranscriptionCallbacks{})
			if err != nil || file.Text != "Hello 世界" || calls != 2 {
				t.Fatalf("file: %#v %v", file, err)
			}
		})
	}
}

func TestQwenTTSSpeechAndVoiceFixtures(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Error("credential missing")
		}
		if r.URL.Path == "/v1/audio/voices" {
			_, _ = w.Write([]byte(`{"voices":["ryan","vivian"],"uploaded_voices":[]}`))
			return
		}
		if r.URL.Path != "/v1/audio/speech" {
			t.Error("wrong speech route")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		want := map[string]any{"model": "selected-alias", "input": "Hello", "voice": "ryan", "speed": 1.25, "response_format": "wav", "stream": false, "language": "Japanese", "instructions": "Warm delivery", "task_type": "CustomVoice"}
		if len(body) != len(want) {
			t.Errorf("unexpected request fields: %v", body)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("%s: got %v want %v", key, body[key], value)
			}
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("fixture WAV"))
	}))
	defer server.Close()
	c := New()
	v := c.ListVoices(t.Context(), compatibility.VLLMOmni, server.URL+"/v1", "fixture-key", "selected-alias")
	if v.ErrorKind != "" || len(v.Voices) != 2 || v.Voices[0].ID != "ryan" {
		t.Fatalf("voices: %#v", v)
	}
	req := SpeechRequest{CompatibilityProfile: compatibility.VLLMOmni, ModelProfile: modelprofile.Qwen3TTS, Model: "selected-alias", Voice: "ryan", Input: "Hello", Speed: 1.25, Options: modelprofile.SpeechOptions{Language: "ja", Instructions: "Warm delivery"}}
	if _, err := c.SynthesizeSpeech(t.Context(), server.URL+"/v1", "fixture-key", req); err != nil {
		t.Fatal(err)
	}
	req.ModelProfile = modelprofile.Generic
	if _, err := c.SynthesizeSpeech(t.Context(), server.URL+"/v1", "fixture-key", req); err == nil || calls != 2 {
		t.Fatal("unsupported options reached network")
	}
}
