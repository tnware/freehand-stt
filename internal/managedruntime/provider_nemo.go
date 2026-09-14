package managedruntime

import (
	"errors"
	"runtime"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

type nemoProvider struct{}

func (nemoProvider) newAdapter(root string) runtimeAdapter { return newAdapter(root) }
func (n nemoProvider) descriptor() ProviderDescriptor {
	models := make([]Model, 0, len(qualified))
	for _, id := range []string{"nemotron-3.5", "parakeet-tdt"} {
		m := n.model(id)
		models = append(models, m)
	}
	_, supported := recipeFor(NeMoSpeechCPP, runtime.GOOS, runtime.GOARCH, "cpu")
	return ProviderDescriptor{ID: NeMoSpeechCPP, Name: "NeMo-Speech.cpp", Version: Version, Supported: supported, Backends: hostBackends(NeMoSpeechCPP), Models: models, Source: runtimeSource(NeMoSpeechCPP)}
}
func (n nemoProvider) model(id string) Model {
	m := qualified[id]
	m.Contracts = []Contract{}
	for _, role := range []compatibility.Role{compatibility.Transcription, compatibility.Realtime, compatibility.PostProcessing, compatibility.Speech} {
		c, err := n.qualify(id, role)
		if err != nil {
			continue
		}
		m.Contracts = append(m.Contracts, c)
		if role == compatibility.Transcription {
			m.Behavior = &c.Behavior
		}
	}
	m.SizeBytes = modelSpecs[id].size
	if spec, ok := modelSpecs[id]; ok {
		m.Source = spec.source(NeMoModelManager)
	}
	return m
}

func (nemoProvider) qualify(model string, role compatibility.Role) (Contract, error) {
	q, ok := qualified[model]
	if !ok {
		return Contract{}, errors.New("Choose a supported managed speech model.")
	}
	if role != compatibility.Transcription && role != compatibility.Realtime {
		return Contract{}, errors.New("This runtime model does not support the selected role.")
	}
	c, err := modelprofile.Resolve(modelprofile.ID(q.Profile), compatibility.NeMoSpeechV1, role)
	if err != nil {
		return Contract{}, err
	}
	return Contract{Role: role, CompatibilityProfile: compatibility.NeMoSpeechV1, ModelProfile: modelprofile.ID(q.Profile), Behavior: c.Profile}, nil
}

// NeMo v0.1.0 qualified model metadata; never used by the generic supervisor.
var qualified = map[string]Model{
	"nemotron-3.5": {ID: "nemotron-3.5", Name: "Nemotron 3.5 Streaming", Description: "Multilingual speech recognition with realtime dictation.", Recommended: true, Realtime: true, Profile: "nemotron-3.5-streaming"},
	"parakeet-tdt": {ID: "parakeet-tdt", Name: "Parakeet TDT v3", Description: "Multilingual completed speech recognition.", Profile: "parakeet-tdt-v3"},
}
