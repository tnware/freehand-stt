package activity

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func result(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("admission did not settle")
		return nil
	}
}

func TestAdmissionRulesReadOwnersWithoutCaching(t *testing.T) {
	for _, state := range []string{"idle", "dictation", "file", "both"} {
		t.Run(state, func(t *testing.T) {
			voice, file := state == "dictation" || state == "both", state == "file" || state == "both"
			stops := 0
			c := New(Sources{DictationActive: func() bool { return voice }, FileActive: func() bool { return file }, StopPlayback: func() error { stops++; return nil }})
			defer c.Close()
			for _, test := range []struct {
				name    string
				begin   func() (func(), error)
				blocked bool
			}{
				{"recording", c.BeginRecording, file}, {"file", c.BeginFileTranscription, voice}, {"playback", c.BeginPlayback, voice || file},
			} {
				release, err := test.begin()
				if (err != nil) != test.blocked {
					t.Fatalf("%s blocked=%v error=%v", test.name, test.blocked, err)
				}
				if release != nil {
					release()
					release()
				}
			}
			if (c.CheckShortcutCapture() != nil) != (voice || file) {
				t.Fatal("shortcut admission disagrees with owners")
			}
			if (stops == 0) != file {
				t.Fatal("recording preempted playback despite active file, or skipped preemption")
			}
			voice, file = false, false
			release, err := c.BeginPlayback()
			if err != nil {
				t.Fatal("cached stale activity", err)
			}
			release()
		})
	}
}

func TestCompetingStartReadsPublishedStateAfterAdmissionRelease(t *testing.T) {
	var file atomic.Bool
	c := New(Sources{FileActive: file.Load})
	defer c.Close()
	release, err := c.BeginFileTranscription()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		r, e := c.BeginRecording()
		if r != nil {
			r()
		}
		done <- e
	}()
	// A successful file start must publish its own activity before release.
	file.Store(true)
	release()
	if result(t, done) == nil {
		t.Fatal("competing recording was admitted")
	}
	file.Store(false)
	r, err := c.BeginRecording()
	if err != nil {
		t.Fatal(err)
	}
	r()
}

func TestFailedPreemptionReleasesAdmission(t *testing.T) {
	c := New(Sources{StopPlayback: func() error { return errors.New("private native detail") }})
	defer c.Close()
	release, err := c.BeginRecording()
	if release != nil || err == nil || err.Error() != "speech playback could not be stopped before recording" {
		t.Fatal("failed preemption admitted capture or exposed native error")
	}
	done := make(chan error, 1)
	go func() {
		r, e := c.BeginFileTranscription()
		if r != nil {
			r()
		}
		done <- e
	}()
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestCloseWakesWaitersWithoutWaitingForPreemption(t *testing.T) {
	entered, unblock := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(unblock) }) }
	t.Cleanup(release)
	c := New(Sources{StopPlayback: func() error { close(entered); <-unblock; return nil }})
	done := make(chan error, 1)
	go func() {
		r, e := c.BeginRecording()
		if r != nil {
			r()
		}
		done <- e
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		release()
		t.Fatal("preemption not reached")
	}
	waiting := make(chan error, 1)
	go func() {
		r, e := c.BeginPlayback()
		if r != nil {
			r()
		}
		waiting <- e
	}()
	c.Close()
	c.Close()
	// Closure must wake other callers while preemption is still blocked.
	waitErr := result(t, waiting)
	release()
	if !errors.Is(waitErr, ErrClosed) {
		t.Fatal("queued playback survived closure")
	}
	if !errors.Is(result(t, done), ErrClosed) {
		t.Fatal("late preemption admitted capture after closure")
	}
	for _, begin := range []func() (func(), error){c.BeginRecording, c.BeginFileTranscription, c.BeginPlayback} {
		r, e := begin()
		if r != nil || !errors.Is(e, ErrClosed) {
			t.Fatal("closed coordinator admitted a start")
		}
	}
	if !errors.Is(c.CheckShortcutCapture(), ErrClosed) {
		t.Fatal("closed coordinator admitted shortcut capture")
	}
}
