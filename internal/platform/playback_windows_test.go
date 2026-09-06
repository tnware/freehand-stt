//go:build windows

package platform

import (
	"bytes"
	"github.com/tnware/freehand-stt/internal/audio"
	"testing"
	"time"
)

func TestPlaybackPositionWaitsForAudibleDurationAfterBufferSubmission(t *testing.T) {
	const sampleRate = 16_000
	player := &Playback{
		data:     make([]byte, sampleRate*2),
		position: sampleRate * 2,
		channels: 1,
		rate:     sampleRate,
		elapsed:  25 * time.Millisecond,
	}

	position, duration, done := player.Position()
	if position != 25 || duration != 1000 || done {
		t.Fatalf("queued buffer completed early: position=%d duration=%d done=%v", position, duration, done)
	}

	player.elapsed = time.Second + playbackDrainGrace
	position, duration, done = player.Position()
	if position != 1000 || duration != 1000 || !done {
		t.Fatalf("drained buffer remained active: position=%d duration=%d done=%v", position, duration, done)
	}
}

func TestPlaybackSnapshotIsIndependentOfUnload(t *testing.T) {
	pcm := []byte{1, 0, 2, 0}
	player := &Playback{data: append([]byte(nil), pcm...), rate: 16000, channels: 1}
	wav, err := player.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Unload(); err != nil {
		t.Fatal(err)
	}
	want, _ := audio.WAV(pcm)
	if !bytes.Equal(wav, want) {
		t.Fatal("unload changed the export snapshot")
	}
	if _, err := player.Snapshot(); err == nil {
		t.Fatal("unloaded player retained exportable audio")
	}
	clear(wav)
}
