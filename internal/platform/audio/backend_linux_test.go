//go:build linux && cgo

package audio

import (
	"context"
	"errors"
	"testing"

	"github.com/gen2brain/malgo"
)

func TestLinuxPrefersPulseAudioWithALSAFallback(t *testing.T) {
	backends := nativeAudioBackends()
	if len(backends) != 2 || backends[0] != malgo.BackendPulseaudio || backends[1] != malgo.BackendAlsa {
		t.Fatalf("native backends = %v, want PulseAudio then ALSA only", backends)
	}
	// Callers cannot mutate the backend policy for future device contexts.
	backends[0] = malgo.BackendNull
	if nativeAudioBackends()[0] != malgo.BackendPulseaudio {
		t.Fatal("backend policy was mutated")
	}
}

func TestLinuxAuthorizationPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, request := range []bool{false, true} {
		if err := authorizeMicrophone(ctx, request); !errors.Is(err, context.Canceled) {
			t.Fatalf("authorization request=%v: %v", request, err)
		}
	}
	if err := authorizeMicrophone(context.Background(), false); err != nil {
		t.Fatal(err)
	}
}
