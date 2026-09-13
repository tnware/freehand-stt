package config

import (
	"math"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestTranscriptionOptionsValidation(t *testing.T) {
	for _, options := range []TranscriptionOptions{
		{Prompt: strings.Repeat("x", 8193)}, {Prompt: "bad\x00hint"}, {Prompt: string([]byte{0xff})},
		{Temperature: math.NaN()}, {Temperature: math.Inf(1)}, {Temperature: -0.1}, {Temperature: 1.1},
	} {
		cfg := Default()
		cfg.CompatibilityProfile = compatibility.Speaches
		cfg.TranscriptionOptions = options
		if Validate(cfg) == nil {
			t.Fatal("accepted invalid transcription controls")
		}
	}
	cfg := Default()
	cfg.TranscriptionOptions = TranscriptionOptions{Prompt: strings.Repeat("x", 8192), TemperatureOverride: true, Temperature: 1}
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.CompatibilityProfile = compatibility.Speaches
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
}
