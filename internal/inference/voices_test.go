package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestVoiceDiscoveryProviderShapesAndScope(t *testing.T) {
	for _, tc := range []struct {
		name           string
		backend        compatibility.ID
		models, voices string
		scope          VoiceScope
		calls          int
	}{
		{"kokoro objects", compatibility.KokoroFastAPI, "", `{"voices":[{"id":"af_heart","name":"Heart"}]}`, VoiceScopeServer, 1},
		{"kokoro legacy", compatibility.KokoroFastAPI, "", `{"voices":["af_heart"]}`, VoiceScopeServer, 1},
		{"speaches model", compatibility.Speaches, `{"data":[{"id":"other","voices":[{"id":"wrong"}]},{"id":"chosen","voices":[{"id":"af_heart","name":"Heart","language":"en-us"}]}]}`, "", VoiceScopeModel, 1},
		{"speaches fallback", compatibility.Speaches, `{"data":[{"id":"alias"}]}`, `{"voices":[{"id":"af_heart","name":"Heart"}]}`, VoiceScopeServer, 2},
		{"speaches empty model", compatibility.Speaches, `{"data":[{"id":"chosen","voices":[]}]}`, "", VoiceScopeModel, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Error("unsafe metadata request")
				}
				switch r.URL.Path {
				case "/proxy/v1/models":
					fmt.Fprint(w, tc.models)
				case "/proxy/v1/audio/voices":
					fmt.Fprint(w, tc.voices)
				default:
					t.Error("unexpected route")
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			result := New().ListVoices(context.Background(), tc.backend, server.URL+"/proxy/v1", "fixture-key", "chosen")
			if result.ErrorKind != "" || result.Scope != tc.scope || calls != tc.calls {
				t.Fatalf("result=%+v calls=%d", result, calls)
			}
			if tc.name != "speaches empty model" && (len(result.Voices) != 1 || result.Voices[0].ID != "af_heart" || result.Voices[0].Name == "") {
				t.Fatal(result)
			}
			if tc.name == "speaches empty model" && len(result.Voices) != 0 {
				t.Fatal("empty model list replaced by unrelated voices")
			}
		})
	}
}
func TestVoiceDiscoveryRejectsUnsafeResponsesAndRemainsBounded(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"malformed", `<html>login</html>`, "response", 200},
		{"wrong shape", `{"data":[]}`, "response", 200},
		{"unauthorized", `secret error body`, "http", 401},
		{"redirect", "", "http", 302},
		{"oversized", strings.Repeat("x", maxResponse+1), "response_too_large", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Location", "/other")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			result := New().ListVoices(context.Background(), compatibility.KokoroFastAPI, server.URL+"/v1", "secret", "")
			if result.ErrorKind != tc.want || calls != 1 || len(result.Voices) != 0 {
				t.Fatal(result, calls)
			}
		})
	}
	rows := []map[string]string{{"id": "secret"}, {"id": "safe", "name": "secret", "language": "secret"}, {"id": "safe"}, {"id": "line\nbreak"}, {"id": strings.Repeat("x", 201)}}
	for i := 0; i < MaxDiscoveredVoices+5; i++ {
		rows = append(rows, map[string]string{"id": fmt.Sprintf("voice-%d", i)})
	}
	body, _ := json.Marshal(map[string]any{"voices": rows})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	result := New().ListVoices(context.Background(), compatibility.KokoroFastAPI, server.URL, "secret", "")
	encoded, _ := json.Marshal(result)
	if len(result.Voices) != MaxDiscoveredVoices || !result.Truncated || strings.Contains(string(encoded), "secret") || result.Voices[0].Name != "safe" {
		t.Fatal("unsafe/unbounded voices")
	}
	result = New().ListVoices(context.Background(), compatibility.Generic, server.URL, "", "")
	if result.ErrorKind != "unsupported" {
		t.Fatal("generic discovery enabled")
	}
}
func TestKokoroSpeechExplicitlyDisablesStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Method != "POST" || r.URL.Path != "/v1/audio/speech" || json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("invalid speech request")
		}
		if len(body) != 6 || body["stream"] != false || body["response_format"] != "wav" || body["voice"] != "af_heart" || body["speed"] != 1.25 {
			t.Error(body)
		}
		w.Header().Set("Content-Type", "audio/wav")
		fmt.Fprint(w, "fixture")
	}))
	defer server.Close()
	_, err := New().SynthesizeSpeech(context.Background(), server.URL+"/v1", "", SpeechRequest{CompatibilityProfile: compatibility.KokoroFastAPI, Model: "kokoro", Voice: "af_heart", Input: "Fixed fixture", Speed: 1.25})
	if err != nil {
		t.Fatal(err)
	}
}
