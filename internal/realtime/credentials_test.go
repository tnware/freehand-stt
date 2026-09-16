package realtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestRealtimeRejectsReflectedCredentials(t *testing.T) {
	for _, profile := range []modelprofile.ID{modelprofile.Nemotron35, modelprofile.Qwen3ASR, modelprofile.VoxtralRealtime} {
		t.Run(string(profile), func(t *testing.T) {
			delta := func(text string) string {
				kind := "transcription.delta"
				if profile == modelprofile.Nemotron35 {
					kind = "conversation.item.input_audio_transcription.delta"
				}
				return `{"type":` + strconv.Quote(kind) + `,"delta":` + strconv.Quote(text) + `}`
			}
			final := func(text string) string {
				if profile == modelprofile.Nemotron35 {
					return `{"type":"conversation.item.input_audio_transcription.completed","transcript":` + strconv.Quote(text) + `}`
				}
				return `{"type":"transcription.done","text":` + strconv.Quote(text) + `}`
			}
			type scenario struct {
				name, key, want string
				events          []string
			}
			cases := []scenario{
				{name: "delta", key: "fixture-key", events: []string{delta("fixture-key"), final("safe")}},
				{name: "split-delta", key: "fixture-key", events: []string{delta("fixture-"), delta("key"), final("safe")}},
				{name: "final", key: "fixture-key", events: []string{delta("safe"), final("fixture-key")}},
				{name: "ordinary", key: "fixture-key", events: []string{delta("fixture-"), delta("text"), final("safe")}, want: "safe"},
				{name: "unauthenticated", events: []string{delta("fixture-key"), final("fixture-key")}, want: "fixture-key"},
			}
			if profile == modelprofile.Nemotron35 {
				cases = append(cases,
					scenario{name: "language", key: "en-US", events: []string{`{"type":"conversation.item.input_audio_transcription.completed","transcript":"safe","language":"en-US"}`}},
					scenario{name: "languages", key: "fixture-key", events: []string{`{"type":"conversation.item.input_audio_transcription.completed","transcript":"safe","languages":["fixture-key"]}`}},
					scenario{name: "joined-finals", key: "fixture key", events: []string{final("fixture"), final("key")}},
					scenario{name: "joined-final-preview", key: "fixture key", events: []string{final("fixture"), delta("key")}},
				)
			}
			if profile == modelprofile.Qwen3ASR {
				cases = append(cases,
					scenario{name: "parsed-delta", key: "fixture key", events: []string{delta("fixturelanguage English<asr_text>key"), final("safe")}},
					scenario{name: "parsed-final", key: "fixture key", events: []string{final("fixturelanguage English<asr_text>key")}},
				)
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					server := credentialFixture(t, profile, tc.key, func(ctx context.Context, conn *websocket.Conn) {
						for {
							_, message, err := conn.Read(ctx)
							if err != nil {
								return
							}
							if !strings.Contains(string(message), "input_audio_buffer.commit") {
								continue
							}
							for _, event := range tc.events {
								if conn.Write(ctx, websocket.MessageText, []byte(event)) != nil {
									return
								}
							}
							if profile == modelprofile.Nemotron35 {
								_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"input_audio_buffer.committed"}`))
							}
							return
						}
					})
					var updates []Update
					session, err := Open(t.Context(), credentialFixtureConfig(server, profile, tc.key), tc.key, func(update Update) { updates = append(updates, update) })
					if err != nil {
						t.Fatal(err)
					}
					session.Pipe.Close()
					result := session.Wait()
					if tc.want != "" {
						if result.Err != nil || result.Text != tc.want {
							t.Fatalf("ordinary transcript rejected: %+v", result)
						}
						return
					}
					if diagnostics.ErrorKind(result.Err) != "credential_reflection" || result.Text != "" || result.Language != "" || len(result.Languages) != 0 {
						t.Fatal("reflected credential did not produce an empty, classified failure")
					}
					if strings.Contains(result.Err.Error(), tc.key) {
						t.Fatal("credential exposed through error")
					}
					for _, update := range updates {
						if strings.Contains(update.Final, tc.key) || strings.Contains(update.Partial, tc.key) || strings.Contains(update.Final+" "+update.Partial, tc.key) {
							t.Fatal("credential exposed through live publication")
						}
					}
				})
			}
		})
	}
}

func TestReflectedCredentialStopsRealtimeCaptureAndDrainsAudio(t *testing.T) {
	for _, profile := range []modelprofile.ID{modelprofile.Nemotron35, modelprofile.VoxtralRealtime} {
		t.Run(string(profile), func(t *testing.T) {
			server := credentialFixture(t, profile, "fixture-key", func(ctx context.Context, conn *websocket.Conn) {
				if _, _, err := conn.Read(ctx); err != nil {
					return
				}
				kind := "transcription.delta"
				if profile == modelprofile.Nemotron35 {
					kind = "conversation.item.input_audio_transcription.delta"
				}
				_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":`+strconv.Quote(kind)+`,"delta":"fixture-key"}`))
				_, _, _ = conn.Read(ctx)
			})
			session, err := Open(t.Context(), credentialFixtureConfig(server, profile, "fixture-key"), "fixture-key", func(Update) { t.Error("reflected text published") })
			if err != nil {
				t.Fatal(err)
			}
			defer session.AbortBeforeCapture()
			if !session.Pipe.WritePCM(make([]byte, audio.VADFrameBytes)) {
				t.Fatal("frame refused")
			}
			select {
			case <-session.Failed:
			case <-time.After(2 * time.Second):
				t.Fatal("capture was not notified of rejected credentials")
			}
			session.Pipe.WritePCM(make([]byte, audio.VADFrameBytes))
			session.Pipe.Close()
			if result := session.Wait(); diagnostics.ErrorKind(result.Err) != "credential_reflection" || result.Text != "" {
				t.Fatal("rejected session yielded deliverable text")
			}
		})
	}
}

func credentialFixture(t *testing.T, profile modelprofile.ID, key string, after func(context.Context, *websocket.Conn)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantHeader := ""
		if key != "" {
			wantHeader = "Bearer " + key
		}
		if r.Header.Get("Authorization") != wantHeader {
			t.Error("unexpected credential header")
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"session.created","session":{"model":"fixture-model"}}`))
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		if profile == modelprofile.Nemotron35 {
			_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"session.updated"}`))
		} else if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		after(ctx, conn)
	}))
	t.Cleanup(server.Close)
	return server
}

func credentialFixtureConfig(server *httptest.Server, profile modelprofile.ID, key string) config.VoiceTranscriptionSettings {
	cfg := fixtureConfig(server)
	cfg.ModelProfile = profile
	if profile != modelprofile.Nemotron35 {
		cfg.CompatibilityProfile = compatibility.VLLM
	}
	if key == "" {
		cfg.AuthenticationMode = config.AuthenticationModeNone
	}
	return cfg
}
