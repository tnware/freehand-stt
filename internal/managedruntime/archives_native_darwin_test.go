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

	"github.com/tnware/freehand-stt/internal/managedruntime/internal/artifact"
)

// macOS exposes /var as an OS symlink. Use its physical temporary directory
// for fixtures; production managed storage still rejects linked ancestors.
func TestMain(m *testing.M) {
	dir, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil || os.Setenv("TMPDIR", dir) != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

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
			bundle := recipe.Bundle
			bundle.Archives = append([]artifact.Asset(nil), bundle.Archives...)
			for i, a := range bundle.Archives {
				u, _ := url.Parse(a.URL)
				filename := filepath.Join(cache, path.Base(u.Path))
				if err := artifact.VerifyFile(t.Context(), filename, a.Size, a.SHA256); err != nil {
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
			defer p.Kill()
			if err = p.Wait(ctx); err != nil {
				t.Fatalf("metadata command failed: %v; %s", err, p.Stderr())
			}
			if key.provider == NeMoSpeechCPP {
				if !strings.Contains(string(p.Stdout()), "nemo-speech") {
					t.Fatal("CLI help missing")
				}
			} else if !strings.Contains(string(p.Stdout()), "--gpu-layers") {
				t.Fatal("server help missing")
			}
		})
	}
}
