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
