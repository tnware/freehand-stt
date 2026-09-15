package managedruntime

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// Pinned from the v0.1.0 release's share/nemo-speech/model-index.json.
// The CLI list is metadata-only and does not expose sizes, filenames or hashes.
// These pins supply integrity and exact cache ownership, not an invented inventory.
type modelSpec struct {
	repo, revision, filename, sha256 string
	size                             int64
}

var modelSpecs = map[string]modelSpec{
	"nemotron-3.5": {"nvidia/nemotron-3.5-asr-streaming-0.6b", "1c8deaecc64b91f034d73e08dd8b64625eb3395d", "nemotron-3.5-asr-streaming-0.6b.q8_0.gguf", "a5c435f294eea8f88ce68dd27b8c3bfea7f777cb2fbba04fcd30eaa555f429ae", 741548352},
	"parakeet-tdt": {"nvidia/parakeet-tdt-0.6b-v3", "541d1f99c6b0c3cd0b11a95167540bb8edefd82b", "parakeet-tdt-0.6b-v3.q8_0.gguf", "e3880d0aaaaf2c308ea2c35016b2b895c423eb3fda924c1b463d1c19b7f4d32e", 713975456},
	"magpie-tts":   {"nvidia/magpie_tts_multilingual_357m", "452ef560f972c38d5fc16476259aac9456453547", "magpie_tts_multilingual_357m.v2602.f16.gguf", "901d299a8b1df016cf81cae0089a7a7c15627b9633d033357e15a47d9a219a75", 448604832},
}

func (m modelSpec) directory(root string) string {
	return filepath.Join(root, "models", filepath.FromSlash(m.repo), m.revision)
}
func (m modelSpec) path(root string) string { return filepath.Join(m.directory(root), m.filename) }

// QualifiedBehavior resolves the selected managed model's metadata without
// requiring an installed runtime, catalog discovery, or a running endpoint.
func QualifiedBehavior(id string) (modelprofile.Profile, error) {
	contract, err := Qualify(NeMoSpeechCPP, id, compatibility.Transcription)
	return contract.Behavior, err
}

func parseCatalog(b []byte) ([]Model, error) {
	var raw struct {
		Models []struct {
			Repo, Revision                       string
			Aliases, Roles, Commands, Companions []string
		}
	}
	if len(b) > 256<<10 || json.Unmarshal(b, &raw) != nil || len(raw.Models) > 128 {
		return nil, errors.New("The runtime returned invalid model metadata. Reinstall the runtime.")
	}
	out := []Model{}
	for _, id := range []string{"nemotron-3.5", "parakeet-tdt", "magpie-tts"} {
		spec := modelSpecs[id]
		for _, m := range raw.Models {
			if m.Repo != spec.repo {
				continue
			}
			role := "asr"
			companionsValid := len(m.Companions) == 0
			if id == "magpie-tts" {
				role = "tts"
				companionsValid = slices.Equal(m.Companions, []string{nemoCodecSpec.repo}) && slices.Contains(m.Roles, "tokenizer")
				codecValid := false
				for _, companion := range raw.Models {
					if companion.Repo == nemoCodecSpec.repo && companion.Revision == nemoCodecSpec.revision && slices.Contains(companion.Roles, "codec") && len(companion.Companions) == 0 {
						codecValid = true
					}
				}
				companionsValid = companionsValid && codecValid
			}
			if m.Revision != spec.revision || !slices.Contains(m.Aliases, id) || !slices.Contains(m.Roles, role) || !slices.Contains(m.Commands, "serve") || !companionsValid {
				return nil, errors.New("The runtime catalog is not qualified for this release. Reinstall the runtime.")
			}
			q := (nemoProvider{}).model(id)
			out = append(out, q)
			break
		}
	}
	if len(out) == 0 {
		return nil, errors.New("No qualified speech models are available in this runtime.")
	}
	return out, nil
}
