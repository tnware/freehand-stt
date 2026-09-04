package audio

import (
	"encoding/binary"
	"testing"
)

func TestWAV(t *testing.T) {
	b, e := WAV([]byte{1, 2, 3, 4})
	if e != nil || string(b[:4]) != "RIFF" || binary.LittleEndian.Uint32(b[24:]) != 16000 || len(b) != 48 {
		t.Fatalf("bad WAV: %v %d", e, len(b))
	}
}

func TestPCM16WAVPreservesPlaybackFormat(t *testing.T) {
	b, err := PCM16WAV([]byte{1, 0, 2, 0, 3, 0, 4, 0}, 24000, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(b[22:]); got != 2 {
		t.Fatalf("channels = %d", got)
	}
	if got := binary.LittleEndian.Uint32(b[24:]); got != 24000 {
		t.Fatalf("sample rate = %d", got)
	}
	if got := binary.LittleEndian.Uint32(b[28:]); got != 96000 {
		t.Fatalf("byte rate = %d", got)
	}
	if got := binary.LittleEndian.Uint16(b[32:]); got != 4 {
		t.Fatalf("block align = %d", got)
	}
}
func TestBounds(t *testing.T) {
	n, e := MaxPCMBytes(120)
	if e != nil || n != 3840000 {
		t.Fatalf("%d %v", n, e)
	}
	if _, e = MaxPCMBytes(0); e == nil {
		t.Fatal("expected error")
	}
}
