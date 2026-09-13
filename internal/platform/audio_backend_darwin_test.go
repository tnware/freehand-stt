//go:build darwin

package platform

import (
 "testing"
 "github.com/gen2brain/malgo"
)

func TestDarwinSelectsCoreAudioWithoutFallback(t *testing.T) {
 if nativeAudioBackend != malgo.BackendCoreaudio {
  t.Fatalf("native backend = %v, want CoreAudio", nativeAudioBackend)
 }
}
