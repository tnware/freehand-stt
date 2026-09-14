package managedruntime

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (g ggmlProvider) bundle(backend string) (runtimeBundle, error) {
	if r, ok := recipeFor(g.id, runtime.GOOS, runtime.GOARCH, backend); ok {
		// Existing consumers project these fields from the registry; retain
		// fixture overrides without maintaining a second set of release pins.
		if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
			if backend == "cpu" {
				r.archives = []asset{g.release}
			} else {
				r.runtimeBundle = g.cuda
			}
		}
		return r.runtimeBundle, nil
	}
	return runtimeBundle{}, errors.New("Choose a supported CPU or NVIDIA CUDA backend.")
}
func (g ggmlProvider) backendArguments(model string, s modelSpec, root string, port int, backend string) ([]string, error) {
	if _, err := g.bundle(backend); err != nil {
		return nil, err
	}
	args := g.arguments(model, s, root, port)
	if backend == "cpu" {
		return args, nil
	}
	if g.id == LlamaCPP {
		args[slices.Index(args, "--gpu-layers")+1] = "auto"
		args = slices.DeleteFunc(args, func(s string) bool { return s == "--no-op-offload" || s == "--no-warmup" })
		args = append(args, "--device", "CUDA0", "--split-mode", "none", "--main-gpu", "0")
	} else {
		args = slices.DeleteFunc(args, func(s string) bool { return s == "--no-gpu" })
	}
	return args, nil
}

var errCUDAUnavailable = errors.New("NVIDIA CUDA requires a compatible GPU and driver. Update the NVIDIA driver or choose CPU.")

// Metadata only: never instantiate a backend or load a model for admission.
// An uncertain probe fails closed rather than silently selecting CPU or GPU.
func cudaMetadataSupported(output string, minDriver, minCapability float64) bool {
	parts := strings.Split(strings.TrimSpace(output), ",")
	if len(parts) != 2 {
		return false
	}
	driver, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil || math.IsNaN(driver) || math.IsInf(driver, 0) {
		return false
	}
	capability, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil || math.IsNaN(capability) || math.IsInf(capability, 0) {
		return false
	}
	return driver >= minDriver && capability >= minCapability
}
func (a *ggmlAdapter) admitCUDA(ctx context.Context) error {
	if err := safeRoot(a.root); err != nil {
		return err
	}
	recipe, ok := recipeFor(a.recipe.id, runtime.GOOS, runtime.GOARCH, "cuda")
	if !ok {
		return errCUDAUnavailable
	}
	gpu := probeGGMLGPU(ctx, a.launch)
	if err := ctx.Err(); err != nil {
		return err
	}
	if !cudaAvailable(recipe, gpu) {
		return errCUDAUnavailable
	}
	return nil
}

// Query the same default NVIDIA ordinal used by CUDA_VISIBLE_DEVICES=0 and
// CUDA0 at model launch. No PATH search, inherited overrides, or raw output DTO.
func probeGGMLGPU(ctx context.Context, launch func(context.Context, string, []string, string, []string) (*ownedProcess, error)) gpuMetadata {
	unknown := gpuMetadata{}
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "nvidia-smi.exe")
	if runtime.GOOS != "windows" || !filepath.IsAbs(exe) {
		return unknown
	}
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p, err := launch(probe, exe, []string{"--id=0", "--query-gpu=driver_version,compute_cap", "--format=csv,noheader,nounits"}, filepath.Dir(exe), ggmlEnvironment(os.Environ()))
	if err != nil {
		return unknown
	}
	if err := p.wait(probe); err != nil {
		return unknown
	}
	p.stdout.mu.Lock()
	overflow := p.stdout.overflow
	p.stdout.mu.Unlock()
	if overflow {
		return unknown
	}
	return gpuMetadata{vendor: "nvidia", ordinal: 0, output: string(p.stdout.bytes()), known: true}
}
