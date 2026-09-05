package config

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func TestTranscriptionOptionsMigrationAndRoundTrip(t *testing.T) {
	raw, _ := json.Marshal(Default())
	var document map[string]json.RawMessage
	json.Unmarshal(raw, &document)
	delete(document, "transcriptionOptions")
	raw, _ = json.Marshal(document)
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := os.WriteFile(store.Path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TranscriptionOptions != (compatibility.TranscriptionOptions{}) {
		t.Fatal("old settings enabled controls")
	}
	loaded.CompatibilityProfile = compatibility.Speaches
	loaded.TranscriptionOptions = compatibility.TranscriptionOptions{Prompt: "Project 日本語", Hotwords: "Freehand", TemperatureOverride: true}
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	again, err := store.Load()
	if err != nil || again.TranscriptionOptions != loaded.TranscriptionOptions {
		t.Fatalf("round trip: %v", err)
	}
	again.TranscriptionOptions.TemperatureOverride = false
	again.TranscriptionOptions.Temperature = 0.4
	if err := store.Save(again); err != nil {
		t.Fatal(err)
	}
	final, err := store.Load()
	if err != nil || final.TranscriptionOptions.TemperatureOverride || final.TranscriptionOptions.Temperature != 0.4 {
		t.Fatal("disabled override was not retained")
	}
}

func TestTranscriptionOptionsValidation(t *testing.T) {
	for _, options := range []compatibility.TranscriptionOptions{
		{Prompt: strings.Repeat("x", 8193)}, {Prompt: "bad\x00hint"}, {Prompt: string([]byte{0xff})},
		{Hotwords: strings.Repeat("語", 683)}, {Hotwords: "bad\x1bterm"},
		{Temperature: math.NaN()}, {Temperature: math.Inf(1)}, {Temperature: -0.1}, {Temperature: 1.1},
	} {
		cfg := Default()
		cfg.CompatibilityProfile = compatibility.Speaches
		cfg.TranscriptionOptions = options
		if Validate(cfg) == nil {
			t.Fatal("accepted invalid transcription controls")
		}
	}
	cfg := Default()
	cfg.TranscriptionOptions = compatibility.TranscriptionOptions{Prompt: strings.Repeat("x", 8192), Hotwords: strings.Repeat("x", 2048), TemperatureOverride: true, Temperature: 1}
	if Validate(cfg) == nil {
		t.Fatal("generic accepted hotwords")
	}
	cfg.CompatibilityProfile = compatibility.Speaches
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
}
