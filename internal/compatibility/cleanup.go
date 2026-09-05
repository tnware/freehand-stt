package compatibility

import "errors"

// MaxCleanupOutputTokens bounds Freehand's optional request override, not the
// model's context window or the server's own generation limit.
const MaxCleanupOutputTokens = 65536

// CleanupOptions contains only values so each job owns a coherent snapshot.
// Disabled limits may retain a valid number locally, but send no request field.
type CleanupOptions struct {
	LimitOutputTokens bool `json:"limitOutputTokens"`
	MaxOutputTokens   int  `json:"maxOutputTokens"`
	DisableReasoning  bool `json:"disableReasoning"`
}

func ValidateCleanupOptions(id ID, options CleanupOptions) error {
	contract, err := Resolve(id, PostProcessing)
	if err != nil {
		return err
	}
	if options.MaxOutputTokens < 0 || options.MaxOutputTokens > MaxCleanupOutputTokens || (options.LimitOutputTokens && options.MaxOutputTokens == 0) {
		return errors.New("cleanup output limit must be between 1 and 65536 tokens when enabled; a disabled limit may also be zero")
	}
	if options.LimitOutputTokens && !contract.Capabilities.CleanupOutputLimit {
		return errors.New("cleanup output limits are unavailable for this profile")
	}
	if options.DisableReasoning && !contract.Capabilities.CleanupDisableReasoning {
		return errors.New("disabling cleanup reasoning requires the llama.cpp profile; turn off the override before choosing another profile")
	}
	return nil
}
