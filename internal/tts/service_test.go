package tts

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type speechClientFake struct {
	mu      sync.Mutex
	calls   int
	request inference.SpeechRequest
	wav     []byte
}

func (f *speechClientFake) SynthesizeSpeech(_ context.Context, _, _ string, request inference.SpeechRequest) ([]byte, error) {
	f.mu.Lock()
	f.calls++
	f.request = request
	f.mu.Unlock()
	return append([]byte(nil), f.wav...), nil
}

type playerFake struct {
	mu       sync.Mutex
	loaded   bool
	playing  bool
	position int64
	duration int64
	saved    string
}

func (p *playerFake) Load([]byte, uint32, uint32) error {
	p.mu.Lock()
	p.loaded = true
	p.duration = 1000
	p.mu.Unlock()
	return nil
}
func (p *playerFake) Play() error  { p.mu.Lock(); p.playing = true; p.mu.Unlock(); return nil }
func (p *playerFake) Pause() error { p.mu.Lock(); p.playing = false; p.mu.Unlock(); return nil }
func (p *playerFake) Rewind() error {
	p.mu.Lock()
	p.position = 0
	p.playing = false
	p.mu.Unlock()
	return nil
}
func (p *playerFake) Seek(position int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.position = position
	p.playing = false
	return nil
}
func (p *playerFake) Position() (int64, int64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.playing {
		p.position = p.duration
	}
	return p.position, p.duration, p.position >= p.duration
}
func (p *playerFake) OutputName() string { return "Test speakers" }
func (p *playerFake) Snapshot() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.loaded {
		return nil, errors.New("no audio loaded")
	}
	return audio.WAV([]byte{1, 0, 2, 0})
}

func (p *playerFake) Stop() error {
	p.mu.Lock()
	p.playing = false
	p.position = 0
	p.mu.Unlock()
	return nil
}
func (p *playerFake) Unload() error {
	p.mu.Lock()
	p.loaded = false
	p.mu.Unlock()
	return nil
}
func (p *playerFake) Close() error { return nil }

func TestPlayHistoryEntryUsesBackendOwnedTranscript(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	client := &speechClientFake{wav: wav}
	player := &playerFake{}
	store := history.NewStore(true, nil)
	id := store.Begin("backend transcript", history.HistoryInserted, false, time.Now(), history.HistoryRunDetails{})
	profile := settings.TextToSpeechProfile{Settings: config.TextToSpeechSettings{Enabled: true, BaseURL: "https://example.test/v1", AuthenticationMode: config.AuthenticationModeNone, Model: "tts-model", Voice: "voice", Speed: 1, TimeoutSeconds: config.DefaultTextToSpeechTimeoutSeconds}}
	statuses := make(chan Status, 8)
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) { return profile, nil }, client, player, store, nil, nil, nil, func(status Status) { statuses <- status }, nil)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.PlayHistoryEntry(id, history.HistoryTextFinal); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case status := <-statuses:
			if status.Phase == Completed {
				client.mu.Lock()
				request := client.request
				calls := client.calls
				client.mu.Unlock()
				if calls != 1 || request.Input != "backend transcript" {
					t.Fatalf("calls=%d request=%#v", calls, request)
				}
				return
			}
		case <-deadline:
			t.Fatal("playback did not complete")
		}
	}
}

func TestDisabledSpeechPlaybackNeverInvokesInference(t *testing.T) {
	client := &speechClientFake{}
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		return settings.TextToSpeechProfile{}, context.Canceled
	}, client, &playerFake{}, nil, nil, nil, nil, nil, nil)
	if err := service.PreviewVoice(nil); err == nil {
		t.Fatal("expected disabled profile to fail")
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.calls != 0 {
		t.Fatalf("inference calls = %d", client.calls)
	}
}

func TestSpeakTextUsesBoundedUserInputWithoutWritingHistory(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	client := &speechClientFake{wav: wav}
	store := history.NewStore(true, nil)
	profile := settings.TextToSpeechProfile{Settings: config.TextToSpeechSettings{
		Enabled: true, BaseURL: "https://example.test/v1",
		AuthenticationMode: config.AuthenticationModeNone,
		Model:              "tts-model", Voice: "voice", Speed: 1,
		TimeoutSeconds: config.DefaultTextToSpeechTimeoutSeconds,
	}}
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) { return profile, nil }, client, &playerFake{}, store, nil, nil, nil, nil, nil)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.ServiceShutdown() }()

	if err := service.SpeakText("  first-class speech  "); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for service.CurrentStatus().Phase != Completed && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	client.mu.Lock()
	request := client.request
	client.mu.Unlock()
	if request.Input != "first-class speech" || service.CurrentStatus().Source != SourceCompose {
		t.Fatalf("request=%#v status=%+v", request, service.CurrentStatus())
	}
	if entries := store.Entries(); len(entries) != 0 {
		t.Fatalf("TTS composer wrote transcript history: %#v", entries)
	}

	unicodeText := strings.Repeat("😀", config.MaxTTSInputCharacters)
	if err := service.SpeakText(unicodeText); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for service.CurrentStatus().Phase != Completed && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	client.mu.Lock()
	received := client.request.Input
	client.mu.Unlock()
	if received != unicodeText {
		t.Fatal("Unicode input was truncated")
	}
	if err := service.SpeakText(unicodeText + "😀"); err == nil {
		t.Fatal("oversized Unicode accepted")
	}
	tooLong := strings.Repeat("a", config.MaxTTSInputCharacters+1)
	if err := service.SpeakText(tooLong); err == nil {
		t.Fatal("expected oversized speech input to fail")
	}
}

func TestActiveCaptureRejectsPlaybackBeforeProfileOrInference(t *testing.T) {
	client := &speechClientFake{}
	profileCalls := 0
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		profileCalls++
		return settings.TextToSpeechProfile{}, nil
	}, client, &playerFake{}, nil, nil, nil, activity.New(activity.Sources{DictationActive: func() bool { return true }}), nil, nil)
	if err := service.PreviewVoice(nil); err == nil {
		t.Fatal("expected active capture to reject playback")
	}
	if profileCalls != 0 || client.calls != 0 {
		t.Fatalf("profile calls=%d inference calls=%d", profileCalls, client.calls)
	}
}

func TestStopReleasesCompletedSessionAndRemovesRestart(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	profile := settings.TextToSpeechProfile{Settings: config.TextToSpeechSettings{
		Enabled: true, BaseURL: "https://example.test/v1",
		AuthenticationMode: config.AuthenticationModeNone,
		Model:              "tts-model", Voice: "voice", Speed: 1,
		TimeoutSeconds: config.DefaultTextToSpeechTimeoutSeconds,
	}}
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) { return profile, nil }, &speechClientFake{wav: wav}, &playerFake{}, nil, nil, nil, nil, nil, nil)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.PreviewVoice(nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for service.CurrentStatus().Phase != Completed && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if status := service.CurrentStatus(); status.Phase != Completed || !status.CanRestart {
		t.Fatalf("completed status = %+v", status)
	}
	if err := service.Stop(); err != nil {
		t.Fatal(err)
	}
	if status := service.CurrentStatus(); status.Phase != Cancelled || status.CanRestart {
		t.Fatalf("stopped status = %+v", status)
	}
}

func TestCompletedSpeechCanBeSavedThenExplicitlyCleared(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	profile := settings.TextToSpeechProfile{Settings: config.TextToSpeechSettings{
		Enabled: true, BaseURL: "https://example.test/v1",
		AuthenticationMode: config.AuthenticationModeNone,
		Model:              "tts-model", Voice: "voice", Speed: 1,
		TimeoutSeconds: config.DefaultTextToSpeechTimeoutSeconds,
	}}
	player := &playerFake{}
	service := NewService(
		func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) { return profile, nil },
		&speechClientFake{wav: wav}, player, nil, nil,
		func() (string, error) { return `C:\chosen\speech.wav`, nil },
		nil, nil, nil,
	)
	service.writeAudio = func(_ context.Context, path string, _ []byte) error {
		player.mu.Lock()
		defer player.mu.Unlock()
		player.saved = path
		return nil
	}
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.PreviewVoice(nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for service.CurrentStatus().Phase != Completed && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if status := service.CurrentStatus(); !status.CanSave || !status.CanClear {
		t.Fatalf("completed audio actions unavailable: %+v", status)
	}
	saved, err := service.SaveAudio()
	if err != nil || !saved {
		t.Fatalf("save result = %v, %v", saved, err)
	}
	player.mu.Lock()
	savedPath := player.saved
	player.mu.Unlock()
	if savedPath != `C:\chosen\speech.wav` {
		t.Fatalf("saved path = %q", savedPath)
	}
	if err := service.ClearAudio(); err != nil {
		t.Fatal(err)
	}
	if status := service.CurrentStatus(); status.Phase != Idle || status.CanSave || status.CanClear {
		t.Fatalf("cleared status = %+v", status)
	}
	player.mu.Lock()
	loaded := player.loaded
	player.mu.Unlock()
	if loaded {
		t.Fatal("player retained audio after clear")
	}
}

func TestStaleCompletionCannotPauseReplacement(t *testing.T) {
	player := &playerFake{playing: true, loaded: true}
	service := NewService(nil, nil, player, nil, nil, nil, nil, nil, nil)
	service.generation = 2
	service.status = Status{Generation: 2, Phase: Playing}
	service.finish(1, Completed, "old completion", "")
	player.mu.Lock()
	playing := player.playing
	player.mu.Unlock()
	if !playing || service.CurrentStatus().Phase != Playing {
		t.Fatal("stale completion altered replacement")
	}
}

func TestStopFencesDelayedGeneration(t *testing.T) {
	player := &playerFake{}
	service := NewService(nil, nil, player, nil, nil, nil, nil, nil, nil)
	service.generation = 1
	service.status = Status{Generation: 1, Phase: Generating, CanStop: true}
	if err := service.Stop(); err != nil {
		t.Fatal(err)
	}
	service.update(1, Status{Generation: 1, Phase: Playing})
	if service.CurrentStatus().Phase != Cancelled {
		t.Fatal("delayed generation revived stopped playback")
	}
}

func TestSaveDialogDoesNotBlockStopOrSaveReplacement(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	player := &playerFake{loaded: true}
	service := NewService(nil, nil, player, nil, nil, func() (string, error) {
		close(entered)
		<-release
		return "chosen.wav", nil
	}, nil, nil, nil)
	service.generation = 1
	service.status = Status{Generation: 1, Phase: Playing, CanSave: true}
	saved := make(chan error, 1)
	go func() { _, err := service.SaveAudio(); saved <- err }()
	<-entered
	stopped := make(chan error, 1)
	go func() { stopped <- service.Stop() }()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("dialog blocked Stop")
	}
	close(release)
	if err := <-saved; err == nil {
		t.Fatal("saved a changed session")
	}
	if player.saved != "" {
		t.Fatal("stale save reached player")
	}
}

func TestSaveDialogDoesNotBlockShutdown(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	service := NewService(nil, nil, &playerFake{}, nil, nil, func() (string, error) {
		close(entered)
		<-release
		return "chosen.wav", nil
	}, nil, nil, nil)
	service.status = Status{Generation: 1, Phase: Completed, CanSave: true}
	saved := make(chan error, 1)
	go func() { _, err := service.SaveAudio(); saved <- err }()
	<-entered
	done := make(chan error, 1)
	go func() { done <- service.ServiceShutdown() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("dialog blocked shutdown")
	}
	close(release)
	if err := <-saved; err == nil {
		t.Fatal("save after shutdown accepted")
	}
}

type cancellationIgnoringSpeechClient struct{ entered, release chan struct{} }

func (c *cancellationIgnoringSpeechClient) SynthesizeSpeech(context.Context, string, string, inference.SpeechRequest) ([]byte, error) {
	close(c.entered)
	<-c.release
	return nil, context.Canceled
}
func TestShutdownBoundsUncooperativeInferenceWorker(t *testing.T) {
	client := &cancellationIgnoringSpeechClient{make(chan struct{}), make(chan struct{})}
	cfg := config.Default().TextToSpeech
	cfg.Enabled, cfg.BaseURL, cfg.Model, cfg.Voice = true, "https://example.test/v1", "speech-model", "voice"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		return settings.TextToSpeechProfile{Settings: cfg}, nil
	}, client, &playerFake{}, nil, nil, nil, nil, nil, nil)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := service.SpeakText("Fixed test text"); err != nil {
		t.Fatal(err)
	}
	<-client.entered
	done := make(chan error, 1)
	go func() { done <- service.ServiceShutdown() }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown error = %v", err)
		}
	case <-time.After(shutdownTimeout + time.Second):
		close(client.release)
		t.Fatal("uncooperative worker prevented shutdown")
	}
	close(client.release)
	service.workers.Wait()
	if !service.closed.Load() {
		t.Fatal("service remained open")
	}
}

func TestRestartPlaysTheRetainedSessionAfterRewind(t *testing.T) {
	player := newPlayerGate("")
	player.loaded, player.position = true, 1000
	service, _ := testSpeechService(t, player)
	service.generation = 1
	service.status = Status{Generation: 1, Phase: Completed, CanRestart: true}
	if err := service.Restart(); err != nil {
		t.Fatal(err)
	}
	if player.plays.Load() != 1 {
		t.Fatal("Restart did not explicitly start playback after rewind")
	}
	if service.CurrentStatus().Generation != 2 {
		t.Fatal("Restart did not advance the generation")
	}
}

func TestPreviewVoiceForwardsDraftIntoTheCapturedPlaybackRequest(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	client := &speechClientFake{wav: wav}
	draft := &settings.TextToSpeechPreview{Enabled: true, ConnectionID: "speech", Model: "unsaved-model", Voice: "unsaved-voice", Speed: 1.5, TimeoutSeconds: 30}
	service := NewService(func(options *settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		if options == nil || options.ConnectionID != "speech" {
			t.Fatal("missing preview draft")
		}
		cfg := config.Default().TextToSpeech
		cfg.Enabled, cfg.BaseURL, cfg.AuthenticationMode = options.Enabled, "https://fixture.test/v1", config.AuthenticationModeNone
		cfg.Model, cfg.Voice, cfg.Speed, cfg.TimeoutSeconds = options.Model, options.Voice, options.Speed, options.TimeoutSeconds
		return settings.TextToSpeechProfile{Settings: cfg}, nil
	}, client, &playerFake{}, nil, nil, nil, nil, nil, nil)
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.PreviewVoice(draft); err != nil {
		t.Fatal(err)
	}
	draft.Model, draft.Voice, draft.Speed = "later-model", "later-voice", 2
	deadline := time.Now().Add(time.Second)
	for service.CurrentStatus().Phase != Completed && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.calls != 1 || client.request.Model != "unsaved-model" || client.request.Voice != "unsaved-voice" || client.request.Speed != 1.5 {
		t.Fatal("playback did not retain the preview snapshot")
	}
	if service.CurrentStatus().Source != SourcePreview {
		t.Fatal("preview used another workflow")
	}
}

func TestPlayVoiceTranscriptWithoutHistory(t *testing.T) {
	wav, _ := audio.WAV([]byte{1, 0, 2, 0})
	client := &speechClientFake{wav: wav}
	store := history.NewStore(false, nil)
	profileCalls := 0
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		profileCalls++
		return settings.TextToSpeechProfile{Settings: config.TextToSpeechSettings{Enabled: true, BaseURL: "https://example.test/v1", AuthenticationMode: config.AuthenticationModeNone, Model: "tts-model", Voice: "voice", Speed: 1, TimeoutSeconds: 30}}, nil
	}, client, &playerFake{}, store, &TranscriptSources{Voice: func(generation uint64) (string, error) {
		if generation != 3 {
			return "", errors.New("stale result")
		}
		return "backend voice result", nil
	}}, nil, nil, nil, nil)
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.PlayVoiceTranscript(2); err == nil || profileCalls != 0 {
		t.Fatal("stale selection reached speech profile")
	}
	if err := service.PlayVoiceTranscript(3); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for service.CurrentStatus().Phase != Completed && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	client.mu.Lock()
	request, calls := client.request, client.calls
	client.mu.Unlock()
	status := service.CurrentStatus()
	if calls != 1 || request.Input != "backend voice result" || status.Source != SourceVoice || status.Phase != Completed {
		t.Fatalf("calls=%d request=%+v status=%+v", calls, request, status)
	}
	if len(store.Entries()) != 0 {
		t.Fatal("playback retained history")
	}
	service.texts.Voice = nil
	if err := service.PlayVoiceTranscript(3); err == nil {
		t.Fatal("missing source accepted")
	}
}
