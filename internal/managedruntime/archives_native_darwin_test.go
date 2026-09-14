//go:build darwin

package managedruntime

import (
	"context"
	"debug/macho"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Opt-in release qualification: archives supplied by the maintainer, never
// downloaded in CI. Only --help executes; no model is loaded or enumerated.
func TestDarwinOfficialRuntimeArchives(t *testing.T) {
	cache := os.Getenv("FREEHAND_MACOS_RUNTIME_ARCHIVES")
	if cache == "" {
		t.Skip("set FREEHAND_MACOS_RUNTIME_ARCHIVES to pinned release archives")
	}
	for key, recipe := range platformRecipes {
		if key.os != "darwin" {
			continue
		}
		t.Run(string(key.provider)+"/"+key.arch+"/"+key.backend, func(t *testing.T) {
			bundle := recipe.runtimeBundle
			bundle.archives = append([]asset(nil), bundle.archives...)
			for i, a := range bundle.archives {
				u, _ := url.Parse(a.url)
				filename := filepath.Join(cache, path.Base(u.Path))
				if err := verifyFile(t.Context(), filename, a.size, a.sha256); err != nil {
					t.Fatal(err)
				}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, filename) }))
				t.Cleanup(srv.Close)
				bundle.archives[i].url = srv.URL
			}
			root := t.TempDir()
			if err := installRuntimeBundle(t.Context(), root, bundle, http.DefaultClient, nil); err != nil {
				t.Fatal(err)
			}
			if err := verifyRuntimeBundle(t.Context(), filepath.Join(root, "runtime"), bundle); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(root, "runtime", filepath.FromSlash(bundle.executable))
			m, err := macho.Open(exe)
			if err != nil {
				t.Fatal(err)
			}
			for _, load := range m.Loads {
				b := load.Raw()
				if len(b) >= 16 && m.ByteOrder.Uint32(b) == 0x32 {
					t.Logf("Mach-O minimum macOS: %d.%d", m.ByteOrder.Uint32(b[12:16])>>16, (m.ByteOrder.Uint32(b[12:16])>>8)&255)
				}
			}
			m.Close()
			if key.arch != runtime.GOARCH {
				return
			}
			args, env := []string{"--help"}, ggmlEnvironment(os.Environ())
			if key.provider == NeMoSpeechCPP {
				env = childEnvironment(root, os.Environ())
			}
			// First launch can include macOS executable/dylib security checks.
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			p, err := launchOwned(ctx, exe, args, filepath.Dir(exe), env)
			if err != nil {
				t.Fatal(err)
			}
			defer p.kill()
			if err = p.wait(ctx); err != nil {
				t.Fatalf("metadata command failed: %v; %s", err, p.stderr.bytes())
			}
			if key.provider == NeMoSpeechCPP {
				if !strings.Contains(string(p.stdout.bytes()), "nemo-speech") {
					t.Fatal("CLI help missing")
				}
			} else if !strings.Contains(string(p.stdout.bytes()), "--gpu-layers") {
				t.Fatal("server help missing")
			}
		})
	}
}
