package managedruntime

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// A bundle is private recipe data, not renderer input. Each archive is pinned
// independently and extracted without overwrites into the same fresh directory.
// Archive zero retains the CPU-era name, preserving existing installations.
type runtimeBundle struct {
	archives   []asset
	executable string
	required   []string
}

func bundleArchive(index int) string {
	if index == 0 {
		return ".release.zip"
	}
	return fmt.Sprintf(".release-%d.zip", index)
}
func verifyRuntimeBundle(ctx context.Context, base string, b runtimeBundle) error {
	if len(b.archives) == 0 {
		return errIntegrity
	}
	if err := safeRoot(base); err != nil {
		return err
	}
	marker, err := os.ReadFile(filepath.Join(base, ".backend"))
	if err != nil {
		return err
	}
	if string(marker) != b.archives[0].backend {
		return errIntegrity
	}
	allowed := map[string]bool{".backend": true}
	for i, a := range b.archives {
		name := bundleArchive(i)
		allowed[strings.ToLower(name)] = true
		if err := verifyRuntimeArchive(ctx, base, filepath.Join(base, name), a); err != nil {
			return err
		}
		names, err := archiveFileNames(ctx, filepath.Join(base, name))
		if err != nil {
			return err
		}
		for _, entry := range names {
			allowed[strings.ToLower(entry)] = true
		}
	}
	for _, name := range append([]string{b.executable}, b.required...) {
		st, err := os.Stat(filepath.Join(base, filepath.FromSlash(name)))
		if err != nil || !st.Mode().IsRegular() {
			return errIntegrity
		}
	}
	if err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		if !allowed[strings.ToLower(filepath.ToSlash(rel))] {
			return errIntegrity
		}
		return ctx.Err()
	}); err != nil {
		return err
	}
	return ctx.Err()
}

// Called only while the worker owns a stopped runtime. If publication was
// interrupted after moving the old installation aside, restore it on next use.
func recoverRuntime(root string) error {
	if err := safeRoot(root); err != nil {
		return err
	}
	current, previous := filepath.Join(root, "runtime"), filepath.Join(root, ".runtime-previous")
	if _, err := os.Stat(current); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(previous); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return os.Rename(previous, current)
}
func installRuntimeBundle(ctx context.Context, root string, b runtimeBundle, client *http.Client, progress func(float64)) error {
	if len(b.archives) == 0 {
		return errIntegrity
	}
	if err := recoverRuntime(root); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(root, ".install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	var total, completed int64
	for _, a := range b.archives {
		if a.backend != b.archives[0].backend || a.size <= 0 {
			return errIntegrity
		}
		total += a.size
	}
	for i, a := range b.archives {
		report := func(p float64) {
			if progress != nil {
				progress((float64(completed) + p*float64(a.size)) / float64(total))
			}
		}
		archive := filepath.Join(stage, bundleArchive(i))
		if err := downloadBundleArchive(ctx, archive, a, client, report); err != nil {
			return err
		}
		if err := extractArchive(ctx, archive, stage); err != nil {
			return err
		}
		completed += a.size
	}
	if err := os.WriteFile(filepath.Join(stage, ".backend"), []byte(b.archives[0].backend), 0600); err != nil {
		return err
	}
	if err := verifyRuntimeBundle(ctx, stage, b); err != nil {
		return err
	}
	if err := safeRoot(root); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, previous := filepath.Join(root, "runtime"), filepath.Join(root, ".runtime-previous")
	// A previous successful publication may have left a backup during shutdown.
	// Never touch it until the replacement has fully verified.
	if err := os.RemoveAll(previous); err != nil {
		return err
	}
	hadOld := false
	if _, err := os.Stat(current); err == nil {
		if err := os.Rename(current, previous); err != nil {
			return err
		}
		hadOld = true
	} else if !os.IsNotExist(err) {
		return err
	}
	rollback := func(cause error) error {
		if hadOld {
			if err := os.Rename(previous, current); err != nil {
				return fmt.Errorf("runtime restoration required: %w", err)
			}
		}
		return cause
	}
	if err := ctx.Err(); err != nil {
		return rollback(err)
	}
	if err := os.Rename(stage, current); err != nil {
		return rollback(err)
	}
	// Publication is the commit point. Cancellation after it cannot undo success.
	// Backup cleanup failure is recoverable and must not report a failed switch.
	_ = os.RemoveAll(previous)
	return nil
}
func downloadBundleArchive(ctx context.Context, path string, a asset, client *http.Client, progress func(float64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || res.ContentLength > 0 && res.ContentLength != a.size {
		return errIntegrity
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	n, err := io.Copy(&progressWriter{w: f, total: a.size, changed: progress}, io.LimitReader(contextReader{ctx, res.Body}, a.size+1))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n != a.size {
		return errIntegrity
	}
	return verifyFile(ctx, path, a.size, a.sha256)
}
