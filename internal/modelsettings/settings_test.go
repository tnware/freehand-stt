package modelsettings

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"testing"
)

func TestOptionsNeverReplaceConnectionOrWorkflow(t *testing.T) {
	v := config.Default()
	v.BaseURL = "https://example.test/v1"
	v.PostProcessing.Enabled = true
	v.TextToSpeech.Enabled = true
	v.MicrophoneID = "chosen-device"
	v.TranscriptionTimeoutSeconds = 240
	for p, o := range Defaults() {
		got := Apply(v, p, "model", o)
		if got.BaseURL != v.BaseURL || !got.PostProcessing.Enabled || !got.TextToSpeech.Enabled || got.MicrophoneID != v.MicrophoneID || got.TranscriptionTimeoutSeconds != 240 {
			t.Fatal("model options changed connection or workflow")
		}
		if Extract(got, p) != o {
			t.Fatal("options failed round trip")
		}
	}
}
func TestModelOptionsValidateWhileFeatureDisabled(t *testing.T) {
	d := savedconnection.Details{BaseURL: "https://example.test/v1", AuthenticationMode: config.AuthenticationModeNone}
	e := Entry{Model: "cleanup", Purpose: savedconnection.Cleanup, Options: Defaults()[savedconnection.Cleanup]}
	e.Options.Profile = modelprofile.S1Mini
	e.Options.Styling = "invented"
	if Validate(e, d) == nil {
		t.Fatal("invalid S1 controls accepted")
	}
	e.Options = Defaults()[savedconnection.Cleanup]
	e.Options.Voice = "cross-role"
	if Validate(e, d) == nil {
		t.Fatal("cross-role options accepted")
	}
	e.Options = Defaults()[savedconnection.Cleanup]
	e.Model = "bad\nmodel"
	if Validate(e, d) == nil {
		t.Fatal("control characters accepted")
	}
}

func TestSelectionPreservesTaskIntent(t *testing.T) {
	v := config.Default()
	v.Language = "ja"
	v.TranscriptionOptions.Prompt = "Proper names"
	v.TranscriptionOptions.Hotwords = "Freehand"
	v.PostProcessing.SystemPrompt = "Keep punctuation."
	v.PostProcessing.Styling = "formal"
	v.TextToSpeech.Speed = 1.5
	for p, o := range Defaults() {
		o.Voice = "model-voice"
		next := Select(v, p, "new-model", o)
		if next.Language != v.Language || next.PostProcessing.SystemPrompt != v.PostProcessing.SystemPrompt || next.PostProcessing.Styling != v.PostProcessing.Styling || next.TextToSpeech.Speed != v.TextToSpeech.Speed {
			t.Fatal("model selection replaced task intent")
		}
		if Model(next, p) != "new-model" {
			t.Fatal("model was not selected")
		}
		if p == savedconnection.Speech && next.TextToSpeech.Voice != "model-voice" {
			t.Fatal("model voice was not restored")
		}
	}
}
