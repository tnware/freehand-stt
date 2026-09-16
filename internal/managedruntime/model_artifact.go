package managedruntime

import "path/filepath"

// modelSpec pins a provider-qualified model and its owned cache location.
type modelSpec struct {
	repo, revision, filename, sha256 string
	size                             int64
}

func (m modelSpec) directory(root string) string {
	return filepath.Join(root, "models", filepath.FromSlash(m.repo), m.revision)
}
func (m modelSpec) path(root string) string { return filepath.Join(m.directory(root), m.filename) }
