package config

import (
	"encoding/json"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"os"
	"path/filepath"
	"testing"
)

func TestCompatibilityProfilesMigrateAndRoundTrip(t *testing.T) {
	settings := Default()
	raw, _ := json.Marshal(settings)
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	delete(document, "compatibilityProfile")
	delete(document["postProcessing"].(map[string]any), "compatibilityProfile")
	delete(document["textToSpeech"].(map[string]any), "compatibilityProfile")
	raw, _ = json.Marshal(document)
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := os.WriteFile(store.Path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CompatibilityProfile != compatibility.Generic || loaded.PostProcessing.CompatibilityProfile != compatibility.Generic || loaded.TextToSpeech.CompatibilityProfile != compatibility.Generic {
		t.Fatal("old document did not retain generic defaults")
	}
	loaded.CompatibilityProfile = compatibility.Speaches
	loaded.PostProcessing.CompatibilityProfile = compatibility.LlamaCPP
	loaded.TextToSpeech.CompatibilityProfile = compatibility.Speaches
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	again, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if again.CompatibilityProfile != compatibility.Speaches || again.PostProcessing.CompatibilityProfile != compatibility.LlamaCPP || again.TextToSpeech.CompatibilityProfile != compatibility.Speaches {
		t.Fatal("separate selections were lost")
	}
}

func TestUnavailableCompatibilityProfilesFailEvenWhenFeatureDisabled(t *testing.T) {
	for _, id := range []compatibility.ID{compatibility.VLLM, compatibility.ID("future-profile")} {
		for _, role := range []string{"stt", "processing", "speech"} {
			settings := Default()
			switch role {
			case "stt":
				settings.CompatibilityProfile = id
			case "processing":
				settings.PostProcessing.CompatibilityProfile = id
			case "speech":
				settings.TextToSpeech.CompatibilityProfile = id
			}
			if Validate(settings) == nil {
				t.Fatalf("accepted %s/%s", role, id)
			}
		}
	}
	settings := Default()
	settings.CompatibilityProfile = compatibility.LlamaCPP
	if Validate(settings) == nil {
		t.Fatal("chat profile accepted for STT")
	}
	// Existing recovery behavior must preserve a future profile on disk.
	raw, _ := json.Marshal(settings)
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := os.WriteFile(store.Path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("invalid profile did not require recovery")
	}
	after, err := os.ReadFile(store.Path)
	if err != nil || string(after) != string(raw) {
		t.Fatal("invalid document was overwritten")
	}
}
