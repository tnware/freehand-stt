package managedruntime

import (
	"os"
	"path/filepath"
)

// NeMo v0.1.0 (4f9676226f667d14608487df744f375db87127f8),
// app/model_store.cpp:716-722 suppresses curl progress for --json AND for
// redirected stderr. There is no byte-progress output to parse safely.
// materialize():1014-1041 writes <pinned filename>.partial, resumes/restarts it,
// verifies it and renames it to the pinned destination. Observe only metadata
// for these exact files while the owned CLI runs; never replace its downloader,
// open model contents here, scan inventories, or read its diagnostic output.
func nemoAcquiredBytes(root string, spec modelSpec) AcquisitionProgress {
	p := AcquisitionProgress{Phase: "downloading", TotalBytes: spec.size}
	for _, path := range []string{spec.path(root) + ".partial", spec.path(root)} {
		// Check ancestors without walking the concurrently changing cache tree.
		// Do not follow links/reparse points, including an untrusted ancestor.
		safe := true
		for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
			info, err := os.Lstat(dir)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || isReparse(dir) {
				safe = false
				break
			}
			if dir == filepath.Dir(dir) {
				break
			}
		}
		if !safe {
			continue
		}
		info, err := os.Lstat(path)
		if err == nil && info.Mode().IsRegular() && !isReparse(path) {
			p.Bytes = info.Size()
			return boundedAcquisition(p)
		}
	}
	return p
}
