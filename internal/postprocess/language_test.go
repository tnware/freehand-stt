package postprocess

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
)

func TestS1MiniLanguageAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, selected string
		detected       []string
		blocked        bool
	}{
		{"default assumes English", "", nil, false},
		{"automatic assumes English", "auto", nil, false},
		{"selected English", "en", nil, false},
		{"English region", "en-US", nil, false},
		{"English name", "English", nil, false},
		{"English report", "auto", []string{"en", "English", "eng", "en_GB"}, false},
		{"unknown report", "auto", []string{"und", "unknown"}, false},
		{"selected Spanish", "es", nil, true},
		{"detected Spanish", "auto", []string{"es"}, true},
		{"conflicting report", "en", []string{"French"}, true},
		{"mixed report", "auto", []string{"en", "ja"}, true},
		{"custom selected non-English", "custom-value", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default().PostProcessing
			cfg.Preset = config.PostProcessingPresetS1Mini
			err := ValidateLanguage(cfg, tc.selected, tc.detected)
			if (err != nil) != tc.blocked {
				t.Fatalf("blocked=%t err=%v", tc.blocked, err)
			}
			if err != nil && diagnostics.ErrorKind(err) != "unsupported_language" {
				t.Fatal("language rejection lacks a bounded category")
			}
			cfg.Preset = config.PostProcessingPresetGeneric
			if err = ValidateLanguage(cfg, tc.selected, tc.detected); err != nil {
				t.Fatal("generic cleanup inherited S1-mini restriction")
			}
		})
	}
}
