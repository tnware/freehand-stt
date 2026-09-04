package tts

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

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
func (p *playerFake) Restart() error {
	p.mu.Lock()
	p.position = 0
	p.playing = true
	p.mu.Unlock()
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
func (p *playerFake) Save(path string) error {
	p.mu.Lock()
	p.saved = path
	p.mu.Unlock()
	return nil
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
	service := NewService(func() (settings.TextToSpeechProfile, error) { return profile, nil }, client, player, store, nil, nil, nil, func(status Status) { statuses <- status }, nil)
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
	service := NewService(func() (settings.TextToSpeechProfile, error) { return settings.TextToSpeechProfile{}, context.Canceled }, client, &playerFake{}, nil, nil, nil, nil, nil, nil)
	if err := service.PreviewVoice(); err == nil {
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
	service := NewService(func() (settings.TextToSpeechProfile, error) { return profile, nil }, client, &playerFake{}, store, nil, nil, nil, nil, nil)
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

	tooLong := strings.Repeat("a", config.MaxTTSInputCharacters+1)
	if err := service.SpeakText(tooLong); err == nil {
		t.Fatal("expected oversized speech input to fail")
	}
}

func TestActiveCaptureRejectsPlaybackBeforeProfileOrInference(t *testing.T) {
	client := &speechClientFake{}
	profileCalls := 0
	service := NewService(func() (settings.TextToSpeechProfile, error) {
		profileCalls++
		return settings.TextToSpeechProfile{}, nil
	}, client, &playerFake{}, nil, nil, nil, func() bool { return true }, nil, nil)
	if err := service.PreviewVoice(); err == nil {
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
	service := NewService(func() (settings.TextToSpeechProfile, error) { return profile, nil }, &speechClientFake{wav: wav}, &playerFake{}, nil, nil, nil, nil, nil, nil)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.PreviewVoice(); err != nil {
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
		func() (settings.TextToSpeechProfile, error) { return profile, nil },
		&speechClientFake{wav: wav}, player, nil, nil,
		func() (string, error) { return `C:\chosen\speech.wav`, nil },
		nil, nil, nil,
	)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.PreviewVoice(); err != nil {
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
