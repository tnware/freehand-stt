package inference

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestMagpieSpeechRequestUsesQualifiedBufferedWAV(t *testing.T) {
	calls := 0
	wantLanguage := "fr-FR"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/proxy/v1/audio/speech" || r.Header.Get("Authorization") != "Bearer selected-key" {
			t.Errorf("incorrect request route or credential")
		}
		body, _ := io.ReadAll(r.Body)
		var request map[string]any
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatal(err)
		}
		wantFields := 6
		if wantLanguage == "" {
			wantFields = 5
		}
		if (wantLanguage == "" && request["language"] != nil) || (wantLanguage != "" && request["language"] != wantLanguage) {
			t.Fatalf("incorrect language projection: %v", request)
		}
		if len(request) != wantFields || request["model"] != "magpietts" || request["voice"] != "John" || request["speed"] != float64(1) || request["response_format"] != "wav" || request["input"] != "Bonjour." {
			t.Fatalf("request=%v", request)
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.Write([]byte("RIFFfixture"))
	}))
	defer server.Close()
	request := SpeechRequest{CompatibilityProfile: compatibility.NeMoSpeechV1, ModelProfile: modelprofile.MagpieTTS, Model: "magpietts", Voice: "John", Input: "Bonjour.", Speed: 1, Options: modelprofile.SpeechOptions{Language: "fr-FR"}}
	audio, err := New().SynthesizeSpeech(t.Context(), server.URL+"/proxy/v1", "selected-key", request)
	if err != nil || string(audio) != "RIFFfixture" {
		t.Fatal(err)
	}
	wantLanguage = ""
	request.Options.Language = ""
	if _, err := New().SynthesizeSpeech(t.Context(), server.URL+"/proxy/v1", "selected-key", request); err != nil {
		t.Fatal(err)
	}
	request.Speed = 1.25
	if _, err := New().SynthesizeSpeech(t.Context(), server.URL, "", request); err == nil {
		t.Fatal("unsupported speed admitted")
	}
	request.Speed = 1
	request.Options.Instructions = "Whisper"
	if _, err := New().SynthesizeSpeech(t.Context(), server.URL, "", request); err == nil {
		t.Fatal("unsupported style admitted")
	}
	request.Options = modelprofile.SpeechOptions{Language: "auto"}
	if _, err := New().SynthesizeSpeech(t.Context(), server.URL, "", request); err == nil {
		t.Fatal("unsupported autodetection admitted")
	}
	if calls != 2 {
		t.Fatal("invalid settings reached server", calls)
	}
}

func TestNeMoSpeechVoiceDiscoveryUsesSelectedModelMetadataOnly(t *testing.T) {
	for _, tc := range []struct {
		name, model, body, errorKind string
		voices, languages            []string
	}{
		{"matching speech", "magpietts", `{"data":[{"id":"asr","capability":"transcription","voices":["wrong"]},{"id":"magpietts","capability":"speech","voices":["John","Sofia","John","selected-key","bad\nvoice"],"languages":["en-US","fr-FR","fr-FR","selected-key","ar","unknown"]}]}`, "", []string{"John", "Sofia"}, []string{"en-US", "fr-FR"}},
		{"empty voices", "magpietts", `{"data":[{"id":"magpietts","capability":"speech","voices":[],"languages":["en-US"]}]}`, "", nil, []string{"en-US"}},
		{"no selected model", "missing", `{"data":[{"id":"magpietts","capability":"speech","voices":["John"]}]}`, "model_not_listed", nil, nil},
		{"wrong capability", "chosen", `{"data":[{"id":"chosen","capability":"transcription","voices":["wrong"]}]}`, "model_not_listed", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/proxy/v1/models" || r.Header.Get("Authorization") != "Bearer selected-key" {
					t.Errorf("unexpected discovery request: %s %s", r.Method, r.URL.Path)
				}
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			result := New().ListVoices(t.Context(), compatibility.NeMoSpeechV1, server.URL+"/proxy/v1", "selected-key", tc.model)
			got := []string{}
			for _, v := range result.Voices {
				got = append(got, v.ID)
			}
			if calls != 1 || result.ErrorKind != tc.errorKind || !slices.Equal(got, tc.voices) || !slices.Equal(result.Languages, tc.languages) {
				t.Fatalf("result=%+v calls=%d", result, calls)
			}
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "selected-key") {
				t.Fatal("credential reflected")
			}
		})
	}
}
