package tts

import (
	"encoding/binary"
	"errors"
)

type PCM struct {
	Data       []byte
	SampleRate uint32
	Channels   uint32
}

func decodeWAV(data []byte) (PCM, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return PCM{}, errors.New("speech endpoint did not return a WAV file")
	}
	var formatTag, channels, bits uint16
	var sampleRate uint32
	var samples []byte
	for offset := 12; offset+8 <= len(data); {
		name := string(data[offset : offset+4])
		declaredSize := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		size := int(declaredSize)
		start := offset + 8
		// Streaming WAV writers cannot know the final data length when they
		// emit the header, so 0xffffffff conventionally means "to EOF".
		// The HTTP response is fully buffered and bounded before decoding.
		if name == "data" && declaredSize == ^uint32(0) {
			size = len(data) - start
		}
		end := start + size
		if size < 0 || end < start || end > len(data) {
			return PCM{}, errors.New("speech WAV contains an invalid chunk")
		}
		switch name {
		case "fmt ":
			if size < 16 {
				return PCM{}, errors.New("speech WAV format is incomplete")
			}
			formatTag = binary.LittleEndian.Uint16(data[start : start+2])
			channels = binary.LittleEndian.Uint16(data[start+2 : start+4])
			sampleRate = binary.LittleEndian.Uint32(data[start+4 : start+8])
			bits = binary.LittleEndian.Uint16(data[start+14 : start+16])
		case "data":
			samples = data[start:end]
		}
		offset = end + size%2
	}
	if formatTag != 1 || bits != 16 || (channels != 1 && channels != 2) || sampleRate < 8000 || sampleRate > 192000 || len(samples) == 0 {
		return PCM{}, errors.New("speech WAV must contain mono or stereo 16-bit PCM audio")
	}
	frameBytes := int(channels) * 2
	if len(samples)%frameBytes != 0 {
		return PCM{}, errors.New("speech WAV audio is truncated")
	}
	// The decoded PCM must own its memory. The caller clears the bounded HTTP
	// response as soon as decoding finishes; returning a view into that buffer
	// would silently turn every sample into zero before native playback loads it.
	pcm := append([]byte(nil), samples...)
	return PCM{Data: pcm, SampleRate: sampleRate, Channels: uint32(channels)}, nil
}
