package managedruntime

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGGMLBackendAdmissionAndArguments(t *testing.T) {
	for _, g := range []ggmlProvider{llamaProvider, whisperProvider} {
		if _, err := g.bundle("auto"); err == nil {
			t.Fatal("accepted automatic GPU selection")
		}
		for _, backend := range []string{"cpu", "cuda"} {
			b, err := g.bundle(backend)
			if err != nil {
				t.Fatal(err)
			}
			if len(b.archives) == 0 || b.archives[0].backend != backend {
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
				if backend == "cuda" {
					want = "auto"
				}
				if i < 0 || args[i+1] != want || slices.Contains(args, "--no-warmup") != (backend == "cpu") || !slices.Contains(args, "--offline") {
					t.Fatal("unsafe llama flags", args)
				}
				if backend == "cuda" && (!slices.Contains(args, "CUDA0") || !slices.Contains(args, "--split-mode")) {
					t.Fatal("unbounded devices", args)
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
	w.process = &ownedProcess{done: make(chan struct{})}
	if err := m.InstallBackend(BackendRequest{InstanceID: i.ID, Backend: "cuda"}); err == nil {
		t.Fatal("switched running installation")
	}
	w.process = nil
	w.providerProcess.process = &ownedProcess{done: make(chan struct{})}
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
	w := newWorker(t.TempDir(), llamaProvider, workerConfig{Model: "s1-mini"}, nil, nil)
	if !w.status.Supported {
		t.Skip("Windows x64 provider")
	}
	w.ctx = t.Context()
	w.status.Backend = "cpu"
	a := w.adapter.(*ggmlAdapter)
	a.launch = func(context.Context, string, []string, string, []string) (*ownedProcess, error) {
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

func TestRuntimeBundleSwitchPreservesCacheAndRejectsFailedReplacement(t *testing.T) {
	root := t.TempDir()
	cpuBytes := zipFixture(t, map[string]string{"server.exe": "cpu"})
	gpuBytes := zipFixture(t, map[string]string{"server.exe": "gpu", "ggml-cuda.dll": "backend"})
	dllBytes := zipFixture(t, map[string]string{"cudart.dll": "cuda runtime"})
	payloads := map[string][]byte{"/cpu": cpuBytes, "/gpu": gpuBytes, "/dll": dllBytes}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payloads[r.URL.Path]) }))
	defer srv.Close()
	pin := func(backend, path string) asset {
		b := payloads[path]
		return asset{backend, srv.URL + path, fmt.Sprintf("%x", sha256.Sum256(b)), int64(len(b))}
	}
	cpu := runtimeBundle{archives: []asset{pin("cpu", "/cpu")}, executable: "server.exe"}
	gpu := runtimeBundle{archives: []asset{pin("cuda", "/gpu"), pin("cuda", "/dll")}, executable: "server.exe", required: []string{"ggml-cuda.dll", "cudart.dll"}}
	if err := installRuntimeBundle(t.Context(), root, cpu, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "models", "keep.gguf")
	if err := os.MkdirAll(filepath.Dir(cache), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte("existing model"), 0600); err != nil {
		t.Fatal(err)
	}
	check := func(bundle runtimeBundle, backend string) {
		t.Helper()
		if err := verifyRuntimeBundle(t.Context(), filepath.Join(root, "runtime"), bundle); err != nil {
			t.Fatal(err)
		}
		b, e := os.ReadFile(filepath.Join(root, "runtime", ".backend"))
		if e != nil || string(b) != backend {
			t.Fatal("selection changed", e, string(b))
		}
		b, e = os.ReadFile(cache)
		if e != nil || string(b) != "existing model" {
			t.Fatal("cache changed", e)
		}
	}
	payloads["/dll"] = []byte("corrupt")
	if err := installRuntimeBundle(t.Context(), root, gpu, srv.Client(), nil); err == nil {
		t.Fatal("accepted corrupt companion")
	}
	check(cpu, "cpu")
	payloads["/dll"] = dllBytes
	ctx, cancel := context.WithCancel(t.Context())
	if err := installRuntimeBundle(ctx, root, gpu, srv.Client(), func(float64) { cancel() }); err == nil {
		t.Fatal("accepted cancelled switch")
	}
	check(cpu, "cpu")
	if err := installRuntimeBundle(t.Context(), root, gpu, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	check(gpu, "cuda")
	extra := filepath.Join(root, "runtime", "foreign.dll")
	if err := os.WriteFile(extra, []byte("unverified plugin"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeBundle(t.Context(), filepath.Join(root, "runtime"), gpu); err == nil {
		t.Fatal("accepted unverified DLL")
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime", "cudart.dll"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeBundle(t.Context(), filepath.Join(root, "runtime"), gpu); err == nil {
		t.Fatal("accepted changed companion DLL")
	}
	if err := installRuntimeBundle(t.Context(), root, cpu, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	check(cpu, "cpu")
	if _, err := os.Stat(filepath.Join(root, "runtime", "ggml-cuda.dll")); !os.IsNotExist(err) {
		t.Fatal("GPU DLL leaked into CPU installation")
	}
	// Simulate interruption between the two directory renames.
	if err := os.Rename(filepath.Join(root, "runtime"), filepath.Join(root, ".runtime-previous")); err != nil {
		t.Fatal(err)
	}
	if err := recoverRuntime(root); err != nil {
		t.Fatal(err)
	}
	check(cpu, "cpu")
}
