package filetranscription

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type shutdownTransport func(*http.Request) (*http.Response, error)

func (f shutdownTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func awaitFileBoundary(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("file boundary not reached")
	}
}
func awaitFileShutdown(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("shutdown = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("file shutdown ignored its wait deadline")
	}
}
func TestShutdownBudgetIncludesFileStateLock(t *testing.T) {
	service := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	service.fileMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitFileBoundary(t, service.rootContext.Done())
	cancel()
	awaitFileShutdown(t, done)
	service.fileMu.Unlock()
	awaitFileBoundary(t, service.shutdownDone)
}
func TestShutdownDiscardsLateFileTranscriptionSuccess(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	client := inference.New()
	client.HTTP.Transport = shutdownTransport(func(r *http.Request) (*http.Response, error) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		close(entered)
		<-release
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"text":"late file transcript"}`)), Request: r}, nil
	})
	cfg := config.Default()
	cfg.SetupCompleted, cfg.HistoryEnabled = true, true
	cfg.BaseURL, cfg.Model = "https://fixture.invalid/v1", "speech"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	transcripts := history.NewStore(true, nil)
	var publications atomic.Int32
	service := NewService(func() config.Settings { return cfg }, func() (settings.RequestProfile, error) { return settings.RequestProfile{Settings: cfg}, nil }, client, nil, transcripts, nil, nil, func(FileTranscriptionStatus) { publications.Add(1) }, nil, nil, nil)
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "recording.wav")
	t.Cleanup(func() { unblock(); _ = service.ServiceShutdown() })
	if err := os.WriteFile(path, []byte("RIFF audio"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.selectAudioFile(path); err != nil {
		t.Fatal(err)
	}
	if err := service.StartFileTranscription(false); err != nil {
		t.Fatal(err)
	}
	awaitFileBoundary(t, entered)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.shutdown(ctx) }()
	awaitFileBoundary(t, service.rootContext.Done())
	count := publications.Load()
	cancel()
	awaitFileShutdown(t, done)
	unblock()
	awaitFileBoundary(t, service.shutdownDone)
	if publications.Load() != count || len(transcripts.Entries()) != 0 {
		t.Fatal("late file work was published or retained")
	}
	if status := service.CurrentFileTranscription(); status.Transcript != "" || status.CanCopy || status.CanStart {
		t.Fatalf("closed file result remained active: %+v", status)
	}
}
