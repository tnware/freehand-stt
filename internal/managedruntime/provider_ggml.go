package managedruntime

import (
	"errors"
	"runtime"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// These recipes target qualified Windows/macOS hosts and one task per provider. CPU remains
// the default; CUDA is an explicit, durable selection.
// Release asset sizes and SHA256 are from the official GitHub release API,
// independently checked against downloaded ZIPs. Model pins are HF LFS metadata.
// No catalog operation downloads or executes a model.
type ggmlProvider struct {
	id   ProviderID
	name string
	platformRecipe
	backend compatibility.ID
	profile modelprofile.ID
	role    compatibility.Role
	models  []Model
	specs   map[string]modelSpec
}

var llamaProvider = ggmlProvider{
	id: LlamaCPP, name: "llama.cpp", platformRecipe: hostCPURecipe(LlamaCPP),
	backend: compatibility.LlamaCPP, profile: modelprofile.S1Mini, role: compatibility.PostProcessing,
	models: []Model{{ID: "s1-mini", Name: "S1-mini by Superwhisper", Description: "v1 Q4_K_M. English transcript cleanup; not speech recognition.", Recommended: true}},
	specs:  map[string]modelSpec{"s1-mini": {"superwhisper/s1-mini-GGUF", "34add00a48a2e5d24e5a4ee5405a99620a3a240c", "s1-mini-q4_k_m.gguf", "3b41ebe2502cbd03e811d5d16b022f5ab551eda58d62597d152f89535003c634", 484219808}},
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
		models[i].Source = g.specs[models[i].ID].source(FreehandHuggingFace)
	}
	_, supported := recipeFor(g.id, runtime.GOOS, runtime.GOARCH, "cpu")
	supported = supported && supportsOSVersion(g.id, runtime.GOOS, hostOSVersion())
	reason := ""
	if !supported && runtime.GOOS == "darwin" {
		if g.id == WhisperCPP {
			reason = "Upstream does not publish a macOS server binary. Use a manual Connection."
		}
		if g.id == LlamaCPP && (runtime.GOARCH == "arm64" || runtime.GOARCH == "amd64") {
			reason = "The pinned llama.cpp binary requires macOS 13.3 or later."
		}
	}
	return ProviderDescriptor{ID: g.id, Name: g.name, Version: g.version, Supported: supported, UnavailableReason: reason, Backends: hostBackends(g.id), Models: models, Source: runtimeSource(g.id)}
}
func (g ggmlProvider) newAdapter(root string) runtimeAdapter {
	return &ggmlAdapter{root: root, recipe: g, launch: launchOwned, listenerOwner: ownsListener}
}
