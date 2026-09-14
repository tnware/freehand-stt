//go:build windows

package managedruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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
			if err := verifyFile(t.Context(), archive, g.release.size, g.release.sha256); err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, archive) }))
			defer srv.Close()
			local := g.release
			local.url = srv.URL
			root := t.TempDir()
			if err := installBinaryAsset(t.Context(), root, local, g.executable, srv.Client(), nil); err != nil {
				t.Fatal(err)
			}
			a := g.newAdapter(root).(*ggmlAdapter)
			if err := a.installed(t.Context()); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			model := g.models[0].ID
			args := append(g.arguments(model, g.specs[model], root, 18080), "--help")
			p, err := a.launch(ctx, a.executable(), args, filepath.Dir(a.executable()), ggmlEnvironment(os.Environ()))
			if err != nil {
				t.Fatal(err)
			}
			if err = p.wait(ctx); err != nil {
				t.Fatal(err)
			}
			if len(p.stdout.bytes())+len(p.stderr.bytes()) == 0 {
				t.Fatal("no CLI help output")
			}
			t.Log("official ZIP installed and reverified; owned Windows --help exited successfully")
		})
	}
}
