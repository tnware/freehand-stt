package modelprofile

import "github.com/tnware/freehand-stt/internal/compatibility"

// VocabularyMode resolves only qualified request fields, never a model ID.
func VocabularyMode(id ID, backend compatibility.ID, realtime bool) string {
	c, err := Resolve(id, backend, compatibility.Transcription)
	if err != nil {
		return ""
	}
	if id == Nemotron35 && backend == compatibility.NeMoSpeechV1 {
		return "speech-contexts"
	}
	if realtime {
		return ""
	}
	if c.Capabilities.TranscriptionHotwords {
		return "hotwords"
	}
	if c.Capabilities.TranscriptionPrompt {
		return "prompt"
	}
	return ""
}
