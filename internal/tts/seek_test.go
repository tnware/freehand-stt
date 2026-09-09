package tts

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

type seekPlayer struct {
	playerFake
	seeks            int
	seekErr, playErr error
}

func (p *seekPlayer) SeekTo(position int64) error {
	p.seeks++
	if p.seekErr != nil {
		return p.seekErr
	}
	return p.playerFake.SeekTo(position)
}
func (p *seekPlayer) Play() error {
	if p.playErr != nil {
		return p.playErr
	}
	return p.playerFake.Play()
}
func (p *seekPlayer) Position() (int64, int64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.position, p.duration, false
}
func preparedSeekService(t *testing.T, phase Phase) (*Service, *seekPlayer, *speechClientFake) {
	t.Helper()
	player := &seekPlayer{playerFake: playerFake{loaded: true, duration: 10000, position: 4000, playing: phase == Playing}}
	service, client := testSpeechService(t, player)
	service.generation = 7
	service.status = Status{Generation: 7, Phase: phase, Source: SourceCompose, DurationMilliseconds: 10000, PositionMilliseconds: 4000, CanSeek: true, CanRestart: true, CanSave: true, CanClear: true}
	return service, player, client
}
func TestSeekReusesAudioAndPreservesPlaybackIntent(t *testing.T) {
	for _, phase := range []Phase{Playing, Paused, Completed} {
		t.Run(string(phase), func(t *testing.T) {
			service, player, client := preparedSeekService(t, phase)
			status, err := service.Seek(SeekRequest{Generation: 7, PositionMilliseconds: 1500})
			if err != nil {
				t.Fatal(err)
			}
			want := phase
			if phase == Completed {
				want = Paused
			}
			if status.Phase != want || status.PositionMilliseconds != 1500 || status.Generation != 7 || !status.CanSeek || !status.CanSave || !status.CanRestart {
				t.Fatalf("unexpected sought status: %+v", status)
			}
			player.mu.Lock()
			playing := player.playing
			player.mu.Unlock()
			if playing != (want == Playing) || status.CanPause != (want == Playing) || status.CanResume != (want == Paused) {
				t.Fatal("seek changed playback intent")
			}
			if client.calls != 0 || player.seeks != 1 {
				t.Fatal("seek must only reposition retained audio")
			}
			service.finish(6, Completed, "stale completion", "")
			if service.CurrentStatus().Phase != want {
				t.Fatal("stale completion replaced seek")
			}
		})
	}
}
func TestSeekRejectsStaleInvalidOrUnavailableRequestsWithoutTouchingAudio(t *testing.T) {
	for _, request := range []SeekRequest{{6, 500}, {7, -1}, {7, 10001}, {7, math.MaxInt64}} {
		service, player, _ := preparedSeekService(t, Paused)
		if _, err := service.Seek(request); err == nil {
			t.Fatalf("accepted %+v", request)
		}
		if player.seeks != 0 || service.CurrentStatus().PositionMilliseconds != 4000 {
			t.Fatal("invalid seek touched playback")
		}
	}
	for _, phase := range []Phase{Idle, Generating, Failed, Cancelled} {
		service, player, _ := preparedSeekService(t, phase)
		if _, err := service.Seek(SeekRequest{7, 500}); err == nil || player.seeks != 0 {
			t.Fatalf("accepted phase %s", phase)
		}
	}
}
func TestSeekEndCanBeRepositionedAndResumed(t *testing.T) {
	service, _, _ := preparedSeekService(t, Playing)
	end, err := service.Seek(SeekRequest{7, 10000})
	if err != nil || end.Phase != Completed || end.CanResume || end.CanPause || end.CanStop {
		t.Fatalf("end = %+v, %v", end, err)
	}
	middle, err := service.Seek(SeekRequest{7, 1000})
	if err != nil || middle.Phase != Paused || !middle.CanResume {
		t.Fatalf("middle = %+v, %v", middle, err)
	}
	if err := service.Resume(); err != nil {
		t.Fatal(err)
	}
	if service.CurrentStatus().Phase != Playing {
		t.Fatal("could not resume from sought position")
	}
}
func TestSeekFailuresKeepRetainedAudioRecoverable(t *testing.T) {
	service, player, _ := preparedSeekService(t, Playing)
	player.seekErr = errors.New("native stop failed")
	if _, err := service.Seek(SeekRequest{7, 500}); err == nil {
		t.Fatal("ignored seek failure")
	}
	if service.CurrentStatus().PositionMilliseconds != 4000 {
		t.Fatal("failed seek changed position")
	}
	player.seekErr = nil
	player.playErr = errors.New("native start failed")
	if _, err := service.Seek(SeekRequest{7, 500}); err == nil {
		t.Fatal("ignored resume failure")
	}
	if status := service.CurrentStatus(); status.Phase != Paused || !status.CanSave || !status.CanResume {
		t.Fatalf("audio not recoverable: %+v", status)
	}
}
func TestSeekCannotResumeAfterShutdownDuringNativeStop(t *testing.T) {
	player := newPlayerGate("SeekTo")
	service, _ := testSpeechService(t, player)
	service.generation = 1
	service.status = Status{Generation: 1, Phase: Playing, CanSeek: true, DurationMilliseconds: 1000}
	done := make(chan error, 1)
	go func() { _, err := service.Seek(SeekRequest{1, 500}); done <- err }()
	awaitSignal(t, player.entered)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := service.shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown: %v", err)
	}
	player.unblock()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("seek: %v", err)
	}
	if player.plays.Load() != 0 {
		t.Fatal("seek resumed after shutdown")
	}
}
