package managedruntime

import (
	"errors"
	"runtime"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// These recipes qualify only CPU Windows x64 and one task per provider.
// Release asset sizes and SHA256 are from the official GitHub release API,
// independently checked against downloaded ZIPs. Model pins are HF LFS metadata.
// No catalog operation downloads or executes a model.
type ggmlProvider struct {
	id                        ProviderID
	name, version, executable string
	release                   asset
	backend                   compatibility.ID
	profile                   modelprofile.ID
	role                      compatibility.Role
	models                    []Model
	specs                     map[string]modelSpec
}

var llamaProvider = ggmlProvider{
	id: LlamaCPP, name: "llama.cpp (CPU)", version: "b10809", executable: "llama-server.exe",
	release: asset{"cpu", "https://github.com/ggml-org/llama.cpp/releases/download/b10809/llama-b10809-bin-win-cpu-x64.zip", "9df3158ed228a641a4b127942d7f459f24c9e13f04682659d05c00c80099b6b5", 18407457},
	backend: compatibility.LlamaCPP, profile: modelprofile.S1Mini, role: compatibility.PostProcessing,
	models: []Model{{ID: "s1-mini", Name: "S1-mini by Superwhisper", Description: "v1 Q4_K_M. English transcript cleanup on CPU; not speech recognition.", Recommended: true}},
	specs:  map[string]modelSpec{"s1-mini": {"superwhisper/s1-mini-GGUF", "34add00a48a2e5d24e5a4ee5405a99620a3a240c", "s1-mini-q4_k_m.gguf", "3b41ebe2502cbd03e811d5d16b022f5ab551eda58d62597d152f89535003c634", 484219808}},
}
var whisperProvider = ggmlProvider{
	id: WhisperCPP, name: "whisper.cpp (CPU)", version: "v1.8.3", executable: "Release/whisper-server.exe",
	release: asset{"cpu", "https://github.com/ggml-org/whisper.cpp/releases/download/v1.8.3/whisper-bin-x64.zip", "d824b1e37599f882b396e73f1ee0bfd5d0529f700314c48311dcbd00b803321d", 3968674},
	backend: compatibility.WhisperCPP, profile: modelprofile.Generic, role: compatibility.Transcription,
	models: []Model{
		{ID: "base", Name: "Whisper Base", Description: "Multilingual completed transcription on CPU.", Recommended: true},
		{ID: "small", Name: "Whisper Small", Description: "Multilingual completed transcription on CPU; more memory and time than Base."},
		{ID: "medium", Name: "Whisper Medium", Description: "Multilingual completed transcription on CPU; substantial memory and processing time."},
	},
	specs: map[string]modelSpec{
		"base":   {"ggerganov/whisper.cpp", "5359861c739e955e79d9a303bcbc70fb988958b1", "ggml-base.bin", "60ed5bc3dd14eea856493d334349b405782ddcaf0028d4b5df4088345fba2efe", 147951465},
		"small":  {"ggerganov/whisper.cpp", "5359861c739e955e79d9a303bcbc70fb988958b1", "ggml-small.bin", "1be3a9b2063867b937e64e2ec7483364a79917e157fa98c5d94b5c1fffea987b", 487601967},
		"medium": {"ggerganov/whisper.cpp", "5359861c739e955e79d9a303bcbc70fb988958b1", "ggml-medium.bin", "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208", 1533763059},
	},
}

func (g ggmlProvider) qualify(id string, role compatibility.Role) (Contract, error) {
	if _, ok := g.specs[id]; !ok || role != g.role {
		return Contract{}, errors.New("This runtime model does not support the selected role.")
	}
	c, err := modelprofile.Resolve(g.profile, g.backend, role)
	if err != nil {
		return Contract{}, err
	}
	return Contract{Role: role, CompatibilityProfile: g.backend, ModelProfile: g.profile, Behavior: c.Profile}, nil
}
func (g ggmlProvider) descriptor() ProviderDescriptor {
	models := append([]Model(nil), g.models...)
	for i := range models {
		c, _ := g.qualify(models[i].ID, g.role)
		models[i].Contracts = []Contract{c}
		models[i].Behavior = &c.Behavior
		models[i].Profile = string(g.profile)
		models[i].SizeBytes = g.specs[models[i].ID].size
	}
	return ProviderDescriptor{ID: g.id, Name: g.name, Version: g.version, Supported: runtime.GOOS == "windows" && runtime.GOARCH == "amd64", Models: models}
}
func (g ggmlProvider) newAdapter(root string) runtimeAdapter {
	return &ggmlAdapter{root: root, recipe: g, launch: launchOwned, listenerOwner: ownsListener}
}
