package input

import (
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/hotkey"
)

func TestActivityRejectionsPreserveShortcutResultContract(t *testing.T) {
	for _, state := range []string{"dictation", "file", "shutdown"} {
		t.Run(state, func(t *testing.T) {
			c := activity.New(activity.Sources{DictationActive: func() bool { return state == "dictation" }, FileActive: func() bool { return state == "file" }})
			defer c.Close()
			if state == "shutdown" {
				c.Close()
			}
			native := &shortcutFake{entered: make(chan struct{})}
			guard := &guardFake{}
			s := &Service{activity: c, shortcutCapture: native, shortcutGuard: guard}
			result, err := s.CaptureShortcut(captureRequest(hotkey.ToggleRecording))
			if state == "shutdown" {
				if !errors.Is(err, activity.ErrClosed) {
					t.Fatal("closure not returned", err)
				}
			} else if err != nil || result.Outcome != ShortcutRejected || result.RejectionKind != hotkey.RejectionUnavailable {
				t.Fatalf("result = %#v, error = %v", result, err)
			}
			if guard.suspended || guard.resumed {
				t.Fatal("rejected capture changed shortcuts")
			}
			select {
			case <-native.entered:
				t.Fatal("rejected capture reached native input")
			default:
			}
		})
	}
}
