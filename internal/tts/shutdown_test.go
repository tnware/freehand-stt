package tts

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/settings"
)

type playerGate struct {
	playerFake
	block            string
	entered, release chan struct{}
	once             sync.Once
	closes, plays    atomic.Int32
	releaseOnce      sync.Once
}

func newPlayerGate(operation string) *playerGate {
	return &playerGate{block: operation, entered: make(chan struct{}), release: make(chan struct{})}
}
func (p *playerGate) unblock() { p.releaseOnce.Do(func() { close(p.release) }) }
func (p *playerGate) gate(operation string) {
	if p.block == operation {
		p.once.Do(func() { close(p.entered); <-p.release })
	}
}
func (p *playerGate) Load(data []byte, rate, channels uint32) error {
	p.gate("Load")
	return p.playerFake.Load(data, rate, channels)
}
func (p *playerGate) Play() error   { p.plays.Add(1); return p.playerFake.Play() }
func (p *playerGate) Pause() error  { p.gate("Pause"); return p.playerFake.Pause() }
func (p *playerGate) Stop() error   { p.gate("Stop"); return p.playerFake.Stop() }
func (p *playerGate) Unload() error { p.gate("Unload"); return p.playerFake.Unload() }
func (p *playerGate) Close() error  { p.gate("Close"); p.closes.Add(1); return nil }

func testSpeechService(t *testing.T, player Player) (*Service, *speechClientFake) {
	wav, _ := audio.WAV([]byte{1, 0, 2, 0})
	client := &speechClientFake{wav: wav}
	cfg := config.Default().TextToSpeech
	cfg.Enabled, cfg.BaseURL, cfg.Model, cfg.Voice = true, "https://fixture.invalid/v1", "speech", "voice"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		return settings.TextToSpeechProfile{Settings: cfg}, nil
	}, client, player, nil, nil, nil, nil, nil, nil)
	t.Cleanup(func() {
		if p, ok := player.(*playerGate); ok {
			p.unblock()
		}
		_ = service.ServiceShutdown()
	})
	return service, client
}
func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("operation did not reach expected boundary")
	}
}
func awaitShutdownError(t *testing.T, done <-chan error, want error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Fatalf("shutdown error = %v, want %v", err, want)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown ignored its cancelled wait context")
	}
}

func TestShutdownBoundsEveryNativeTeardownStep(t *testing.T) {
	for _, operation := range []string{"Stop", "Unload", "Close"} {
		t.Run(operation, func(t *testing.T) {
			player := newPlayerGate(operation)
			service, _ := testSpeechService(t, player)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			defer player.unblock()
			done := make(chan error, 1)
			go func() { done <- service.shutdown(ctx) }()
			awaitSignal(t, player.entered)
			awaitSignal(t, service.rootContext.Done())
			if err := service.SpeakText("Must not start"); err == nil {
				t.Fatal("shutdown admitted speech")
			}
			cancel()
			awaitShutdownError(t, done, context.Canceled)
			// No concurrent Close is allowed to free a blocked native call's resources.
			if player.closes.Load() != 0 {
				t.Fatal("native resources closed out of order")
			}
		})
	}
}

func TestShutdownDuringLoadCannotStartLatePlayback(t *testing.T) {
	player := newPlayerGate("Load")
	service, _ := testSpeechService(t, player)
	var published atomic.Int32
	service.changed = func(Status) { published.Add(1) }
	if err := service.SpeakText("Fixed synthetic text"); err != nil {
		t.Fatal(err)
	}
	awaitSignal(t, player.entered)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitSignal(t, service.rootContext.Done())
	before := published.Load()
	cancel()
	awaitShutdownError(t, done, context.Canceled)
	if err := service.Stop(); err == nil {
		t.Fatal("closed player accepted Stop")
	}
	player.unblock()
	awaitSignal(t, service.shutdownDone)
	if player.plays.Load() != 0 || player.closes.Load() != 1 || published.Load() != before {
		t.Fatalf("late playback/publication or wrong close count: plays=%d closes=%d publications=%d", player.plays.Load(), player.closes.Load(), published.Load())
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	if player.closes.Load() != 1 {
		t.Fatal("repeated shutdown closed the player twice")
	}
}

func TestShutdownCancelsBeforeAnAdmittedStartReleasesPlayerControl(t *testing.T) {
	player := newPlayerGate("Stop")
	service, client := testSpeechService(t, player)
	started := make(chan error, 1)
	go func() { started <- service.SpeakText("Fixed synthetic text") }()
	awaitSignal(t, player.entered)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitSignal(t, service.rootContext.Done())
	cancel()
	awaitShutdownError(t, done, context.Canceled)
	player.unblock()
	if err := <-started; err == nil {
		t.Fatal("late admitted start succeeded")
	}
	awaitSignal(t, service.shutdownDone)
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.calls != 0 {
		t.Fatal("late admitted start invoked inference")
	}
}

func TestShutdownDuringPauseKeepsCleanupSerialized(t *testing.T) {
	player := newPlayerGate("Pause")
	service, _ := testSpeechService(t, player)
	service.status = Status{Phase: Playing, Generation: 1}
	paused := make(chan error, 1)
	go func() { paused <- service.Pause() }()
	awaitSignal(t, player.entered)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitSignal(t, service.rootContext.Done())
	cancel()
	awaitShutdownError(t, done, context.Canceled)
	if player.closes.Load() != 0 {
		t.Fatal("closed a player still in use")
	}
	player.unblock()
	<-paused
	awaitSignal(t, service.shutdownDone)
	if status := service.CurrentStatus(); status.Phase != Idle || status.CanResume {
		t.Fatalf("late pause revived state: %+v", status)
	}
}

func TestAudioExportDoesNotHoldPlayerControlAndCannotReportLateSuccess(t *testing.T) {
	player := newPlayerGate("")
	player.loaded = true
	service, _ := testSpeechService(t, player)
	service.status = Status{Generation: 1, Phase: Completed, CanSave: true}
	service.saveFile = func() (string, error) { return "synthetic.wav", nil }
	entered, release := make(chan struct{}), make(chan struct{})
	var retained []byte
	service.writeAudio = func(ctx context.Context, _ string, wav []byte) error {
		retained = wav
		close(entered)
		<-release // Simulate an OS write that does not respond to cancellation.
		return nil
	}
	saved := make(chan error, 1)
	go func() {
		ok, err := service.SaveAudio()
		if ok {
			err = errors.New("reported late save success")
		}
		saved <- err
	}()
	awaitSignal(t, entered)
	stopped := make(chan error, 1)
	go func() { stopped <- service.Stop() }()
	awaitShutdownError(t, stopped, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitSignal(t, service.rootContext.Done())
	cancel()
	awaitShutdownError(t, done, context.Canceled)
	close(release)
	select {
	case err := <-saved:
		if err == nil || err.Error() != "speech audio save cancelled during shutdown" {
			t.Fatalf("export result = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("export did not release")
	}
	awaitSignal(t, service.shutdownDone)
	for _, b := range retained {
		if b != 0 {
			t.Fatal("export retained audio after completion")
		}
	}
	if player.closes.Load() != 1 {
		t.Fatal("player not closed")
	}
}
