//go:build windows

package platform

import (
	"context"
	"os"
	"testing"
	"time"
)

// Opt-in hardware acceptance. Capture is RAM-only and immediately discarded;
// playback uses silence. No endpoint, inference, user settings or file is used.
func TestNativeAudioShutdown(t *testing.T) {
	if os.Getenv("FREEHAND_NATIVE_AUDIO_ACCEPTANCE") != "1" {
		t.Skip("set FREEHAND_NATIVE_AUDIO_ACCEPTANCE=1 to exercise the default Windows audio devices")
	}
	closeWithin := func(t *testing.T, closeAudio func() error) {
		t.Helper()
		done := make(chan error, 1)
		go func() { done <- closeAudio() }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("native audio teardown exceeded three seconds")
		}
	}
	t.Run("active microphone", func(t *testing.T) {
		capture := &Capture{}
		if _, err := capture.Start(context.Background(), "", 1); err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond) // Allow actual WASAPI callbacks to run.
		closeWithin(t, capture.Close)
		if capture.dev != nil || capture.ctx != nil || len(capture.pcm) != 0 {
			t.Fatal("microphone retained resources after Close")
		}
	})
	for _, name := range []string{"playing", "paused", "restarted"} {
		t.Run(name, func(t *testing.T) {
			player := &Playback{}
			if err := player.Load(make([]byte, 16000*2), 16000, 1); err != nil {
				t.Fatal(err)
			}
			if err := player.Play(); err != nil {
				t.Fatal(err)
			}
			time.Sleep(100 * time.Millisecond)
			if name == "paused" {
				if err := player.Pause(); err != nil {
					t.Fatal(err)
				}
			}
			if name == "restarted" {
				if err := player.Rewind(); err != nil {
					t.Fatal(err)
				}
				if err := player.Play(); err != nil {
					t.Fatal(err)
				}
			}
			closeWithin(t, player.Close)
			if player.dev != nil || player.ctx != nil || len(player.data) != 0 {
				t.Fatal("playback retained resources after Close")
			}
		})
	}
}
