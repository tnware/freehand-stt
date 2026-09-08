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
	Realtime                    bool `json:"realtime"`
	ServerLoadedModel           bool `json:"serverLoadedModel"`
	VLLMTranscriptionEvents     bool `json:"vllmTranscriptionEvents"`
	CleanupOutputLimit          bool `json:"cleanupOutputLimit"`
	CleanupDisableReasoning     bool `json:"cleanupDisableReasoning"`
	FileStreaming               bool `json:"fileStreaming"`
	TypedTranscriptionEvents    bool `json:"typedTranscriptionEvents"`
	LegacyTranscriptionSegments bool `json:"legacyTranscriptionSegments"`
	LanguageHint                bool `json:"languageHint"`
	VoiceDiscovery              bool `json:"voiceDiscovery"`
	SpeechInstructions          bool `json:"speechInstructions"`
	SpeechLanguage              bool `json:"speechLanguage"`
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
	Realtime       []Profile `json:"realtime"`
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
	return Catalog{Realtime: realtimeProfiles(), Transcription: options(Transcription), PostProcessing: options(PostProcessing), Speech: options(Speech)}
}

func options(role Role) []Profile {
	if role == Realtime {
		return realtimeProfiles()
	}
	caps := Capabilities{}
	genericDescription := ""
	switch role {
	case Transcription:
		caps = Capabilities{FileStreaming: true, TypedTranscriptionEvents: true, LegacyTranscriptionSegments: true, LanguageHint: true, TranscriptionPrompt: true, TranscriptionTemperature: true}
		genericDescription = "Completed transcription and optional file streaming. Preserves support for typed events and legacy text segments; streaming and language hints depend on the selected model."
	case PostProcessing:
		caps.CleanupOutputLimit = true
		genericDescription = "Text chat completions with system/user messages. Choose the model profile separately in feature settings."
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
		caps.VoiceDiscovery = true
		result = append(result, Profile{ID: Speaches, Label: "Speaches", Available: true, Description: "Buffered PCM16 WAV speech using the installed model and voice IDs. Speed support depends on the model.", Capabilities: caps})
	} else {
		caps.CleanupDisableReasoning = true
		result = append(result, Profile{ID: LlamaCPP, Label: "llama.cpp", Available: true, Description: "Text cleanup with an optional output-token limit and disable-reasoning override. Reasoning control requires a compatible llama.cpp build and model template; S1-mini behavior uses a separate model profile.", Capabilities: caps})
	}
	planned := func(id ID, label, reason string) {
		result = append(result, Profile{ID: id, Label: label, Description: "Dedicated profile not implemented. " + reason})
	}
	planned(OpenAI, "OpenAI hosted", "Model-specific fields and limits need qualification.")
	planned(LocalAI, "LocalAI", "Backend-specific capabilities need qualification.")
	switch role {
	case Transcription:
		result = append(result,
			Profile{ID: NeMoSpeechV1, Label: "NeMo-Speech.cpp", Available: true, Description: "Completed transcription and qualified realtime dictation with the server-loaded speech model; v0.1.0.", Capabilities: Capabilities{ServerLoadedModel: true, LanguageHint: true, Realtime: true}},
			Profile{ID: WhisperCPP, Label: "whisper.cpp", Available: true, Description: "Completed transcription through the native /inference route. Uses the model already loaded by the server; connection checks use /health. File streaming is unavailable.", Capabilities: Capabilities{ServerLoadedModel: true, LanguageHint: true, TranscriptionPrompt: true, TranscriptionTemperature: true}},
			Profile{ID: VLLM, Label: "vLLM", Available: true, Description: "Completed transcription, file streams, and Qwen3-ASR and Voxtral realtime. Context, language, and temperature depend on the model and mode; v0.28.0.", Capabilities: Capabilities{Realtime: true, FileStreaming: true, VLLMTranscriptionEvents: true, LanguageHint: true, TranscriptionPrompt: true, TranscriptionTemperature: true}},
		)
	case PostProcessing:
		result = append(result, Profile{ID: VLLM, Label: "vLLM", Available: true, Description: "Text cleanup with output-token and reasoning-off controls, qualified against v0.28.0. Reasoning control requires a compatible model template.", Capabilities: Capabilities{CleanupOutputLimit: true, CleanupDisableReasoning: true}})
	case Speech:
		result = append(result, Profile{ID: VLLMOmni, Label: "vLLM-Omni", Available: true, Description: "Buffered WAV speech, voice discovery, and Qwen3-TTS language and style instructions; v0.18.0.", Capabilities: Capabilities{SpeechSpeed: true, VoiceDiscovery: true, SpeechInstructions: true, SpeechLanguage: true}})
		result = append(result, Profile{ID: KokoroFastAPI, Label: "Kokoro-FastAPI", Available: true, Description: "Buffered PCM16 WAV speech, selectable server voices, and speed control. Requests explicitly disable streaming.", Capabilities: Capabilities{SpeechSpeed: true, VoiceDiscovery: true}})
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
		if role == Realtime {
			route = "realtime"
		}
		if role == Transcription {
			route = "audio/transcriptions"
			if profile.ID == WhisperCPP {
				route = "inference"
			}
		}
		if role == Speech {
			route = "audio/speech"
		}
		return Contract{Profile: profile, Path: route}, nil
	}
	return Contract{}, errors.New("compatibility profile is invalid for this operation")
}

// TranscriptionHealthPath preserves explicit custom paths and supplies only
// the native whisper.cpp metadata default. It never loads or switches a model.
func TranscriptionHealthPath(id ID, custom string) string {
	if Effective(id) == WhisperCPP && custom == "" {
		return "/health"
	}
	return custom
}
