package postprocess

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

// ValidateLanguage is admission policy for the existing English-only S1-mini
// preset. Unknown language assumes English by product choice. Any explicit or
// reported non-English input prevents a cleanup request and preserves raw text.
// It never adds an untrained language token to S1-mini's prompt.
func ValidateLanguage(cfg config.PostProcessingSettings, selected string, detected []string) error {
	if cfg.Preset != config.PostProcessingPresetS1Mini {
		return nil
	}
	mismatch := !speechlanguage.Unspecified(selected) && !speechlanguage.English(selected)
	for _, value := range detected {
		if !speechlanguage.Unspecified(value) && !speechlanguage.English(value) {
			mismatch = true
		}
	}
	if mismatch {
		return &inference.Error{Kind: "unsupported_language", Message: "S1-mini supports English only; raw transcript preserved"}
	}
	return nil
}
