//go:build windows

package platform

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/tnware/freehand-stt/internal/audio"
)

// miniaudio can submit the final PCM frames to WASAPI before the hardware has
// rendered them. Keep the device alive briefly after the audible duration so
// short previews do not get stopped while their only buffer is still queued.
const playbackDrainGrace = 100 * time.Millisecond

type Playback struct {
	mu       sync.Mutex
	ctx      *malgo.AllocatedContext
	dev      *malgo.Device
	data     []byte
	position int
	channels uint32
	rate     uint32
	output   string
	elapsed  time.Duration
	started  time.Time
	playing  bool
	closed   bool
}

func (p *Playback) Load(data []byte, sampleRate, channels uint32) error {
	p.release()
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errors.New("audio playback is closed")
	}
	p.mu.Unlock()
	ctx, err := newContext()
	if err != nil {
		return err
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = ctx.Uninit()
		ctx.Free()
		return errors.New("audio playback is closed")
	}
	p.ctx = ctx
	p.data = append([]byte(nil), data...)
	p.rate, p.channels = sampleRate, channels
	p.output = defaultPlaybackName(ctx)
	p.mu.Unlock()
	cfg := malgo.DefaultDeviceConfig(malgo.Playback)
	cfg.SampleRate = sampleRate
	cfg.Playback.Format = malgo.FormatS16
	cfg.Playback.Channels = channels
	cfg.Playback.ShareMode = malgo.Shared
	dev, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{Data: p.render})
	if err != nil {
		p.release()
		return err
	}
	p.mu.Lock()
	p.dev = dev
	p.mu.Unlock()
	return nil
}

func (p *Playback) render(output, _ []byte, _ uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.position >= len(p.data) {
		clear(output)
		return
	}
	n := copy(output, p.data[p.position:])
	clear(output[n:])
	p.position += n
}

func (p *Playback) Play() error {
	p.mu.Lock()
	dev := p.dev
	playing := p.playing
	p.mu.Unlock()
	if dev == nil {
		return errors.New("no speech audio is loaded")
	}
	if playing {
		return nil
	}
	if err := dev.Start(); err != nil {
		return err
	}
	p.mu.Lock()
	if p.dev == dev && !p.playing {
		p.started = time.Now()
		p.playing = true
	}
	p.mu.Unlock()
	return nil
}

func (p *Playback) Pause() error {
	p.mu.Lock()
	dev := p.dev
	p.mu.Unlock()
	if dev == nil {
		return errors.New("no speech audio is loaded")
	}
	if err := dev.Stop(); err != nil {
		return err
	}
	p.mu.Lock()
	p.pauseClock(time.Now())
	p.mu.Unlock()
	return nil
}

func (p *Playback) Restart() error {
	p.mu.Lock()
	dev := p.dev
	p.mu.Unlock()
	if dev == nil {
		return errors.New("no speech audio is loaded")
	}
	_ = dev.Stop()
	p.mu.Lock()
	p.position = 0
	p.elapsed = 0
	p.started = time.Time{}
	p.playing = false
	p.mu.Unlock()
	return p.Play()
}

func (p *Playback) Position() (int64, int64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	frameBytes := int(p.channels) * 2
	if frameBytes == 0 || p.rate == 0 {
		return 0, 0, false
	}
	durationValue := time.Duration(len(p.data)/frameBytes) * time.Second / time.Duration(p.rate)
	played := p.elapsed
	if p.playing && !p.started.IsZero() {
		played += time.Since(p.started)
	}
	positionValue := min(played, durationValue)
	return positionValue.Milliseconds(), durationValue.Milliseconds(), p.position >= len(p.data) && played >= durationValue+playbackDrainGrace
}

// OutputName identifies the Windows default playback endpoint selected when
// the current device was created. Windows may still apply a per-app output
// override, but this makes the native selection visible in diagnostics.
func (p *Playback) OutputName() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.output == "" {
		return "System default"
	}
	return p.output
}

// Save writes a canonical WAV snapshot of the current in-memory session. The
// caller supplies a path chosen by the native save dialog; no temporary file
// or WebView audio payload is involved.
func (p *Playback) Save(path string) error {
	p.mu.Lock()
	if p.dev == nil || len(p.data) == 0 {
		p.mu.Unlock()
		return errors.New("no speech audio is loaded")
	}
	data := append([]byte(nil), p.data...)
	rate, channels := p.rate, p.channels
	p.mu.Unlock()
	defer clear(data)

	wav, err := audio.PCM16WAV(data, rate, channels)
	if err != nil {
		return err
	}
	defer clear(wav)
	return os.WriteFile(path, wav, 0o666)
}

func (p *Playback) Stop() error {
	p.mu.Lock()
	dev := p.dev
	p.mu.Unlock()
	if dev != nil {
		_ = dev.Stop()
	}
	p.mu.Lock()
	p.position = 0
	p.elapsed = 0
	p.started = time.Time{}
	p.playing = false
	p.mu.Unlock()
	return nil
}

func (p *Playback) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()
	p.release()
	return nil
}

func (p *Playback) Unload() error {
	p.release()
	return nil
}

func (p *Playback) release() {
	p.mu.Lock()
	dev, ctx := p.dev, p.ctx
	p.dev, p.ctx = nil, nil
	clear(p.data)
	p.data = nil
	p.position = 0
	p.rate, p.channels = 0, 0
	p.output = ""
	p.elapsed = 0
	p.started = time.Time{}
	p.playing = false
	p.mu.Unlock()
	if dev != nil {
		_ = dev.Stop()
		dev.Uninit()
	}
	if ctx != nil {
		_ = ctx.Uninit()
		ctx.Free()
	}
}

func (p *Playback) pauseClock(now time.Time) {
	if !p.playing {
		return
	}
	if !p.started.IsZero() {
		p.elapsed += now.Sub(p.started)
	}
	p.started = time.Time{}
	p.playing = false
}

func defaultPlaybackName(ctx *malgo.AllocatedContext) string {
	devices, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return "System default"
	}
	for index := range devices {
		if devices[index].IsDefault != 0 {
			if name := devices[index].Name(); name != "" {
				return name
			}
		}
	}
	return "System default"
}
