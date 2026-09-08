package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestVLLMProtocolAndFinalAuthority(t *testing.T) {
	for _, profile := range []modelprofile.ID{modelprofile.Qwen3ASR, modelprofile.VoxtralRealtime} {
		t.Run(string(profile), func(t *testing.T) { testVLLMProtocol(t, profile) })
	}
}
func testVLLMProtocol(t *testing.T, profile modelprofile.ID) {
	for _, scenario := range []string{"final", "missing-text", "disconnect", "error", "early-final", "oversize", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			preview := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/realtime" || r.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Error("wrong upgrade")
					return
				}
				c, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer c.CloseNow()
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()
				send := func(value string) { _ = c.Write(ctx, websocket.MessageText, []byte(value)) }
				send(`{"type":"session.created","id":"fixture"}`)
				_, b, err := c.Read(ctx)
				if err != nil {
					return
				}
				var selected map[string]any
				if json.Unmarshal(b, &selected) != nil || len(selected) != 2 || selected["model"] != "fixture-model" || selected["type"] != "session.update" {
					t.Error("session must contain only type and model")
					return
				}
				_, b, err = c.Read(ctx)
				if err != nil {
					return
				}
				if string(b) != `{"type":"input_audio_buffer.commit","final":false}` {
					t.Error("missing generation start")
					return
				}
				if scenario == "early-final" {
					send(`{"type":"transcription.done","text":"unrequested"}`)
					return
				}
				for {
					kind, b, err := c.Read(ctx)
					if err != nil {
						return
					}
					var event struct {
						Type  string `json:"type"`
						Audio string `json:"audio"`
						Final bool   `json:"final"`
					}
					if kind != websocket.MessageText || json.Unmarshal(b, &event) != nil {
						t.Error("expected JSON audio")
						return
					}
					if event.Type == "input_audio_buffer.append" {
						pcm, err := base64.StdEncoding.DecodeString(event.Audio)
						if err != nil || len(pcm) != audio.VADFrameBytes {
							t.Error("wrong PCM frame")
						}
						send(`{"type":"transcription.delta","delta":"language English<asr_te"}`)
						send(`{"type":"transcription.delta","delta":"xt>Provisional."}`)
						if scenario == "oversize" {
							value, _ := json.Marshal(map[string]string{"type": "transcription.delta", "delta": strings.Repeat("x", MaxTranscriptBytes)})
							send(string(value))
							return
						}
						continue
					}
					if event.Type != "input_audio_buffer.commit" || !event.Final {
						t.Error("wrong stop")
						return
					}
					switch scenario {
					case "final":
						send(`{"type":"transcription.done","text":"language English<asr_text>Correct.language English<asr_text>Final."}`)
					case "missing-text":
						send(`{"type":"transcription.done"}`)
					case "error":
						send(`{"type":"error","error":"private peer content"}`)
					}
					return
				}
			}))
			defer server.Close()
			cfg := fixtureConfig(server)
			cfg.CompatibilityProfile = compatibility.VLLM
			cfg.ModelProfile = profile
			cfg.Language = "auto"
			cfg.Options = modelprofile.NemotronOptions{}
			// Completed hints are saved but must never leak into realtime messages.
			if profile == modelprofile.Qwen3ASR {
				cfg.TranscriptionOptions = compatibility.TranscriptionOptions{Prompt: "private context", TemperatureOverride: true, Temperature: .3}
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var updates []Update
			s, err := Open(ctx, cfg, "fixture-key", func(u Update) {
				updates = append(updates, u)
				if u.Partial != "" {
					select {
					case preview <- struct{}{}:
					default:
					}
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "early-final" {
				select {
				case <-s.Failed:
				case <-time.After(2 * time.Second):
					t.Fatal("early final not rejected")
				}
			} else {
				if !s.Pipe.WritePCM(make([]byte, audio.VADFrameBytes)) {
					t.Fatal("frame refused")
				}
				if scenario == "cancel" {
					select {
					case <-preview:
					case <-time.After(2 * time.Second):
						t.Fatal("no preview")
					}
					cancel()
				}
			}
			s.Pipe.Close()
			select {
			case <-s.Done:
			case <-time.After(3 * time.Second):
				t.Fatal("workers did not stop")
			}
			result := s.Wait()
			if scenario == "final" {
				expected, language := "Correct. Final.", "en"
				if profile == modelprofile.VoxtralRealtime {
					expected, language = "language English<asr_text>Correct.language English<asr_text>Final.", ""
				}
				if result.Err != nil || result.Text != expected || result.Language != language || result.AudioMilliseconds != 20 {
					t.Fatalf("wrong final: %#v", result)
				}
				if len(updates) != 3 || (profile == modelprofile.Qwen3ASR && (updates[0].Partial != "" || updates[1].Partial != "Provisional.")) || updates[2].Final != result.Text {
					t.Fatal("wrong preview replacement")
				}
			} else if result.Err == nil || result.Text != "" || strings.Contains(result.Err.Error(), "private") {
				t.Fatalf("unsafe failure: %#v", result)
			}
		})
	}
}

func TestVLLMRequiresExplicitModelProfileBeforeConnecting(t *testing.T) {
	cfg := config.DefaultVoiceTranscription()
	cfg.Realtime = true
	cfg.CompatibilityProfile = compatibility.VLLM
	if _, err := Open(t.Context(), cfg, "", nil); err == nil {
		t.Fatal("unqualified profile admitted")
	}
}
