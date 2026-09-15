package managedruntime

import (
	"encoding/hex"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestWhisperCatalogPublishedVariants(t *testing.T) {
	// Standard server models published in ggerganov/whisper.cpp at 5359861.
	// Keep this explicit: the repository does not publish every combination.
	want := []string{
		"tiny", "tiny-q5_1", "tiny-q8_0", "tiny.en", "tiny.en-q5_1", "tiny.en-q8_0",
		"base", "base-q5_1", "base-q8_0", "base.en", "base.en-q5_1", "base.en-q8_0",
		"small", "small-q5_1", "small-q8_0", "small.en", "small.en-q5_1", "small.en-q8_0",
		"medium", "medium-q5_0", "medium-q8_0", "medium.en", "medium.en-q5_0", "medium.en-q8_0",
		"large-v1", "large-v2", "large-v2-q5_0", "large-v2-q8_0", "large-v3", "large-v3-q5_0",
		"large-v3-turbo", "large-v3-turbo-q5_0", "large-v3-turbo-q8_0",
	}
	models := whisperProvider.descriptor().Models
	if len(models) != len(want) || len(whisperProvider.specs) != len(want) {
		t.Fatalf("published model count: descriptors=%d, pins=%d, want=%d", len(models), len(whisperProvider.specs), len(want))
	}
	for i, id := range []string{"base", "small", "medium"} {
		if models[i].ID != id {
			t.Fatalf("original catalog order changed at %d: %s", i, models[i].ID)
		}
	}
	seen := make(map[string]bool, len(models))
	for _, model := range models {
		if !slices.Contains(want, model.ID) || seen[model.ID] {
			t.Fatalf("unexpected or duplicate model %q", model.ID)
		}
		seen[model.ID] = true
		spec, ok := whisperProvider.specs[model.ID]
		if !ok || spec.repo != "ggerganov/whisper.cpp" || spec.revision != "5359861c739e955e79d9a303bcbc70fb988958b1" || spec.filename != "ggml-"+model.ID+".bin" {
			t.Fatalf("wrong pinned model identity for %q: %+v", model.ID, spec)
		}
		checksum, err := hex.DecodeString(spec.sha256)
		if err != nil || len(checksum) != 32 || spec.size <= 0 || model.SizeBytes != spec.size {
			t.Fatalf("invalid integrity metadata for %q", model.ID)
		}
		if model.Recommended != (model.ID == "base") || model.Installed {
			t.Fatalf("catalog changed default or invented an installation: %+v", model)
		}
		if strings.Contains(model.ID, ".en") && !strings.Contains(model.Description, "English-only") {
			t.Fatalf("English-only model must disclose its language limit: %q", model.ID)
		}
	}
}

func TestWhisperCatalogCompletedTranscriptionOnly(t *testing.T) {
	for _, model := range whisperProvider.descriptor().Models {
		t.Run(model.ID, func(t *testing.T) {
			contract, err := Qualify(WhisperCPP, model.ID, compatibility.Transcription)
			if err != nil {
				t.Fatal(err)
			}
			if contract.ModelProfile != modelprofile.Generic || contract.CompatibilityProfile != compatibility.WhisperCPP || model.Realtime || len(model.Contracts) != 1 || model.Contracts[0].Role != compatibility.Transcription {
				t.Fatalf("variant changed the completed transcription contract: %+v", model)
			}
			for _, role := range []compatibility.Role{compatibility.Realtime, compatibility.PostProcessing, compatibility.Speech} {
				if _, err := Qualify(WhisperCPP, model.ID, role); err == nil {
					t.Fatalf("admitted unsupported role %q", role)
				}
			}
			spec := whisperProvider.specs[model.ID]
			root := t.TempDir()
			want := []string{"-m", spec.path(root), "--host", "127.0.0.1", "--port", "12345", "--no-gpu", "--no-flash-attn"}
			if got := whisperProvider.arguments(model.ID, spec, root, 12345); !slices.Equal(got, want) {
				t.Fatalf("variant requires an unqualified launch change: %v", got)
			}
		})
	}
	for _, id := range []string{"large", "large-v1-q5_0", "large-v3-q8_0", "small.en-tdrz", "tiny-encoder.mlmodelc.zip", "silero-v6.2.0", "for-tests-ggml-tiny", "../foreign"} {
		if _, err := Qualify(WhisperCPP, id, compatibility.Transcription); err == nil {
			t.Fatalf("admitted unpublished variant, companion asset, or foreign model %q", id)
		}
	}
}

func TestWhisperVariantRemovalPreservesSiblingModels(t *testing.T) {
	root := t.TempDir()
	ids := []string{"base", "base-q5_1", "base.en-q5_1", "large-v3-turbo-q8_0"}
	for _, id := range ids {
		spec := whisperProvider.specs[id]
		if err := os.MkdirAll(spec.directory(root), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(spec.path(root), []byte(id), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := whisperProvider.newAdapter(root).RemoveModel(t.Context(), "base-q5_1"); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		data, err := os.ReadFile(whisperProvider.specs[id].path(root))
		if id == "base-q5_1" {
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("selected variant remains after removal: %v", err)
			}
		} else if err != nil || string(data) != id {
			t.Fatalf("removing one variant changed sibling %q: %v", id, err)
		}
	}
}
