//go:build linux

package managedruntime

import (
	"context"
	"debug/elf"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/managedruntime/internal/artifact"
)

// Explicit opt-in supplied archives only. Default tests never download, execute,
// enumerate, or load runtime models. Execution additionally requires its own flag.
func TestLinuxOfficialRuntimeArchives(t *testing.T) {
	cache := os.Getenv("FREEHAND_LINUX_RUNTIME_ARCHIVES")
	if cache == "" {
		t.Skip("set FREEHAND_LINUX_RUNTIME_ARCHIVES to pinned release archives")
	}
	for key, recipe := range platformRecipes {
		if key.os != "linux" || key.arch != runtime.GOARCH {
			continue
		}
		t.Run(string(key.provider)+"/"+key.backend, func(t *testing.T) {
			bundle := recipe.Bundle
			bundle.Archives = append([]artifact.Asset(nil), bundle.Archives...)
			for i, asset := range bundle.Archives {
				u, err := url.Parse(asset.URL)
				if err != nil {
					t.Fatal(err)
				}
				filename := filepath.Join(cache, path.Base(u.Path))
				if err := artifact.VerifyFile(t.Context(), filename, asset.Size, asset.SHA256); err != nil {
					t.Fatal(err)
				}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, filename) }))
				t.Cleanup(srv.Close)
				bundle.Archives[i].URL = srv.URL
			}
			root := t.TempDir()
			if err := artifact.Install(t.Context(), root, bundle, http.DefaultClient, nil); err != nil {
				t.Fatal(err)
			}
			if err := artifact.Verify(t.Context(), filepath.Join(root, "runtime"), bundle); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(root, "runtime", filepath.FromSlash(bundle.Executable))
			binary, err := elf.Open(exe)
			if err != nil {
				t.Fatal(err)
			}
			if binary.Machine != elf.EM_X86_64 {
				t.Fatal("unexpected executable machine")
			}
			binary.Close()
			if os.Getenv("FREEHAND_LINUX_RUNTIME_EXECUTE") != "1" {
				return
			}
			env, err := managedGGMLEnvironment(root, key.provider, os.Environ())
			if err != nil {
				t.Fatal(err)
			}
			if key.provider == NeMoSpeechCPP {
				env = childEnvironment(root, os.Environ())
			}
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			p, err := launchOwned(ctx, exe, []string{"--help"}, filepath.Dir(exe), env)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Kill()
			if err := p.Wait(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}
