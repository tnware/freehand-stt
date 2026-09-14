package managedruntime

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

type ModelAcquisitionMethod string

const (
	FreehandHuggingFace ModelAcquisitionMethod = "freehand_hugging_face"
	NeMoModelManager    ModelAcquisitionMethod = "nemo_model_manager"
)

// ModelSource discloses acquisition ownership and the pins Freehand verifies.
// For NeMo, Repository is the model manager's logical repo identity and the
// pins originate in the release model index, not CLI list output. That index
// does not establish a Hugging Face publisher or direct download URL here;
// those optional fields are deliberately absent for delegated acquisition.
// Publisher for direct downloads is the hosting repository namespace, not an
// assertion about the model's original training author.
type ModelSource struct {
	AcquisitionMethod ModelAcquisitionMethod `json:"acquisitionMethod"`
	MetadataOrigin    string                 `json:"metadataOrigin"`
	Publisher         string                 `json:"publisher,omitempty"`
	Repository        string                 `json:"repository"`
	RepositoryURL     string                 `json:"repositoryURL,omitempty"`
	Revision          string                 `json:"revision"`
	Filename          string                 `json:"filename"`
	SHA256            string                 `json:"sha256"`
	URL               string                 `json:"url,omitempty"`
}

func (s modelSpec) source(method ModelAcquisitionMethod) *ModelSource {
	source := &ModelSource{AcquisitionMethod: method, Repository: s.repo, Revision: s.revision, Filename: s.filename, SHA256: s.sha256}
	switch method {
	case FreehandHuggingFace:
		source.MetadataOrigin = "freehand_pinned_catalog"
		source.Publisher, _, _ = strings.Cut(s.repo, "/")
		source.RepositoryURL = "https://huggingface.co/" + s.repo
		source.URL = source.RepositoryURL + "/resolve/" + s.revision + "/" + s.filename
	case NeMoModelManager:
		source.MetadataOrigin = "nemo_release_model_index"
	default:
		return nil
	}
	return source
}

// RuntimeSource describes this build's pinned release, not the installed state
// or host availability. Artifact backend/host labels identify bundle members;
// e.g. llama.cpp CUDA requires both its server and CUDA dependency archives.
// Installation admission remains authoritative for available binary choices.
type RuntimeSource struct {
	RepositoryURL string            `json:"repositoryURL"`
	ReleaseURL    string            `json:"releaseURL"`
	Artifacts     []RuntimeArtifact `json:"artifacts"`
}

type RuntimeArtifact struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Backend      string `json:"backend"`
	Filename     string `json:"filename"`
	URL          string `json:"url"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"sizeBytes"`
}

// Project only built-in acquisition recipes. No renderer URL, filesystem state,
// subprocess, network lookup, or independent frontend pin registry is involved.
func runtimeSource(provider ProviderID) *RuntimeSource {
	source := &RuntimeSource{Artifacts: []RuntimeArtifact{}}
	appendAsset := func(os, arch string, a asset) bool {
		u, err := url.Parse(a.url)
		if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return false
		}
		repo, release, ok := strings.Cut(u.Path, "/releases/download/")
		tag, filename, hasFile := strings.Cut(release, "/")
		if !ok || !hasFile || tag == "" || filename == "" || strings.Count(repo, "/") != 2 {
			return false
		}
		repositoryURL := "https://github.com" + repo
		releaseURL := repositoryURL + "/releases/tag/" + tag
		if source.RepositoryURL != "" && (source.RepositoryURL != repositoryURL || source.ReleaseURL != releaseURL) {
			return false
		}
		source.RepositoryURL, source.ReleaseURL = repositoryURL, releaseURL
		source.Artifacts = append(source.Artifacts, RuntimeArtifact{OS: os, Architecture: arch, Backend: a.backend, Filename: path.Base(u.Path), URL: a.url, SHA256: a.sha256, SizeBytes: a.size})
		return true
	}
	for key, recipe := range platformRecipes {
		if key.provider != provider {
			continue
		}
		for _, a := range recipe.archives {
			if !appendAsset(key.os, key.arch, a) {
				return nil
			}
		}
	}
	if len(source.Artifacts) == 0 {
		return nil
	}
	sort.Slice(source.Artifacts, func(i, j int) bool {
		a, b := source.Artifacts[i], source.Artifacts[j]
		return a.OS+"/"+a.Architecture+"/"+a.Backend+"/"+a.Filename < b.OS+"/"+b.Architecture+"/"+b.Backend+"/"+b.Filename
	})
	return source
}
