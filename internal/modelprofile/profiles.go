// Package modelprofile owns explicit, qualified model behavior independently
// of server adapters. Generic is a permissive baseline, not model discovery.
package modelprofile

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

type ID string

const (
	Generic ID = "generic"
	S1Mini  ID = "s1-mini"
)

// Profile is renderer-safe metadata for a model/backend combination. Its
// capabilities are the intersection, never an assertion about an inventory ID.
type Profile struct {
	ID                    ID                         `json:"id"`
	Name                  string                     `json:"name"`
	Description           string                     `json:"description"`
	Language              string                     `json:"language,omitempty"`
	AssumeUnknownLanguage string                     `json:"assumeUnknownLanguage,omitempty"`
	ReasoningOffRequired  bool                       `json:"reasoningOffRequired"`
	Capabilities          compatibility.Capabilities `json:"capabilities"`
}

type Catalog struct {
	Transcription  []Profile `json:"transcription"`
	PostProcessing []Profile `json:"postProcessing"`
	Speech         []Profile `json:"speech"`
}

// Contract retains the backend path/response encoding, but narrows request
// capabilities through a model behavior profile.
type Contract struct {
	Profile
	Backend compatibility.Contract
}

func Effective(id ID) ID {
	if id == "" {
		return Generic
	}
	return id
}

func definition(id ID, role compatibility.Role) (Profile, error) {
	id = Effective(id)
	p := Profile{ID: id, Name: "Generic"}
	switch role {
	case compatibility.Transcription:
		p.Description = "Standard transcription behavior. Language, context, temperature, and streaming depend on your server and model; no model-specific support is assumed."
		p.Capabilities = compatibility.Capabilities{LanguageHint: true, FileStreaming: true, TranscriptionPrompt: true, TranscriptionHotwords: true, TranscriptionTemperature: true}
	case compatibility.PostProcessing:
		p.Description = "Standard chat cleanup with your own instruction. Generation controls depend on your backend and model."
		p.Capabilities = compatibility.Capabilities{CleanupOutputLimit: true, CleanupDisableReasoning: true}
	case compatibility.Speech:
		p.Description = "Standard WAV speech generation with a provider voice ID. Available voices, languages, and speed depend on your server and model."
		p.Capabilities = compatibility.Capabilities{SpeechSpeed: true}
	default:
		return Profile{}, errors.New("model profile operation is invalid")
	}
	if id == Generic {
		return p, nil
	}
	if role == compatibility.PostProcessing && id == S1Mini {
		p.Name = "S1-mini by Superwhisper"
		p.Description = "English-only cleanup with a fixed prompt and trained output controls. Reasoning must be off; unknown input language is assumed English."
		p.Language = "en"
		p.AssumeUnknownLanguage = "en"
		p.ReasoningOffRequired = true
		return p, nil
	}
	return Profile{}, errors.New("model profile is not supported for this operation")
}

// constrain preserves transport facts (routes/event encoding/server-loaded
// models). Only user-facing model capabilities participate in the intersection.
func constrain(backend, model compatibility.Capabilities) compatibility.Capabilities {
	backend.LanguageHint = backend.LanguageHint && model.LanguageHint
	backend.FileStreaming = backend.FileStreaming && model.FileStreaming
	backend.TranscriptionPrompt = backend.TranscriptionPrompt && model.TranscriptionPrompt
	backend.TranscriptionHotwords = backend.TranscriptionHotwords && model.TranscriptionHotwords
	backend.TranscriptionTemperature = backend.TranscriptionTemperature && model.TranscriptionTemperature
	backend.CleanupOutputLimit = backend.CleanupOutputLimit && model.CleanupOutputLimit
	backend.CleanupDisableReasoning = backend.CleanupDisableReasoning && model.CleanupDisableReasoning
	backend.SpeechSpeed = backend.SpeechSpeed && model.SpeechSpeed
	return backend
}

func Resolve(id ID, backend compatibility.ID, role compatibility.Role) (Contract, error) {
	p, err := definition(id, role)
	if err != nil {
		return Contract{}, err
	}
	wire, err := compatibility.Resolve(backend, role)
	if err != nil {
		return Contract{}, err
	}
	p.Capabilities = constrain(wire.Capabilities, p.Capabilities)
	wire.Capabilities = p.Capabilities
	return Contract{Profile: p, Backend: wire}, nil
}

func options(backend compatibility.ID, role compatibility.Role) []Profile {
	ids := []ID{Generic}
	if role == compatibility.PostProcessing {
		ids = append(ids, S1Mini)
	}
	result := make([]Profile, 0, len(ids))
	for _, id := range ids {
		if p, err := Resolve(id, backend, role); err == nil {
			result = append(result, p.Profile)
		}
	}
	return result
}

func Profiles(transcription, cleanup, speech compatibility.ID) Catalog {
	return Catalog{Transcription: options(transcription, compatibility.Transcription), PostProcessing: options(cleanup, compatibility.PostProcessing), Speech: options(speech, compatibility.Speech)}
}

// ValidateLanguage preserves the accepted S1-mini policy for explicit and
// reported input. It does not invent a language hint or rewrite a prompt.
func ValidateLanguage(id ID, role compatibility.Role, selected string, detected []string) error {
	p, err := definition(id, role)
	if err != nil {
		return err
	}
	if p.Language == "" {
		return nil
	}
	accepts := func(value string) bool {
		if speechlanguage.Unspecified(value) {
			return p.AssumeUnknownLanguage == p.Language
		}
		return p.Language == "en" && speechlanguage.English(value)
	}
	if !accepts(selected) {
		return errors.New("model profile does not support the input language")
	}
	for _, value := range detected {
		if !accepts(value) {
			return errors.New("model profile does not support the input language")
		}
	}
	return nil
}

func ValidateTranscription(id ID, backend compatibility.ID, language string, options compatibility.TranscriptionOptions) error {
	c, err := Resolve(id, backend, compatibility.Transcription)
	if err != nil {
		return err
	}
	if err = compatibility.ValidateTranscriptionOptions(backend, options); err != nil {
		return err
	}
	if (options.Prompt != "" && !c.Capabilities.TranscriptionPrompt) || (options.Hotwords != "" && !c.Capabilities.TranscriptionHotwords) || (options.TemperatureOverride && !c.Capabilities.TranscriptionTemperature) {
		return errors.New("transcription options are unavailable for this model profile")
	}
	if !speechlanguage.Unspecified(language) && !c.Capabilities.LanguageHint {
		return errors.New("language hints are unavailable for this model profile")
	}
	return ValidateLanguage(id, compatibility.Transcription, language, nil)
}

func ValidateCleanup(id ID, backend compatibility.ID, options compatibility.CleanupOptions) error {
	c, err := Resolve(id, backend, compatibility.PostProcessing)
	if err != nil {
		return err
	}
	if err = compatibility.ValidateCleanupOptions(backend, options); err != nil {
		return err
	}
	if (options.LimitOutputTokens && !c.Capabilities.CleanupOutputLimit) || (options.DisableReasoning && !c.Capabilities.CleanupDisableReasoning) {
		return errors.New("cleanup options are unavailable for this model profile")
	}
	return nil
}

func ValidateSpeech(id ID, backend compatibility.ID, speed float64) error {
	c, err := Resolve(id, backend, compatibility.Speech)
	if err != nil {
		return err
	}
	if speed != 0 && speed != 1 && !c.Capabilities.SpeechSpeed {
		return errors.New("speech speed is unavailable for this model profile")
	}
	return nil
}
