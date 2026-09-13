package managedruntime_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/realtime"
)

// Exercise the production adapter's published URL with the production clients.
// The fixture implements NeMo's versioned routes; no real model is loaded.
func TestManagedEndpointReachesSpeechClients(t *testing.T) {
	managedruntime.WithManagedEndpointFixture(t, func(ep managedruntime.Endpoint) {
		for _, profile := range []modelprofile.ID{modelprofile.ParakeetTDT, modelprofile.Nemotron35} {
			t.Run(string(profile), func(t *testing.T) {
				client := inference.New().WithCompatibility(compatibility.NeMoSpeechV1).WithModelProfile(profile)
				t.Run("completed microphone", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
					defer cancel()
					result, err := client.Transcribe(ctx, ep.BaseURL, ep.Model, "", "", nil, []byte("fixture"))
					if err != nil || result.Text != "fixture transcript" {
						t.Fatalf("managed microphone request: result=%+v error=%v", result, err)
					}
				})
				t.Run("audio file", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
					defer cancel()
					wav := []byte("fixture")
					result, err := client.TranscribeFile(ctx, ep.BaseURL, ep.Model, "", "", nil, "fixture.wav", int64(len(wav)), bytes.NewReader(wav), false, inference.FileTranscriptionCallbacks{})
					if err != nil || result.Text != "fixture transcript" {
						t.Fatalf("managed file request: result=%+v error=%v", result, err)
					}
				})
			})
		}
		t.Run("realtime handshake", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			cfg := config.DefaultVoiceTranscription()
			cfg.BaseURL = ep.BaseURL
			cfg.Model = ep.Model
			cfg.ModelProfile = modelprofile.ID(ep.Profile)
			cfg.CompatibilityProfile = compatibility.NeMoSpeechV1
			cfg.AuthenticationMode = config.AuthenticationModeNone
			cfg.AllowInsecureHTTP = true
			cfg.Realtime = ep.Realtime
			session, err := realtime.Open(ctx, cfg, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			session.AbortBeforeCapture()
		})
	})
}
