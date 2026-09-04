package dictation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type completionCapture struct {
	pcm []byte
}

func (*completionCapture) List(context.Context) ([]audio.Device, error) { return nil, nil }
func (*completionCapture) Start(context.Context, string, int) (<-chan error, error) {
	return make(chan error), nil
}
func (c *completionCapture) Stop(context.Context) (audio.Result, error) {
	return audio.Result{PCM: append([]byte(nil), c.pcm...)}, nil
}
func (*completionCapture) Cancel(context.Context) error { return nil }
func (*completionCapture) Close() error                 { return nil }

type completionPlatform struct {
	mu       sync.Mutex
	inserted []string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func (*completionPlatform) CaptureTarget() (insertion.Target, error) { return validTarget(), nil }
func (*completionPlatform) Foreground() (insertion.Target, error)    { return validTarget(), nil }
func (p *completionPlatform) InsertUnicode(_ context.Context, _ insertion.Target, text string) error {
	p.mu.Lock()
	p.inserted = append(p.inserted, text)
	p.mu.Unlock()
	return nil
}
func (*completionPlatform) Copy(context.Context, string) error { return nil }

func (p *completionPlatform) insertion() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.inserted) == 0 {
		return ""
	}
	return p.inserted[len(p.inserted)-1]
}

func newCompletionService(serverURL string, clients ...*inference.Client) (*Service, *completionPlatform) {
	cfg := config.Default()
	cfg.SetupCompleted = true
	cfg.BaseURL = serverURL
	cfg.Model = "speech/stt"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	cfg.AutoInsert = true
	settingsSource := settings.Source(func() config.Settings { return cfg })
	profiles := settings.ProfileSource(func() (settings.RequestProfile, error) {
		return settings.RequestProfile{Settings: cfg}, nil
	})
	client := inference.New()
	if len(clients) > 0 {
		client = clients[0]
	}
	platform := &completionPlatform{}
	service := NewService(
		&completionCapture{pcm: make([]byte, audio.SampleRate/10)},
		platform,
		client,
		nil,
		settingsSource,
		profiles,
		nil,
		func() bool { return false },
		nil,
		nil,
		nil,
	)
	return service, platform
}

func startCompletionService(t *testing.T, service *Service) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	if err := service.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
		cancel()
		t.Fatal(err)
	}
	return cancel
}

func waitForServiceState(t *testing.T, service *Service, want State) Status {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		status := service.CurrentStatus()
		if status.State == want {
			return status
		}
		select {
		case <-deadline.C:
			t.Fatalf("state = %s, want %s", status.State, want)
		case <-ticker.C:
		}
	}
}

func TestStopRecordingReturnsAfterCaptureWhileCompletionContinues(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	var releaseOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		_, _ = w.Write([]byte(`{"text":"managed transcript"}`))
	}))
	defer server.Close()
	defer releaseOnce.Do(func() { close(releaseRequest) })

	service, platform := newCompletionService(server.URL)
	cancelRoot := startCompletionService(t, service)
	defer cancelRoot()
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.StartRecording(RecordingToggle); err != nil {
		t.Fatal(err)
	}

	stopDone := make(chan error, 1)
	go func() { stopDone <- service.StopRecording() }()
	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(250 * time.Millisecond):
		releaseOnce.Do(func() { close(releaseRequest) })
		t.Fatal("StopRecording remained blocked on remote transcription")
	}
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("managed completion did not start transcription")
	}
	if status := service.CurrentStatus(); status.State != Transcribing || platform.insertion() != "" {
		t.Fatalf("in-flight completion status=%+v insertion=%q", status, platform.insertion())
	}

	releaseOnce.Do(func() { close(releaseRequest) })
	waitForServiceState(t, service, Idle)
	if got := platform.insertion(); got != "managed transcript" {
		t.Fatalf("inserted = %q", got)
	}
}

func TestBeforeRecordingHookCompletesBeforeCaptureStarts(t *testing.T) {
	service, _ := newCompletionService("http://127.0.0.1")
	cancelRoot := startCompletionService(t, service)
	defer cancelRoot()
	defer func() { _ = service.ServiceShutdown() }()
	preempted := false
	SetBeforeRecording(service, func() { preempted = true })

	if err := service.StartRecording(RecordingToggle); err != nil {
		t.Fatal(err)
	}
	if !preempted || service.CurrentStatus().State != Recording {
		t.Fatalf("preempted=%v status=%+v", preempted, service.CurrentStatus())
	}
}

func TestCancelStopsManagedCompletion(t *testing.T) {
	requestStarted := make(chan struct{})
	requestCancelled := make(chan struct{})
	client := &inference.Client{HTTP: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		close(requestStarted)
		<-request.Context().Done()
		close(requestCancelled)
		return nil, request.Context().Err()
	})}}

	service, platform := newCompletionService("http://example.test/v1", client)
	cancelRoot := startCompletionService(t, service)
	defer cancelRoot()
	defer func() { _ = service.ServiceShutdown() }()
	if err := service.StartRecording(RecordingToggle); err != nil {
		t.Fatal(err)
	}
	if err := service.StopRecording(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("managed completion did not start transcription")
	}
	if err := service.Cancel(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-requestCancelled:
	case <-time.After(time.Second):
		t.Fatal("managed completion request was not cancelled")
	}
	if status := service.CurrentStatus(); status.State != Idle || platform.insertion() != "" {
		t.Fatalf("cancelled completion status=%+v insertion=%q", status, platform.insertion())
	}
}

func TestServiceShutdownWaitsForTheManagedCompletionWorker(t *testing.T) {
	service, _ := newCompletionService("http://127.0.0.1")
	cancelRoot := startCompletionService(t, service)
	defer cancelRoot()
	workerStarted := make(chan struct{})
	releaseWorker := make(chan struct{})
	if !service.scheduleCompletion(func() {
		close(workerStarted)
		<-releaseWorker
	}) {
		t.Fatal("completion worker rejected work while active")
	}
	<-workerStarted

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- service.ServiceShutdown() }()
	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before its worker: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(releaseWorker)
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish after the managed worker")
	}
	if service.scheduleCompletion(func() {}) {
		t.Fatal("shutdown service accepted new completion work")
	}
}
