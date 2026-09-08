package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
)

type speechLogBuffer struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *speechLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}

func (b *speechLogBuffer) records(t *testing.T) []map[string]any {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(b.Buffer.String()), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

type loggingSpeechClient func(context.Context) ([]byte, error)

func (f loggingSpeechClient) SynthesizeSpeech(ctx context.Context, _, _ string, _ inference.SpeechRequest) ([]byte, error) {
	return f(ctx)
}

func newLoggingSpeechService(t *testing.T, client SpeechClient, player Player) (*Service, *speechLogBuffer) {
	t.Helper()
	cfg := config.Default().TextToSpeech
	cfg.Enabled, cfg.BaseURL, cfg.Model, cfg.Voice = true, "https://fixture.invalid/private-path", "private-model", "private-voice"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	logs := &speechLogBuffer{}
	service := NewService(func(*settings.TextToSpeechPreview) (settings.TextToSpeechProfile, error) {
		return settings.TextToSpeechProfile{Settings: cfg, Credential: "private-credential"}, nil
	}, client, player, nil, nil, nil, nil, nil, slog.New(slog.NewJSONHandler(logs, nil)))
	t.Cleanup(func() {
		if err := service.ServiceShutdown(); err != nil {
			t.Error(err)
		}
	})
	return service, logs
}

func assertSpeechLifecycle(t *testing.T, logs *speechLogBuffer, lifecycle string, generation uint64, outcome string) {
	t.Helper()
	starts, terminals := 0, 0
	for _, record := range logs.records(t) {
		if record["generation"] != float64(generation) {
			continue
		}
		message, _ := record["msg"].(string)
		if message == lifecycle+" started" {
			starts++
		}
		if message == lifecycle+" completed" || message == lifecycle+" failed" || message == lifecycle+" cancelled" {
			terminals++
			if message != lifecycle+" "+outcome {
				t.Errorf("unexpected terminal: %v", record)
			}
			wantOutcome := outcome
			if lifecycle == "speech playback" && outcome == "completed" {
				wantOutcome = "played"
			}
			if record["outcome"] != wantOutcome {
				t.Errorf("terminal outcome = %v, want %s", record["outcome"], wantOutcome)
			}
			if outcome == "cancelled" && record["error_kind"] != "cancelled" {
				t.Errorf("cancelled terminal lost error_kind: %v", record)
			}
			if duration, ok := record["duration_ms"].(float64); !ok || duration < 0 {
				t.Errorf("missing/invalid duration_ms: %v", record)
			}
		}
	}
	if starts != 1 || terminals != 1 {
		t.Fatalf("%s generation %d: starts=%d terminals=%d; records=%v", lifecycle, generation, starts, terminals, logs.records(t))
	}
}

func TestSpeechLoggingGenerationCancellation(t *testing.T) {
	entered := make(chan struct{})
	client := loggingSpeechClient(func(ctx context.Context) ([]byte, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	service, logs := newLoggingSpeechService(t, client, &playerFake{})
	if err := service.SpeakText("private-transcript"); err != nil {
		t.Fatal(err)
	}
	awaitSignal(t, entered)
	generation := service.CurrentStatus().Generation
	if err := service.Stop(); err != nil {
		t.Fatal(err)
	}
	service.workers.Wait()
	assertSpeechLifecycle(t, logs, "speech generation", generation, "cancelled")
}

func TestSpeechLoggingDiscardsLateGeneration(t *testing.T) {
	for _, action := range []string{"replacement", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			wav, err := audio.WAV([]byte{1, 0, 2, 0})
			if err != nil {
				t.Fatal(err)
			}
			var calls int
			var mu sync.Mutex
			client := loggingSpeechClient(func(context.Context) ([]byte, error) {
				mu.Lock()
				calls++
				first := calls == 1
				mu.Unlock()
				if first {
					close(entered)
					<-release
				}
				return append([]byte(nil), wav...), nil
			})
			service, logs := newLoggingSpeechService(t, client, &playerFake{})
			defer unblock()
			if err := service.SpeakText("private-transcript"); err != nil {
				t.Fatal(err)
			}
			awaitSignal(t, entered)
			generation := service.CurrentStatus().Generation
			if action == "replacement" {
				if err := service.SpeakText("replacement"); err != nil {
					t.Fatal(err)
				}
				unblock()
				service.workers.Wait()
				assertSpeechLifecycle(t, logs, "speech generation", service.CurrentStatus().Generation, "completed")
			} else {
				done := make(chan error, 1)
				go func() { done <- service.ServiceShutdown() }()
				awaitSignal(t, service.rootContext.Done())
				unblock()
				if err := <-done; err != nil {
					t.Fatal(err)
				}
			}
			assertSpeechLifecycle(t, logs, "speech generation", generation, "cancelled")
		})
	}
}

func TestSpeechLoggingShutdownDuringNativeLoad(t *testing.T) {
	player := newPlayerGate("Load")
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	service, logs := newLoggingSpeechService(t, &speechClientFake{wav: wav}, player)
	defer player.unblock()
	if err := service.SpeakText("private-transcript"); err != nil {
		t.Fatal(err)
	}
	awaitSignal(t, player.entered)
	generation := service.CurrentStatus().Generation
	done := make(chan error, 1)
	go func() { done <- service.ServiceShutdown() }()
	awaitSignal(t, service.rootContext.Done())
	player.unblock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	assertSpeechLifecycle(t, logs, "speech generation", generation, "cancelled")
	for _, record := range logs.records(t) {
		if record["msg"] == "speech playback started" {
			t.Fatalf("shutdown started playback: %v", record)
		}
	}
}

type loggingPlayer struct {
	playerFake
	loadErr error
	playErr error
	hold    bool
}

func (*loggingPlayer) OutputName() string { return "private-output-device" }

func (p *loggingPlayer) Position() (int64, int64, bool) {
	if p.hold {
		return 0, 1000, false
	}
	return p.playerFake.Position()
}

func TestSpeechLoggingPlaybackCancellation(t *testing.T) {
	for _, action := range []string{"stop", "clear", "replacement", "restart", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			wav, err := audio.WAV([]byte{1, 0, 2, 0})
			if err != nil {
				t.Fatal(err)
			}
			service, logs := newLoggingSpeechService(t, &speechClientFake{wav: wav}, &loggingPlayer{hold: true})
			if err := service.SpeakText("private-transcript"); err != nil {
				t.Fatal(err)
			}
			generation := service.CurrentStatus().Generation
			awaitSpeechLog(t, logs, "speech playback started", generation)
			switch action {
			case "stop":
				err = service.Stop()
			case "clear":
				err = service.ClearAudio()
			case "replacement":
				err = service.SpeakText("replacement")
			case "restart":
				err = service.Restart()
			case "shutdown":
				err = service.ServiceShutdown()
			}
			if err != nil {
				t.Fatal(err)
			}
			if action == "replacement" || action == "restart" {
				awaitSpeechLog(t, logs, "speech playback cancelled", generation)
				awaitSpeechLog(t, logs, "speech playback started", generation+1)
				if err := service.ServiceShutdown(); err != nil {
					t.Fatal(err)
				}
			}
			service.workers.Wait()
			assertSpeechLifecycle(t, logs, "speech generation", generation, "completed")
			assertSpeechLifecycle(t, logs, "speech playback", generation, "cancelled")
			if action == "replacement" {
				assertSpeechLifecycle(t, logs, "speech generation", generation+1, "completed")
				assertSpeechLifecycle(t, logs, "speech playback", generation+1, "cancelled")
			}
			if action == "restart" {
				assertSpeechLifecycle(t, logs, "speech playback", generation+1, "cancelled")
				for _, record := range logs.records(t) {
					if record["generation"] == float64(generation+1) && record["msg"] == "speech generation started" {
						t.Errorf("restart synthesized speech: %v", record)
					}
				}
			}
		})
	}
}

func (p *loggingPlayer) Load(data []byte, rate, channels uint32) error {
	if p.loadErr != nil {
		return p.loadErr
	}
	return p.playerFake.Load(data, rate, channels)
}

func (p *loggingPlayer) Play() error {
	if p.playErr != nil {
		return p.playErr
	}
	return p.playerFake.Play()
}

func TestSpeechLoggingGenerationFailure(t *testing.T) {
	for _, stage := range []string{"inference", "decode", "load", "play"} {
		t.Run(stage, func(t *testing.T) {
			wav, err := audio.WAV([]byte{1, 0, 2, 0})
			if err != nil {
				t.Fatal(err)
			}
			privateErr := errors.New("private-provider-error https://fixture.invalid/private-path?token=secret")
			player := &loggingPlayer{}
			client := loggingSpeechClient(func(context.Context) ([]byte, error) {
				if stage == "inference" {
					return nil, privateErr
				}
				if stage == "decode" {
					return []byte("private-invalid-audio"), nil
				}
				return append([]byte(nil), wav...), nil
			})
			if stage == "load" {
				player.loadErr = privateErr
			}
			if stage == "play" {
				player.playErr = privateErr
			}
			service, logs := newLoggingSpeechService(t, client, player)
			if err := service.SpeakText("private-transcript"); err != nil {
				t.Fatal(err)
			}
			generation := service.CurrentStatus().Generation
			service.workers.Wait()
			assertSpeechLifecycle(t, logs, "speech generation", generation, "failed")
			for _, record := range logs.records(t) {
				if record["msg"] == "speech generation failed" {
					if record["level"] != "ERROR" || record["stage"] != stage || record["error_kind"] == nil {
						t.Errorf("unexpected failure fields: %v", record)
					}
				}
				if record["msg"] == "speech playback started" {
					t.Errorf("failed generation started playback: %v", record)
				}
			}
			data, _ := json.Marshal(logs.records(t))
			if strings.Contains(string(data), "private-") {
				t.Errorf("sensitive failure log: %s", data)
			}
		})
	}
}

func TestSpeechLoggingSuccessfulLifecycles(t *testing.T) {
	wav, err := audio.WAV([]byte{1, 0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	service, logs := newLoggingSpeechService(t, &speechClientFake{wav: wav}, &loggingPlayer{})
	if err := service.SpeakText("private-transcript"); err != nil {
		t.Fatal(err)
	}
	generation := service.CurrentStatus().Generation
	service.workers.Wait()
	// Releasing a completed session is not another playback terminal.
	if err := service.Stop(); err != nil {
		t.Fatal(err)
	}
	assertSpeechLifecycle(t, logs, "speech generation", generation, "completed")
	assertSpeechLifecycle(t, logs, "speech playback", generation, "completed")
	data, err := json.Marshal(logs.records(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-transcript", "private-model", "private-voice", "private-credential", "private-path", "private-output-device", "output_device", "generation_ms"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("logs contain forbidden content %q: %s", secret, data)
		}
	}
}

// Keep waits bounded without relying on a status callback running after a log.
func awaitSpeechLog(t *testing.T, logs *speechLogBuffer, message string, generation uint64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, record := range logs.records(t) {
			if record["msg"] == message && record["generation"] == float64(generation) {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("missing %q: %v", message, logs.records(t))
}
