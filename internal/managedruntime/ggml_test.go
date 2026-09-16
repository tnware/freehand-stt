package managedruntime

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestGGMLProviderRoleAdmission(t *testing.T) {
	for _, tc := range []struct {
		provider ProviderID
		model    string
		role     compatibility.Role
		profile  modelprofile.ID
		backend  compatibility.ID
	}{
		{"llama-cpp", "s1-mini", compatibility.PostProcessing, modelprofile.S1Mini, compatibility.LlamaCPP},
		{"whisper-cpp", "base", compatibility.Transcription, modelprofile.Generic, compatibility.WhisperCPP},
		{"whisper-cpp", "small", compatibility.Transcription, modelprofile.Generic, compatibility.WhisperCPP},
		{"whisper-cpp", "medium", compatibility.Transcription, modelprofile.Generic, compatibility.WhisperCPP},
	} {
		c, err := Qualify(tc.provider, tc.model, tc.role)
		if err != nil {
			t.Fatal(err)
		}
		if c.ModelProfile != tc.profile || c.CompatibilityProfile != tc.backend {
			t.Fatalf("wrong contract: %+v", c)
		}
		for _, role := range []compatibility.Role{compatibility.Transcription, compatibility.Realtime, compatibility.PostProcessing, compatibility.Speech} {
			if role != tc.role {
				if _, err := Qualify(tc.provider, tc.model, role); err == nil {
					t.Fatal("unqualified role admitted")
				}
			}
		}
		if _, err := Qualify(tc.provider, "../foreign", tc.role); err == nil {
			t.Fatal("arbitrary model admitted")
		}
	}
}
