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
	Voices                []string                   `json:"voices,omitempty"`
	ID                    ID                         `json:"id"`
	Name                  string                     `json:"name"`
	Description           string                     `json:"description"`
	Language              string                     `json:"language,omitempty"`
	AssumeUnknownLanguage string                     `json:"assumeUnknownLanguage,omitempty"`
	ReasoningOffRequired  bool                       `json:"reasoningOffRequired"`
	Capabilities          compatibility.Capabilities `json:"capabilities"`
	Languages             []speechlanguage.Option    `json:"languages,omitempty"`
	RealtimeLanguageHint  bool                       `json:"realtimeLanguageHint,omitempty"`
}

type Catalog struct {
	VoiceTranscription []Profile `json:"voiceTranscription"`
	Realtime           []Profile `json:"realtime"`
	Transcription      []Profile `json:"transcription"`
	PostProcessing     []Profile `json:"postProcessing"`
	Speech             []Profile `json:"speech"`
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
	if p, ok := familyProfile(id, role); ok {
		return p, nil
	}
	if id == Qwen3TTS && role == compatibility.Speech {
		return qwenTTSProfile(), nil
	}
	if id == Qwen3ASR && (role == compatibility.Transcription || role == compatibility.Realtime) {
		return qwenProfile(role), nil
	}
	if role == compatibility.Realtime || (role == compatibility.Transcription && id == Nemotron35) {
		if id != Nemotron35 {
			return Profile{}, errors.New("choose the qualified Nemotron 3.5 streaming model profile")
		}
		return Profile{ID: id, Name: "Nemotron 3.5 ASR streaming", Description: "32 base-model locales and vocabulary boosting. Live text is provisional until finalized.", Languages: NemotronLanguages(), RealtimeLanguageHint: true, Capabilities: compatibility.Capabilities{LanguageHint: true, Realtime: true}}, nil
	}
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
	backend.Realtime = backend.Realtime && model.Realtime
	backend.LanguageHint = backend.LanguageHint && model.LanguageHint
	backend.FileStreaming = backend.FileStreaming && model.FileStreaming
	backend.TranscriptionPrompt = backend.TranscriptionPrompt && model.TranscriptionPrompt
	backend.TranscriptionHotwords = backend.TranscriptionHotwords && model.TranscriptionHotwords
	backend.TranscriptionTemperature = backend.TranscriptionTemperature && model.TranscriptionTemperature
	backend.CleanupOutputLimit = backend.CleanupOutputLimit && model.CleanupOutputLimit
	backend.CleanupDisableReasoning = backend.CleanupDisableReasoning && model.CleanupDisableReasoning
	backend.SpeechSpeed = backend.SpeechSpeed && model.SpeechSpeed
	backend.SpeechInstructions = backend.SpeechInstructions && model.SpeechInstructions
	backend.SpeechLanguage = backend.SpeechLanguage && model.SpeechLanguage
	return backend
}

func Resolve(id ID, backend compatibility.ID, role compatibility.Role) (Contract, error) {
	if (id == Qwen3ASR || id == CohereTranscribe || id == VoxtralRealtime) && backend != compatibility.VLLM {
		return Contract{}, errors.New("this model profile requires the vLLM backend")
	}
	if (id == Nemotron35 || id == ParakeetTDT) && backend != compatibility.NeMoSpeechV1 {
		return Contract{}, errors.New("this model profile requires the NeMo-Speech.cpp backend")
	}
	if id == Qwen3TTS && backend != compatibility.VLLMOmni {
		return Contract{}, errors.New("Qwen3-TTS profile requires the vLLM-Omni backend")
	}
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
	if role == compatibility.Transcription && backend == compatibility.VLLM {
		ids = append(ids, Qwen3ASR, CohereTranscribe, VoxtralRealtime)
	}
	if role == compatibility.Transcription && backend == compatibility.NeMoSpeechV1 {
		ids = append(ids, Nemotron35, ParakeetTDT)
	}
	if role == compatibility.Realtime {
		ids = []ID{Nemotron35, Qwen3ASR, VoxtralRealtime}
	}
	if role == compatibility.Speech && backend == compatibility.VLLMOmni {
		ids = append(ids, Qwen3TTS)
	}
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
	return Catalog{Realtime: append(options(compatibility.NeMoSpeechV1, compatibility.Realtime), options(compatibility.VLLM, compatibility.Realtime)...), Transcription: options(transcription, compatibility.Transcription), PostProcessing: options(cleanup, compatibility.PostProcessing), Speech: options(speech, compatibility.Speech)}
}

// ValidateLanguage preserves the accepted S1-mini policy for explicit and
// reported input. It does not invent a language hint or rewrite a prompt.
func ValidateLanguage(id ID, role compatibility.Role, selected string, detected []string) error {
	if _, ok := familyProfile(id, role); ok {
		return validateFamilyLanguage(id, selected)
	}
	if id == Qwen3ASR && role == compatibility.Transcription {
		return validateQwenLanguage(selected)
	}
	p, err := definition(id, role)
	if err != nil {
		return err
	}
	if id == Nemotron35 {
		// An omitted language delegates automatic detection to the server.
		if selected == "" {
			return nil
		}
		return ValidateNemotron(selected, NemotronOptions{})
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
	if options.Vocabulary != "" || options.VocabularyBoost != 0 {
		if VocabularyMode(id, backend, false) != "speech-contexts" {
			return errors.New("speech contexts require the qualified Nemotron and NeMo profiles")
		}
		if err := ValidateNemotron("auto", NemotronOptions{Vocabulary: options.Vocabulary, Boost: options.VocabularyBoost}); err != nil {
			return err
		}
	}
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

func VoiceProfiles(backend compatibility.ID) []Profile {
	return options(backend, compatibility.Transcription)
}
