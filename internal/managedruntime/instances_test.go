package managedruntime

import (
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestInstanceQualificationAndInventoryBounds(t *testing.T) {
	valid := Instance{ID: "speech-one", Name: "Local speech", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	if err := ValidateInstance(valid); err != nil {
		t.Fatal(err)
	}
	for _, role := range []compatibility.Role{compatibility.Transcription, compatibility.Realtime} {
		c, err := Qualify(valid.Provider, valid.Model, role)
		if err != nil || c.Role != role || c.CompatibilityProfile != compatibility.NeMoSpeechV1 || c.ModelProfile != modelprofile.Nemotron35 || !c.Behavior.Capabilities.Realtime {
			t.Fatalf("contract %+v: %v", c, err)
		}
	}
	for _, role := range []compatibility.Role{compatibility.PostProcessing, compatibility.Speech, "unknown"} {
		if _, err := Qualify(valid.Provider, valid.Model, role); err == nil {
			t.Fatalf("accepted role %s", role)
		}
	}
	if _, err := Qualify(NeMoSpeechCPP, "parakeet-tdt", compatibility.Realtime); err == nil {
		t.Fatal("accepted completed-only model for realtime")
	}
	for _, id := range []string{"", "../escape", "a/b", "A", "con", "nul", strings.Repeat("x", 65)} {
		v := valid
		v.ID = id
		if ValidateInstance(v) == nil {
			t.Fatalf("accepted unsafe ID %q", id)
		}
	}
	for _, name := range []string{"", "\xff", strings.Repeat("é", 41)} {
		v := valid
		v.Name = name
		if ValidateInstance(v) == nil {
			t.Fatalf("accepted invalid name %q", name)
		}
	}
	for _, change := range []Instance{{ID: valid.ID, Name: valid.Name, Provider: "llama-cpp", Model: valid.Model}, {ID: valid.ID, Name: valid.Name, Provider: valid.Provider, Model: "unqualified"}} {
		if ValidateInstance(change) == nil {
			t.Fatal("accepted unregistered provider/model")
		}
	}
	if ValidateInstances([]Instance{valid, valid}) == nil {
		t.Fatal("accepted duplicate IDs")
	}
	if ValidateInstances(make([]Instance, MaxInstances+1)) == nil {
		t.Fatal("accepted unbounded inventory")
	}
}

func TestNeMoInstanceSeparatesOptionalSpeechFromASR(t *testing.T) {
	valid := Instance{ID: "combined", Name: "Combined", Provider: NeMoSpeechCPP, Model: "nemotron-3.5", SpeechModel: "magpie-tts"}
	if err := ValidateInstance(valid); err != nil {
		t.Fatal(err)
	}
	if valid.ModelForRole(compatibility.Speech) != "magpie-tts" || valid.ModelForRole(compatibility.Transcription) != "nemotron-3.5" || valid.ModelForRole(compatibility.Realtime) != "nemotron-3.5" {
		t.Fatal("model selection crossed role boundaries")
	}
	for _, invalid := range []Instance{
		{ID: valid.ID, Name: valid.Name, Provider: NeMoSpeechCPP, Model: "magpie-tts"},
		{ID: valid.ID, Name: valid.Name, Provider: NeMoSpeechCPP, Model: valid.Model, SpeechModel: "parakeet-tdt"},
		{ID: valid.ID, Name: valid.Name, Provider: NeMoSpeechCPP, Model: valid.Model, SpeechModel: "unknown"},
		{ID: valid.ID, Name: valid.Name, Provider: LlamaCPP, Model: "s1-mini", SpeechModel: "magpie-tts"},
	} {
		if ValidateInstance(invalid) == nil {
			t.Fatalf("accepted unqualified model combination: %+v", invalid)
		}
	}
	valid.SpeechModel = ""
	if err := ValidateInstance(valid); err != nil || valid.ModelForRole(compatibility.Speech) != "" {
		t.Fatal("speech must be optional without changing the ASR selection", err)
	}
	if Validate(Preferences{Model: "magpie-tts"}) == nil {
		t.Fatal("legacy ASR preferences accepted a speech-only model")
	}
}
