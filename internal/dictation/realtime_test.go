package dictation

import (
	"context"
	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/realtime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type liveCaptureFixture struct {
	capFake
	sink audio.PCMStreamSink
}

func (c *liveCaptureFixture) StartStream(_ context.Context, _ string, _ int, sink audio.PCMStreamSink) (<-chan error, error) {
	c.sink = sink
	return nil, nil
}
func (c *liveCaptureFixture) Stop(context.Context) (audio.Result, error) {
	c.sink.Close()
	return audio.Result{}, nil
}
func (c *liveCaptureFixture) Cancel(context.Context) error {
	if c.sink != nil {
		c.sink.Close()
	}
	return nil
}

type liveSettingsFixture struct{ cfg config.Settings }

func (s liveSettingsFixture) Current() config.Settings { return s.cfg }
func TestRealtimeNeverDeliversPreviewAndCancellationDiscardsIt(t *testing.T) {
	for _, cancelRecording := range []bool{false, true} {
		name := "finalize"
		if cancelRecording {
			name = "cancel"
		}
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer conn.CloseNow()
				ctx, stop := context.WithTimeout(r.Context(), 5*time.Second)
				defer stop()
				send := func(text string) { _ = conn.Write(ctx, websocket.MessageText, []byte(text)) }
				send(`{"type":"session.created","session":{"model":"fixture"}}`)
				if _, _, err = conn.Read(ctx); err != nil {
					return
				}
				send(`{"type":"session.updated"}`)
				for {
					kind, b, err := conn.Read(ctx)
					if err != nil {
						return
					}
					if kind == websocket.MessageBinary {
						send(`{"type":"conversation.item.input_audio_transcription.delta","delta":"wrong preview"}`)
					} else if strings.Contains(string(b), "input_audio_buffer.commit") {
						send(`{"type":"conversation.item.input_audio_transcription.completed","transcript":"Correct final."}`)
						send(`{"type":"input_audio_buffer.committed"}`)
						return
					}
				}
			}))
			defer server.Close()
			cfg := config.Default()
			cfg.VoiceTranscription.AllowInsecureHTTP = true
			cfg.VoiceTranscription.Realtime = true
			cfg.VoiceTranscription.CompatibilityProfile = "nemo-speech-v1"
			cfg.VoiceTranscription.ModelProfile = "nemotron-3.5-streaming"
			cfg.VoiceTranscription.BaseURL = server.URL + "/v1"
			cfg.VoiceTranscription.Model = "fixture"
			cfg.AutoInsert = true
			capture := &liveCaptureFixture{}
			target := &platFake{}
			updates := make(chan Status, 20)
			c := New(capture, target, nil, nil, liveSettingsFixture{cfg}, func(s Status) { updates <- s })
			defer c.Cancel()
			if err := c.Start(); err != nil {
				t.Fatal(err)
			}
			if !capture.sink.WritePCM(make([]byte, audio.VADFrameBytes)) {
				t.Fatal("frame refused")
			}
			timeout := time.NewTimer(2 * time.Second)
			defer timeout.Stop()
			for {
				select {
				case s := <-updates:
					if s.LivePartial != "wrong preview" {
						continue
					}
					if s.Transcript != "" || s.CanCopy || target.inserts != 0 {
						t.Fatal("preview gained delivery authority")
					}
				case <-timeout.C:
					t.Fatal("live preview never arrived")
				}
				break
			}
			generation := c.Status().Generation
			if cancelRecording {
				if err := c.Cancel(); err != nil {
					t.Fatal(err)
				}
				c.publishLive(generation, realtime.Update{Partial: "late preview"})
				if c.Status().LivePartial != "" || target.inserts != 0 || len(c.History()) != 0 {
					t.Fatal("cancel retained or delivered preview")
				}
			} else {
				if err := c.Stop(); err != nil {
					t.Fatal(err)
				}
				if target.inserts != 1 || target.lastInsert != "Correct final." {
					t.Fatal("final did not replace provisional text for delivery")
				}
			}
		})
	}
}
