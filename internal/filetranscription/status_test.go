package filetranscription

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestInitialFileCapabilityEventsMatchEmptyBindingSnapshot(t *testing.T) {
	var events []FileTranscriptionStatus
	service := NewService(nil, nil, nil, nil, nil, nil,
		func() (string, error) { return "", nil },
		func(status FileTranscriptionStatus) { events = append(events, status) },
		nil, nil, nil)
	assertEmpty := func(status FileTranscriptionStatus) {
		t.Helper()
		payload, err := json.Marshal(status)
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(payload, &wire); err != nil {
			t.Fatal(err)
		}
		if wire["phase"] != "empty" {
			t.Fatalf("initial renderer payload phase = %v, want empty", wire["phase"])
		}
		if status.Generation != 0 || status.FileName != "" || status.FileSize != 0 || status.CanStart || status.CanCancel || status.CanCopy || service.fileSelection != nil {
			t.Fatal("initial status invented a selection or file action")
		}
	}
	assertEmpty(service.CurrentFileTranscription())
	// Settings changes can publish after the initial binding response. Both
	// paths must describe the same empty selection, including repeated refreshes.
	for range 2 {
		ApplySettings(service, config.Default())
		if len(events) == 0 {
			t.Fatal("settings refresh did not publish status")
		}
		event := events[len(events)-1]
		assertEmpty(event)
		if !reflect.DeepEqual(event, service.CurrentFileTranscription()) {
			t.Fatal("settings event differs from binding snapshot")
		}
	}
	cancelled, err := service.ChooseAudioFile()
	if err != nil {
		t.Fatal(err)
	}
	assertEmpty(cancelled)
	service.pickAudioFile = func() (string, error) { return "", errors.New("picker failed") }
	if _, err := service.ChooseAudioFile(); err == nil {
		t.Fatal("picker failure was not reported")
	}
	assertEmpty(service.CurrentFileTranscription())
}

func TestFileSnapshotPreservesEveryEstablishedPhase(t *testing.T) {
	for _, phase := range []FileTranscriptionPhase{
		FileTranscriptionEmpty, FileTranscriptionSelected, FileTranscriptionUploading,
		FileTranscriptionProcessing, FileTranscriptionStreaming, FileTranscriptionCompleted,
		FileTranscriptionFailed, FileTranscriptionCancelling,
	} {
		t.Run(string(phase), func(t *testing.T) {
			stored := FileTranscriptionStatus{
				Generation: 3, Phase: phase, FileName: "selected.wav", FileSize: 42,
				CanStart: true, CanCancel: true, CanCopy: true, Message: "retained state",
				Transcript: "partial transcript", TranscriptRevision: 2,
			}
			selection := &audioFileSelection{path: "/private/selected.wav"}
			service := &Service{fileStatus: stored, fileSelection: selection, fileTranscriptRevision: 2}
			_, _ = service.fileTranscript.WriteString(stored.Transcript)
			if got := service.CurrentFileTranscription(); !reflect.DeepEqual(got, stored) {
				t.Fatalf("snapshot changed established %s status", phase)
			}
			if service.fileSelection != selection {
				t.Fatal("snapshot changed backend selection")
			}
		})
	}
}
