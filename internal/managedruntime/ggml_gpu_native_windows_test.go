//go:build windows

package managedruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/managedruntime/internal/artifact"
)

// Opt-in qualification of official GPU distributions through the production
// installer and owned process boundary. No model download, load or inference.
func TestGGMLPinnedCUDABundles(t *testing.T) {
	dir := os.Getenv("FREEHAND_TEST_GGML_GPU_ZIPS")
	if dir == "" {
		t.Skip("set FREEHAND_TEST_GGML_GPU_ZIPS to predownloaded official CUDA ZIPs, retaining release filenames")
	}
	for _, g := range []ggmlProvider{llamaProvider, whisperProvider} {
		t.Run(string(g.id), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, filepath.Join(dir, filepath.Base(r.URL.Path)))
			}))
			defer srv.Close()
			g.cuda.Archives = append([]artifact.Asset(nil), g.cuda.Archives...)
			for i, pin := range g.cuda.Archives {
				u, err := url.Parse(pin.URL)
				if err != nil {
					t.Fatal(err)
				}
				filename := filepath.Base(u.Path)
				if err := artifact.VerifyFile(t.Context(), filepath.Join(dir, filename), pin.Size, pin.SHA256); err != nil {
					t.Fatal(err)
				}
				g.cuda.Archives[i].URL = srv.URL + "/" + filename
			}
			a := g.newAdapter(t.TempDir()).(*ggmlAdapter)
			a.downloadClient = srv.Client()
			backend, err := a.InstallBackend(t.Context(), "cuda", nil)
			if err != nil || backend != "cuda" {
				t.Fatalf("CUDA installation: %q %v", backend, err)
			}
			// Recreate the adapter to exercise the durable installation marker.
			a = g.newAdapter(a.root).(*ggmlAdapter)
			backend, models, err := a.Inspect(t.Context())
			if err != nil || backend != "cuda" {
				t.Fatalf("reopen CUDA installation: %q %v", backend, err)
			}
			for _, model := range models {
				if model.Installed {
					t.Fatal("binary setup unexpectedly installed a model")
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			id := g.models[0].ID
			args, err := g.backendArguments(id, g.specs[id], a.root, 18080, "cuda")
			if err != nil {
				t.Fatal(err)
			}
			args = append(args, "--help")
			env := append(ggmlEnvironment(os.Environ()), "CUDA_VISIBLE_DEVICES=0")
			p, err := a.launch(ctx, a.executable(), args, filepath.Dir(a.executable()), env)
			if err != nil {
				t.Fatal(err)
			}
			if err := p.Wait(ctx); err != nil {
				t.Fatal(err)
			}
			if len(p.Stdout())+len(p.Stderr()) == 0 {
				t.Fatal("no metadata output from CUDA distribution")
			}
			if g.id == LlamaCPP {
				p, err = a.launch(ctx, a.executable(), []string{"--list-devices"}, filepath.Dir(a.executable()), env)
				if err != nil {
					t.Fatal(err)
				}
				if err := p.Wait(ctx); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(p.Stdout()), "CUDA0") {
					t.Fatal("the pinned CUDA device was not enumerated")
				}
			}
			t.Log("official CUDA bundle installed, reopened and verified; owned CLI metadata succeeded without models")
		})
	}
}
