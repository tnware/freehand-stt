package tts

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/tnware/freehand-stt/internal/audio"
)

func TestDecodeWAVAcceptsFreehandPCM(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	pcm, err := decodeWAV(wav)
	if err != nil {
		t.Fatal(err)
	}
	if pcm.SampleRate != audio.SampleRate || pcm.Channels != 1 || len(pcm.Data) != 4 {
		t.Fatalf("pcm = %#v", pcm)
	}
}

func TestDecodedPCMDoesNotAliasClearedResponse(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0, 3, 0})
	if err != nil {
		t.Fatal(err)
	}
	pcm, err := decodeWAV(wav)
	if err != nil {
		t.Fatal(err)
	}
	clear(wav)
	if got, want := pcm.Data, []byte{1, 0, 2, 0, 3, 0}; !bytes.Equal(got, want) {
		t.Fatalf("decoded PCM was erased with response buffer: got %v want %v", got, want)
	}
}

func TestDecodeWAVRejectsCompressedAudio(t *testing.T) {
	if _, err := decodeWAV([]byte("not wave audio")); err == nil {
		t.Fatal("expected malformed WAV to fail")
	}
}

func TestDecodeWAVAcceptsStreamingDataLength(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0, 3, 0})
	if err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(wav[40:44], ^uint32(0))

	pcm, err := decodeWAV(wav)
	if err != nil {
		t.Fatalf("decode streaming WAV: %v", err)
	}
	if got, want := len(pcm.Data), 6; got != want {
		t.Fatalf("sample bytes = %d, want %d", got, want)
	}
}
