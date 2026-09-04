package dictation

import (
	"context"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type capFake struct{}

func (capFake) List(context.Context) ([]audio.Device, error) { return nil, nil }
func (capFake) Start(context.Context, string, int) (<-chan error, error) {
	return nil, nil
}
func (capFake) Stop(context.Context) (audio.Result, error) { return audio.Result{}, nil }
func (capFake) Cancel(context.Context) error               { return nil }
func (capFake) Close() error                               { return nil }

type namedCapture struct {
	capFake
	listCalls int
}

func (c *namedCapture) List(context.Context) ([]audio.Device, error) {
	c.listCalls++
	return []audio.Device{{ID: "microphone-id", Name: "Test microphone"}}, nil
}
func (*namedCapture) DeviceName() string { return "Test microphone" }

type platFake struct {
	inserts, copies int
	lastInsert      string
}

func (*platFake) CaptureTarget() (insertion.Target, error) {
	return validTarget(), nil
}
func (*platFake) Foreground() (insertion.Target, error) {
	return validTarget(), nil
}
func (p *platFake) InsertUnicode(_ context.Context, _ insertion.Target, text string) error {
	p.inserts++
	p.lastInsert = text
	return nil
}
func (p *platFake) Copy(context.Context, string) error { p.copies++; return nil }

type settingsFake struct{}

func (settingsFake) Current() config.Settings { return config.Default() }
func validTarget() insertion.Target {
	return insertion.Target{HWND: 1, FocusHWND: 2, ThreadID: 3, ProcessID: 4, ProcessCreationTime: 5}
}

func TestStartRejectsUnknownRecordingMode(t *testing.T) {
	c := New(capFake{}, &platFake{}, nil, nil, settingsFake{}, nil)
	if err := c.StartWithMode(RecordingMode("unexpected")); err == nil {
		t.Fatal("unknown renderer-controlled recording mode was accepted")
	}
	if status := c.Status(); status.State != Idle || status.Generation != 0 {
		t.Fatalf("invalid start mutated recorder status: %+v", status)
	}
}

func TestStartUsesPreparedDeviceNameWithoutEnumerating(t *testing.T) {
	capture := &namedCapture{}
	c := New(capture, &platFake{}, nil, nil, settingsFake{}, nil)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	if capture.listCalls != 0 {
		t.Fatalf("recording start enumerated devices %d times", capture.listCalls)
	}
	if status := c.Status(); status.RecordingMode != RecordingToggle {
		t.Fatalf("recording mode = %q, want %q", status.RecordingMode, RecordingToggle)
	}
	c.mu.Lock()
	details := c.runDetails[c.status.Generation]
	c.mu.Unlock()
	if details.Microphone != "Test microphone" {
		t.Fatalf("microphone = %q, want prepared device name", details.Microphone)
	}
}

func TestCancelAdvancesGeneration(t *testing.T) {
	p := &platFake{}
	c := New(capFake{}, p, nil, nil, settingsFake{}, nil)
	if e := c.Start(); e != nil {
		t.Fatal(e)
	}
	g := c.Status().Generation
	_ = c.Cancel()
	if c.Status().State != Idle || c.Status().Generation <= g {
		t.Fatal("cancel did not fence generation")
	}
	if p.inserts != 0 {
		t.Fatal("cancel inserted")
	}
}

func TestNewRecordingClearsPending(t *testing.T) {
	p := &platFake{}
	c := New(capFake{}, p, nil, nil, settingsFake{}, nil)
	c.mu.Lock()
	c.pending = "old"
	c.status = Status{State: Failed, CanCopy: true}
	c.mu.Unlock()
	if e := c.Start(); e != nil {
		t.Fatal(e)
	}
	if c.Status().CanCopy {
		t.Fatal("stale pending transcript survived a new recording")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending != "" {
		t.Fatal("pending transcript was not cleared")
	}
}

func TestCopyPendingIsExplicitAndClearsMemory(t *testing.T) {
	p := &platFake{}
	c := New(capFake{}, p, nil, nil, settingsFake{}, nil)
	c.mu.Lock()
	c.pending = "pending"
	c.status = Status{State: Failed, Generation: 7, CanCopy: true}
	c.mu.Unlock()
	if e := c.CopyPending(); e != nil {
		t.Fatal(e)
	}
	if p.copies != 1 || c.Status().CanCopy {
		t.Fatal("explicit copy did not clear pending state")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending != "" {
		t.Fatal("pending text remained in memory")
	}
}

type blockingCapture struct {
	started    chan struct{}
	release    chan struct{}
	cancelled  chan struct{}
	startOnce  sync.Once
	cancelOnce sync.Once
}

func (c *blockingCapture) List(context.Context) ([]audio.Device, error) { return nil, nil }
func (c *blockingCapture) Start(context.Context, string, int) (<-chan error, error) {
	c.startOnce.Do(func() { close(c.started) })
	<-c.release
	return nil, nil
}
func (c *blockingCapture) Stop(context.Context) (audio.Result, error) {
	return audio.Result{}, nil
}
func (c *blockingCapture) Cancel(context.Context) error {
	c.cancelOnce.Do(func() { close(c.cancelled) })
	return nil
}
func (c *blockingCapture) Close() error { return nil }

func TestCancelWaitsForCaptureStartTransition(t *testing.T) {
	cap := &blockingCapture{started: make(chan struct{}), release: make(chan struct{}), cancelled: make(chan struct{})}
	c := New(cap, &platFake{}, nil, nil, settingsFake{}, nil)
	startDone := make(chan error, 1)
	go func() { startDone <- c.Start() }()
	<-cap.started
	cancelDone := make(chan error, 1)
	go func() { cancelDone <- c.Cancel() }()
	select {
	case <-cap.cancelled:
		t.Fatal("cancel raced microphone startup")
	case <-time.After(25 * time.Millisecond):
	}
	close(cap.release)
	if err := <-startDone; err != nil {
		t.Fatal(err)
	}
	if err := <-cancelDone; err != nil {
		t.Fatal(err)
	}
	if c.Status().State != Idle {
		t.Fatalf("state = %s, want idle", c.Status().State)
	}
}

type interruptCapture struct {
	mu          sync.Mutex
	sessions    []chan error
	cancelCount int
	partial     []byte
}

func (c *interruptCapture) List(context.Context) ([]audio.Device, error) { return nil, nil }
func (c *interruptCapture) Start(context.Context, string, int) (<-chan error, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	interrupted := make(chan error, 1)
	c.sessions = append(c.sessions, interrupted)
	return interrupted, nil
}
func (*interruptCapture) Stop(context.Context) (audio.Result, error) {
	return audio.Result{}, nil
}
func (c *interruptCapture) Cancel(context.Context) error {
	c.mu.Lock()
	c.cancelCount++
	for i := range c.partial {
		c.partial[i] = 0
	}
	c.mu.Unlock()
	return audio.ErrDeviceInterrupted
}
func (*interruptCapture) Close() error { return nil }

func (c *interruptCapture) session(index int) chan error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessions[index]
}

func waitForState(t *testing.T, recorder *testRecorder, want State) Status {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		status := recorder.Status()
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

func TestDeviceInterruptionFailsPromptlyDiscardsAndAllowsRetry(t *testing.T) {
	capture := &interruptCapture{partial: []byte{1, 2, 3, 4}}
	recorder := New(capture, &platFake{}, nil, nil, settingsFake{}, nil)
	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	firstGeneration := recorder.Status().Generation
	capture.session(0) <- audio.ErrDeviceInterrupted

	status := waitForState(t, recorder, Failed)
	if status.Generation != firstGeneration || status.Message != "Microphone: "+audio.ErrDeviceInterrupted.Error() {
		t.Fatalf("interruption status = %+v", status)
	}
	recorder.mu.Lock()
	_, targetRetained := recorder.targets[firstGeneration]
	pending := recorder.pending
	recordingCancel := recorder.recordingCancel
	recorder.mu.Unlock()
	capture.mu.Lock()
	cancelCount := capture.cancelCount
	partial := append([]byte(nil), capture.partial...)
	capture.mu.Unlock()
	if targetRetained || pending != "" || recordingCancel != nil || cancelCount != 1 || !allZero(partial) {
		t.Fatalf("interruption cleanup: target=%v pending=%q watcher=%v cancels=%d partial=%v", targetRetained, pending, recordingCancel != nil, cancelCount, partial)
	}

	if err := recorder.Start(); err != nil {
		t.Fatalf("retry start: %v", err)
	}
	second := recorder.Status()
	if second.State != Recording || second.Generation <= firstGeneration {
		t.Fatalf("retry status = %+v", second)
	}

	// A delayed notification from the first recording is tied to its old
	// channel and cannot fail the new generation.
	capture.session(0) <- audio.ErrDeviceInterrupted
	time.Sleep(25 * time.Millisecond)
	if got := recorder.Status(); got.State != Recording || got.Generation != second.Generation {
		t.Fatalf("stale interruption changed retry: %+v", got)
	}
	if err := recorder.Cancel(); err != nil {
		t.Fatal(err)
	}
}

func allZero(value []byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}

type staticSettings struct{ value config.Settings }

func (s staticSettings) Current() config.Settings { return s.value }

type staticCredential struct{ value string }

func (s staticCredential) Get() (string, error) { return s.value, nil }
func (staticCredential) Set(string) error       { return nil }
func (staticCredential) Delete() error          { return nil }
func (staticCredential) Configured() bool       { return true }

func TestReflectedCredentialNeverBecomesInsertOrPendingText(t *testing.T) {
	const secret = "recorder-credential-canary"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"text":"prefix ` + secret + ` suffix"}`))
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.BaseURL = server.URL
	cfg.Model = "speech/stt"
	cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	platform := &platFake{}
	c := New(capFake{}, platform, inference.New(), staticCredential{value: secret}, staticSettings{value: cfg}, nil)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(); err == nil {
		t.Fatal("expected reflected credential to fail transcription")
	}
	status := c.Status()
	if platform.inserts != 0 || platform.copies != 0 || status.CanCopy {
		t.Fatalf("credential reached delivery state: inserts=%d copies=%d status=%+v", platform.inserts, platform.copies, status)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending != "" {
		t.Fatal("credential retained as pending text")
	}
}
