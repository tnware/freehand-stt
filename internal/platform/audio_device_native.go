//go:build windows || darwin

package platform

/*
#include <stdlib.h>
*/
import "C"

import (
	"errors"

	"github.com/gen2brain/malgo"
	"github.com/tnware/freehand-stt/internal/audio"
)

// captureDeviceConfig leaves a nil ID for system-default routing. An explicit
// selection must still exist and never silently falls back to another input.
// The caller releases the C ID allocation after InitDevice has copied it.
func captureDeviceConfig(id string, devices []malgo.DeviceInfo) (malgo.DeviceConfig, string, func(), error) {
	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.SampleRate = audio.SampleRate
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = 1
	cfg.Capture.ShareMode = malgo.Shared
	if id == "" {
		return cfg, "System default", func() {}, nil
	}
	for i := range devices {
		if devices[i].ID.String() == id {
			pointer := devices[i].ID.Pointer() // malgo uses C.CBytes, not Go memory.
			cfg.Capture.DeviceID = pointer
			return cfg, devices[i].Name(), func() { C.free(pointer) }, nil
		}
	}
	return cfg, "", nil, errors.New("selected microphone is unavailable")
}
