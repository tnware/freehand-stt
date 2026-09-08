//go:build windows

package platform

import (
	"bytes"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
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

func TestPlaybackSeekAlignsPCMAndRetainsFullExport(t *testing.T) {
	device := &restartDevice{}
	pcm := make([]byte, 44100*4*2)
	for i := range pcm {
		pcm[i] = byte(i % 251)
	}
	player := &Playback{dev: device, data: pcm, rate: 44100, channels: 2, playing: true}
	before, err := player.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Seek(123); err != nil {
		t.Fatal(err)
	}
	wantFrame := 123 * 44100 / 1000
	if player.position != wantFrame*4 || player.playing || device.starts.Load() != 0 {
		t.Fatal("seek did not stop at a whole stereo frame")
	}
	output := make([]byte, 40)
	player.render(output, nil, 10)
	if !bytes.Equal(output, pcm[wantFrame*4:wantFrame*4+40]) {
		t.Fatal("render did not read the sought PCM")
	}
	after, _ := player.Snapshot()
	if !bytes.Equal(before, after) {
		t.Fatal("seek changed retained export")
	}
	if err := player.Seek(2000); err != nil || player.position != len(pcm) {
		t.Fatal("end seek did not reach the last frame")
	}
	if _, _, done := player.Position(); done {
		t.Fatal("paused seek claimed an audible drain")
	}
	for _, invalid := range []int64{-1, 2001, 1 << 62} {
		if err := player.Seek(invalid); err == nil {
			t.Fatal("invalid native seek accepted")
		}
	}
}
