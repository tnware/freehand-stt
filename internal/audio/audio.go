package audio

import (
	"context"
	"encoding/binary"
	"errors"
)

const SampleRate = 16000
const MaxSingleRequestDurationSeconds = 262
const MaxSegmentedRecordingDurationSeconds = 3600

var ErrDurationLimit = errors.New("maximum recording duration reached")
var ErrDeviceInterrupted = errors.New("device was removed or stopped unexpectedly")

type Device struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}
type Result struct {
	PCM          []byte
	LimitReached bool
}

type Capture interface {
	List(context.Context) ([]Device, error)
	// Start returns a per-recording interruption channel. Native callbacks must
	// signal it without blocking; a nil channel means the implementation cannot
	// report asynchronous device loss.
	Start(context.Context, string, int) (<-chan error, error)
	Stop(context.Context) (Result, error)
	Cancel(context.Context) error
	Close() error
}

// DeviceNamer is implemented by capture backends that can report the device
// they already resolved while opening capture. It avoids enumerating devices
// again on the latency-sensitive recording path just to populate run metadata.
type DeviceNamer interface {
	DeviceName() string
}

// PreparableCapture may initialize a stopped native device before the first
// recording. Preparation must not start capture or emit PCM.
type PreparableCapture interface {
	Prepare(context.Context, string) error
}

// StreamCapture is implemented by capture backends that can hand PCM to a
// bounded, non-blocking sink instead of retaining an entire recording.
type StreamCapture interface {
	Capture
	StartStream(context.Context, string, int, PCMStreamSink) (<-chan error, error)
}

func MaxPCMBytes(seconds int) (int, error) {
	if seconds < 1 || seconds > MaxSingleRequestDurationSeconds {
		return 0, errors.New("duration must be between 1 and 262 seconds")
	}
	return seconds * SampleRate * 2, nil
}

func WAV(pcm []byte) ([]byte, error) {
	return PCM16WAV(pcm, SampleRate, 1)
}

// PCM16WAV wraps mono or stereo little-endian PCM16 in a canonical RIFF/WAVE
// container. It is shared by STT uploads and native TTS export so saved speech
// never depends on a provider's streaming or non-canonical WAV header.
func PCM16WAV(pcm []byte, sampleRate, channels uint32) ([]byte, error) {
	if sampleRate == 0 || (channels != 1 && channels != 2) {
		return nil, errors.New("PCM16 format is invalid")
	}
	if len(pcm)%2 != 0 {
		return nil, errors.New("PCM16 payload has odd length")
	}
	blockAlign := channels * 2
	if len(pcm)%int(blockAlign) != 0 {
		return nil, errors.New("PCM16 payload has an incomplete frame")
	}
	if uint64(len(pcm)) > uint64(^uint32(0))-36 {
		return nil, errors.New("PCM payload too large")
	}
	b := make([]byte, 44+len(pcm))
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+len(pcm)))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], uint16(channels))
	binary.LittleEndian.PutUint32(b[24:], sampleRate)
	binary.LittleEndian.PutUint32(b[28:], sampleRate*blockAlign)
	binary.LittleEndian.PutUint16(b[32:], uint16(blockAlign))
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(pcm)))
	copy(b[44:], pcm)
	return b, nil
}
