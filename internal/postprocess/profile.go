package postprocess

import "github.com/tnware/freehand-stt/internal/config"

// S1MiniSystemInstruction is the fixed instruction from the S1-mini v1 model
// contract. It is exported so the renderer can display the effective request
// without maintaining a second copy of model-specific protocol text.
const S1MiniSystemInstruction = "You are a text normalizer for speech-to-text transcripts. The input begins with a control line specifying the styling, structure, and context settings; clean the transcript to match those settings and output only the cleaned text."

// ProfileDescriptor describes a request-building profile supported by
// Freehand. Endpoint model IDs remain independent: the user explicitly chooses
// both the model and the request behavior because compatible servers may use
// arbitrary model names.
type ProfileDescriptor struct {
	ID                      config.PostProcessingPreset `json:"id"`
	Name                    string                      `json:"name"`
	Description             string                      `json:"description"`
	InstructionEditable     bool                        `json:"instructionEditable"`
	RecommendedInstruction  string                      `json:"recommendedInstruction,omitempty"`
	SystemInstruction       string                      `json:"systemInstruction,omitempty"`
	MaximumInstructionBytes int                         `json:"maximumInstructionBytes,omitempty"`
	Controls                *ProfileControlOptions      `json:"controls,omitempty"`
}

// ProfileControlOptions contains the complete trained choice vocabulary for a
// specialized profile. An absent value means the profile has no such control.
type ProfileControlOptions struct {
	Styling   []string `json:"styling,omitempty"`
	Structure []string `json:"structure,omitempty"`
	Context   []string `json:"context,omitempty"`
}

// Profiles returns a fresh copy of the supported processing-profile catalog.
// The custom profile's instruction lives in durable user settings; a built-in
// profile supplies its exact fixed instruction here for transparent UI display.
func Profiles() []ProfileDescriptor {
	return []ProfileDescriptor{
		{
			ID:                      config.PostProcessingPresetGeneric,
			Name:                    "Custom instruction",
			Description:             "Use any OpenAI-compatible chat model with your own transcript-cleanup instruction.",
			InstructionEditable:     true,
			RecommendedInstruction:  config.DefaultPostProcessingInstruction,
			MaximumInstructionBytes: config.MaxPromptBytes,
		},
		{
			ID:                config.PostProcessingPresetS1Mini,
			Name:              "S1-mini by Superwhisper",
			Description:       "Use S1-mini's fixed normalization contract and its trained output controls.",
			SystemInstruction: S1MiniSystemInstruction,
			Controls: &ProfileControlOptions{
				Styling:   config.S1MiniStylingValues(),
				Structure: config.S1MiniStructureValues(),
				Context:   config.S1MiniContextValues(),
			},
		},
	}
}
