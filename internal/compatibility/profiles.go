// Package compatibility defines the bounded server contracts implemented by
// Freehand. A selected contract is configuration, never inferred from a URL or
// model inventory. Capabilities describe client support, not every server model.
//go:generate go run ../../build/scripts/compatibility -out ../../site/src/data/compatibility.generated.json

package compatibility

import "errors"

type ID string

const (
	Generic        ID = "generic"
	Speaches       ID = "speaches"
	LlamaCPP       ID = "llama-cpp"
	OpenAI         ID = "openai"
	LocalAI        ID = "localai"
	WhisperCPP     ID = "whisper-cpp"
	VLLM           ID = "vllm"
	VLLMOmni       ID = "vllm-omni"
	KokoroFastAPI  ID = "kokoro-fastapi"
	OpenedAISpeech ID = "openedai-speech"
)

type Role string

const (
	Transcription  Role = "transcription"
	PostProcessing Role = "post-processing"
	Speech         Role = "speech"
)

// Capabilities are the implemented wire contract. Model-specific advanced
// parameters must acquire their own qualified rules before being exposed.
type Capabilities struct {
	CleanupOutputLimit          bool `json:"cleanupOutputLimit"`
	CleanupDisableReasoning     bool `json:"cleanupDisableReasoning"`
	FileStreaming               bool `json:"fileStreaming"`
	TypedTranscriptionEvents    bool `json:"typedTranscriptionEvents"`
	LegacyTranscriptionSegments bool `json:"legacyTranscriptionSegments"`
	LanguageHint                bool `json:"languageHint"`
	SpeechSpeed                 bool `json:"speechSpeed"`
	TranscriptionPrompt         bool `json:"transcriptionPrompt"`
	TranscriptionHotwords       bool `json:"transcriptionHotwords"`
	TranscriptionTemperature    bool `json:"transcriptionTemperature"`
}

type Profile struct {
	ID           ID           `json:"id"`
	Label        string       `json:"label"`
	Available    bool         `json:"available"`
	Description  string       `json:"description"`
	Capabilities Capabilities `json:"capabilities"`
}

type Catalog struct {
	Transcription  []Profile `json:"transcription"`
	PostProcessing []Profile `json:"postProcessing"`
	Speech         []Profile `json:"speech"`
}

// Contract remains backend-owned; the renderer receives only Profile metadata.
type Contract struct {
	Profile
	Path string
}

// Effective preserves the zero-value contract used by older callers/settings.
func Effective(id ID) ID {
	if id == "" {
		return Generic
	}
	return id
}

func Profiles() Catalog {
	return Catalog{Transcription: options(Transcription), PostProcessing: options(PostProcessing), Speech: options(Speech)}
}

func options(role Role) []Profile {
	caps := Capabilities{}
	genericDescription := ""
	switch role {
	case Transcription:
		caps = Capabilities{FileStreaming: true, TypedTranscriptionEvents: true, LegacyTranscriptionSegments: true, LanguageHint: true, TranscriptionPrompt: true, TranscriptionTemperature: true}
		genericDescription = "Completed transcription and optional file streaming. Preserves support for typed events and legacy text segments; streaming and language hints depend on the selected model."
	case PostProcessing:
		caps.CleanupOutputLimit = true
		genericDescription = "Text chat completions with system/user messages. Choose the cleanup prompt preset separately."
	case Speech:
		caps.SpeechSpeed = true
		genericDescription = "Buffered WAV speech with a voice ID and speed. The server must return PCM16 audio; model support varies."
	default:
		return []Profile{}
	}
	result := []Profile{{ID: Generic, Label: "Generic OpenAI-compatible", Available: true, Description: genericDescription, Capabilities: caps}}
	if role == Transcription {
		caps.TranscriptionHotwords = true
		result = append(result, Profile{ID: Speaches, Label: "Speaches", Available: true, Description: "Completed transcription, typed file events, and older Speaches text segments that finish at end of stream. Language and streaming support depend on the model and server version.", Capabilities: caps})
	} else if role == Speech {
		result = append(result, Profile{ID: Speaches, Label: "Speaches", Available: true, Description: "Buffered PCM16 WAV speech using the installed model and voice IDs. Speed support depends on the model.", Capabilities: caps})
	} else {
		caps.CleanupDisableReasoning = true
		result = append(result, Profile{ID: LlamaCPP, Label: "llama.cpp", Available: true, Description: "Text cleanup with an optional output-token limit and disable-reasoning override. Reasoning control requires a compatible llama.cpp build and model template; S1-mini remains a separate prompt preset.", Capabilities: caps})
	}
	planned := func(id ID, label, reason string) {
		result = append(result, Profile{ID: id, Label: label, Description: "Dedicated profile not implemented. " + reason})
	}
	planned(OpenAI, "OpenAI hosted", "Model-specific fields and limits need qualification.")
	planned(LocalAI, "LocalAI", "Backend-specific capabilities need qualification.")
	switch role {
	case Transcription:
		planned(WhisperCPP, "whisper.cpp", "Native routing and server-loaded model behavior need an adapter.")
		planned(VLLM, "vLLM", "Transcription streaming needs its own response decoder.")
	case PostProcessing:
		planned(VLLM, "vLLM", "The dedicated text processing contract needs qualification.")
	case Speech:
		planned(VLLMOmni, "vLLM-Omni", "Model-specific voice inputs and audio output need qualification.")
		planned(KokoroFastAPI, "Kokoro-FastAPI", "Voice, speed, and WAV output need qualification.")
		planned(OpenedAISpeech, "openedai-speech", "Server-configured voices and WAV output need qualification.")
	}
	return result
}

func Resolve(id ID, role Role) (Contract, error) {
	id = Effective(id)
	for _, profile := range options(role) {
		if profile.ID != id {
			continue
		}
		if !profile.Available {
			return Contract{}, errors.New("dedicated compatibility profile is not implemented")
		}
		route := "chat/completions"
		if role == Transcription {
			route = "audio/transcriptions"
		}
		if role == Speech {
			route = "audio/speech"
		}
		return Contract{Profile: profile, Path: route}, nil
	}
	return Contract{}, errors.New("compatibility profile is invalid for this operation")
}
