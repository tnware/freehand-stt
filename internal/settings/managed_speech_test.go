package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"testing"
)

func TestManagedSpeechUnavailableDoesNotReadManualKey(t *testing.T) {
	h, _ := previewHarness()
	s := h.service
	s.cfg.TextToSpeech.Enabled = true
	s.cfg.TextToSpeech.ManagedInstanceID = "stopped-speech"
	if _, err := TextToSpeechProfiles(s).Capture(); !errors.Is(err, ErrManagedUnavailable) {
		t.Fatalf("unavailable speech admitted: %v", err)
	}
	if len(*h.log) != 0 {
		t.Fatal("managed speech read manual credentials")
	}
	s.cfg.TextToSpeech.Enabled = false
	if _, err := TextToSpeechProfiles(s).Capture(); err == nil {
		t.Fatal("disabled speech admitted")
	}
}
func TestManagedSpeechPreviewCannotChangeInstanceModel(t *testing.T) {
	h, d := previewHarness()
	h.service.cfg.TextToSpeech.ManagedInstanceID = "speech"
	h.service.cfg.TextToSpeech.AuthenticationMode = config.AuthenticationModeNone
	if _, err := TextToSpeechProfiles(h.service).CapturePreview(&d); err == nil {
		t.Fatal("preview replaced instance model")
	}
}
