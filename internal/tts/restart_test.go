package tts

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type restartPlayer struct {
	playerFake
	rewindErr, playErr error
	plays              int
	block              string
	entered, release   chan struct{}
}

func (p *restartPlayer) gate(operation string) {
	if p.block == operation {
		close(p.entered)
		<-p.release
	}
}

func (p *restartPlayer) Rewind() error {
	p.gate("rewind")
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rewindErr != nil {
		return p.rewindErr
	}
	p.position, p.playing = 0, false
	return nil
}

func (p *restartPlayer) Play() error {
	p.gate("playback")
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plays++
	if p.playErr != nil {
		return p.playErr
	}
	p.playing = true
	return nil
}

func (p *restartPlayer) Position() (int64, int64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.position, p.duration, p.position >= p.duration
}

func prepareRestart(t *testing.T, phase Phase, player *restartPlayer) (*Service, *speechClientFake, <-chan Status, context.Context) {
	t.Helper()
	player.loaded, player.playing, player.position, player.duration = true, phase == Playing, 4000, 10000
	if phase == Completed {
		player.position = player.duration
	}
	service, client := testSpeechService(t, player)
	service.generation = 7
	service.status = Status{
		Generation: 7, Phase: phase, Source: SourceCompose,
		PositionMilliseconds: player.position, DurationMilliseconds: player.duration,
		CanPause: phase == Playing, CanResume: phase == Paused, CanStop: phase != Completed,
		CanRestart: true, CanSeek: true, CanSave: true, CanClear: true,
	}
	statuses := make(chan Status, 32)
	service.changed = func(status Status) {
		select {
		case statuses <- status:
		default:
		}
	}
	ctx, cancel := service.operationContext()
	service.operation = cancel
	return service, client, statuses, ctx
}

func awaitRestartPhase(t *testing.T, statuses <-chan Status, phase Phase) Status {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case status := <-statuses:
			if status.Phase == phase {
				return status
			}
		case <-timer.C:
			t.Fatalf("playback monitor did not publish %s", phase)
		}
	}
}

func TestRestartFailuresPreserveAudioAndObservation(t *testing.T) {
	for _, stage := range []string{"rewind", "playback"} {
		for _, phase := range []Phase{Playing, Paused, Completed} {
			t.Run(stage+"/"+string(phase), func(t *testing.T) {
				player := &restartPlayer{}
				privateError := errors.New("native failure at C:/private/audio/session.wav")
				if stage == "rewind" {
					player.rewindErr = privateError
				} else {
					player.playErr = privateError
				}
				service, client, statuses, oldContext := prepareRestart(t, phase, player)
				previous := service.CurrentStatus()
				logs := &speechLogBuffer{}
				service.logger = slog.New(slog.NewJSONHandler(logs, nil))
				if phase != Completed {
					service.workers.Go(func() { service.monitor(oldContext, 7) })
				}
				err := service.Restart()
				if err == nil || strings.Contains(err.Error(), "private") {
					t.Fatalf("restart did not return a safe failure: %v", err)
				}
				status := service.CurrentStatus()
				if !status.CanSave || !status.CanClear || !status.CanSeek || !status.CanRestart || status.Source != SourceCompose {
					t.Fatalf("retained audio controls lost: %+v", status)
				}
				if wav, err := player.Snapshot(); err != nil || len(wav) == 0 {
					t.Fatalf("restart failure discarded audio: %v", err)
				}
				if stage == "rewind" {
					if status != previous || oldContext.Err() != nil || player.plays != 0 {
						t.Fatalf("failed rewind replaced the original session: %+v, context=%v, plays=%d", status, oldContext.Err(), player.plays)
					}
				} else {
					if status.Generation != 8 || status.Phase != Paused || status.PositionMilliseconds != 0 || status.CanPause || !status.CanResume || !status.CanStop || oldContext.Err() == nil {
						t.Fatalf("failed start did not retain paused audio: %+v, old context=%v", status, oldContext.Err())
					}
				}
				failureLogged := false
				for _, record := range logs.records(t) {
					if record["msg"] == "speech restart failed" {
						failureLogged = record["stage"] == stage && record["error_kind"] != ""
					}
					for _, value := range record {
						if text, ok := value.(string); ok && strings.Contains(text, "private") {
							t.Fatalf("raw native error reached diagnostics: %v", record)
						}
					}
				}
				if !failureLogged {
					t.Fatal("restart failure lost its bounded diagnostic")
				}
				player.mu.Lock()
				player.rewindErr, player.playErr = nil, nil
				player.mu.Unlock()
				if status.Phase == Paused {
					if err := service.Resume(); err != nil {
						t.Fatal(err)
					}
				} else if status.Phase == Completed {
					if err := service.Restart(); err != nil {
						t.Fatal(err)
					}
				}
				player.mu.Lock()
				player.position = player.duration
				player.mu.Unlock()
				completed := awaitRestartPhase(t, statuses, Completed)
				if !completed.CanRestart || !completed.CanSave || completed.CanPause || completed.CanResume {
					t.Fatalf("recovered playback did not finish coherently: %+v", completed)
				}
				client.mu.Lock()
				defer client.mu.Unlock()
				if client.calls != 0 {
					t.Fatal("restart recovery synthesized replacement audio")
				}
			})
		}
	}
}

func TestRestartShutdownDuringNativeFailureCannotRevivePlayback(t *testing.T) {
	for _, stage := range []string{"rewind", "playback"} {
		t.Run(stage, func(t *testing.T) {
			player := &restartPlayer{block: stage, entered: make(chan struct{}), release: make(chan struct{})}
			player.rewindErr = errors.New("private native failure")
			if stage == "playback" {
				player.rewindErr, player.playErr = nil, player.rewindErr
			}
			service, _, _, oldContext := prepareRestart(t, Playing, player)
			unblock := sync.OnceFunc(func() { close(player.release) })
			t.Cleanup(unblock)
			var published atomic.Int32
			service.changed = func(Status) { published.Add(1) }
			service.workers.Go(func() { service.monitor(oldContext, 7) })
			done := make(chan error, 1)
			go func() { done <- service.Restart() }()
			awaitSignal(t, player.entered)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if err := service.shutdown(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("shutdown did not honor its deadline: %v", err)
			}
			before := published.Load()
			unblock()
			awaitShutdownError(t, done, context.Canceled)
			awaitSignal(t, service.shutdownDone)
			status := service.CurrentStatus()
			if status.Phase != Idle || status.CanResume || status.CanRestart || published.Load() != before {
				t.Fatalf("late restart revived playback: %+v, publications=%d -> %d", status, before, published.Load())
			}
			if stage == "rewind" && player.plays != 0 {
				t.Fatal("restart played after shutdown during rewind")
			}
		})
	}
}
