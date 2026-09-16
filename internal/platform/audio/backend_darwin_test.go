//go:build darwin

package audio

import (
	"github.com/gen2brain/malgo"
	"testing"
)

func TestDarwinSelectsCoreAudioWithoutFallback(t *testing.T) {
	if nativeAudioBackend != malgo.BackendCoreaudio {
		t.Fatalf("native backend = %v, want CoreAudio", nativeAudioBackend)
	}
}
