package config

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// TranscriptionOptions is durable model behavior. Vocabulary is projected only
// into private request state from Settings.Vocabulary, never accepted on the wire.
type TranscriptionOptions struct {
	NeMo                compatibility.NeMoOptions `json:"nemo"`
	Prompt              string                    `json:"prompt"`
	TemperatureOverride bool                      `json:"temperatureOverride"`
	Temperature         float64                   `json:"temperature"`
	hotwords            string
	vocabulary          string
	vocabularyBoost     float64
}

func (o TranscriptionOptions) Inference() compatibility.TranscriptionOptions {
	return compatibility.TranscriptionOptions{NeMo: o.NeMo, Prompt: o.Prompt, TemperatureOverride: o.TemperatureOverride, Temperature: o.Temperature, Hotwords: o.hotwords, Vocabulary: o.vocabulary, VocabularyBoost: o.vocabularyBoost}
}

func (v VoiceTranscriptionSettings) RealtimeOptions() modelprofile.NemotronOptions {
	return v.realtimeOptions
}
