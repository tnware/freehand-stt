package postprocess

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// S1MiniSystemInstruction is the fixed instruction from the S1-mini v1 model
// contract. It is exported so the renderer can display the effective request
// without maintaining a second copy of model-specific protocol text.
const S1MiniSystemInstruction = "You are a text normalizer for speech-to-text transcripts. The input begins with a control line specifying the styling, structure, and context settings; clean the transcript to match those settings and output only the cleaned text."

// ProfileDescriptor describes a request-building profile supported by
// Freehand. Endpoint model IDs remain independent: the user explicitly chooses
// both the model and the request behavior because compatible servers may use
// arbitrary model names.
type ProfileDescriptor struct {
	Language                string                      `json:"language,omitempty"`
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
	catalog := modelprofile.Profiles(compatibility.Generic, compatibility.Generic, compatibility.Generic).PostProcessing
	return []ProfileDescriptor{
		{
			ID:                      config.PostProcessingPresetGeneric,
			Name:                    catalog[0].Name,
			Description:             catalog[0].Description,
			InstructionEditable:     true,
			RecommendedInstruction:  config.DefaultPostProcessingInstruction,
			MaximumInstructionBytes: config.MaxPromptBytes,
		},
		{
			ID:                config.PostProcessingPresetS1Mini,
			Name:              catalog[1].Name,
			Language:          catalog[1].Language,
			Description:       catalog[1].Description,
			SystemInstruction: S1MiniSystemInstruction,
			Controls: &ProfileControlOptions{
				Styling:   config.S1MiniStylingValues(),
				Structure: config.S1MiniStructureValues(),
				Context:   config.S1MiniContextValues(),
			},
		},
	}
}
