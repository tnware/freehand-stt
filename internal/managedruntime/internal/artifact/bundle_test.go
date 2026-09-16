package artifact

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeBundleSwitchPreservesCacheAndRejectsFailedReplacement(t *testing.T) {
	root := t.TempDir()
	cpuBytes := zipFixture(t, map[string]string{"server.exe": "cpu"})
	gpuBytes := zipFixture(t, map[string]string{"server.exe": "gpu", "ggml-cuda.dll": "backend"})
	dllBytes := zipFixture(t, map[string]string{"cudart.dll": "cuda runtime"})
	payloads := map[string][]byte{"/cpu": cpuBytes, "/gpu": gpuBytes, "/dll": dllBytes}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payloads[r.URL.Path]) }))
	defer srv.Close()
	pin := func(backend, path string) Asset {
		b := payloads[path]
		return Asset{backend, srv.URL + path, fmt.Sprintf("%x", sha256.Sum256(b)), int64(len(b))}
	}
	cpu := Bundle{Archives: []Asset{pin("cpu", "/cpu")}, Executable: "server.exe"}
	gpu := Bundle{Archives: []Asset{pin("cuda", "/gpu"), pin("cuda", "/dll")}, Executable: "server.exe", Required: []string{"ggml-cuda.dll", "cudart.dll"}}
	if err := Install(t.Context(), root, cpu, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "models", "keep.gguf")
	if err := os.MkdirAll(filepath.Dir(cache), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte("existing model"), 0600); err != nil {
		t.Fatal(err)
	}
	check := func(bundle Bundle, backend string) {
		t.Helper()
		if err := Verify(t.Context(), filepath.Join(root, "runtime"), bundle); err != nil {
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
	if err := Install(t.Context(), root, gpu, srv.Client(), nil); err == nil {
		t.Fatal("accepted corrupt companion")
	}
	check(cpu, "cpu")
	payloads["/dll"] = dllBytes
	ctx, cancel := context.WithCancel(t.Context())
	if err := Install(ctx, root, gpu, srv.Client(), func(float64) { cancel() }); err == nil {
		t.Fatal("accepted cancelled switch")
	}
	check(cpu, "cpu")
	if err := Install(t.Context(), root, gpu, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	check(gpu, "cuda")
	extra := filepath.Join(root, "runtime", "foreign.dll")
	if err := os.WriteFile(extra, []byte("unverified plugin"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Verify(t.Context(), filepath.Join(root, "runtime"), gpu); err == nil {
		t.Fatal("accepted unverified DLL")
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime", "cudart.dll"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Verify(t.Context(), filepath.Join(root, "runtime"), gpu); err == nil {
		t.Fatal("accepted changed companion DLL")
	}
	if err := Install(t.Context(), root, cpu, srv.Client(), nil); err != nil {
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
	if err := Recover(root); err != nil {
		t.Fatal(err)
	}
	check(cpu, "cpu")
}
