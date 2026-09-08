package compatibility

// NeMoSpeechV1 is the NeMo-Speech.cpp v0.1.0 project-specific WebSocket
// protocol. It is deliberately separate from OpenAI Realtime compatibility.
const NeMoSpeechV1 ID = "nemo-speech-v1"
const Realtime Role = "realtime"

func realtimeProfiles() []Profile {
	return []Profile{{ID: NeMoSpeechV1, Label: "NeMo-Speech.cpp", Available: true,
		Description:  "Live PCM16 transcription using the v0.1.0 WebSocket contract. Uses the server's loaded model.",
		Capabilities: Capabilities{Realtime: true, ServerLoadedModel: true, LanguageHint: true}}}
}
