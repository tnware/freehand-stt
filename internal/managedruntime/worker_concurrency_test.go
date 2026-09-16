package managedruntime

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/activity"
)

func runningConcurrencyWorker(t *testing.T) *worker {
	t.Helper()
	s := newTestWorker(t, true)
	s.status.Supported = true
	s.adapter = &workerAdapter{installed: true, downloaded: true}
	if err := s.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.ServiceShutdown() })
	waitWorker(t, s, "running")
	return s
}

func awaitConcurrency(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("barrier timed out")
	}
}

func TestSpeechAdmissionFencedAfterIdleCheck(t *testing.T) {
	for _, kind := range []string{"recording", "file"} {
		t.Run(kind, func(t *testing.T) {
			s := runningConcurrencyWorker(t)
			c := activity.New(activity.Sources{})
			checked, resume := make(chan struct{}), make(chan struct{})
			s.checkIdle = func() error { err := c.CheckShortcutCapture(); close(checked); <-resume; return err }
			stopped := make(chan error, 1)
			go func() { stopped <- s.Stop() }()
			awaitConcurrency(t, checked)
			begin := c.BeginRecording
			if kind == "file" {
				begin = c.BeginFileTranscription
			}
			release, err := begin()
			if err != nil {
				close(resume)
				t.Fatal(err)
			}
			_, resolveErr := resolveWorker(s)
			release()
			close(resume)
			if err := <-stopped; err != nil {
				t.Fatal(err)
			}
			waitWorker(t, s, "stopped")
			if resolveErr == nil {
				t.Fatal("speech captured endpoint after idle guard released and before stop transition")
			}
		})
	}
}

func TestSpeechPublicationBeforeIdleCheckRejectsMutation(t *testing.T) {
	for _, kind := range []string{"recording", "file"} {
		t.Run(kind, func(t *testing.T) {
			s := runningConcurrencyWorker(t)
			var active atomic.Bool
			sources := activity.Sources{}
			if kind == "file" {
				sources.FileActive = active.Load
			} else {
				sources.DictationActive = active.Load
			}
			c := activity.New(sources)
			begin := c.BeginRecording
			if kind == "file" {
				begin = c.BeginFileTranscription
			}
			release, err := begin()
			if err != nil {
				t.Fatal(err)
			}
			endpoint, err := resolveWorker(s)
			if err != nil {
				release()
				t.Fatal(err)
			}
			entered := make(chan struct{})
			s.checkIdle = func() error { close(entered); return c.CheckShortcutCapture() }
			stopped := make(chan error, 1)
			go func() { stopped <- s.Stop() }()
			awaitConcurrency(t, entered)
			active.Store(true) // Feature publication occurs before releasing real admission.
			release()
			if err := <-stopped; err == nil {
				t.Fatal("runtime stopped admitted speech")
			}
			got, err := resolveWorker(s)
			if err != nil || got != endpoint {
				t.Fatalf("admitted endpoint lost: %+v %v", got, err)
			}
		})
	}
}
