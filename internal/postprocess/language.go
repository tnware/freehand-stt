package postprocess

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// ValidateLanguage is admission policy for the existing English-only S1-mini
// preset. Unknown language assumes English by product choice. Any explicit or
// reported non-English input prevents a cleanup request and preserves raw text.
// It never adds an untrained language token to S1-mini's prompt.
func ValidateLanguage(cfg config.PostProcessingSettings, selected string, detected []string) error {
	if err := modelprofile.ValidateLanguage(modelprofile.ID(cfg.Preset), compatibility.PostProcessing, selected, detected); err != nil {
		return &inference.Error{Kind: "unsupported_language", Message: "S1-mini supports English only; raw transcript preserved"}
	}

	return nil
}
