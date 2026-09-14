package managedruntime

import (
	"errors"
	"runtime"
)

// ProviderRequest admits only provider identity, never paths or launch flags.
type ProviderRequest struct {
	Provider ProviderID `json:"provider"`
}
type BinaryOption struct {
	Backend   string `json:"backend"`
	Supported bool   `json:"supported"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}
type BinaryOptions struct {
	Provider           ProviderID     `json:"provider"`
	OS                 string         `json:"os"`
	Architecture       string         `json:"architecture"`
	Supported          bool           `json:"supported"`
	RecommendedBackend string         `json:"recommendedBackend"`
	Reason             string         `json:"reason"`
	Options            []BinaryOption `json:"options"`
}

type gpuMetadata struct {
	vendor  string
	ordinal int
	output  string
	known   bool
}
type binaryHost struct {
	os, arch string
	gpu      gpuMetadata
}

func cudaAvailable(r platformRecipe, g gpuMetadata) bool {
	return g.known && g.vendor == "nvidia" && g.ordinal == 0 && cudaMetadataSupported(g.output, r.minDriver, r.minCapability)
}
func selectBinaryOptions(provider ProviderID, host binaryHost) (BinaryOptions, error) {
	if provider != LlamaCPP && provider != WhisperCPP {
		return BinaryOptions{}, errors.New("Choose llama.cpp or whisper.cpp for binary selection.")
	}
	result := BinaryOptions{Provider: provider, OS: host.os, Architecture: host.arch, Options: []BinaryOption{}}
	for _, backend := range []string{"cpu", "cuda"} {
		recipe, supported := recipeFor(provider, host.os, host.arch, backend)
		option := BinaryOption{Backend: backend, Supported: supported, Reason: "No qualified binary is available for this operating system and architecture."}
		if supported {
			if backend == "cpu" {
				option.Available = true
				option.Reason = "The pinned CPU binary is available without an NVIDIA GPU."
			} else if cudaAvailable(recipe, host.gpu) {
				option.Available = true
				option.Reason = "NVIDIA GPU 0 and its driver meet the pinned CUDA 12.4 requirements."
			} else {
				option.Reason = "NVIDIA GPU 0 or its driver is unknown or unsupported by the pinned CUDA 12.4 binary; CPU is recommended."
			}
		}
		result.Options = append(result.Options, option)
	}
	result.Supported = result.Options[0].Supported
	result.Reason = result.Options[0].Reason
	if result.Supported {
		result.RecommendedBackend = "cpu"
		result.Reason = result.Options[1].Reason
		if result.Options[1].Available {
			result.RecommendedBackend = "cuda"
		}
	}
	return result, nil
}

// GetBinaryOptions is advisory metadata only: never inspects/replaces an
// installation, downloads an archive, or launches a model. Install keeps its
// CPU default; explicit acceptance uses InstallBackend with the chosen backend.
func (m *Manager) GetBinaryOptions(request ProviderRequest) (BinaryOptions, error) {
	host := binaryHost{os: runtime.GOOS, arch: runtime.GOARCH}
	if _, err := selectBinaryOptions(request.Provider, host); err != nil {
		return BinaryOptions{}, err
	}
	m.mu.Lock()
	if m.closed || m.ctx == nil {
		m.mu.Unlock()
		return BinaryOptions{}, errNotReady
	}
	ctx := m.ctx
	m.mu.Unlock()
	if _, ok := recipeFor(request.Provider, host.os, host.arch, "cuda"); ok {
		host.gpu = probeGGMLGPU(ctx, launchOwned)
	}
	if err := ctx.Err(); err != nil {
		return BinaryOptions{}, errNotReady
	}
	return selectBinaryOptions(request.Provider, host)
}
