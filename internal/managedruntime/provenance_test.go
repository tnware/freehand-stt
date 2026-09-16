package managedruntime

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

type provenanceTransport func(*http.Request) (*http.Response, error)

func (f provenanceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestModelSourceMatchesRealDirectRequest(t *testing.T) {
	for _, g := range []ggmlProvider{llamaProvider, whisperProvider} {
		for _, model := range g.descriptor().Models {
			t.Run(string(g.id)+"/"+model.ID, func(t *testing.T) {
				stopped := errors.New("stop before any network or artifact transfer")
				calls := 0
				client := &http.Client{Transport: provenanceTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodGet || req.URL.String() != model.Source.URL {
						t.Fatalf("actual acquisition request does not match disclosure: %s %s", req.Method, req.URL)
					}
					return nil, stopped
				})}
				err := acquireModel(t.Context(), t.TempDir(), g.specs[model.ID], client, nil)
				if !errors.Is(err, stopped) || calls != 1 {
					t.Fatalf("did not exercise the real request boundary: calls=%d err=%v", calls, err)
				}
			})
		}
	}
}

func TestNeMoCatalogRetainsReleaseIndexProvenance(t *testing.T) {
	data, err := os.ReadFile("testdata/catalog-v0.1.0.json")
	if err != nil {
		t.Fatal(err)
	}
	models, err := parseCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	expected := (nemoProvider{}).descriptor().Models
	if len(models) != len(expected) {
		t.Fatalf("catalog has %d models, want %d", len(models), len(expected))
	}
	for i, m := range models {
		if m.Source == nil || !reflect.DeepEqual(m.Source, expected[i].Source) {
			t.Fatalf("catalog dropped or changed release-index provenance: %+v", m)
		}
		wire, err := json.Marshal(m.Source)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(wire, &fields); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"url", "repositoryURL", "publisher"} {
			if _, present := fields[key]; present {
				t.Fatalf("unestablished NeMo %s must be omitted", key)
			}
		}
	}
}

func TestSourceDescriptorsAreIndependentSnapshots(t *testing.T) {
	manager := &Manager{}
	before, err := json.Marshal(manager.GetProviders())
	if err != nil {
		t.Fatal(err)
	}
	for _, descriptor := range manager.GetProviders() {
		descriptor.Source.Artifacts[0].URL = "modified"
		descriptor.Models[0].Source.Repository = "modified"
	}
	after, err := json.Marshal(manager.GetProviders())
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("caller mutation or map iteration changed source descriptors")
	}
	if runtimeSource(ProviderID("unknown")) != nil {
		t.Fatal("unknown providers must not invent provenance")
	}
}

func TestModelSourceProjectsAcquisitionPins(t *testing.T) {
	for _, d := range (&Manager{}).GetProviders() {
		for _, m := range d.Models {
			t.Run(string(d.ID)+"/"+m.ID, func(t *testing.T) {
				wire, err := json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				var decoded struct {
					Source *struct {
						AcquisitionMethod string `json:"acquisitionMethod"`
						MetadataOrigin    string `json:"metadataOrigin"`
						Publisher         string `json:"publisher"`
						Repository        string `json:"repository"`
						RepositoryURL     string `json:"repositoryURL"`
						Revision          string `json:"revision"`
						Filename          string `json:"filename"`
						SHA256            string `json:"sha256"`
						URL               string `json:"url"`
					} `json:"source"`
				}
				if err := json.Unmarshal(wire, &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.Source == nil {
					t.Fatal("model has no acquisition source disclosure")
				}
				got := decoded.Source
				var spec modelSpec
				switch d.ID {
				case LlamaCPP:
					spec = llamaProvider.specs[m.ID]
				case WhisperCPP:
					spec = whisperProvider.specs[m.ID]
				case NeMoSpeechCPP:
					spec = modelSpecs[m.ID]
				default:
					t.Fatalf("uncovered provider %q", d.ID)
				}
				wantSize := spec.size
				if d.ID == NeMoSpeechCPP {
					wantSize = nemoModelSize(m.ID)
				}
				if got.Repository != spec.repo || got.Revision != spec.revision || got.Filename != spec.filename || got.SHA256 != spec.sha256 || m.SizeBytes != wantSize {
					t.Fatalf("model source differs from acquisition/verification pins: %+v", got)
				}
				if d.ID == NeMoSpeechCPP {
					if got.AcquisitionMethod != "nemo_model_manager" || got.MetadataOrigin != "nemo_release_model_index" {
						t.Fatalf("NeMo must disclose delegated acquisition and release-index pins: %+v", got)
					}
					if got.URL != "" || got.RepositoryURL != "" || got.Publisher != "" {
						t.Fatalf("NeMo logical repository must not become an invented publisher or download URL: %+v", got)
					}
				} else {
					publisher, _, _ := strings.Cut(spec.repo, "/")
					if got.AcquisitionMethod != "freehand_hugging_face" || got.MetadataOrigin != "freehand_pinned_catalog" || got.Publisher != publisher || got.RepositoryURL != "https://huggingface.co/"+spec.repo || got.URL != "https://huggingface.co/"+spec.repo+"/resolve/"+spec.revision+"/"+spec.filename {
						t.Fatalf("direct download disclosure does not match pinned HF spec: %+v", got)
					}
				}
			})
		}
	}
}

// Exercise the actual renderer boundary without starting a manager or adapter.
func TestProviderRuntimeSourceProjectsPinnedArchives(t *testing.T) {
	descriptors := (&Manager{}).GetProviders()
	for _, d := range descriptors {
		t.Run(string(d.ID), func(t *testing.T) {
			wire, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			var decoded struct {
				Source *struct {
					RepositoryURL string `json:"repositoryURL"`
					ReleaseURL    string `json:"releaseURL"`
					Artifacts     []struct {
						OS           string `json:"os"`
						Architecture string `json:"architecture"`
						Backend      string `json:"backend"`
						Filename     string `json:"filename"`
						URL          string `json:"url"`
						SHA256       string `json:"sha256"`
						SizeBytes    int64  `json:"sizeBytes"`
					} `json:"artifacts"`
				} `json:"source"`
			}
			if err := json.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Source == nil {
				t.Fatal("provider has no runtime source disclosure")
			}
			expected := map[string]RuntimeArtifact{}
			for key, recipe := range platformRecipes {
				if key.provider != d.ID {
					continue
				}
				for _, a := range recipe.Archives {
					u, _ := url.Parse(a.URL)
					id := key.os + "/" + key.arch + "/" + a.Backend + "/" + a.URL
					expected[id] = RuntimeArtifact{OS: key.os, Architecture: key.arch, Backend: a.Backend, Filename: path.Base(u.Path), URL: a.URL, SHA256: a.SHA256, SizeBytes: a.Size}
				}
			}
			if len(decoded.Source.Artifacts) != len(expected) {
				t.Fatalf("artifacts: got %d, want %d", len(decoded.Source.Artifacts), len(expected))
			}
			for _, got := range decoded.Source.Artifacts {
				id := got.OS + "/" + got.Architecture + "/" + got.Backend + "/" + got.URL
				a, ok := expected[id]
				if !ok {
					t.Fatalf("unexpected or duplicate artifact %q", got.URL)
				}
				delete(expected, id)
				repo, release, _ := strings.Cut(a.URL, "/releases/download/")
				tag, _, _ := strings.Cut(release, "/")
				if decoded.Source.RepositoryURL != repo || decoded.Source.ReleaseURL != repo+"/releases/tag/"+tag {
					t.Fatal("release identity does not match downloaded artifact")
				}
				if got.OS != a.OS || got.Architecture != a.Architecture || got.Backend != a.Backend || got.Filename != a.Filename || got.SHA256 != a.SHA256 || got.SizeBytes != a.SizeBytes {
					t.Fatalf("artifact does not match acquisition recipe: %+v", got)
				}
			}
		})
	}
}
