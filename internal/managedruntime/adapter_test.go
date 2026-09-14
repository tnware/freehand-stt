package managedruntime

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestAdapterCatalogPullStartLifecycle(t *testing.T) {
	WithManagedEndpointFixture(t, nil)
}

// WithManagedEndpointFixture exposes only a test adapter to external-package
// tests, allowing the real clients to consume the real adapter's endpoint.
// No runtime binary, downloaded weights, or inference engine is executed.
func WithManagedEndpointFixture(t *testing.T, exercise func(Endpoint)) {
	t.Helper()
	root := t.TempDir()
	ctx := context.Background()
	a, calls := managedAdapterFixture(t, root)
	backend, models, err := a.Inspect(ctx)
	if err != nil || backend != "cpu" || len(models) != 2 || models[0].Installed {
		t.Fatalf("inspect %s %+v %v", backend, models, err)
	}
	if len(*calls) != 1 || (*calls)[0] != "--json model list" {
		t.Fatal(calls)
	}
	if err := a.Pull(ctx, "nemotron-3.5", nil); err != nil {
		t.Fatal(err)
	}
	_, models, err = a.Inspect(ctx)
	if err != nil || !models[0].Installed {
		t.Fatalf("verified model absent: %+v %v", models, err)
	}
	a.listenerOwner = func(int, int) (bool, error) { return true, nil }
	p, ep, err := a.Start(ctx, "nemotron-3.5")
	if err != nil {
		t.Fatal(err)
	}
	defer p.kill()
	if ep.Model != "actual GGUF name" || !ep.Enabled || !ep.Realtime || ep.Profile != qualified["nemotron-3.5"].Profile {
		t.Fatal(ep)
	}
	if exercise != nil {
		exercise(ep)
	}
	p.kill()
	if err = a.RemoveModel(ctx, "nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = a.Start(ctx, "nemotron-3.5"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing model not rejected: %v", err)
	}
	if err = a.Remove(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(root); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// managedAdapterFixture retains the real integrity, cache, catalog and HTTP
// readiness paths, replacing only download bytes and the OS process boundary.
func managedAdapterFixture(t *testing.T, root string) (*nemoAdapter, *[]string) {
	t.Helper()
	ctx := context.Background()
	a := newAdapter(root)
	key := platformRecipeKey{NeMoSpeechCPP, runtime.GOOS, runtime.GOARCH, "cpu"}
	originalRecipe, existed := platformRecipes[key]
	recipe := originalRecipe
	if recipe.executable == "" {
		recipe = platformRecipes[platformRecipeKey{NeMoSpeechCPP, "windows", "amd64", "cpu"}]
	}
	data := zipFixture(t, map[string]string{recipe.executable: "fixture, never executed"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	t.Cleanup(srv.Close)
	pin := asset{backend: "cpu", url: srv.URL, size: int64(len(data)), sha256: fmt.Sprintf("%x", sha256.Sum256(data))}
	recipe.archives = []asset{pin}
	platformRecipes[key] = recipe
	t.Cleanup(func() {
		if existed {
			platformRecipes[key] = originalRecipe
		} else {
			delete(platformRecipes, key)
		}
	})
	if err := installRuntimeBundle(ctx, root, recipe.runtimeBundle, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	catalog, err := os.ReadFile("testdata/catalog-v0.1.0.json")
	if err != nil {
		t.Fatal(err)
	}
	original := modelSpecs["nemotron-3.5"]
	spec := original
	spec.size = 7
	spec.sha256 = fmt.Sprintf("%x", sha256.Sum256([]byte("fixture")))
	modelSpecs["nemotron-3.5"] = spec
	t.Cleanup(func() { modelSpecs["nemotron-3.5"] = original })
	calls := []string{}
	a.launch = func(ctx context.Context, exe string, args []string, dir string, env []string) (*ownedProcess, error) {
		calls = append(calls, strings.Join(args, " "))
		p := &ownedProcess{done: make(chan struct{}), closeJob: func() {}}
		if args[0] == "serve" {
			if args[1] != "--host" || args[2] != "127.0.0.1" || args[5] != "--asr-model" || args[6] != spec.path(root) {
				t.Fatalf("unsafe serve argv: %v", args)
			}
			port, _ := strconv.Atoi(args[4])
			listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
			if err != nil {
				return nil, err
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/ready" {
					w.Write([]byte(`{"ready":true}`))
				} else if r.URL.Path == "/v1/models" {
					w.Write([]byte(`{"data":[{"id":"actual GGUF name","capability":"transcription"}]}`))
				} else if r.URL.Path == "/v1/audio/transcriptions" && r.Method == http.MethodPost {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"text":"fixture transcript"}`))
				} else if r.URL.Path == "/v1/realtime" {
					conn, err := websocket.Accept(w, r, nil)
					if err != nil {
						return
					}
					defer conn.CloseNow()
					ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
					defer cancel()
					if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"session.created","session":{"model":"actual GGUF name"}}`)); err != nil {
						return
					}
					if _, _, err := conn.Read(ctx); err != nil {
						return
					}
					if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"session.updated"}`)); err != nil {
						return
					}
					kind, _, err := conn.Read(ctx)
					if err == nil && kind == websocket.MessageBinary {
						t.Error("route-only test unexpectedly sent audio")
					}
				} else {
					http.NotFound(w, r)
				}
			})}
			p.closeJob = func() { server.Close(); close(p.done) }
			go server.Serve(listener)
			return p, nil
		}
		if strings.Join(args, " ") == "--json model list" {
			p.stdout.Write(catalog)
		} else if strings.Join(args, " ") == "--json model pull "+spec.repo {
			if err := os.MkdirAll(spec.directory(root), 0700); err != nil {
				return nil, err
			}
			if err := os.WriteFile(spec.path(root), []byte("fixture"), 0600); err != nil {
				return nil, err
			}
		} else {
			t.Fatalf("unexpected command %v", args)
		}
		close(p.done)
		return p, nil
	}

	a.listenerOwner = func(int, int) (bool, error) { return true, nil }
	return a, &calls
}

func TestAdapterReadinessUsesServerIdentity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready":
			w.Write([]byte(`{"ready":true}`))
		case "/v1/models":
			w.Write([]byte(`{"data":[{"id":"GGUF general.name, not alias","capability":"transcription"}]}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	p := &ownedProcess{done: make(chan struct{})}
	a := newAdapter(t.TempDir())
	a.listenerOwner = func(int, int) (bool, error) { return true, nil }
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	id, err := a.waitReady(ctx, p, srv.URL, 123)
	if err != nil || id != "GGUF general.name, not alias" {
		t.Fatalf("%q %v", id, err)
	}
	a.listenerOwner = func(int, int) (bool, error) { return false, nil }
	short, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	if _, err := a.waitReady(short, p, srv.URL, 123); err == nil {
		t.Fatal("adopted foreign listener")
	}
	close(p.done)
	if _, err := a.waitReady(ctx, p, srv.URL, 123); err == nil {
		t.Fatal("adopted dead process")
	}
}
func TestAdapterReleaseRedirectPolicy(t *testing.T) {
	client := releaseClient()
	for _, raw := range []string{"http://release-assets.githubusercontent.com/x", "https://evil.example/x"} {
		req, _ := http.NewRequest("GET", raw, nil)
		if client.CheckRedirect(req, nil) == nil {
			t.Fatal(raw)
		}
	}
	req, _ := http.NewRequest("GET", "https://release-assets.githubusercontent.com/x", nil)
	if err := client.CheckRedirect(req, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterMissingInstallAndIsolation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "owned")
	a := newAdapter(root)
	if _, _, err := a.Inspect(context.Background()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing install: %v", err)
	}
	env := childEnvironment(root, []string{"PATH=system", "NEMO_SPEECH_HTTP_API_KEY=secret", "nemo_speech_model_dir=foreign", "NEMO_SPEECH_HF_BASE_URL=http://bad", "HF_TOKEN=secret"})
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "secret") || strings.Contains(joined, "foreign") || strings.Contains(joined, "http://bad") || !strings.Contains(joined, "NEMO_SPEECH_MODEL_DIR="+filepath.Join(root, "models")) {
		t.Fatal(env)
	}
	if err := a.RemoveModel(context.Background(), "../foreign"); err == nil {
		t.Fatal("accepted unknown model")
	}
	spec := modelSpecs["nemotron-3.5"]
	if err := os.MkdirAll(spec.directory(root), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec.path(root), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, "keep")
	if err := os.WriteFile(other, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveModel(context.Background(), "nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(spec.directory(root)); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatal(err)
	}
}
