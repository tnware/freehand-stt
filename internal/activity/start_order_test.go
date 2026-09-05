package activity_test

import (
	"sync"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/filetranscription"
	"github.com/tnware/freehand-stt/internal/tts"
)

func TestFileAdmissionSpansPreparation(t *testing.T) {
	gate, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	h := newHarness(t, nil, func() { close(gate); <-release })
	t.Cleanup(unblock)
	file := async(func() error { return h.files.StartFileTranscription(false) })
	entered(t, gate)
	voice := async(func() error { return h.voice.StartRecording(dictation.RecordingToggle) })
	// Keep file preparation gated: no competing capture may escape this window.
	select {
	case err := <-voice:
		t.Fatalf("voice returned during file preparation: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	unblock()
	if err := await(t, file); err != nil {
		t.Fatal(err)
	}
	if err := await(t, voice); err == nil {
		t.Fatal("voice overlapped the admitted file start")
	}
	if h.capture.starts.Load() != 0 || !filetranscription.Active(h.files) {
		t.Fatal("wrong feature acquired the runtime")
	}
}

func TestExistingSpeechIsNotPreemptedByFileTranscription(t *testing.T) {
	h := newHarness(t, nil, nil)
	if err := h.speech.PreviewVoice(); err != nil {
		t.Fatal(err)
	}
	before := h.speech.CurrentStatus()
	stops := h.player.stopped.Load()
	if err := h.files.StartFileTranscription(false); err != nil {
		t.Fatal(err)
	}
	after := h.speech.CurrentStatus()
	if after.Generation != before.Generation || after.Phase != tts.Generating || h.player.stopped.Load() != stops {
		t.Fatal("file start preempted existing speech")
	}
}
