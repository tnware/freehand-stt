package managedruntime

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"testing"
)

func TestGGMLBackendAdmissionAndArguments(t *testing.T) {
	for _, g := range []ggmlProvider{llamaProvider, whisperProvider} {
		if _, err := g.bundle("auto"); err == nil {
			t.Fatal("accepted automatic GPU selection")
		}
		for _, backend := range hostBackends(g.id) {
			b, err := g.bundle(backend)
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Archives) == 0 || b.Archives[0].Backend != backend {
				t.Fatal("wrong bundle")
			}
			args, err := g.backendArguments(g.models[0].ID, g.specs[g.models[0].ID], t.TempDir(), 1234, backend)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(args, "127.0.0.1") {
				t.Fatal("nonlocal listener")
			}
			if g.id == LlamaCPP {
				i := slices.Index(args, "--gpu-layers")
				want := "0"
				if backend != "cpu" {
					want = "auto"
				}
				if i < 0 || args[i+1] != want || slices.Contains(args, "--no-warmup") != (backend == "cpu") || !slices.Contains(args, "--offline") {
					t.Fatal("unsafe llama flags", args)
				}
				if backend == "cuda" && (!slices.Contains(args, "CUDA0") || !slices.Contains(args, "--split-mode")) {
					t.Fatal("unbounded devices", args)
				}
				if backend == "metal" && (!slices.Contains(args, "Metal") || slices.Contains(args, "CUDA0")) {
					t.Fatal("wrong Metal device", args)
				}
			} else if slices.Contains(args, "--no-gpu") != (backend == "cpu") || !slices.Contains(args, "--no-flash-attn") {
				t.Fatal("wrong whisper device flags", args)
			}
		}
	}
	if !cudaMetadataSupported("610.62, 12.0", 581, 7.5) || cudaMetadataSupported("broken", 581, 7.5) || cudaMetadataSupported("580.1, 12.0", 581, 7.5) || cudaMetadataSupported("610.62, 6.1", 581, 7.5) {
		t.Fatal("unsafe driver admission")
	}
}

func TestManagerBackendSwitchAdmission(t *testing.T) {
	i := Instance{ID: "llama", Name: "Llama", Provider: LlamaCPP, Model: "s1-mini"}
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{i}})
	w := m.workers[i.ID]
	w.status.Supported = true
	w.ctx = t.Context()
	if err := m.InstallBackend(BackendRequest{InstanceID: i.ID, Backend: "auto"}); err == nil {
		t.Fatal("accepted unknown backend")
	}
	w.process = &fakeProcess{done: make(chan struct{})}
	if err := m.InstallBackend(BackendRequest{InstanceID: i.ID, Backend: "cuda"}); err == nil {
		t.Fatal("switched running installation")
	}
	w.process = nil
	w.providerProcess.process = &fakeProcess{done: make(chan struct{})}
	if err := m.InstallBackend(BackendRequest{InstanceID: i.ID, Backend: "cpu"}); err == nil {
		t.Fatal("switched while stopped child still draining")
	}
	w.providerProcess.process = nil
	w.checkIdle = func() error { return errors.New("active speech") }
	if err := m.InstallBackend(BackendRequest{InstanceID: i.ID, Backend: "cpu"}); err == nil {
		t.Fatal("switched during active speech")
	}
	w.checkIdle = nil
	w.adapter = newAdapter(t.TempDir())
	if err := m.InstallBackend(BackendRequest{InstanceID: i.ID, Backend: "cpu"}); err == nil {
		t.Fatal("changed unsupported NeMo adapter")
	}
}

func TestCUDADriverFailureOffersCPURecovery(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows CUDA recipe")
	}
	w := newWorker(t.TempDir(), llamaProvider, workerConfig{Model: "s1-mini"}, nil, nil)
	if !w.status.Supported {
		t.Skip("Windows x64 provider")
	}
	w.ctx = t.Context()
	w.status.Backend = "cpu"
	a := w.adapter.(*ggmlAdapter)
	a.launch = func(context.Context, string, []string, string, []string) (processHandle, error) {
		return nil, errors.New("private OS diagnostic")
	}
	if err := w.InstallBackend("cuda"); err != nil {
		t.Fatal(err)
	}
	w.wg.Wait()
	status := w.GetStatus()
	if status.Backend != "cpu" || status.Operation.Outcome != "failed" || status.Error != errCUDAUnavailable.Error() {
		t.Fatalf("GPU setup did not preserve CPU selection with bounded recovery advice: %+v", status)
	}
}
