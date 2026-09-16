//go:build windows

package managedruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/managedruntime/internal/artifact"
)

// Optional native qualification using explicitly predownloaded official ZIPs.
// No network download, model file, model loading, or inference is performed.
func TestGGMLPinnedRuntimeZIPs(t *testing.T) {
	dir := os.Getenv("FREEHAND_TEST_GGML_ZIPS")
	if dir == "" {
		t.Skip("set FREEHAND_TEST_GGML_ZIPS to a directory containing llama.zip and whisper.zip")
	}
	for _, tc := range []struct {
		name   string
		recipe ggmlProvider
	}{{"llama", llamaProvider}, {"whisper", whisperProvider}} {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.recipe
			archive := filepath.Join(dir, tc.name+".zip")
			if err := artifact.VerifyFile(t.Context(), archive, g.release.Size, g.release.SHA256); err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, archive) }))
			defer srv.Close()
			local := g.release
			local.URL = srv.URL
			root := t.TempDir()
			if err := artifact.Install(t.Context(), root, artifact.Bundle{Archives: []artifact.Asset{local}, Executable: g.Executable}, srv.Client(), nil); err != nil {
				t.Fatal(err)
			}
			a := g.newAdapter(root).(*ggmlAdapter)
			if err := a.installed(t.Context()); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			model := g.models[0].ID
			args := g.arguments(model, g.specs[model], root, 18080)
			if g.id == LlamaCPP {
				// b10809 arg.cpp emits a LOG_WRN for duplicate flags. Unlike
				// printf-based help, this catches --log-disable without loading
				// a model, serving requests, or depending on a startup failure.
				args = append(args, "--offline")
			}
			args = append(args, "--help")
			files := func() []string {
				t.Helper()
				var paths []string
				if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					paths = append(paths, path)
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				return paths
			}
			before := files()
			p, err := a.launch(ctx, a.executable(), args, filepath.Dir(a.executable()), ggmlEnvironment(os.Environ()))
			if err != nil {
				t.Fatal(err)
			}
			if err = p.Wait(ctx); err != nil {
				t.Fatal(err)
			}
			if len(p.Stdout())+len(p.Stderr()) == 0 {
				t.Fatal("no CLI help output")
			}
			if g.id == LlamaCPP {
				if !strings.Contains(string(p.Stderr()), "argument '--offline' specified multiple times") {
					t.Fatal("normal upstream warning missing from owned stderr capture")
				}
				// Prefixes default off, but the warning still emits its SGR reset.
				if !strings.Contains(string(p.Stderr()), "\x1b[0m") {
					t.Fatal("forced upstream ANSI colors missing from private capture")
				}
			}
			if !slices.Equal(before, files()) {
				t.Fatal("metadata-only launch created files (including possible disk logs)")
			}
			t.Log("official ZIP installed and reverified; owned Windows --help exited successfully")
		})
	}
}
