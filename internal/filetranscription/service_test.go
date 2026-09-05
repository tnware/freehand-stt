package filetranscription

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
)

type lateSuccessProcessor struct {
	started chan struct{}
	release chan struct{}
}

func (p *lateSuccessProcessor) ProcessWithCredential(context.Context, config.PostProcessingSettings, string, string) (postprocess.Result, error) {
	close(p.started)
	<-p.release
	return postprocess.Result{Text: "late processed transcript"}, nil
}

func TestChooseAudioFileOwnsPickerGrantAndReturnsOnlyMetadata(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "private meeting.wav")
	if err := os.WriteFile(path, []byte("RIFF"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := &Service{pickAudioFile: func() (string, error) { return path, nil }}
	selected, err := service.ChooseAudioFile()
	if err != nil {
		t.Fatal(err)
	}
	if selected.FileName != "private meeting.wav" || selected.FileSize != 4 || !selected.CanStart {
		t.Fatalf("selected = %#v", selected)
	}
	if strings.Contains(fmt.Sprintf("%#v", selected), directory) {
		t.Fatalf("renderer status exposed the selected path: %#v", selected)
	}
	if service.fileSelection == nil || service.fileSelection.path != path {
		t.Fatal("Go did not retain the native picker grant")
	}
}

func TestBoundFileSelectionHasNoRendererPathArgument(t *testing.T) {
	serviceType := reflect.TypeOf(&Service{})
	if _, exposed := serviceType.MethodByName("SelectAudioFile"); exposed {
		t.Fatal("renderer-controlled path method is exported")
	}
	choose, exposed := serviceType.MethodByName("ChooseAudioFile")
	if !exposed || choose.Type.NumIn() != 1 {
		t.Fatalf("ChooseAudioFile accepts %d renderer arguments, want 0", choose.Type.NumIn()-1)
	}
}

func TestChooseAudioFileCancellationPreservesSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kept.wav")
	if err := os.WriteFile(path, []byte("RIFF"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	selected, err := service.selectAudioFile(path)
	if err != nil {
		t.Fatal(err)
	}
	service.pickAudioFile = func() (string, error) { return "", nil }
	after, err := service.ChooseAudioFile()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, selected) || service.fileSelection == nil || service.fileSelection.path != path {
		t.Fatalf("cancelled picker changed selection: before=%#v after=%#v", selected, after)
	}
}

func TestStreamingDeltasAreIncrementalAndSnapshotsRemainRecoverable(t *testing.T) {
	statusPublishes := 0
	var received strings.Builder
	deltaCount := 0
	service := &Service{
		fileStatus: FileTranscriptionStatus{
			Generation: 7,
			Phase:      FileTranscriptionStreaming,
			Streaming:  true,
		},
		fileChanged: func(FileTranscriptionStatus) { statusPublishes++ },
		fileDelta: func(delta FileTranscriptionDelta) {
			deltaCount++
			if delta.Generation != 7 || delta.Revision != uint64(deltaCount) {
				t.Errorf("delta %d = %#v", deltaCount, delta)
			}
			_, _ = received.WriteString(delta.Text)
		},
	}

	service.appendFileDelta(6, "stale")
	const chunks = 4096
	for range chunks {
		service.appendFileDelta(7, "word ")
	}

	want := strings.Repeat("word ", chunks)
	if statusPublishes != 0 {
		t.Fatalf("full status published %d times while streaming", statusPublishes)
	}
	if deltaCount != chunks || received.String() != want {
		t.Fatalf("received %d deltas and %d bytes, want %d deltas and %d bytes", deltaCount, received.Len(), chunks, len(want))
	}
	snapshot := service.CurrentFileTranscription()
	if snapshot.Transcript != want || snapshot.TranscriptRevision != chunks {
		t.Fatalf("recovery snapshot revision=%d bytes=%d", snapshot.TranscriptRevision, len(snapshot.Transcript))
	}

	service.fileMu.Lock()
	service.fileStatus.Phase = FileTranscriptionCompleted
	service.replaceFileTranscriptLocked("authoritative final transcript")
	final := service.snapshotFileStatusLocked()
	service.fileMu.Unlock()
	if final.Transcript != "authoritative final transcript" || final.TranscriptRevision != chunks+1 {
		t.Fatalf("final reconciliation = %#v", final)
	}
}

func TestStreamingTranscriptLimitPublishesExplicitPartialState(t *testing.T) {
	var status FileTranscriptionStatus
	service := &Service{
		fileStatus: FileTranscriptionStatus{
			Generation: 9,
			Phase:      FileTranscriptionStreaming,
			Streaming:  true,
		},
		fileChanged: func(value FileTranscriptionStatus) { status = value },
	}
	_, _ = service.fileTranscript.WriteString(strings.Repeat("x", maxFileTranscriptBytes-1))

	service.appendFileDelta(9, "too large")

	if !service.fileTranscriptLimitHit || service.fileTranscript.Len() != maxFileTranscriptBytes-1 {
		t.Fatalf("limit state hit=%v bytes=%d", service.fileTranscriptLimitHit, service.fileTranscript.Len())
	}
	if !strings.Contains(status.Message, "8 MiB safety limit") || len(status.Transcript) != maxFileTranscriptBytes-1 {
		t.Fatalf("published status = %#v", status)
	}
}

func TestCancellationOwnsLatePostProcessingSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"text":"raw transcript"}`))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "recording.wav")
	if err := os.WriteFile(path, []byte("RIFF audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.BaseURL = server.URL + "/v1"
	cfg.Model = "speech/stt"
	cfg.AuthenticationMode = config.AuthenticationModeNone
	cfg.SetupCompleted = true
	cfg.HistoryEnabled = true
	cfg.PostProcessing.Enabled = true
	cfg.PostProcessing.BaseURL = "https://processor.example/v1"
	cfg.PostProcessing.Model = "cleanup"
	processor := &lateSuccessProcessor{started: make(chan struct{}), release: make(chan struct{})}
	transcripts := history.NewStore(true, nil)
	service := NewService(
		settings.Source(func() config.Settings { return cfg }),
		settings.ProfileSource(func() (settings.RequestProfile, error) {
			return settings.RequestProfile{Settings: cfg}, nil
		}),
		&inference.Client{HTTP: server.Client()},
		processor,
		transcripts,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	if _, err := service.selectAudioFile(path); err != nil {
		t.Fatal(err)
	}
	if err := service.StartFileTranscription(false); err != nil {
		t.Fatal(err)
	}
	select {
	case <-processor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("post-processing did not start")
	}
	service.fileMu.Lock()
	done := service.fileDone
	service.fileMu.Unlock()
	if err := service.CancelFileTranscription(); err != nil {
		t.Fatal(err)
	}
	close(processor.release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled transcription did not finish")
	}

	status := service.CurrentFileTranscription()
	if status.Phase != FileTranscriptionSelected || status.Transcript != "" || status.CanCopy {
		t.Fatalf("cancelled status = %#v", status)
	}
	entries := transcripts.Entries()
	if len(entries) != 1 {
		t.Fatalf("history entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Text != "raw transcript" || entry.ProcessedText != "" || entry.ProcessingStatus != history.HistoryProcessingCancelled || entry.Outcome != history.HistoryCancelled {
		t.Fatalf("cancelled history entry = %#v", entry)
	}
}
