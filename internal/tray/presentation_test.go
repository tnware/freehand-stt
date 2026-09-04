package tray

import (
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/filetranscription"
)

func TestPresentRecordingUsesVoiceActivityWithoutExposingMessage(t *testing.T) {
	state := snapshot{
		dictation: dictation.Status{
			State:     dictation.Recording,
			Message:   "private renderer detail",
			CanCancel: true,
			VADState:  dictation.VADSpeech,
		},
		file: filetranscription.FileTranscriptionStatus{Phase: filetranscription.FileTranscriptionEmpty},
	}
	view := present(state)
	if view.status != "Recording" || view.detail != "Speech detected" {
		t.Fatalf("recording presentation = %#v", view)
	}
	if view.cancel != sourceDictation || view.cancelLabel != "Cancel recording" {
		t.Fatalf("recording controls = %#v", view)
	}
	if strings.Contains(view.status+view.detail+view.tooltip, state.dictation.Message) {
		t.Fatal("renderer message leaked into tray presentation")
	}
}

func TestPresentCopyActionMatchesDisplayedTerminalSource(t *testing.T) {
	state := snapshot{
		dictation: dictation.Status{State: dictation.Failed, CanCopy: true},
		file: filetranscription.FileTranscriptionStatus{
			Phase:   filetranscription.FileTranscriptionCompleted,
			CanCopy: true,
		},
		latest: sourceFile,
	}
	view := present(state)
	if view.status != "Audio transcript ready" || view.copy != sourceFile {
		t.Fatalf("terminal source presentation = %#v", view)
	}

	state.file.Phase = filetranscription.FileTranscriptionUploading
	state.file.CanCancel = true
	view = present(state)
	if view.copy != sourceNone || view.cancel != sourceFile {
		t.Fatalf("active operation controls = %#v", view)
	}
}

func TestPresentSilenceCountdownIsExplicit(t *testing.T) {
	view := present(snapshot{dictation: dictation.Status{
		State:         dictation.Recording,
		CanCancel:     true,
		VADState:      dictation.VADSilence,
		AutoStopState: dictation.AutoStopCountdown,
	}})
	if view.detail != "Silence detected · automatic stop pending" {
		t.Fatalf("detail = %q", view.detail)
	}
}

func TestPresentPostProcessingHasDistinctState(t *testing.T) {
	view := present(snapshot{dictation: dictation.Status{
		State:     dictation.PostProcessing,
		CanCancel: true,
	}})
	if view.status != "Cleaning transcript" || view.detail != "Applying post-processing" {
		t.Fatalf("post-processing presentation = %#v", view)
	}
}

func TestPresentReadyToInsertIsDistinctFromIdle(t *testing.T) {
	view := present(snapshot{dictation: dictation.Status{
		State:     dictation.Ready,
		CanCancel: true,
	}})
	if view.status != "Finishing dictation" || view.detail != "Preparing transcript delivery" {
		t.Fatalf("ready-to-insert presentation = %#v", view)
	}
	if view.status == present(snapshot{dictation: dictation.Status{State: dictation.Idle}}).status {
		t.Fatal("ready-to-insert and idle states have the same tray label")
	}
}

func TestPresentUploadShowsBoundedProgressWithoutFileIdentity(t *testing.T) {
	state := snapshot{
		file: filetranscription.FileTranscriptionStatus{
			Phase:         filetranscription.FileTranscriptionUploading,
			FileName:      "private-recording.wav",
			FileSize:      400,
			BytesUploaded: 101,
			CanCancel:     true,
		},
	}
	view := present(state)
	if view.status != "Uploading audio" || view.detail != "25% uploaded" {
		t.Fatalf("upload presentation = %#v", view)
	}
	if view.cancel != sourceFile || strings.Contains(view.status+view.detail+view.tooltip, state.file.FileName) {
		t.Fatalf("upload controls or privacy = %#v", view)
	}
}

func TestPresentCopyRecoveryUsesLatestTerminalSource(t *testing.T) {
	state := snapshot{
		dictation: dictation.Status{State: dictation.Idle},
		file: filetranscription.FileTranscriptionStatus{
			Phase:      filetranscription.FileTranscriptionFailed,
			CanCopy:    true,
			Transcript: "private partial transcript",
			Message:    "private endpoint failure",
		},
		latest: sourceFile,
	}
	view := present(state)
	if view.status != "Partial transcript available" || view.copy != sourceFile {
		t.Fatalf("recovery presentation = %#v", view)
	}
	all := view.status + view.detail + view.tooltip
	if strings.Contains(all, state.file.Transcript) || strings.Contains(all, state.file.Message) {
		t.Fatal("file content leaked into tray presentation")
	}
}

func TestPresentReadyRetainsBoundedLastActivityAndWindowAction(t *testing.T) {
	view := present(snapshot{
		dictation:  dictation.Status{State: dictation.Idle},
		file:       filetranscription.FileTranscriptionStatus{Phase: filetranscription.FileTranscriptionEmpty},
		last:       "Last dictation completed · 3.2s",
		mainWindow: true,
	})
	if view.status != "Ready" || view.detail != "Last dictation completed · 3.2s" {
		t.Fatalf("ready presentation = %#v", view)
	}
	if view.windowLabel != "Hide Freehand" {
		t.Fatalf("window presentation = %#v", view)
	}
}

func TestActivityDurationsAreStableAndBounded(t *testing.T) {
	started := time.Date(2026, time.August, 30, 17, 0, 0, 0, time.UTC)
	if got := withElapsed("Last dictation completed", started, started.Add(3200*time.Millisecond)); got != "Last dictation completed · 3.2s" {
		t.Fatalf("short duration = %q", got)
	}
	if got := withElapsed("Last audio transcription completed", started, started.Add(2*time.Minute+5*time.Second)); got != "Last audio transcription completed · 2m 05s" {
		t.Fatalf("long duration = %q", got)
	}
	if got := boundedTooltip(strings.Repeat("x", 200)); len([]rune(got)) != 120 {
		t.Fatalf("bounded tooltip rune length = %d", len([]rune(got)))
	}
}

func TestSettledTransitionsIgnoreInitialAndCrossGenerationSnapshots(t *testing.T) {
	active := dictation.Status{State: dictation.Transcribing, Generation: 4}
	if !dictationSettled(active, dictation.Status{State: dictation.Idle, Generation: 4}) {
		t.Fatal("same-generation terminal transition was not settled")
	}
	if dictationSettled(active, dictation.Status{State: dictation.Idle, Generation: 5}) {
		t.Fatal("cross-generation transition was treated as settled")
	}
	if fileSettled(
		filetranscription.FileTranscriptionStatus{Phase: filetranscription.FileTranscriptionEmpty},
		filetranscription.FileTranscriptionStatus{Phase: filetranscription.FileTranscriptionCompleted},
	) {
		t.Fatal("initial file snapshot was treated as settled")
	}
}
