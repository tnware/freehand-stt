package managedruntime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exercise actual adapter verification, launch admission and readiness; replace
// only archive/model bytes and the external child with a loopback HTTP fixture.
func WithGGMLEndpointFixture(t *testing.T, id ProviderID, handler http.Handler, exercise func(Endpoint)) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows provider")
	}
	g := llamaProvider
	if id == WhisperCPP {
		g = whisperProvider
	}
	key := g.models[0].ID
	g.specs = map[string]modelSpec{key: g.specs[key]}
	s := g.specs[key]
	s.size = 7
	s.sha256 = fmt.Sprintf("%x", sha256.Sum256([]byte("fixture")))
	g.specs[key] = s
	root := t.TempDir()
	data := zipFixture(t, map[string]string{g.executable: "test executable, not run"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()
	g.release = asset{"cpu", srv.URL, fmt.Sprintf("%x", sha256.Sum256(data)), int64(len(data))}
	if err := installBinaryAsset(t.Context(), root, g.release, g.executable, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(s.directory(root), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path(root), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	a := g.newAdapter(root).(*ggmlAdapter)
	a.launch = func(ctx context.Context, exe string, args []string, dir string, env []string) (*ownedProcess, error) {
		if exe != filepath.Join(root, "runtime", filepath.FromSlash(g.executable)) || dir != filepath.Dir(exe) {
			t.Fatal("unowned executable")
		}
		value := func(flag string) string {
			i := slices.Index(args, flag)
			if i < 0 || i+1 >= len(args) {
				t.Fatalf("missing %s", flag)
			}
			return args[i+1]
		}
		if value("-m") != s.path(root) || value("--host") != "127.0.0.1" {
			t.Fatal("unsafe model/listener")
		}
		if g.id == LlamaCPP {
			if value("--reasoning") != "off" || value("--parallel") != "1" || value("--ctx-size") != "4096" || value("--gpu-layers") != "0" || !slices.Contains(args, "--no-warmup") || !slices.Contains(args, "--offline") {
				t.Fatal("unsafe llama arguments")
			}
		} else if !slices.Contains(args, "--no-gpu") || !slices.Contains(args, "--no-flash-attn") {
			t.Fatal("unsafe whisper arguments")
		}
		l, err := net.Listen("tcp4", "127.0.0.1:"+value("--port"))
		if err != nil {
			return nil, err
		}
		server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/health":
				w.Write([]byte(`{"status":"ok"}`))
			case "/v1/models":
				if g.id != LlamaCPP {
					t.Error("whisper inventory probe")
				}
				fmt.Fprintf(w, `{"data":[{"id":%q}]}`, key)
			default:
				handler.ServeHTTP(w, r)
			}
		})}
		p := &ownedProcess{done: make(chan struct{}), pid: 123}
		p.closeJob = func() { server.Close() }
		go func() { _ = server.Serve(l); close(p.done) }()
		return p, nil
	}
	a.listenerOwner = func(int, int) (bool, error) { return true, nil }
	p, ep, err := a.Start(t.Context(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer p.kill()
	exercise(ep)
	p.kill()
	// Exact bytes remain required at each start, even after an earlier success.
	if err = os.WriteFile(s.path(root), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = a.Start(t.Context(), key); err == nil {
		t.Fatal("started modified model")
	}
}

func TestGGMLReadinessRejectsForeignDeadAndWrongAlias(t *testing.T) {
	for _, g := range []ggmlProvider{llamaProvider, whisperProvider} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.Write([]byte(`{"status":"ok"}`))
			} else {
				w.Write([]byte(`{"data":[{"id":"wrong"}]}`))
			}
		}))
		a := g.newAdapter(t.TempDir()).(*ggmlAdapter)
		p := &ownedProcess{done: make(chan struct{}), pid: 123}
		for _, owned := range []bool{false, true} {
			if owned && g.id == WhisperCPP {
				continue
			}
			a.listenerOwner = func(int, int) (bool, error) { return owned, nil }
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
			err := a.waitReady(ctx, p, srv.URL, 123, "s1-mini")
			cancel()
			if err == nil {
				t.Fatal("adopted wrong listener/model")
			}
		}
		close(p.done)
		if err := a.waitReady(t.Context(), p, srv.URL, 123, "s1-mini"); err == nil {
			t.Fatal("adopted dead process")
		}
		srv.Close()
	}
}
func TestGGMLDownloadIntegrityCancellationAndSiblingRemoval(t *testing.T) {
	g := whisperProvider
	root := t.TempDir()
	s := g.specs["base"]
	s.size = 7
	s.sha256 = fmt.Sprintf("%x", sha256.Sum256([]byte("fixture")))
	body := "corrupt"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
	defer srv.Close()
	client := srv.Client()
	client.Transport = ggmlRewriteTransport{srv.URL, client.Transport}
	if err := acquireModel(t.Context(), root, s, client, nil); err == nil {
		t.Fatal("published corrupt download")
	}
	entries, _ := os.ReadDir(s.directory(root))
	if len(entries) != 0 {
		t.Fatal("partial download retained")
	}
	body = "fixture"
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := acquireModel(ctx, root, s, client, nil); err == nil {
		t.Fatal("ignored cancellation")
	}
	if err := acquireModel(t.Context(), root, s, client, nil); err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(s.directory(root), g.specs["small"].filename)
	if err := os.WriteFile(sibling, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	a := g.newAdapter(root)
	if err := a.RemoveModel(t.Context(), "base"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Fatal("deleted sibling", err)
	}
}

type ggmlRewriteTransport struct {
	url  string
	next http.RoundTripper
}

func (r ggmlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	dest, _ := http.NewRequest("GET", r.url, nil)
	req.URL.Scheme = dest.URL.Scheme
	req.URL.Host = dest.URL.Host
	return r.next.RoundTrip(req)
}
func TestGGMLPinnedSourcesAndEnvironment(t *testing.T) {
	for _, g := range []ggmlProvider{llamaProvider, whisperProvider} {
		if len(g.release.sha256) != 64 || g.release.size <= 0 || !strings.HasPrefix(g.release.url, "https://github.com/ggml-org/") || !strings.Contains(g.release.url, "/releases/download/"+g.version+"/") {
			t.Fatal("unpinned runtime")
		}
		for _, s := range g.specs {
			if len(s.revision) != 40 || len(s.sha256) != 64 || s.size <= 0 || strings.ContainsAny(s.filename, "/\\:") {
				t.Fatal("unpinned model")
			}
			if _, err := strconv.ParseUint(s.revision[:16], 16, 64); err != nil {
				t.Fatal(err)
			}
		}
	}
	env := ggmlEnvironment([]string{"SystemRoot=C:\\Windows", "TEMP=C:\\Temp", "LLAMA_ARG_HOST=0.0.0.0", "LLAMA_ARG_MODEL_URL=bad", "GGML_BACKEND_PATH=bad", "HF_TOKEN=secret", "HUGGING_FACE_HUB_TOKEN=secret", "HTTPS_PROXY=bad", "PATH=foreign", "WHISPER_MODEL=bad"})
	if len(env) != 2 {
		t.Fatalf("inherited runtime config: %v", env)
	}
	c := modelClient()
	defer c.CloseIdleConnections()
	// The pinned public model URLs currently redirect to this HF CDN.
	r, _ := http.NewRequest("GET", "https://us.aws.cdn.hf.co/model", nil)
	r.Header.Set("Authorization", "canary")
	r.Header.Set("Cookie", "canary")
	if err := c.CheckRedirect(r, nil); err != nil || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
		t.Fatal("public model CDN rejected or credentials forwarded", err)
	}
	for _, u := range []string{"https://evil.example/x", "http://huggingface.co/x", "https://user@huggingface.co/x", "https://huggingface.co:443/x", "https://cas-bridge.xethub.hf.co.evil.example/x"} {
		r, _ := http.NewRequest("GET", u, nil)
		if c.CheckRedirect(r, nil) == nil {
			t.Fatal("unsafe redirect", u)
		}
	}
}
