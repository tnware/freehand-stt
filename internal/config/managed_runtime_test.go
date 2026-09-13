package config

import (
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"testing"
)

func TestManagedRuntimeDefaultsAndValidation(t *testing.T) {
	v := Default()
	if len(v.ManagedRuntimes) != 0 {
		t.Fatal("fresh inventory")
	}
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}}
	if e := Validate(v); e != nil {
		t.Fatal(e)
	}
	v.ManagedRuntimes[0].Model = "unqualified"
	if Validate(v) == nil {
		t.Fatal("unqualified model accepted")
	}
}
