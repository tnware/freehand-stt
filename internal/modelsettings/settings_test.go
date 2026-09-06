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
