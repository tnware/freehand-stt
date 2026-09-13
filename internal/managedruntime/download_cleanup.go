package managedruntime

import (
	"errors"
	"os"
)

// NeMo v0.1.0 app/model_store.cpp:683-749 redirects curl failures to a
// temporary diagnostic beside the .partial model download. Job cancellation
// bypasses its own removal, so clean exact qualified paths after pulls and
// during inspection on the next launch. Never read or publish their contents.
func (a *nemoAdapter) clearDownloadDiagnostics() error {
	if err := safeRoot(a.root); err != nil {
		return err
	}
	for _, spec := range modelSpecs {
		path := spec.path(a.root) + ".partial.curl-errors"
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("Could not clean temporary model download diagnostics.")
		}
		if err := os.Remove(path); err != nil {
			return errors.New("Could not clean temporary model download diagnostics.")
		}
	}
	return nil
}
