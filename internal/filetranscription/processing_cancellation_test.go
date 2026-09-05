package filetranscription

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
)

// Formatting occurs when history projects the failed attempt, after the shared
// outcome decision. Pause there to exercise cancellation during finalization.
type finalizationFailure struct {
	entered, release chan struct{}
	once             sync.Once
}

func (e *finalizationFailure) Error() string {
	e.once.Do(func() { close(e.entered) })
	<-e.release
	return "cleanup failed"
}
func (e *finalizationFailure) ProcessWithCredential(context.Context, config.PostProcessingSettings, string, string) (postprocess.Result, error) {
	return postprocess.Result{}, e
}

func TestCancellationDuringFailedProcessingFinalization(t *testing.T) {
	cfg := config.Default()
	cfg.BaseURL = "https://stt.example/v1"
	cfg.Model = "speech"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	cfg.SetupCompleted = true
	cfg.PostProcessing.Enabled = true
	cfg.PostProcessing.BaseURL = "https://cleanup.example/v1"
	cfg.PostProcessing.Model = "cleanup"
	failure := &finalizationFailure{entered: make(chan struct{}), release: make(chan struct{})}
	var release sync.Once
	unblock := func() { release.Do(func() { close(failure.release) }) }
	transcripts := history.NewStore(true, nil)
	client := inference.New()
	client.HTTP.Transport = processingTransport(func(r *http.Request) (*http.Response, error) {
		if r.Body != nil {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"text":"raw transcript"}`)), Request: r}, nil
	})
	service := NewService(settings.Source(func() config.Settings { return cfg }), settings.ProfileSource(func() (settings.RequestProfile, error) { return settings.RequestProfile{Settings: cfg}, nil }), client, failure, transcripts, nil, nil, nil, nil, nil, nil, nil)
	t.Cleanup(func() { unblock(); _ = service.ServiceShutdown() })
	path := filepath.Join(t.TempDir(), "recording.wav")
	if err := os.WriteFile(path, []byte("RIFF audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.selectAudioFile(path); err != nil {
		t.Fatal(err)
	}
	if err := service.StartFileTranscription(false); err != nil {
		t.Fatal(err)
	}
	select {
	case <-failure.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("processing did not reach finalization")
	}
	if err := service.CancelFileTranscription(); err != nil {
		t.Fatal(err)
	}
	unblock()
	done := make(chan struct{})
	go func() { service.workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled run did not finish")
	}
	status := service.CurrentFileTranscription()
	if status.Phase != FileTranscriptionSelected || status.Transcript != "" || status.CanCopy {
		t.Fatalf("cancelled status = %+v", status)
	}
	entries := transcripts.Entries()
	if len(entries) != 1 || entries[0].Outcome != history.HistoryCancelled || entries[0].RawText != "raw transcript" || entries[0].ProcessedText != "" {
		t.Fatalf("cancelled history = %+v", entries)
	}
	// The cleanup had already failed before cancellation; its stage result and
	// the subsequently cancelled workflow are separate truthful outcomes.
	if entries[0].ProcessingStatus != history.HistoryProcessingFailed {
		t.Fatal("late cancellation rewrote the completed processing decision")
	}
}
