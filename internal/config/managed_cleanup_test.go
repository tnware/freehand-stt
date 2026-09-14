package config

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"testing"
)

func managedCleanupSettings() Settings {
	s := Default()
	s.ManagedRuntimes = []managedruntime.Instance{{ID: "cleanup", Name: "Local cleanup", Provider: managedruntime.LlamaCPP, Model: "s1-mini"}}
	s.PostProcessing.ManagedInstanceID = "cleanup"
	s.PostProcessing.Model = "s1-mini"
	s.PostProcessing.Preset = PostProcessingPresetS1Mini
	s.PostProcessing.CompatibilityProfile = compatibility.LlamaCPP
	s.PostProcessing.Enabled = true
	return s
}

func TestManagedCleanupResolvedValidation(t *testing.T) {
	p := managedCleanupSettings().PostProcessing
	for _, baseURL := range []string{"", "https://remote.example/v1", "http://127.0.0.1/v1", "http://127.0.0.1:43210/v1?"} {
		p.BaseURL, p.AllowInsecureHTTP = baseURL, true
		if err := ValidatePostProcessing(p); err == nil {
			t.Errorf("unresolved or foreign managed endpoint accepted: %q", baseURL)
		}
	}
	p.BaseURL = "http://127.0.0.1:43210/v1"
	if err := ValidatePostProcessing(p); err != nil {
		t.Fatal(err)
	}
	p.AllowInsecureHTTP = false
	if err := ValidatePostProcessing(p); err == nil {
		t.Fatal("HTTP opt-in bypassed")
	}
	p.ManagedInstanceID, p.BaseURL = "", "https://remote.example/v1"
	if err := ValidatePostProcessing(p); err != nil {
		t.Fatalf("manual cleanup rejected: %v", err)
	}
}

func TestManagedCleanupDurableValidation(t *testing.T) {
	s := managedCleanupSettings()
	if err := Validate(s); err != nil {
		t.Fatalf("transport-free enabled cleanup: %v", err)
	}
	for _, enabled := range []bool{false, true} {
		for name, mutate := range map[string]func(*PostProcessingSettings){
			"generation": func(p *PostProcessingSettings) { p.GenerationOptions.MaxOutputTokens = -1 },
			"URL":        func(p *PostProcessingSettings) { p.BaseURL = "https://remote.example/v1" },
			"insecure":   func(p *PostProcessingSettings) { p.AllowInsecureHTTP = true },
			"model":      func(p *PostProcessingSettings) { p.Model = "other" },
			"preset":     func(p *PostProcessingSettings) { p.Preset = PostProcessingPresetGeneric },
			"timeout":    func(p *PostProcessingSettings) { p.TimeoutSeconds = 0 },
			"styling":    func(p *PostProcessingSettings) { p.Styling = "untrained" },
			"structure":  func(p *PostProcessingSettings) { p.Structure = "untrained" },
			"context":    func(p *PostProcessingSettings) { p.Context = "untrained" },
		} {
			t.Run(name+map[bool]string{false: "Disabled", true: "Enabled"}[enabled], func(t *testing.T) {
				bad := managedCleanupSettings()
				bad.PostProcessing.Enabled = enabled
				mutate(&bad.PostProcessing)
				if err := Validate(bad); err == nil {
					t.Fatal("invalid managed cleanup accepted")
				}
			})
		}
	}
}
