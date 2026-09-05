package dictation

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
)

type streamingCapture struct {
	mu       sync.Mutex
	sink     audio.PCMStreamSink
	observed chan struct{}
}

func newStreamingCapture() *streamingCapture {
	return &streamingCapture{observed: make(chan struct{}, 1)}
}

func (*streamingCapture) List(context.Context) ([]audio.Device, error) { return nil, nil }
func (*streamingCapture) Start(context.Context, string, int) (<-chan error, error) {
	return nil, errors.New("buffered capture was not expected")
}
func (c *streamingCapture) StartStream(_ context.Context, _ string, _ int, sink audio.PCMStreamSink) (<-chan error, error) {
	c.mu.Lock()
	c.sink = sink
	c.mu.Unlock()
	return make(chan error, 1), nil
}
func (c *streamingCapture) Stop(context.Context) (audio.Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Model native capture stop: no admitted callback may outlive sink close.
	if c.sink != nil {
		c.sink.Close()
		c.sink = nil
	}
	return audio.Result{}, nil
}
func (c *streamingCapture) Cancel(ctx context.Context) error {
	_, err := c.Stop(ctx)
	return err
}
func (c *streamingCapture) Close() error { return c.Cancel(context.Background()) }

func (c *streamingCapture) write(frame []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sink != nil && c.sink.WritePCM(frame)
}

type sampleDetector struct{ observed chan<- struct{} }

func (d *sampleDetector) Speech(samples []int16) (bool, error) {
	defer func() { d.observed <- struct{}{} }()
	for _, sample := range samples {
		if sample != 0 {
			return true, nil
		}
	}
	return false, nil
}
func (*sampleDetector) Close() error { return nil }

type mutableCredential struct {
	mu    sync.Mutex
	value string
}

func (c *mutableCredential) Get() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value, nil
}
func (c *mutableCredential) Set(value string) error {
	c.mu.Lock()
	c.value = value
	c.mu.Unlock()
	return nil
}
func (c *mutableCredential) Delete() error { return c.Set("") }
func (c *mutableCredential) Configured() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value != ""
}

func pcmFrame(value int16) []byte {
	frame := make([]byte, audio.VADFrameBytes)
	for offset := 0; offset < len(frame); offset += 2 {
		binary.LittleEndian.PutUint16(frame[offset:], uint16(value))
	}
	return frame
}

func feedFrames(t *testing.T, capture *streamingCapture, frame []byte, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		if !capture.write(frame) {
			t.Fatalf("stream rejected frame %d of %d", i+1, count)
		}
		// Wait until the real segmenter reaches this frame's detector call.
		// It may not have released the frame yet, but at most this frame and
		// the next can occupy the pipe. This tests workflow ordering, not
		// synthetic producer throughput; never retry a rejected write.
		select {
		case <-capture.observed:
		case <-time.After(time.Second):
			t.Fatalf("detector did not observe frame %d of %d", i+1, count)
		}
	}
}

func segmentedSettings(baseURL string) config.Settings {
	cfg := config.Default()
	cfg.BaseURL = baseURL
	cfg.Model = "speech/stt"
	cfg.SilenceSplitting = true
	cfg.SegmentSeconds = 15
	cfg.SegmentSilenceMS = 200
	cfg.VADActivitySilenceMS = 200
	cfg.MaxDurationSeconds = 600
	cfg.HistoryEnabled = true
	return cfg
}

func TestSegmentedDictationTranscribesDuringCaptureAndDeliversInOrder(t *testing.T) {
	var requests atomic.Int32
	var active atomic.Int32
	var maximumActive atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			prior := maximumActive.Load()
			if current <= prior || maximumActive.CompareAndSwap(prior, current) {
				break
			}
		}
		request := requests.Add(1)
		time.Sleep(5 * time.Millisecond)
		_, _ = fmt.Fprintf(w, `{"text":"segment-%d"}`, request)
	}))
	defer server.Close()

	capture := newStreamingCapture()
	platform := &platFake{}
	cfg := segmentedSettings(server.URL)
	cfg.VADMode = config.VADModeVeryAggressive
	var statusMu sync.Mutex
	var statuses []Status
	recorder := New(capture, platform, nil, staticCredential{value: "secret"}, staticSettings{value: cfg}, func(status Status) {
		statusMu.Lock()
		statuses = append(statuses, status)
		statusMu.Unlock()
	})
	recorder.client = newTestClient(server.Client())
	requestedVADMode := config.VADMode("")
	recorder.newDetector = func(mode config.VADMode) (audio.VoiceDetector, error) {
		requestedVADMode = mode
		return &sampleDetector{observed: capture.observed}, nil
	}
	var logs bytes.Buffer
	recorder.SetLogger(slog.New(slog.NewTextHandler(&logs, nil)))

	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	if requestedVADMode != config.VADModeVeryAggressive {
		t.Fatalf("VAD mode = %q, want %q", requestedVADMode, config.VADModeVeryAggressive)
	}
	feedFrames(t, capture, pcmFrame(1200), 740)
	feedFrames(t, capture, pcmFrame(0), 10)

	deadline := time.Now().Add(time.Second)
	for requests.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if requests.Load() != 1 {
		t.Fatalf("completed segment requests while recording = %d, want 1", requests.Load())
	}
	waitForSegmentCheckpoint(t, &statusMu, &statuses, 1, SegmentCompleted)
	waitForVADState(t, &statusMu, &statuses, VADSilence)
	statusMu.Lock()
	checkpointStatuses := append([]Status(nil), statuses...)
	statusMu.Unlock()
	assertSegmentCheckpoint(t, checkpointStatuses, 1, SegmentTranscribing)
	assertSegmentCheckpoint(t, checkpointStatuses, 1, SegmentCompleted)
	assertVADState(t, checkpointStatuses, VADSpeech)
	assertVADState(t, checkpointStatuses, VADSilence)

	feedFrames(t, capture, pcmFrame(800), 20)
	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 || maximumActive.Load() != 1 {
		t.Fatalf("requests = %d, max concurrent = %d", requests.Load(), maximumActive.Load())
	}
	if platform.inserts != 1 || platform.lastInsert != "segment-1 segment-2" {
		t.Fatalf("delivery = %d × %q", platform.inserts, platform.lastInsert)
	}
	entries := recorder.History()
	if len(entries) != 1 || entries[0].Text != platform.lastInsert {
		t.Fatalf("history = %#v", entries)
	}
	details := entries[0].Details
	if details.SegmentCount != 2 || len(details.Segments) != 2 || details.AudioDurationMilliseconds <= 0 {
		t.Fatalf("segmented run details = %#v", details)
	}
	if details.Segments[0].Boundary != "silence" || details.Segments[1].Boundary != "recording_stopped" {
		t.Fatalf("segment boundaries = %#v", details.Segments)
	}
	logText := logs.String()
	for _, marker := range []string{"dictation segment ready", "boundary=silence", "boundary=recording_stopped", "dictation segment transcription completed", "dictation completed"} {
		if !strings.Contains(logText, marker) {
			t.Fatalf("diagnostic logs omitted %q:\n%s", marker, logText)
		}
	}
	for _, sensitive := range []string{"secret", "segment-1 segment-2", server.URL} {
		if strings.Contains(logText, sensitive) {
			t.Fatalf("diagnostic logs exposed %q:\n%s", sensitive, logText)
		}
	}
}

func waitForVADState(t *testing.T, mu *sync.Mutex, statuses *[]Status, want VADState) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		for _, status := range *statuses {
			if status.State == Recording && status.VADState == want {
				mu.Unlock()
				return
			}
		}
		mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for recording VAD state %s", want)
}

func assertVADState(t *testing.T, statuses []Status, want VADState) {
	t.Helper()
	for _, status := range statuses {
		if status.State == Recording && status.VADState == want {
			return
		}
	}
	t.Fatalf("missing recording VAD state %s in %#v", want, statuses)
}

func waitForSegmentCheckpoint(t *testing.T, mu *sync.Mutex, statuses *[]Status, segment int, phase SegmentPhase) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		for _, status := range *statuses {
			if status.State == Recording && status.SegmentNumber == segment && status.SegmentPhase == phase {
				mu.Unlock()
				return
			}
		}
		mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for recording checkpoint segment=%d phase=%s", segment, phase)
}

func assertSegmentCheckpoint(t *testing.T, statuses []Status, segment int, phase SegmentPhase) {
	t.Helper()
	for _, status := range statuses {
		if status.State == Recording && status.SegmentNumber == segment && status.SegmentPhase == phase {
			return
		}
	}
	t.Fatalf("missing recording checkpoint segment=%d phase=%s in %#v", segment, phase, statuses)
}

func TestSegmentedDictationWithOnlySilenceSkipsNetworkAndInsertion(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"text":"unexpected"}`))
	}))
	defer server.Close()

	capture := newStreamingCapture()
	platform := &platFake{}
	cfg := segmentedSettings(server.URL)
	recorder := New(capture, platform, nil, staticCredential{value: "secret"}, staticSettings{value: cfg}, nil)
	recorder.client = newTestClient(server.Client())
	recorder.newDetector = func(config.VADMode) (audio.VoiceDetector, error) {
		return &sampleDetector{observed: capture.observed}, nil
	}

	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	feedFrames(t, capture, pcmFrame(0), 25)
	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
	status := recorder.Status()
	if requests.Load() != 0 || platform.inserts != 0 || status.State != Idle || status.Message != "No speech detected" {
		t.Fatalf("requests=%d inserts=%d status=%+v", requests.Load(), platform.inserts, status)
	}
}

func TestSegmentedDictationNoAuthSkipsCredentialStore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authorization := r.Header.Get("Authorization"); authorization != "" {
			t.Errorf("unexpected authorization header %q", authorization)
		}
		_, _ = w.Write([]byte(`{"text":"local segment"}`))
	}))
	defer server.Close()

	capture := newStreamingCapture()
	platform := &platFake{}
	cfg := segmentedSettings(server.URL)
	cfg.AuthenticationMode = config.AuthenticationModeNone
	recorder := New(capture, platform, nil, nil, staticSettings{value: cfg}, nil)
	recorder.client = newTestClient(server.Client())
	recorder.newDetector = func(config.VADMode) (audio.VoiceDetector, error) {
		return &sampleDetector{observed: capture.observed}, nil
	}

	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	feedFrames(t, capture, pcmFrame(1600), 25)
	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
	if platform.lastInsert != "local segment" {
		t.Fatalf("inserted = %q", platform.lastInsert)
	}
}

func TestSegmentedDictationUsesCredentialCapturedBeforeFirstCheckpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer credential-at-start" {
			t.Errorf("authorization = %q", authorization)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
		} else {
			defer r.MultipartForm.RemoveAll()
			if r.FormValue("prompt") != "captured context" || r.FormValue("hotwords") != "captured terms" || r.FormValue("temperature") != "0" {
				t.Error("checkpoint lost captured controls")
			}
		}
		_, _ = w.Write([]byte(`{"text":"captured profile"}`))
	}))
	defer server.Close()

	capture := newStreamingCapture()
	platform := &platFake{}
	cfg := segmentedSettings(server.URL)
	cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	cfg.CompatibilityProfile = compatibility.Speaches
	cfg.TranscriptionOptions = compatibility.TranscriptionOptions{Prompt: "captured context", Hotwords: "captured terms", TemperatureOverride: true}
	credentials := &mutableCredential{value: "credential-at-start"}
	recorder := New(capture, platform, nil, credentials, staticSettings{value: cfg}, nil)
	recorder.client = newTestClient(server.Client())
	recorder.profiles = settings.ProfileSource(func() (settings.RequestProfile, error) {
		key, err := credentials.Get()
		return settings.RequestProfile{Settings: cfg, STTCredential: key}, err
	})
	recorder.newDetector = func(config.VADMode) (audio.VoiceDetector, error) {
		return &sampleDetector{observed: capture.observed}, nil
	}

	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	cfg.TranscriptionOptions = compatibility.TranscriptionOptions{Prompt: "edited context", Temperature: 1}
	if err := credentials.Set("credential-edited-before-checkpoint"); err != nil {
		t.Fatal(err)
	}
	feedFrames(t, capture, pcmFrame(1600), 25)
	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
	if platform.lastInsert != "captured profile" {
		t.Fatalf("inserted = %q", platform.lastInsert)
	}
}

func TestAutomaticStopTrimsSilenceAndCompletesDictation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"text":"automatically completed"}`))
	}))
	defer server.Close()

	capture := newStreamingCapture()
	platform := &platFake{}
	cfg := config.Default()
	cfg.BaseURL = server.URL
	cfg.HistoryEnabled = true
	cfg.SilenceTrimming = true
	cfg.SpeechPaddingMS = 100
	cfg.AutoStopEnabled = true
	cfg.AutoStopSilenceMS = 500
	cfg.AutoStopMinimumSpeechMS = 100
	cfg.VADActivitySilenceMS = 200
	var statusMu sync.Mutex
	var statuses []Status
	recorder := New(capture, platform, nil, staticCredential{value: "secret"}, staticSettings{value: cfg}, func(status Status) {
		statusMu.Lock()
		statuses = append(statuses, status)
		statusMu.Unlock()
	})
	recorder.client = newTestClient(server.Client())
	recorder.newDetector = func(config.VADMode) (audio.VoiceDetector, error) {
		return &sampleDetector{observed: capture.observed}, nil
	}

	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	feedFrames(t, capture, pcmFrame(0), 10)
	feedFrames(t, capture, pcmFrame(1200), 10)
	feedFrames(t, capture, pcmFrame(0), 25)

	deadline := time.Now().Add(2 * time.Second)
	for recorder.Status().State != Idle && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := recorder.Status()
	if status.State != Idle || status.Message != "Silence detected; dictation completed" {
		t.Fatalf("automatic stop status = %+v", status)
	}
	if platform.lastInsert != "automatically completed" {
		t.Fatalf("inserted = %q", platform.lastInsert)
	}
	entries := recorder.History()
	if len(entries) != 1 {
		t.Fatalf("history = %#v", entries)
	}
	details := entries[0].Details
	if !details.AutoStopped || !details.SilenceTrimming || details.AudioDurationMilliseconds != 400 {
		t.Fatalf("automatic stop history details = %#v", details)
	}
	statusMu.Lock()
	observedStatuses := append([]Status(nil), statuses...)
	statusMu.Unlock()
	var waiting, listening, countdown bool
	for _, observed := range observedStatuses {
		switch observed.AutoStopState {
		case AutoStopWaiting:
			waiting = true
		case AutoStopListening:
			listening = true
		case AutoStopCountdown:
			countdown = !observed.AutoStopDeadline.IsZero()
		}
	}
	if !waiting || !listening || !countdown {
		t.Fatalf("automatic stop lifecycle missing: waiting=%v listening=%v countdown=%v statuses=%#v", waiting, listening, countdown, observedStatuses)
	}
}

func TestHoldRecordingWaitsForReleaseWhenAutomaticStopIsConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"text":"released normally"}`))
	}))
	defer server.Close()

	capture := newStreamingCapture()
	platform := &platFake{}
	cfg := config.Default()
	cfg.BaseURL = server.URL
	cfg.HistoryEnabled = true
	cfg.SilenceTrimming = true
	cfg.SpeechPaddingMS = 100
	cfg.AutoStopEnabled = true
	cfg.AutoStopSilenceMS = 500
	cfg.AutoStopMinimumSpeechMS = 100
	cfg.VADActivitySilenceMS = 200
	var statusMu sync.Mutex
	var statuses []Status
	recorder := New(capture, platform, nil, staticCredential{value: "secret"}, staticSettings{value: cfg}, func(status Status) {
		statusMu.Lock()
		statuses = append(statuses, status)
		statusMu.Unlock()
	})
	recorder.client = newTestClient(server.Client())
	recorder.newDetector = func(config.VADMode) (audio.VoiceDetector, error) {
		return &sampleDetector{observed: capture.observed}, nil
	}

	if err := recorder.StartWithMode(RecordingHold); err != nil {
		t.Fatal(err)
	}
	if status := recorder.Status(); status.RecordingMode != RecordingHold {
		t.Fatalf("recording mode = %q, want %q", status.RecordingMode, RecordingHold)
	}
	feedFrames(t, capture, pcmFrame(1200), 10)
	feedFrames(t, capture, pcmFrame(0), 35)
	waitForVADState(t, &statusMu, &statuses, VADSilence)
	time.Sleep(30 * time.Millisecond)
	if status := recorder.Status(); status.State != Recording || status.AutoStopState != "" || !status.AutoStopDeadline.IsZero() {
		t.Fatalf("hold recording ended or exposed a countdown before release: %+v", status)
	}
	if platform.inserts != 0 {
		t.Fatalf("hold recording inserted %d times before release", platform.inserts)
	}

	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
	if platform.lastInsert != "released normally" {
		t.Fatalf("inserted = %q", platform.lastInsert)
	}
	entries := recorder.History()
	if len(entries) != 1 {
		t.Fatalf("history = %#v", entries)
	}
	details := entries[0].Details
	if details.RecordingMode != string(RecordingHold) || !details.AutoStopEnabled || details.AutoStopActive || details.AutoStopped || !details.SilenceTrimming {
		t.Fatalf("hold recording history details = %#v", details)
	}
}

func newTestClient(httpClient *http.Client) *inference.Client {
	return &inference.Client{HTTP: httpClient}
}
