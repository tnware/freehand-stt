//go:build windows || darwin

package audio

import (
	"github.com/gen2brain/malgo"
	"github.com/tnware/freehand-stt/internal/audio"
	"testing"
)

func TestCaptureDeviceSelectionIsExplicitOrSystemDefault(t *testing.T) {
	id := malgo.DeviceID{1, 2, 3}
	devices := []malgo.DeviceInfo{{ID: id, IsDefault: 1}}
	for _, selected := range []string{"", id.String(), "missing-device"} {
		t.Run(selected, func(t *testing.T) {
			cfg, name, release, err := captureDeviceConfig(selected, devices)
			if selected == "missing-device" {
				if err == nil {
					release()
					t.Fatal("missing explicit microphone fell back to default")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			if cfg.SampleRate != audio.SampleRate || cfg.Capture.Format != malgo.FormatS16 || cfg.Capture.Channels != 1 || cfg.Capture.ShareMode != malgo.Shared {
				t.Fatalf("capture format changed: %+v", cfg.Capture)
			}
			if selected == "" {
				if cfg.Capture.DeviceID != nil || name != "System default" {
					t.Fatal("system default was pinned to a physical device")
				}
			} else {
				if cfg.Capture.DeviceID == nil || *(*malgo.DeviceID)(cfg.Capture.DeviceID) != id {
					t.Fatal("explicit microphone ID was not preserved")
				}
			}
		})
	}
}
