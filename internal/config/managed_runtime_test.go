package config

import (
 "testing"
 "github.com/tnware/freehand-stt/internal/managedruntime"
)

func TestManagedRuntimeDefaultsAndValidation(t *testing.T) {
 v := Default()
 if v.ManagedRuntime != managedruntime.Defaults() { t.Fatalf("managed defaults: %#v", v.ManagedRuntime) }
 v.ManagedRuntime.Enabled = true
 if err := Validate(v); err != nil { t.Fatalf("opt-in with empty BYO settings: %v", err) }
 v.ManagedRuntime.Model = "unqualified"
 if err := Validate(v); err == nil { t.Fatal("unqualified runtime model accepted") }
}
