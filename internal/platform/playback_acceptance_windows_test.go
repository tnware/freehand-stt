//go:build windows

package platform

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// This opt-in check exercises the real output device with silence only.
// It does not capture a microphone or contact an inference server.
func TestNativePlaybackSeek(t *testing.T) {
	if os.Getenv("FREEHAND_NATIVE_PLAYBACK_ACCEPTANCE") != "1" {
		t.Skip("set FREEHAND_NATIVE_PLAYBACK_ACCEPTANCE=1 to exercise Windows audio output")
	}
	player := &Playback{}
	t.Cleanup(func() { _ = player.Close() })
	if err := player.Load(make([]byte, 16000*2*2), 16000, 1); err != nil {
		t.Fatal(err)
	}
	before, err := player.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Play(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if err := player.Seek(1250); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if position, duration, done := player.Position(); position != 1250 || duration != 2000 || done {
		t.Fatalf("seek did not hold paused: position=%d duration=%d done=%v", position, duration, done)
	}
	if err := player.Play(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if position, _, _ := player.Position(); position <= 1250 {
		t.Fatalf("resumed output clock did not advance: %d", position)
	}
	if err := player.Seek(1800); err != nil {
		t.Fatal(err)
	}
	if err := player.Play(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, _, done := player.Position(); done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("native output did not drain after seeking")
		}
		time.Sleep(20 * time.Millisecond)
	}
	after, err := player.Snapshot()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("seek changed the full audio export: %v", err)
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}
	if err := player.Seek(0); err == nil {
		t.Fatal("closed output accepted seek")
	}
}
