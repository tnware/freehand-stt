package postprocess

import (
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestProfilesExposeCustomAndS1RequestContracts(t *testing.T) {
	profiles := Profiles()
	if len(profiles) != 2 {
		t.Fatalf("profiles = %#v", profiles)
	}
	custom := profiles[0]
	if custom.ID != config.PostProcessingPresetGeneric || !custom.InstructionEditable || custom.RecommendedInstruction != config.DefaultPostProcessingInstruction || custom.SystemInstruction != "" || custom.MaximumInstructionBytes != config.MaxPromptBytes {
		t.Fatalf("custom profile = %#v", custom)
	}
	s1 := profiles[1]
	if s1.ID != config.PostProcessingPresetS1Mini || s1.InstructionEditable || s1.SystemInstruction != S1MiniSystemInstruction || s1.Controls == nil {
		t.Fatalf("S1-mini profile = %#v", s1)
	}
	if strings.Join(s1.Controls.Styling, ",") != "casual,semi-casual,semi-formal,formal" || strings.Join(s1.Controls.Structure, ",") != "prose,lists" || strings.Join(s1.Controls.Context, ",") != "general,email" {
		t.Fatalf("S1-mini controls = %#v", s1.Controls)
	}

	// Callers may retain or alter their copy without mutating a global catalog.
	profiles[0].Name = "changed"
	if Profiles()[0].Name == "changed" {
		t.Fatal("Profiles returned shared mutable state")
	}
}
