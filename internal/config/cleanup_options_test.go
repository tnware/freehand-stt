package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestCleanupOptionsMigrateAndRoundTrip(t *testing.T) {
	raw, _ := json.Marshal(Default())
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	delete(document["postProcessing"].(map[string]any), "generationOptions")
	raw, _ = json.Marshal(document)
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := os.WriteFile(store.Path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PostProcessing.GenerationOptions != (compatibility.CleanupOptions{}) {
		t.Fatal("old settings enabled new controls")
	}
	cfg.PostProcessing.CompatibilityProfile = compatibility.LlamaCPP
	cfg.PostProcessing.GenerationOptions = compatibility.CleanupOptions{LimitOutputTokens: true, MaxOutputTokens: 2048, DisableReasoning: true}
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	again, err := store.Load()
	if err != nil || again.PostProcessing.GenerationOptions != cfg.PostProcessing.GenerationOptions {
		t.Fatalf("round trip error=%v", err)
	}
	cfg.PostProcessing.GenerationOptions.LimitOutputTokens = false
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	again, err = store.Load()
	if err != nil || again.PostProcessing.GenerationOptions.LimitOutputTokens || again.PostProcessing.GenerationOptions.MaxOutputTokens != 2048 {
		t.Fatal("disabled numeric limit was not retained")
	}
}

func TestCleanupOptionsValidatedWhileProcessingDisabled(t *testing.T) {
	for _, options := range []compatibility.CleanupOptions{
		{DisableReasoning: true}, {LimitOutputTokens: true}, {MaxOutputTokens: -1}, {MaxOutputTokens: 65537},
	} {
		cfg := Default()
		cfg.PostProcessing.GenerationOptions = options
		if Validate(cfg) == nil {
			t.Fatal("invalid disabled cleanup options accepted")
		}
	}
	for _, limit := range []int{1, 65536} {
		cfg := Default()
		cfg.PostProcessing.GenerationOptions = compatibility.CleanupOptions{LimitOutputTokens: true, MaxOutputTokens: limit}
		if err := Validate(cfg); err != nil {
			t.Fatal(err)
		}
	}
}
