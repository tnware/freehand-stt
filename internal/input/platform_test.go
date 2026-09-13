package input

import (
	"github.com/tnware/freehand-stt/internal/hotkey"
	"strings"
	"testing"
)

func TestCaptureSuccessUsesPlatformNeutralMessage(t *testing.T) {
	chord, _ := hotkey.Parse("Ctrl+Shift+K")
	s := &Service{shortcutCapture: &shortcutFake{chord: chord}, shortcutGuard: &guardFake{}}
	got, err := s.CaptureShortcut(captureRequest(hotkey.HoldToTalk))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Message, "Windows") || !strings.Contains(got.Message, "register") {
		t.Fatalf("wrong platform in capture help: %q", got.Message)
	}
}
