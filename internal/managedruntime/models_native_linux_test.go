//go:build linux

package managedruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/tnware/freehand-stt/internal/managedruntime/internal/artifact"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
)

// Explicit selected-model qualification only, never an inventory sweep or CI
// inference test. Each invocation selects exactly one provider and model pair.
func TestLinuxSelectedModelAcceptance(t *testing.T) {
	selected := os.Getenv("FREEHAND_LINUX_SELECTED_MODEL")
	if selected == "" {
		t.Skip("explicitly select nemotron-3.5 or s1-mini to download/load/test")
	}
	root := os.Getenv("FREEHAND_LINUX_SELECTED_ROOT")
	if root == "" {
		root = t.TempDir()
	} else {
		root = filepath.Join(root, selected)
	}
	var adapter runtimeAdapter
	switch selected {
	case "nemotron-3.5":
		adapter = newAdapter(root)
	case "s1-mini":
		adapter = llamaProvider.newAdapter(root)
	default:
		t.Fatal("unsupported explicit test model")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
	defer cancel()
	// The real managed downloader verifies official archive and model pins.
	if fixture := os.Getenv("FREEHAND_LINUX_MODEL_FIXTURES"); fixture != "" {
		provider, spec := LlamaCPP, llamaProvider.specs["s1-mini"]
		if selected == "nemotron-3.5" {
			provider, spec = NeMoSpeechCPP, modelSpecs[selected]
		}
		recipe, ok := recipeFor(provider, "linux", "amd64", "cpu")
		if !ok {
			t.Fatal("missing recipe")
		}
		bundle := recipe.Bundle
		bundle.Archives = append([]artifact.Asset(nil), bundle.Archives...)
		for i, a := range bundle.Archives {
			u, err := url.Parse(a.URL)
			if err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(fixture, path.Base(u.Path))
			if err := artifact.VerifyFile(ctx, filename, a.Size, a.SHA256); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, filename) }))
			defer server.Close()
			bundle.Archives[i].URL = server.URL
		}
		if err := artifact.Install(ctx, root, bundle, http.DefaultClient, nil); err != nil {
			t.Fatal(err)
		}
		source := filepath.Join(fixture, spec.filename)
		if err := artifact.VerifyFile(ctx, source, spec.size, spec.sha256); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(spec.directory(root), 0700); err != nil {
			t.Fatal(err)
		}
		if err := artifact.VerifyFile(ctx, spec.path(root), spec.size, spec.sha256); err != nil {
			in, err := os.Open(source)
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()
			out, err := os.OpenFile(spec.path(root), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			_, copyErr := io.Copy(out, in)
			closeErr := out.Close()
			if copyErr != nil || closeErr != nil {
				t.Fatal(copyErr, closeErr)
			}
		}
		t.Log("pre-downloaded official fixtures verified and installed; direct network downloader not exercised")
	} else {
		if _, err := adapter.Install(ctx, nil); err != nil {
			t.Fatal("install:", err)
		}
		t.Log("official CPU runtime downloaded, installed, and integrity-verified")
	}
	last := ""
	if err := adapter.Pull(ctx, selected, func(p AcquisitionProgress) {
		if p.Phase != last {
			t.Log("selected model acquisition:", p.Phase)
			last = p.Phase
		}
	}); err != nil {
		t.Fatal("selected model acquisition:", err)
	}
	t.Log("selected model acquisition/cache verification passed")
	started := time.Now()
	p, endpoint, err := adapter.Start(ctx, selected)
	if p != nil {
		defer func() {
			p.Kill()
			stopped, done := context.WithTimeout(context.Background(), 5*time.Second)
			defer done()
			_ = p.Wait(stopped)
		}()
	}
	if err != nil {
		t.Fatal("selected model start:", err)
	}
	t.Log("selected model ready with owned loopback listener after", time.Since(started))
	bounded, done := context.WithTimeout(ctx, 90*time.Second)
	defer done()
	var request *http.Request
	if selected == "s1-mini" {
		data, err := json.Marshal(map[string]any{"model": endpoint.Model, "messages": []map[string]string{{"role": "user", "content": "um hello world"}}, "max_tokens": 16, "temperature": 0})
		if err != nil {
			t.Fatal(err)
		}
		request, err = http.NewRequestWithContext(bounded, http.MethodPost, endpoint.BaseURL+"/chat/completions", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
	} else {
		// Synthetic silence stays in memory; there is no recording or retained audio.
		wav, err := audio.PCM16WAV(make([]byte, 16000*2), 16000, 1)
		if err != nil {
			t.Fatal(err)
		}
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		file, err := form.CreateFormFile("file", "qualification.wav")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(wav); err != nil {
			t.Fatal(err)
		}
		if err := form.WriteField("model", endpoint.Model); err != nil {
			t.Fatal(err)
		}
		if err := form.WriteField("response_format", "json"); err != nil {
			t.Fatal(err)
		}
		if err := form.Close(); err != nil {
			t.Fatal(err)
		}
		request, err = http.NewRequestWithContext(bounded, http.MethodPost, endpoint.BaseURL+"/audio/transcriptions", &body)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", form.FormDataContentType())
	}
	client := loopbackClient()
	client.Timeout = 0
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {

		t.Fatal("bounded inference:", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("inference status %d", response.StatusCode)
	}
	var result struct {
		Text    *string           `json:"text"`
		Choices []json.RawMessage `json:"choices"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if selected == "s1-mini" && len(result.Choices) != 1 {
		t.Fatal("missing completion")
	}
	if selected == "nemotron-3.5" && result.Text == nil {
		t.Fatal("missing transcription")
	}
	t.Log("bounded inference returned HTTP 200 with expected response shape; content discarded")
	p.Kill()
	stopped, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_ = p.Wait(stopped)
	select {
	case <-p.Done():
		t.Log("owned runtime stopped")
	default:
		t.Fatal("runtime did not stop")
	}
}
