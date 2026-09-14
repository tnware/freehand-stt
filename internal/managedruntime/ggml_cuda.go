package managedruntime

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (g ggmlProvider) bundle(backend string) (runtimeBundle, error) {
	switch backend {
	case "cpu":
		return runtimeBundle{archives: []asset{g.release}, executable: g.executable}, nil
	case "cuda":
		if len(g.cuda.archives) > 0 {
			return g.cuda, nil
		}
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
		args = slices.DeleteFunc(args, func(s string) bool { return s == "--no-op-offload" })
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
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "nvidia-smi.exe")
	if !filepath.IsAbs(exe) {
		return errCUDAUnavailable
	}
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p, err := a.launch(probe, exe, []string{"--id=0", "--query-gpu=driver_version,compute_cap", "--format=csv,noheader,nounits"}, filepath.Dir(exe), ggmlEnvironment(os.Environ()))
	if err != nil {
		return errCUDAUnavailable
	}
	if err := p.wait(probe); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errCUDAUnavailable
	}
	p.stdout.mu.Lock()
	overflow := p.stdout.overflow
	p.stdout.mu.Unlock()
	// CUDA 12.4 Update 1's bundled Windows driver is 551.78. Use the
	// full-toolkit floor, not CUDA minor compatibility (PTX JIT needs it).
	if overflow || !cudaMetadataSupported(string(p.stdout.bytes()), 551.78, 5.0) {
		return errCUDAUnavailable
	}
	return nil
}
