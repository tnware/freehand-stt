//go:build windows

package platform

import (
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
