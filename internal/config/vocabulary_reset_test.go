package config

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"testing"
)

func TestVocabularyDisabledClearsPreviousRequestProjection(t *testing.T) {
	for _, realtime := range []bool{false, true} {
		v := Default()
		v.VoiceTranscription.ModelProfile = modelprofile.Nemotron35
		v.VoiceTranscription.CompatibilityProfile = compatibility.NeMoSpeechV1
		v.VoiceTranscription.Realtime = realtime
		v.Vocabulary = VocabularySettings{Terms: "Current terms", Voice: true, Boost: 4}
		v, err := WithVocabulary(v, true)
		if err != nil {
			t.Fatal(err)
		}
		v.Vocabulary.Voice = false
		got, err := WithVocabulary(v, true)
		if err != nil {
			t.Fatal(err)
		}
		if got.VoiceTranscription.TranscriptionOptions.Inference().Vocabulary != "" || got.VoiceTranscription.RealtimeOptions().Vocabulary != "" {
			t.Fatal("disabled shared vocabulary retained old request hints")
		}
	}
}
