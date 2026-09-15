package managedruntime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type nemoAdapter struct {
	root          string
	launch        func(context.Context, string, []string, string, []string) (*ownedProcess, error)
	listenerOwner func(int, int) (bool, error)
}

func newAdapter(root string) *nemoAdapter {
	return &nemoAdapter{root: root, launch: launchOwned, listenerOwner: ownsListener}
}

// NeMo's parser accepts arbitrary NEMO_SPEECH_* configuration, including
// model paths, API keys and download origins. Never inherit that configuration.
func childEnvironment(root string, inherited []string) []string {
	if runtime.GOOS == "darwin" {
		// NeMo invokes system curl for explicit model acquisition. Do not inherit
		// shell PATH, DYLD injection, proxy/auth, or model-manager configuration.
		return append(ggmlEnvironment(inherited), "PATH=/usr/bin:/bin", "NEMO_SPEECH_MODEL_DIR="+filepath.Join(root, "models"))
	}
	env := make([]string, 0, len(inherited)+1)
	for _, v := range inherited {
		key, _, _ := strings.Cut(v, "=")
		key = strings.ToUpper(key)
		if strings.HasPrefix(key, "NEMO_") || strings.HasPrefix(key, "HF_") || strings.HasPrefix(key, "HUGGING_FACE_") {
			continue
		}
		env = append(env, v)
	}
	return append(env, "NEMO_SPEECH_MODEL_DIR="+filepath.Join(root, "models"))
}
func (a *nemoAdapter) executable() string {
	r, _ := recipeFor(NeMoSpeechCPP, runtime.GOOS, runtime.GOARCH, "cpu")
	return filepath.Join(a.root, "runtime", filepath.FromSlash(r.executable))
}
func (a *nemoAdapter) command(ctx context.Context, args ...string) ([]byte, error) {
	return a.commandProgress(ctx, nil, args...)
}
func (a *nemoAdapter) commandProgress(ctx context.Context, sample func(), args ...string) ([]byte, error) {
	p, err := a.launch(ctx, a.executable(), args, filepath.Join(a.root, "runtime"), childEnvironment(a.root, os.Environ()))
	if err != nil {
		return nil, err
	}
	if sample != nil {
		tick := time.NewTicker(acquisitionInterval)
		defer tick.Stop()
		sample()
	waitProgress:
		for {
			select {
			case <-ctx.Done():
				break waitProgress
			case <-p.done:
				break waitProgress
			case <-tick.C:
				sample()
			}
		}
	}
	if err = p.wait(ctx); err != nil {
		// Job cancellation can race the child's exit notification. Its OS exit
		// error is not an acquisition failure when our cancellation caused it.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	p.stdout.mu.Lock()
	overflow := p.stdout.overflow
	p.stdout.mu.Unlock()
	if overflow {
		return nil, errors.New("Runtime metadata exceeded its limit.")
	}
	return p.stdout.bytes(), nil
}
func (a *nemoAdapter) installedBackend(ctx context.Context) (string, error) {
	if err := safeRoot(a.root); err != nil {
		return "", err
	}
	marker, err := os.ReadFile(filepath.Join(a.root, "runtime", ".backend"))
	if err != nil {
		return "", err
	}
	selected, ok := recipeFor(NeMoSpeechCPP, runtime.GOOS, runtime.GOARCH, string(marker))
	if !ok {
		return "", errIntegrity
	}
	if err = verifyRuntimeBundle(ctx, filepath.Join(a.root, "runtime"), selected.runtimeBundle); err != nil {
		return "", err
	}
	return string(marker), nil
}
func (a *nemoAdapter) Inspect(ctx context.Context) (string, []Model, error) {
	if err := a.clearDownloadDiagnostics(); err != nil {
		return "", nil, err
	}
	backend, err := a.installedBackend(ctx)
	if err != nil {
		return "", nil, err
	}
	bounded, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	b, err := a.command(bounded, "--json", "model", "list")
	if err != nil {
		return backend, nil, err
	}
	models, err := parseCatalog(b)
	if err != nil {
		return backend, nil, err
	}
	for i := range models {
		err := verifyNeMoModel(ctx, a.root, models[i].ID)
		models[i].Installed = err == nil
		if ctx.Err() != nil {
			return backend, nil, ctx.Err()
		}
	}
	return backend, models, nil
}
func releaseClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Minute, Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 15 * time.Second}).DialContext, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 30 * time.Second}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		host := req.URL.Hostname()
		if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil || req.URL.Port() != "" || (host != "github.com" && host != "release-assets.githubusercontent.com" && host != "objects.githubusercontent.com") {
			return errors.New("Runtime download redirected outside the official release host.")
		}
		return nil
	}}
}

// Probe driver metadata only, never instantiate an inference backend. CUDA 13
// requires a recent driver; uncertain/older systems conservatively use CPU.
func (a *nemoAdapter) chooseAsset(ctx context.Context) asset {
	if runtime.GOOS != "windows" {
		backend := "cpu"
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			backend = "metal"
		}
		r, ok := recipeFor(NeMoSpeechCPP, runtime.GOOS, runtime.GOARCH, backend)
		if !ok {
			return asset{}
		}
		return r.archives[0]
	}
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "nvidia-smi.exe")
	if !filepath.IsAbs(exe) {
		return assets["cpu"]
	}
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p, err := a.launch(probe, exe, []string{"--id=0", "--query-gpu=driver_version,compute_cap", "--format=csv,noheader,nounits"}, a.root, childEnvironment(a.root, os.Environ()))
	if err != nil || p.wait(probe) != nil {
		return assets["cpu"]
	}
	for _, line := range strings.Split(string(p.stdout.bytes()), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			continue
		}
		driver, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		capability, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if driver >= 581 && capability >= 7.5 {
			return assets["cuda"]
		}
	}
	return assets["cpu"]
}
func (a *nemoAdapter) Install(ctx context.Context, progress func(float64)) (string, error) {
	if !(nemoProvider{}).descriptor().Supported {
		return "", errUnsupported
	}
	if err := safeRoot(a.root); err != nil {
		return "", err
	}
	if err := os.MkdirAll(a.root, 0700); err != nil {
		return "", err
	}
	if backend, err := a.installedBackend(ctx); err == nil {
		return backend, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	selected := a.chooseAsset(ctx)
	client := releaseClient()
	defer client.CloseIdleConnections()
	recipe, ok := recipeFor(NeMoSpeechCPP, runtime.GOOS, runtime.GOARCH, selected.backend)
	if !ok {
		return "", errUnsupported
	}
	if err := installRuntimeBundle(ctx, a.root, recipe.runtimeBundle, client, progress); err != nil {
		return "", err
	}
	if err := verifyRuntimeBundle(ctx, filepath.Join(a.root, "runtime"), recipe.runtimeBundle); err != nil {
		return "", err
	}
	return selected.backend, nil
}
func (a *nemoAdapter) Pull(ctx context.Context, id string, progress func(AcquisitionProgress)) error {
	report := acquisitionReporter{emit: progress}
	report.update(AcquisitionProgress{Phase: "preparing"})
	spec, ok := modelSpecs[id]
	if !ok {
		return errors.New("Choose a supported managed speech model.")
	}
	_, models, err := a.Inspect(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, m := range models {
		if m.ID == id {
			found = true
			if m.Installed {
				return nil
			}
		}
	}
	if !found {
		return errors.New("The selected model is not in the qualified runtime catalog.")
	}
	bounded, cancel := context.WithTimeout(ctx, 45*time.Minute)
	defer cancel()
	_, pullErr := a.commandProgress(bounded, func() {
		report.update(nemoModelAcquiredBytes(a.root, id))
	}, "--json", "model", "pull", spec.repo)
	if err := a.clearDownloadDiagnostics(); err != nil {
		return err
	}
	if pullErr != nil {
		return pullErr
	}
	if err = safeRoot(a.root); err != nil {
		return err
	}
	measured := nemoModelAcquiredBytes(a.root, id)
	measured.Phase = "verifying"
	report.update(measured)
	return verifyNeMoModel(ctx, a.root, id)
}
func (a *nemoAdapter) RemoveModel(ctx context.Context, id string) error {
	spec, ok := modelSpecs[id]
	if !ok {
		return errors.New("Choose a supported managed speech model.")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeRoot(a.root); err != nil {
		return err
	}
	// v0.1.0 has list/pull/info, not remove. Delete only this pinned revision.
	if err := os.RemoveAll(spec.directory(a.root)); err != nil {
		return err
	}
	if id == "magpie-tts" {
		// This qualified codec belongs exclusively to the selected TTS bundle.
		return os.RemoveAll(nemoCodecSpec.directory(a.root))
	}
	return nil
}
func (a *nemoAdapter) Remove(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeRoot(a.root); err != nil {
		return err
	}
	return os.RemoveAll(a.root)
}
func loopbackClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
func metadataJSON(ctx context.Context, client *http.Client, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("Runtime is not ready.")
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 64<<10+1))
	if err != nil {
		return err
	}
	if len(b) > 64<<10 {
		return errors.New("Runtime metadata exceeded its limit.")
	}
	return json.Unmarshal(b, target)
}
func (a *nemoAdapter) waitReady(ctx context.Context, p *ownedProcess, base string, port int) (string, error) {
	model, _, err := a.waitReadyModels(ctx, p, base, port, false)
	return model, err
}

func (a *nemoAdapter) waitReadyModels(ctx context.Context, p *ownedProcess, base string, port int, speech bool) (string, string, error) {
	client := loopbackClient()
	defer client.CloseIdleConnections()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-p.done:
			return "", "", errors.New("Managed speech exited before becoming ready.")
		default:
		}
		var ready struct{ Ready bool }
		if metadataJSON(ctx, client, base+"/ready", &ready) == nil && ready.Ready {
			owner, err := a.listenerOwner(port, p.pid)
			if err != nil {
				return "", "", err
			}
			if owner {
				var inventory struct {
					Data []nemoLoadedModel
				}
				if err = metadataJSON(ctx, client, base+"/v1/models", &inventory); err == nil {
					// Each configured engine was given exact verified local inputs.
					// v0.1.0 reports the GGUF general.name as the request model identity.
					model, speechModel, err := nemoModelIdentities(inventory.Data, speech)
					if err != nil {
						return "", "", err
					}
					owner, err = a.listenerOwner(port, p.pid)
					if err != nil {
						return "", "", err
					}
					if owner {
						select {
						case <-p.done:
							return "", "", errors.New("Managed speech stopped during readiness.")
						case <-ctx.Done():
							return "", "", ctx.Err()
						default:
							return model, speechModel, nil
						}
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-p.done:
			return "", "", errors.New("Managed speech exited before becoming ready.")
		case <-tick.C:
		}
	}
}
func (a *nemoAdapter) Start(ctx context.Context, id string) (*ownedProcess, Endpoint, error) {
	return a.StartModels(ctx, id, "")
}

func (a *nemoAdapter) StartModels(ctx context.Context, id, speechID string) (*ownedProcess, Endpoint, error) {
	q, ok := qualified[id]
	if !ok || id == "magpie-tts" {
		return nil, Endpoint{}, errors.New("Choose a supported managed speech model.")
	}
	if speechID != "" && speechID != "magpie-tts" {
		return nil, Endpoint{}, errors.New("Choose a supported managed text-to-speech model.")
	}
	reportStartupProgress(ctx, "verifying_runtime")
	backend, err := a.installedBackend(ctx)
	if err != nil {
		return nil, Endpoint{}, err
	}
	spec := modelSpecs[id]
	reportStartupProgress(ctx, "verifying_model")
	if err = verifyNeMoModel(ctx, a.root, id); err != nil {
		return nil, Endpoint{}, err
	}
	if speechID != "" {
		if err = verifyNeMoModel(ctx, a.root, speechID); err != nil {
			return nil, Endpoint{}, err
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, Endpoint{}, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	device := "cpu"
	if backend != "cpu" {
		device = "gpu:0"
	}
	args := []string{"serve", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--asr-model", spec.path(a.root), "--device", device, "--no-ui", "--access-log", "--log-format", "json"}
	if speechID != "" {
		args = append(args, "--tts-model", modelSpecs[speechID].path(a.root), "--codec-model", nemoCodecSpec.path(a.root), "--tokenizer-dir", nemoTokenizerDirectory(a.root))
	}
	if backend == "cpu" {
		args = append(args, "--no-warmup")
	}
	reportStartupProgress(ctx, "launching")
	p, err := a.launch(runtimeProcessContext(ctx), a.executable(), args, filepath.Join(a.root, "runtime"), childEnvironment(a.root, os.Environ()))
	if err != nil {
		return nil, Endpoint{}, err
	}
	bounded, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	if backend != "cpu" {
		reportStartupProgress(ctx, "loading_warming")
	} else {
		reportStartupProgress(ctx, "waiting_ready")
	}
	model, speechModel, err := a.waitReadyModels(bounded, p, base, port, speechID != "")
	if err != nil {
		p.kill()
		stop, done := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
		defer done()
		_ = p.wait(stop)
		return p, Endpoint{}, err
	}
	// Readiness uses the server origin; speech clients append their routes to
	// the OpenAI-compatible API base, which includes NeMo's /v1 prefix.
	return p, Endpoint{Enabled: true, BaseURL: base + "/v1", Model: model, SpeechModel: speechModel, Realtime: q.Realtime, Profile: q.Profile}, nil
}
