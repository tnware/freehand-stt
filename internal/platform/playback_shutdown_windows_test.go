//go:build windows

package platform

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/tnware/freehand-stt/internal/tts"
)

type restartDevice struct {
	armed                 atomic.Bool
	starts, closes        atomic.Int32
	entered, release      chan struct{}
	stopOnce, releaseOnce sync.Once
}

func (d *restartDevice) Start() error { d.starts.Add(1); return nil }
func (d *restartDevice) Stop() error {
	if d.armed.Load() {
		d.stopOnce.Do(func() { close(d.entered); <-d.release })
	}
	return nil
}
func (d *restartDevice) Uninit()  { d.closes.Add(1) }
func (d *restartDevice) unblock() { d.releaseOnce.Do(func() { close(d.release) }) }

// Only device construction is replaced. Rewind/Play/Stop/Unload/Close below
// run the actual Windows adapter under the actual speech service's control.
type restartPlayback struct {
	Playback
	device *restartDevice
}

func (p *restartPlayback) Load(data []byte, rate, channels uint32) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dev, p.data, p.rate, p.channels = p.device, append([]byte(nil), data...), rate, channels
	return nil
}

type restartSpeechClient struct{}

func (restartSpeechClient) SynthesizeSpeech(context.Context, string, string, inference.SpeechRequest) ([]byte, error) {
	return audio.WAV(make([]byte, 16000*2))
}

func TestRestartCannotStartDeviceAfterShutdownWhileNativeStopWasBlocked(t *testing.T) {
	device := &restartDevice{entered: make(chan struct{}), release: make(chan struct{})}
	player := &restartPlayback{device: device}
	cfg := config.Default().TextToSpeech
	cfg.Enabled, cfg.BaseURL, cfg.Model, cfg.Voice = true, "https://fixture.invalid/v1", "speech", "voice"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	playing := make(chan struct{}, 1)
	service := tts.NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		return settings.TextToSpeechProfile{Settings: cfg}, nil
	}, restartSpeechClient{}, player, nil, nil, nil, nil, func(status tts.Status) {
		if status.Phase == tts.Playing {
			select {
			case playing <- struct{}{}:
			default:
			}
		}
	}, nil)
	t.Cleanup(func() { device.unblock(); _ = service.ServiceShutdown() })
	if err := service.SpeakText("Synthetic fixture"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-playing:
	case <-time.After(time.Second):
		t.Fatal("initial playback did not start")
	}
	if device.starts.Load() != 1 {
		t.Fatal("initial playback did not reach native Start")
	}
	device.armed.Store(true)
	restarted := make(chan error, 1)
	go func() { restarted <- service.Restart() }()
	select {
	case <-device.entered:
	case <-time.After(time.Second):
		t.Fatal("Restart did not enter native Stop")
	}
	shutdown := make(chan error, 1)
	go func() { shutdown <- service.ServiceShutdown() }()
	// Expiry proves shutdown has closed admission and cancelled its root while
	// cleanup still waits for Restart's player-control lock.
	select {
	case err := <-shutdown:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not honor its deadline")
	}
	if device.closes.Load() != 0 {
		t.Fatal("device freed while native Stop was blocked")
	}
	device.unblock()
	select {
	case err := <-restarted:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Restart = %v, want cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Restart did not finish after native Stop")
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	if device.starts.Load() != 1 {
		t.Fatalf("Restart started the device after shutdown: starts=%d", device.starts.Load())
	}
	if device.closes.Load() != 1 {
		t.Fatalf("device closed %d times, want once", device.closes.Load())
	}
	select {
	case <-playing:
		t.Fatal("Restart published playback after shutdown")
	default:
	}
}

func TestPlaybackRewindRequiresExplicitPlay(t *testing.T) {
	device := &restartDevice{}
	player := &Playback{dev: device, position: 32, elapsed: time.Second, playing: true}
	if err := player.Rewind(); err != nil {
		t.Fatal(err)
	}
	if device.starts.Load() != 0 || player.position != 0 || player.playing || player.elapsed != 0 {
		t.Fatal("rewind did not stop/reset without starting")
	}
	if err := player.Play(); err != nil {
		t.Fatal(err)
	}
	if device.starts.Load() != 1 {
		t.Fatal("explicit Play did not start the rewound device")
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}
}
