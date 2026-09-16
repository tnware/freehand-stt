package artifact

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Bundle is private recipe data, not renderer input. Each archive is pinned
// independently and extracted without overwrites into the same fresh directory.
// Archive zero retains the CPU-era name, preserving existing installations.
type Bundle struct {
	Archives   []Asset
	Executable string
	Required   []string
}

func bundleArchive(index int) string {
	if index == 0 {
		return ".release.zip"
	}
	return fmt.Sprintf(".release-%d.zip", index)
}

// Verify admits only the pinned bundle files and the private backend marker.
func Verify(ctx context.Context, base string, b Bundle) error {
	if len(b.Archives) == 0 {
		return ErrIntegrity
	}
	if err := CheckRoot(base); err != nil {
		return err
	}
	marker, err := os.ReadFile(filepath.Join(base, ".backend"))
	if err != nil {
		return err
	}
	if string(marker) != b.Archives[0].Backend {
		return ErrIntegrity
	}
	allowed := map[string]bool{".backend": true}
	for i, a := range b.Archives {
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
	for _, name := range append([]string{b.Executable}, b.Required...) {
		st, err := os.Stat(filepath.Join(base, filepath.FromSlash(name)))
		if err != nil || !st.Mode().IsRegular() {
			return ErrIntegrity
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
			return ErrIntegrity
		}
		return ctx.Err()
	}); err != nil {
		return err
	}
	return ctx.Err()
}

// Called only while the worker owns a stopped runtime. If publication was
// interrupted after moving the old installation aside, restore it on next use.
func Recover(root string) error {
	if err := CheckRoot(root); err != nil {
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

// Install verifies a fresh stage before replacing the current runtime. The
// caller must hold exclusive admission to a stopped runtime for the entire call.
func Install(ctx context.Context, root string, b Bundle, client *http.Client, progress func(float64)) error {
	if len(b.Archives) == 0 {
		return ErrIntegrity
	}
	if err := Recover(root); err != nil {
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
	for _, a := range b.Archives {
		if a.Backend != b.Archives[0].Backend || a.Size <= 0 {
			return ErrIntegrity
		}
		total += a.Size
	}
	for i, a := range b.Archives {
		report := func(p float64) {
			if progress != nil {
				progress((float64(completed) + p*float64(a.Size)) / float64(total))
			}
		}
		archive := filepath.Join(stage, bundleArchive(i))
		if err := downloadRuntimeArchive(ctx, archive, a, client, report); err != nil {
			return err
		}
		if err := extractArchive(ctx, archive, stage); err != nil {
			return err
		}
		completed += a.Size
	}
	if err := os.WriteFile(filepath.Join(stage, ".backend"), []byte(b.Archives[0].Backend), 0600); err != nil {
		return err
	}
	if err := Verify(ctx, stage, b); err != nil {
		return err
	}
	if err := CheckRoot(root); err != nil {
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
