package shortcut

import (
	"github.com/tnware/freehand-stt/internal/config"
	"strings"
	"testing"
)

func TestRegistrationFailureUsesPlatformNeutralMessage(t *testing.T) {
	cfg := config.Default()
	cfg.ToggleShortcut = "Ctrl+Shift+K"
	native := &globalFake{active: map[string]bool{}, fail: cfg.ToggleShortcut}
	err := New(native, nil, func() {}, func() {}).Configure(cfg)
	if err == nil || !strings.Contains(err.Error(), "operating system") || strings.Contains(err.Error(), "Windows") {
		t.Fatalf("wrong platform in registration error: %v", err)
	}
}
