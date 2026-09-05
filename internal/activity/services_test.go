package activity_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/filetranscription"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/tnware/freehand-stt/internal/tts"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type capture struct {
	starts      atomic.Int32
	fail        bool
	beforeStart func()
}

func (*capture) List(context.Context) ([]audio.Device, error) { return nil, nil }
func (c *capture) Start(context.Context, string, int) (<-chan error, error) {
	c.starts.Add(1)
	if c.beforeStart != nil {
		c.beforeStart()
	}
	if c.fail {
		return nil, errors.New("capture unavailable")
	}
	return nil, nil
}
func (*capture) Stop(context.Context) (audio.Result, error) { return audio.Result{}, nil }
func (*capture) Cancel(context.Context) error               { return nil }
func (*capture) Close() error                               { return nil }

type input struct{}

func (input) CaptureTarget() (insertion.Target, error)                      { return insertion.Target{}, nil }
func (input) Foreground() (insertion.Target, error)                         { return insertion.Target{}, nil }
func (input) InsertUnicode(context.Context, insertion.Target, string) error { return nil }
func (input) Copy(context.Context, string) error                            { return nil }

type player struct {
	stopped atomic.Int32
	stop    func() error
}

func (*player) Load([]byte, uint32, uint32) error { return nil }
func (*player) Play() error                       { return nil }
func (*player) Pause() error                      { return nil }
func (*player) Restart() error                    { return nil }
func (*player) Position() (int64, int64, bool)    { return 0, 0, false }
func (*player) OutputName() string                { return "fixture" }
func (*player) Save(string) error                 { return nil }
func (p *player) Stop() error {
	p.stopped.Add(1)
	if p.stop != nil {
		return p.stop()
	}
	return nil
}
func (*player) Unload() error { return nil }
func (*player) Close() error  { return nil }

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type speech struct{ calls atomic.Int32 }

func (s *speech) SynthesizeSpeech(ctx context.Context, _, _ string, _ inference.SpeechRequest) ([]byte, error) {
	s.calls.Add(1)
	<-ctx.Done()
	return nil, ctx.Err()
}

type harness struct {
	admission *activity.Coordinator
	voice     *dictation.Service
	files     *filetranscription.Service
	speech    *tts.Service
	capture   *capture
	player    *player
	client    *speech
}

func newHarness(t *testing.T, beforeSpeechProfile func(), fileProfile func()) *harness {
	t.Helper()
	h := &harness{capture: &capture{}, player: &player{}, client: &speech{}}
	h.admission = activity.New(activity.Sources{DictationActive: func() bool { return dictation.Active(h.voice) }, FileActive: func() bool { return filetranscription.Active(h.files) }, StopPlayback: func() error { return h.speech.Stop() }})
	cfg := config.Default()
	cfg.SetupCompleted = true
	cfg.BaseURL = "https://fixture.invalid/v1"
	cfg.Model = "speech"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	cfg.VADEnabled = false
	cfg.SilenceSplitting = false
	cfg.AutoStopEnabled = false
	source := settings.Source(func() config.Settings { return cfg })
	profiles := settings.ProfileSource(func() (settings.RequestProfile, error) { return settings.RequestProfile{Settings: cfg}, nil })
	client := inference.New()
	client.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	h.voice = dictation.NewService(h.capture, input{}, client, nil, source, profiles, nil, h.admission, nil, nil)
	path := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(path, []byte("RIFF audio"), 0600); err != nil {
		t.Fatal(err)
	}
	h.files = filetranscription.NewService(source, func() (settings.RequestProfile, error) {
		if fileProfile != nil {
			fileProfile()
		}
		return profiles.Capture()
	}, client, nil, nil, input{}, func() (string, error) { return path, nil }, nil, nil, h.admission, nil)
	h.speech = tts.NewService(func() (settings.TextToSpeechProfile, error) {
		if beforeSpeechProfile != nil {
			beforeSpeechProfile()
		}
		return settings.TextToSpeechProfile{Settings: config.TextToSpeechSettings{Enabled: true, BaseURL: "https://fixture.invalid/v1", Model: "speech", Voice: "voice", Speed: 1, AuthenticationMode: config.AuthenticationModeNone, TimeoutSeconds: 30}}, nil
	}, h.client, h.player, nil, nil, nil, h.admission, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	for _, s := range []interface {
		ServiceStartup(context.Context, application.ServiceOptions) error
	}{h.voice, h.files, h.speech} {
		if err := s.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		cancel()
		h.admission.Close()
		_ = h.voice.ServiceShutdown()
		_ = h.files.ServiceShutdown()
		_ = h.speech.ServiceShutdown()
	})
	if _, err := h.files.ChooseAudioFile(); err != nil {
		t.Fatal(err)
	}
	return h
}
func await(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("operation did not settle")
		return nil
	}
}
func entered(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("operation did not reach gate")
	}
}
func async(f func() error) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- f() }()
	return ch
}
func waitIdleFile(t *testing.T, h *harness) {
	t.Helper()
	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for filetranscription.Active(h.files) {
		select {
		case <-deadline:
			t.Fatal("file cancellation did not settle")
		case <-tick.C:
		}
	}
}

func TestRealServicesExcludeTranscriptionAndReleaseAfterCancel(t *testing.T) {
	for _, first := range []string{"voice", "file"} {
		t.Run(first, func(t *testing.T) {
			h := newHarness(t, nil, nil)
			if first == "voice" {
				if err := h.voice.StartRecording(dictation.RecordingToggle); err != nil {
					t.Fatal(err)
				}
				if err := h.files.StartFileTranscription(false); err == nil {
					t.Fatal("file overlapped dictation")
				}
				if err := h.speech.PreviewVoice(); err == nil {
					t.Fatal("speech overlapped dictation")
				}
				if err := h.voice.Cancel(); err != nil {
					t.Fatal(err)
				}
				if err := h.files.StartFileTranscription(false); err != nil {
					t.Fatal("cancelled voice stranded admission", err)
				}
			} else {
				if err := h.files.StartFileTranscription(false); err != nil {
					t.Fatal(err)
				}
				if err := h.voice.StartRecording(dictation.RecordingToggle); err == nil {
					t.Fatal("dictation overlapped file")
				}
				if err := h.speech.PreviewVoice(); err == nil {
					t.Fatal("speech overlapped file")
				}
				if err := h.files.CancelFileTranscription(); err != nil {
					t.Fatal(err)
				}
				waitIdleFile(t, h)
				if err := h.voice.StartRecording(dictation.RecordingToggle); err != nil {
					t.Fatal("cancelled file stranded admission", err)
				}
			}
		})
	}
}

func TestSpeechStartAndRecordingPreemptionShareLockOrder(t *testing.T) {
	gate, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	h := newHarness(t, func() { close(gate); <-release }, nil)
	t.Cleanup(unblock)
	startSpeech := async(h.speech.PreviewVoice)
	entered(t, gate)
	startVoice := async(func() error { return h.voice.StartRecording(dictation.RecordingToggle) })
	h.capture.beforeStart = func() {
		if h.player.stopped.Load() < 2 {
			t.Error("capture began before speech preemption")
		}
	}
	unblock()
	if err := await(t, startSpeech); err != nil {
		t.Fatal(err)
	}
	if err := await(t, startVoice); err != nil {
		t.Fatal(err)
	}
	if !dictation.Active(h.voice) || h.speech.CurrentStatus().Phase != tts.Cancelled {
		t.Fatal("speech escaped recording preemption")
	}
}

func TestRecordingPreemptionBlocksCompetingPlaybackAndShutdown(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(map[bool]string{false: "recording", true: "shutdown"}[shutdown], func(t *testing.T) {
			h := newHarness(t, nil, nil)
			gate, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			t.Cleanup(unblock)
			var stopOnce sync.Once
			h.player.stop = func() error { stopOnce.Do(func() { close(gate); <-release }); return nil }
			recording := async(func() error { return h.voice.StartRecording(dictation.RecordingToggle) })
			entered(t, gate)
			playback := async(h.speech.PreviewVoice)
			if shutdown {
				h.admission.Close()
			}
			unblock()
			recordErr := await(t, recording)
			playErr := await(t, playback)
			if shutdown {
				if !errors.Is(recordErr, activity.ErrClosed) || h.capture.starts.Load() != 0 {
					t.Fatal("late preemption started capture after shutdown")
				}
			} else if recordErr != nil {
				t.Fatal(recordErr)
			}
			if playErr == nil || h.client.calls.Load() != 0 {
				t.Fatal("competing playback escaped admission")
			}
		})
	}
}

func TestFailedStartsDoNotStrandOtherFeatures(t *testing.T) {
	for _, failure := range []string{"capture", "file-selection", "preemption"} {
		t.Run(failure, func(t *testing.T) {
			h := newHarness(t, nil, nil)
			switch failure {
			case "capture":
				h.capture.fail = true
				if err := h.voice.StartRecording(dictation.RecordingToggle); err == nil {
					t.Fatal("expected capture failure")
				}
				if err := h.files.StartFileTranscription(false); err != nil {
					t.Fatal(err)
				}
			case "file-selection":
				if err := h.files.ClearAudioFile(); err != nil {
					t.Fatal(err)
				}
				if err := h.files.StartFileTranscription(false); err == nil {
					t.Fatal("expected missing selection")
				}
				if err := h.voice.StartRecording(dictation.RecordingToggle); err != nil {
					t.Fatal(err)
				}
			case "preemption":
				h.player.stop = func() error { return errors.New("stop failed") }
				if err := h.voice.StartRecording(dictation.RecordingToggle); err == nil || h.capture.starts.Load() != 0 {
					t.Fatal("failed playback stop admitted microphone")
				}
				if err := h.files.StartFileTranscription(false); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestFileStartRechecksShutdownAfterProfileCapture(t *testing.T) {
	gate, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	h := newHarness(t, nil, func() { close(gate); <-release })
	t.Cleanup(unblock)
	start := async(func() error { return h.files.StartFileTranscription(false) })
	entered(t, gate)
	if err := h.files.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	unblock()
	if err := await(t, start); err == nil || filetranscription.Active(h.files) {
		t.Fatal("start published after shutdown")
	}
}
