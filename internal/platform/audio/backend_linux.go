//go:build linux && cgo

package audio

import (
	"context"

	"github.com/gen2brain/malgo"
)

// Prefer the desktop audio server, including PipeWire's PulseAudio-compatible
// service. ALSA is used only if that backend cannot initialize. Do not allow
// miniaudio's null backend: a missing sound system must report an error rather
// than pretend that recording or playback succeeded.
func nativeAudioBackends() []malgo.Backend {
	return []malgo.Backend{malgo.BackendPulseaudio, malgo.BackendAlsa}
}

// Linux audio access is enforced by the user's sound server/device permissions.
// There is no application-owned permission prompt here. This check itself does
// not open a device; the shared capture owner controls explicit start/prepare.
func authorizeMicrophone(ctx context.Context, _ bool) error { return ctx.Err() }
