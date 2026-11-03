package dictation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
)

func TestRecorderLiveTypingLifecycle(t *testing.T) {
	for _, mode := range []string{"stop", "cancel", "disconnect", "conflict", "legacy", "manual", "disabled", "queued-disconnect", "queued-conflict", "stop-disconnect", "stop-conflict"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer conn.CloseNow()
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()
				send := func(s string) { _ = conn.Write(ctx, websocket.MessageText, []byte(s)) }
				send(`{"type":"session.created","session":{"model":"fixture"}}`)
				if _, _, err = conn.Read(ctx); err != nil {
					return
				}
				send(`{"type":"session.updated"}`)
				frames := 0
				for {
					kind, b, err := conn.Read(ctx)
					if err != nil {
						return
					}
					if kind == websocket.MessageBinary {
						frames++
						if frames == 1 {
							event := `{"type":"conversation.item.input_audio_transcription.completed","event_id":"one","transcript":"First."}`
							if mode == "legacy" {
								event = strings.Replace(event, `"event_id":"one",`, "", 1)
							}
							send(event)
						} else {
							if mode == "conflict" || mode == "queued-conflict" {
								send(`{"type":"conversation.item.input_audio_transcription.completed","event_id":"one","transcript":"Unsafe replacement."}`)
							}
							return
						}
					} else if strings.Contains(string(b), "input_audio_buffer.commit") {
						send(`{"type":"conversation.item.input_audio_transcription.completed","event_id":"two","transcript":"Second."}`)
						if mode == "stop-conflict" {
							send(`{"type":"conversation.item.input_audio_transcription.completed","event_id":"two","transcript":"Unsafe replacement."}`)
						}
						if strings.HasPrefix(mode, "stop-") {
							return
						}
						send(`{"type":"input_audio_buffer.committed"}`)
						return
					}
				}
			}))
			defer server.Close()
			cfg := config.Default()
			cfg.LiveInsert = mode != "disabled"
			cfg.AutoInsert = mode != "manual"
			cfg.PostProcessing.Enabled = false
			cfg.VoiceTranscription.AllowInsecureHTTP = true
			cfg.VoiceTranscription.Realtime = true
			cfg.VoiceTranscription.CompatibilityProfile = "nemo-speech-v1"
			cfg.VoiceTranscription.ModelProfile = "nemotron-3.5-streaming"
			cfg.VoiceTranscription.BaseURL = server.URL + "/v1"
			cfg.VoiceTranscription.Model = "fixture"
			capture := &liveCaptureFixture{}
			p := &segmentFixture{sends: make(chan string, 10)}
			queued := strings.HasPrefix(mode, "queued-")
			if strings.HasPrefix(mode, "stop-") {
				p.block = make(chan struct{})
			}
			if queued {
				p.foregroundEntered = make(chan struct{}, 1)
				p.foregroundRelease = make(chan struct{})
			}
			released := false
			defer func() {
				if queued && !released {
					close(p.foregroundRelease)
				}
			}()
			updates := make(chan Status, 100)
			c := New(capture, p, nil, nil, liveSettingsFixture{cfg}, func(s Status) { updates <- s })
			defer c.close()
			if err := c.Start(); err != nil {
				t.Fatal(err)
			}
			if !capture.sink.WritePCM(make([]byte, audio.VADFrameBytes)) {
				t.Fatal("audio refused")
			}
			waitStatus := func(want func(Status) bool) Status {
				t.Helper()
				timer := time.NewTimer(2 * time.Second)
				defer timer.Stop()
				for {
					select {
					case s := <-updates:
						if want(s) {
							return s
						}
					case <-timer.C:
						t.Fatal("status timeout")
						return Status{}
					}
				}
			}
			waitStatus(func(s Status) bool { return s.LiveFinal == "First." })
			live := mode != "legacy" && mode != "manual" && mode != "disabled"
			if queued {
				select {
				case <-p.foregroundEntered:
				case <-time.After(time.Second):
					t.Fatal("validation did not start")
				}
			}
			if live && !queued {
				if got := awaitSend(t, p); got != "First." {
					t.Fatal(got)
				}
			}
			switch mode {
			case "stop-disconnect", "stop-conflict":
				if err := c.Stop(); err == nil {
					t.Fatal("stop-time failure accepted")
				}
				s := c.Status()
				if s.Transcript != "First. Second." || !s.CanCopy || !strings.Contains(s.Message, "has not been removed") {
					t.Fatalf("%+v", s)
				}
			case "cancel":
				if err := c.Cancel(); err != nil {
					t.Fatal(err)
				}
				s := c.Status()
				if s.Transcript != "First." || !s.CanCopy || !strings.Contains(s.Message, "has not been removed") {
					t.Fatalf("%+v", s)
				}
			case "disconnect", "conflict", "queued-disconnect", "queued-conflict":
				if !capture.sink.WritePCM(make([]byte, audio.VADFrameBytes)) {
					t.Fatal("audio refused")
				}
				if queued {
					waitStatus(func(s Status) bool { return strings.Contains(s.Message, "Live typing stopped") })
					close(p.foregroundRelease)
					released = true
				}
				s := waitStatus(func(s Status) bool { return s.State == Failed })
				if s.Transcript != "First." || !s.CanCopy || !strings.Contains(s.Message, "has not been removed") {
					t.Fatalf("%+v", s)
				}
			default:
				if err := c.Stop(); err != nil {
					t.Fatal(err)
				}
				s := c.Status()
				if s.Transcript != "First. Second." {
					t.Fatalf("%+v", s)
				}
				if mode == "manual" && !s.CanCopy {
					t.Fatal("manual transcript lost")
				}
			}
			_ = c.Stop()
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.copies != 0 {
				t.Fatal("automatic clipboard write")
			}
			if live {
				want := 1
				if queued {
					want = 0
				}
				if mode == "stop" {
					want = 2
				}
				if len(p.parts) != want || len(p.ones) != 0 {
					t.Fatalf("parts=%v one-shot=%v", p.parts, p.ones)
				}
				if mode == "stop" && strings.Join(p.parts, "") != "First. Second." {
					t.Fatal(p.parts)
				}
			} else {
				want := 1
				if mode == "manual" {
					want = 0
				}
				if len(p.parts) != 0 || len(p.ones) != want {
					t.Fatalf("fallback parts=%v one-shot=%v", p.parts, p.ones)
				}
			}
		})
	}
}
