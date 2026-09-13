package settings

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestMacAppearanceIgnoresWindowsMica(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native macOS settings projection")
	}
	cfg := config.Default()
	cfg.UseMica = true
	cfg.AppearanceMode = config.AppearanceModeDark
	s := NewService(nil, cfg, &keyFake{}, nil, nil, func() (bool, string) { return false, "" }, nil, nil, nil, nil, nil, nil)
	got := s.GetSettings()
	if got.MicaActive || got.AppearanceModeActive != config.AppearanceModeDark {
		t.Fatalf("native appearance mica=%v mode=%q", got.MicaActive, got.AppearanceModeActive)
	}
}

func TestSettingsSnapshotReportsNativePlatform(t *testing.T) {
	s := NewService(nil, config.Default(), &keyFake{}, nil, nil, func() (bool, string) { return false, "" }, nil, nil, nil, nil, nil, nil)
	data, err := json.Marshal(s.GetSettings())
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["platform"] != runtime.GOOS {
		t.Fatalf("platform = %v, want %s", fields["platform"], runtime.GOOS)
	}
}
