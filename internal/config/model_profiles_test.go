package config

import (
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"testing"
)

func TestModelProfilesRejectWrongRolesAndUnknownIDsWhileDisabled(t *testing.T) {
	for _, id := range []modelprofile.ID{modelprofile.S1Mini, "future"} {
		cfg := Default()
		cfg.ModelProfile = id
		if Validate(cfg) == nil {
			t.Fatal("accepted unavailable transcription profile")
		}
		cfg = Default()
		cfg.TextToSpeech.ModelProfile = id
		if Validate(cfg) == nil {
			t.Fatal("accepted unavailable speech profile")
		}
	}
	cfg := Default()
	cfg.PostProcessing.Preset = "future"
	if Validate(cfg) == nil {
		t.Fatal("accepted unavailable cleanup profile while disabled")
	}
	cfg = Default()
	cfg.PostProcessing.Preset = PostProcessingPresetS1Mini
	if err := Validate(cfg); err != nil {
		t.Fatal("existing S1-mini selection rejected", err)
	}
}
