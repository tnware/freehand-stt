package config

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestCleanupOptionsValidatedWhileProcessingDisabled(t *testing.T) {
	for _, options := range []compatibility.CleanupOptions{
		{DisableReasoning: true}, {LimitOutputTokens: true}, {MaxOutputTokens: -1}, {MaxOutputTokens: 65537},
	} {
		cfg := Default()
		cfg.PostProcessing.GenerationOptions = options
		if Validate(cfg) == nil {
			t.Fatal("invalid disabled cleanup options accepted")
		}
	}
	for _, limit := range []int{1, 65536} {
		cfg := Default()
		cfg.PostProcessing.GenerationOptions = compatibility.CleanupOptions{LimitOutputTokens: true, MaxOutputTokens: limit}
		if err := Validate(cfg); err != nil {
			t.Fatal(err)
		}
	}
}
