package filetranscription

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type cancelledAdmissionTransport struct{}

func (cancelledAdmissionTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()
	return nil, r.Context().Err()
}

func TestShutdownIncludesAdmittedFileWorkerBeforeStartPublication(t *testing.T) {
	cfg := config.Default()
	cfg.SetupCompleted = true
	cfg.AuthenticationMode = config.AuthenticationModeNone
	cfg.BaseURL = "https://fixture.invalid/v1"
	cfg.Model = "speech"
	source := settings.Source(func() config.Settings { return cfg })
	client := inference.New()
	client.HTTP.Transport = cancelledAdmissionTransport{}
	entered, gate := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	service := NewService(source, func() (settings.RequestProfile, error) { return settings.RequestProfile{Settings: cfg}, nil }, client, nil, nil, nil, nil, func(status FileTranscriptionStatus) {
		if status.Phase == FileTranscriptionUploading {
			close(entered)
			<-gate
		}
	}, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	if err := service.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { release(); cancel(); _ = service.ServiceShutdown() })
	path := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(path, []byte("RIFF audio"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.selectAudioFile(path); err != nil {
		t.Fatal(err)
	}
	start := make(chan error, 1)
	go func() { start <- service.StartFileTranscription(false) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("start publication was not reached")
	}
	shutdown := make(chan error, 1)
	go func() { shutdown <- service.ServiceShutdown() }()
	// Root cancellation proves shutdown entered while publication is gated.
	service.lifecycleMu.Lock()
	root := service.rootContext
	service.lifecycleMu.Unlock()
	select {
	case <-root.Done():
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel root")
	}
	select {
	case err := <-shutdown:
		t.Fatalf("shutdown omitted admitted worker: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	release()
	for _, done := range []<-chan error{start, shutdown} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("admitted worker did not drain")
		}
	}
}
