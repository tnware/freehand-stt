package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

const (
	MaxPromptBytes = 8 * 1024

	// DefaultPostProcessingInstruction is the recoverable starting point for
	// the editable custom processing profile. It remains ordinary durable user
	// configuration after the first save.
	DefaultPostProcessingInstruction = "Clean this speech-to-text transcript without changing its meaning. Return only the cleaned transcript."
)

type PostProcessingPreset string

const (
	PostProcessingPresetGeneric PostProcessingPreset = PostProcessingPreset(modelprofile.Generic)
	PostProcessingPresetS1Mini  PostProcessingPreset = PostProcessingPreset(modelprofile.S1Mini)
)

var (
	s1MiniStylingValues   = []string{"casual", "semi-casual", "semi-formal", "formal"}
	s1MiniStructureValues = []string{"prose", "lists"}
	s1MiniContextValues   = []string{"general", "email"}
)

// S1MiniStylingValues returns the exact styling vocabulary trained by the
// supported S1-mini v1 profile.
func S1MiniStylingValues() []string {
	return slices.Clone(s1MiniStylingValues)
}

// S1MiniStructureValues returns the exact supported output structures.
func S1MiniStructureValues() []string {
	return slices.Clone(s1MiniStructureValues)
}

// S1MiniContextValues returns the exact supported transcript contexts.
func S1MiniContextValues() []string {
	return slices.Clone(s1MiniContextValues)
}

type PostProcessingSettings struct {
	ManagedInstanceID string                       `json:"managedInstanceID,omitempty"`
	GenerationOptions compatibility.CleanupOptions `json:"generationOptions"`

	CompatibilityProfile compatibility.ID `json:"compatibilityProfile"`
	Enabled              bool             `json:"enabled"`
	BaseURL              string           `json:"baseURL"`
	AllowInsecureHTTP    bool             `json:"allowInsecureHTTP"`
	Model                string           `json:"model"`
	// Preset selects the cleanup model profile and its supported behavior.
	Preset         PostProcessingPreset `json:"preset"`
	SystemPrompt   string               `json:"systemPrompt"`
	Styling        string               `json:"styling"`
	Structure      string               `json:"structure"`
	Context        string               `json:"context"`
	TimeoutSeconds int                  `json:"timeoutSeconds"`
}

// ValidatePostProcessing admits a resolved request, not a durable managed reference.
func ValidatePostProcessing(s PostProcessingSettings) error {
	if s.ManagedInstanceID != "" {
		if err := validateResolvedManagedTransport(s.BaseURL, AuthenticationModeNone, "", nil); err != nil {
			return err
		}
	}
	if err := validatePostProcessingConnection(s.BaseURL, s.AllowInsecureHTTP, s.Model, true); err != nil {
		return err
	}
	return validatePostProcessingOptions(s)
}

func validatePostProcessingOptions(s PostProcessingSettings) error {
	if err := modelprofile.ValidateCleanup(modelprofile.ID(s.Preset), s.CompatibilityProfile, s.GenerationOptions); err != nil {
		return fieldError("postProcessing.preset", "Choose a compatible cleanup model profile and generation options.", err)
	}
	if _, err := compatibility.Resolve(s.CompatibilityProfile, compatibility.PostProcessing); err != nil {
		return fieldError("postProcessing.compatibilityProfile", "Choose a supported cleanup server profile.", err)
	}
	if err := validateTimeout("post-processing request", s.TimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return fieldError("postProcessing.timeoutSeconds", fmt.Sprintf("Enter a cleanup timeout from %d to %d seconds.", MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds), err)
	}
	if strings.TrimSpace(s.Model) == "" || len(s.Model) > 200 {
		return fieldError("postProcessing.model", "Choose a cleanup model of at most 200 characters.", errors.New("post-processing model is required and must be at most 200 characters"))
	}
	if len(s.SystemPrompt) > MaxPromptBytes {
		return fieldError("postProcessing.systemPrompt", fmt.Sprintf("Enter a cleanup system instruction of at most %d bytes.", MaxPromptBytes), fmt.Errorf("post-processing system prompt must be at most %d bytes", MaxPromptBytes))
	}
	if len(s.Styling) > 32 || len(s.Structure) > 32 || len(s.Context) > 32 {
		return fieldError("postProcessing", "Keep each cleanup profile control to at most 32 bytes.", errors.New("post-processing profile controls must be at most 32 bytes"))
	}
	switch s.Preset {
	case PostProcessingPresetGeneric:
		if strings.TrimSpace(s.SystemPrompt) == "" {
			return fieldError("postProcessing.systemPrompt", "Enter a system instruction for the custom cleanup profile.", errors.New("a system instruction is required for the custom post-processing profile"))
		}
	case PostProcessingPresetS1Mini:
		if !oneOf(s.Styling, s1MiniStylingValues...) {
			return fieldError("postProcessing.styling", "Choose a valid S1-mini styling.", errors.New("S1-mini styling is invalid"))
		}
		if !oneOf(s.Structure, s1MiniStructureValues...) {
			return fieldError("postProcessing.structure", "Choose a valid S1-mini structure.", errors.New("S1-mini structure is invalid"))
		}
		if !oneOf(s.Context, s1MiniContextValues...) {
			return fieldError("postProcessing.context", "Choose a valid S1-mini context.", errors.New("S1-mini context is invalid"))
		}
	default:
		return fieldError("postProcessing.preset", "Choose a compatible cleanup model profile and generation options.", errors.New("post-processing preset is invalid"))
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	return slices.Contains(allowed, value)
}
