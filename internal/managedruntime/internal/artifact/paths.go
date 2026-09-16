package artifact

import (
	"archive/zip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsafePath = errors.New("Managed speech storage is unsafe. Choose an ordinary local directory without links.")

const maxExpandedBytes uint64 = 1 << 30

// The pinned CUDA release uses Windows separators; CPU uses ZIP separators.
// Normalize separators only, never clean away traversal or empty components.
// Validation, duplicate detection, extraction and verification share this name.
func archiveName(name string) string { return strings.ReplaceAll(name, "\\", "/") }
func archiveDirectory(f *zip.File) bool {
	return strings.HasSuffix(archiveName(f.Name), "/") || f.FileInfo().IsDir()
}
func safeArchive(z *zip.ReadCloser) error {
	if len(z.File) > 2048 {
		return ErrIntegrity
	}
	seen := map[string]bool{}
	var total uint64
	for _, f := range z.File {
		n := strings.TrimSuffix(archiveName(f.Name), "/")
		if n == "" || strings.ContainsAny(n, ":\x00") || strings.HasPrefix(n, "/") {
			return ErrUnsafePath
		}
		for part := range strings.SplitSeq(n, "/") {
			if part == "" || part == "." || part == ".." || strings.TrimRight(part, ". ") != part {
				return ErrUnsafePath
			}
			upper := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
			if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" || len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '0' && upper[3] <= '9' {
				return ErrUnsafePath
			}
		}
		key := strings.ToLower(n)
		if seen[key] || key == ".release.zip" || key == ".backend" {
			return ErrUnsafePath
		}
		seen[key] = true
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 || !mode.IsRegular() && !mode.IsDir() {
			return ErrUnsafePath
		}
		if f.UncompressedSize64 > maxExpandedBytes || total > maxExpandedBytes-f.UncompressedSize64 {
			return ErrIntegrity
		}
		total += f.UncompressedSize64
		if f.UncompressedSize64 > 1<<20 && (f.CompressedSize64 == 0 || f.UncompressedSize64/f.CompressedSize64 > 250) {
			return ErrIntegrity
		}
	}
	return nil
}

// Check every ancestor and existing descendant, including Windows junctions.
// The runtime owns this directory exclusively; no renderer-supplied subpaths exist.
func CheckRoot(root string) error {
	if !filepath.IsAbs(root) || filepath.Clean(root) == filepath.VolumeName(root)+string(os.PathSeparator) {
		return ErrUnsafePath
	}
	for p := filepath.Clean(root); ; p = filepath.Dir(p) {
		st, e := os.Lstat(p)
		if e == nil {
			if st.Mode()&os.ModeSymlink != 0 || IsReparse(p) {
				return ErrUnsafePath
			}
		} else if !os.IsNotExist(e) {
			return ErrUnsafePath
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return safeTree(root)
}
func safeTree(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if os.IsNotExist(e) && p == root {
			return nil
		}
		if e != nil {
			return ErrUnsafePath
		}
		if d.Type()&os.ModeSymlink != 0 || IsReparse(p) {
			return ErrUnsafePath
		}
		return nil
	})
}
