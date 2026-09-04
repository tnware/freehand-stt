package audio

import (
	"errors"
	"sync"
	"sync/atomic"
)

const (
	VADFrameMilliseconds = 20
	VADFrameSamples      = SampleRate * VADFrameMilliseconds / 1000
	VADFrameBytes        = VADFrameSamples * 2
	streamFrameCapacity  = 128
)

var ErrStreamOverrun = errors.New("microphone processing could not keep up")

// PCMStreamSink is written only by the native audio callback. WritePCM must
// remain non-blocking because the callback is on miniaudio's realtime thread.
type PCMStreamSink interface {
	WritePCM([]byte) bool
	Close()
}

// FramePipe turns arbitrary callback buffers into fixed 20 ms PCM frames. All
// storage is allocated up front. The callback only copies bytes and performs
// non-blocking channel operations; VAD and networking run on ordinary Go
// goroutines downstream.
type FramePipe struct {
	free    chan []byte
	ready   chan []byte
	current []byte
	used    int
	closed  atomic.Bool
	once    sync.Once
}

func NewFramePipe() *FramePipe {
	p := &FramePipe{
		free:  make(chan []byte, streamFrameCapacity),
		ready: make(chan []byte, streamFrameCapacity),
	}
	for range streamFrameCapacity {
		p.free <- make([]byte, VADFrameBytes)
	}
	return p
}

func (p *FramePipe) Frames() <-chan []byte { return p.ready }

func (p *FramePipe) WritePCM(input []byte) bool {
	if p.closed.Load() {
		return false
	}
	for len(input) > 0 {
		if p.current == nil {
			select {
			case p.current = <-p.free:
			default:
				return false
			}
			p.used = 0
		}
		n := copy(p.current[p.used:], input)
		p.used += n
		input = input[n:]
		if p.used != len(p.current) {
			continue
		}
		select {
		case p.ready <- p.current:
			p.current = nil
			p.used = 0
		default:
			return false
		}
	}
	return true
}

// Release zeroes a consumed frame before returning it to the callback pool.
func (p *FramePipe) Release(frame []byte) {
	for i := range frame {
		frame[i] = 0
	}
	select {
	case p.free <- frame[:VADFrameBytes]:
	default:
		// Every released frame originated in this pool. A full free queue means
		// the pipe has already been closed and all storage has been returned.
	}
}

// Close is called only after the native device has stopped, so it cannot race
// WritePCM. A partial final frame is padded with silence for VAD while its real
// byte count remains unavailable; the padding is harmless because utterance
// boundaries already preserve trailing silence.
func (p *FramePipe) Close() {
	p.once.Do(func() {
		p.closed.Store(true)
		if p.current != nil && p.used > 0 {
			for i := p.used; i < len(p.current); i++ {
				p.current[i] = 0
			}
			p.ready <- p.current
			p.current = nil
			p.used = 0
		}
		close(p.ready)
	})
}
