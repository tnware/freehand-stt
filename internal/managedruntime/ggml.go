package managedruntime

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Shared acquisition and process boundary for the two pinned GGML recipes.
// NeMo keeps its own CLI model manager and automatic accelerator policy.
type ggmlAdapter struct {
	root          string
	recipe        ggmlProvider
	launch        func(context.Context, string, []string, string, []string) (*ownedProcess, error)
	listenerOwner func(int, int) (bool, error)
}

func (a *ggmlAdapter) executable() string {
	return filepath.Join(a.root, "runtime", filepath.FromSlash(a.recipe.executable))
}
func (a *ggmlAdapter) installed(ctx context.Context) error {
	return verifyRuntime(ctx, a.root, a.recipe.release)
}
func (a *ggmlAdapter) Inspect(ctx context.Context) (string, []Model, error) {
	if err := a.installed(ctx); err != nil {
		return "", nil, err
	}
	models := a.recipe.descriptor().Models
	for i := range models {
		s := a.recipe.specs[models[i].ID]
		models[i].Installed = verifyFile(ctx, s.path(a.root), s.size, s.sha256) == nil
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
	}
	return "cpu", models, nil
}
func (a *ggmlAdapter) Install(ctx context.Context, progress func(float64)) (string, error) {
	if !a.recipe.descriptor().Supported {
		return "", errors.New("Managed runtimes require Windows x64.")
	}
	if err := a.installed(ctx); err == nil {
		return "cpu", nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	client := releaseClient()
	defer client.CloseIdleConnections()
	if err := installBinaryAsset(ctx, a.root, a.recipe.release, a.recipe.executable, client, progress); err != nil {
		return "", err
	}
	if err := a.installed(ctx); err != nil {
		return "", err
	}
	return "cpu", nil
}

// Public model files only. Signed CDN redirects are allowed on exact HF hosts;
// never forward credentials, proxy configuration, or arbitrary redirect origins.
func modelClient() *http.Client {
	c := releaseClient()
	c.Timeout = 45 * time.Minute
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		h := req.URL.Hostname()
		if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil || req.URL.Port() != "" || (h != "huggingface.co" && h != "cdn-lfs.huggingface.co" && h != "cdn-lfs-us-1.huggingface.co" && h != "cas-bridge.xethub.hf.co" && h != "us.aws.cdn.hf.co") {
			return errors.New("Model download redirected outside the official model hosts.")
		}
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
		return nil
	}
	return c
}
func (a *ggmlAdapter) Pull(ctx context.Context, id string, progress func(AcquisitionProgress)) error {
	s, ok := a.recipe.specs[id]
	if !ok {
		return errors.New("Choose a qualified runtime model.")
	}
	if err := a.installed(ctx); err != nil {
		return err
	}
	client := modelClient()
	defer client.CloseIdleConnections()
	return acquireModel(ctx, a.root, s, client, progress)
}
func acquireModel(ctx context.Context, root string, s modelSpec, client *http.Client, progress func(AcquisitionProgress)) error {
	report := acquisitionReporter{emit: progress}
	report.update(AcquisitionProgress{Phase: "preparing"})
	if err := safeRoot(root); err != nil {
		return err
	}
	if err := verifyFile(ctx, s.path(root), s.size, s.sha256); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.directory(root), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.directory(root), ".download-")
	if err != nil {
		return err
	}
	stage := f.Name()
	defer os.Remove(stage)
	bounded, cancel := context.WithTimeout(ctx, 45*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(bounded, http.MethodGet, "https://huggingface.co/"+s.repo+"/resolve/"+s.revision+"/"+s.filename, nil)
	if err != nil {
		f.Close()
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		f.Close()
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || res.ContentLength > 0 && res.ContentLength != s.size {
		f.Close()
		return errIntegrity
	}
	report.update(AcquisitionProgress{Phase: "downloading", TotalBytes: s.size})
	n, err := io.Copy(&acquisitionWriter{Writer: f, report: &report, total: s.size}, io.LimitReader(contextReader{bounded, res.Body}, s.size+1))
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	if n != s.size {
		return errIntegrity
	}
	report.update(AcquisitionProgress{Phase: "verifying", Bytes: n, TotalBytes: s.size})
	if err = verifyFile(bounded, stage, s.size, s.sha256); err != nil {
		return err
	}
	if err = safeRoot(root); err != nil {
		return err
	}
	if err = bounded.Err(); err != nil {
		return err
	}
	return os.Rename(stage, s.path(root))
}
func (a *ggmlAdapter) RemoveModel(ctx context.Context, id string) error {
	s, ok := a.recipe.specs[id]
	if !ok {
		return errors.New("Choose a qualified runtime model.")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeRoot(a.root); err != nil {
		return err
	}
	// Whisper variants share a revision directory; never delete their siblings.
	err := os.Remove(s.path(a.root))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
func (a *ggmlAdapter) Remove(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeRoot(a.root); err != nil {
		return err
	}
	return os.RemoveAll(a.root)
}

// Allowlist OS necessities rather than enumerate all current/future LLAMA_,
// WHISPER_, GGML_, HF_, curl/proxy/auth and backend-plugin override variables.
func ggmlEnvironment(inherited []string) []string {
	env := []string{}
	for _, v := range inherited {
		key, _, _ := strings.Cut(v, "=")
		switch strings.ToUpper(key) {
		case "SYSTEMROOT", "WINDIR", "TEMP", "TMP":
			env = append(env, v)
		}
	}
	return env
}
func (g ggmlProvider) arguments(model string, s modelSpec, root string, port int) []string {
	args := []string{"-m", s.path(root), "--host", "127.0.0.1", "--port", strconv.Itoa(port)}
	if g.id == LlamaCPP {
		return append(args, "--alias", model, "--reasoning", "off", "--no-warmup", "--parallel", "1", "--ctx-size", "4096", "--gpu-layers", "0", "--no-op-offload", "--offline", "--no-ui", "--no-agent", "--no-ui-mcp-proxy", "--log-disable")
	}
	return append(args, "--no-gpu", "--no-flash-attn")
}
func (g ggmlProvider) endpoint(origin, model string) Endpoint {
	base := origin
	if g.id == LlamaCPP {
		base += "/v1"
	}
	return Endpoint{Enabled: true, BaseURL: base, Model: model, Profile: string(g.profile)}
}
func (a *ggmlAdapter) waitReady(ctx context.Context, p *ownedProcess, base string, port int, model string) error {
	client := loopbackClient()
	defer client.CloseIdleConnections()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.done:
			return errors.New("Managed runtime exited before becoming ready.")
		default:
		}
		owner, err := a.listenerOwner(port, p.pid)
		if err != nil {
			return err
		}
		var health struct{ Status string }
		if owner && metadataJSON(ctx, client, base+"/health", &health) == nil && health.Status == "ok" {
			identity := true
			if a.recipe.id == LlamaCPP {
				var models struct{ Data []struct{ ID string } }
				identity = metadataJSON(ctx, client, base+"/v1/models", &models) == nil && len(models.Data) == 1 && models.Data[0].ID == model
			}
			// whisper.cpp has no model inventory API. Identity is the exact verified
			// local file passed to the owned PID, not an invented /v1/models route.
			if identity {
				owner, err = a.listenerOwner(port, p.pid)
				if err != nil {
					return err
				}
				if owner {
					select {
					case <-p.done:
						return errors.New("Managed runtime stopped during readiness.")
					case <-ctx.Done():
						return ctx.Err()
					default:
						return nil
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.done:
			return errors.New("Managed runtime exited before becoming ready.")
		case <-tick.C:
		}
	}
}
func (a *ggmlAdapter) Start(ctx context.Context, id string) (*ownedProcess, Endpoint, error) {
	s, ok := a.recipe.specs[id]
	if !ok {
		return nil, Endpoint{}, errors.New("Choose a qualified runtime model.")
	}
	if !a.recipe.descriptor().Supported {
		return nil, Endpoint{}, errors.New("Managed runtimes require Windows x64.")
	}
	if err := a.installed(ctx); err != nil {
		return nil, Endpoint{}, err
	}
	if err := verifyFile(ctx, s.path(a.root), s.size, s.sha256); err != nil {
		return nil, Endpoint{}, err
	}
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, Endpoint{}, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	p, err := a.launch(ctx, a.executable(), a.recipe.arguments(id, s, a.root, port), filepath.Dir(a.executable()), ggmlEnvironment(os.Environ()))
	if err != nil {
		return nil, Endpoint{}, err
	}
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	bounded, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	if err = a.waitReady(bounded, p, base, port, id); err != nil {
		p.kill()
		stop, done := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
		defer done()
		_ = p.wait(stop)
		return p, Endpoint{}, err
	}
	return p, a.recipe.endpoint(base, id), nil
}
