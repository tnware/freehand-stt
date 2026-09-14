package managedruntime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type tarEntry struct {
	name, link, digest string
	size               int64
	mode               os.FileMode
}

func gzipArchive(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()
	var magic [2]byte
	_, err = io.ReadFull(f, magic[:])
	return err == nil && magic == [2]byte{0x1f, 0x8b}
}

func walkTar(ctx context.Context, filename string, visit func(*tar.Header, io.Reader) error) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	z, err := gzip.NewReader(contextReader{ctx, f})
	if err != nil {
		return err
	}
	defer z.Close()
	r := tar.NewReader(io.LimitReader(z, int64(maxExpandedBytes)+8<<20))
	for count := 0; ; count++ {
		h, err := r.Next()
		if err == io.EOF {
			// Read the gzip footer as well, including its checksum.
			_, err = io.Copy(io.Discard, io.LimitReader(z, 1<<20))
			return err
		}
		if err != nil {
			return err
		}
		if count >= 2048 {
			return errIntegrity
		}
		if err := visit(h, r); err != nil {
			return err
		}
	}
}

// Official macOS packages include same-directory dylib aliases. Resolve them
// exclusively inside the verified archive and materialize ordinary file copies.
// No archive link, hard link, special file or privileged permission reaches disk.
func tarManifest(ctx context.Context, filename string) ([]tarEntry, error) {
	entries := []tarEntry{}
	seen := map[string]bool{}
	var total int64
	err := walkTar(ctx, filename, func(h *tar.Header, r io.Reader) error {
		name := strings.TrimSuffix(h.Name, "/")
		if name == "" || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
			return errUnsafePath
		}
		for _, part := range strings.Split(name, "/") {
			if part == "" || part == "." || part == ".." || strings.TrimRight(part, ". ") != part {
				return errUnsafePath
			}
		}
		key := strings.ToLower(name)
		if seen[key] || strings.HasPrefix(key, ".release") || key == ".backend" {
			return errUnsafePath
		}
		seen[key] = true
		if h.Size < 0 || h.Size > int64(maxExpandedBytes)-total {
			return errIntegrity
		}
		total += h.Size
		e := tarEntry{name: name, size: h.Size, mode: 0600}
		if h.Mode&0111 != 0 {
			e.mode = 0700
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if h.Size != 0 {
				return errIntegrity
			}
			return nil
		case tar.TypeReg, tar.TypeRegA:
			hash := sha256.New()
			n, err := io.Copy(hash, contextReader{ctx, r})
			if err != nil {
				return err
			}
			if n != h.Size {
				return errIntegrity
			}
			e.digest = hex.EncodeToString(hash.Sum(nil))
		case tar.TypeSymlink:
			if h.Size != 0 || h.Linkname == "" || h.Linkname == "." || h.Linkname == ".." || strings.ContainsAny(h.Linkname, "/\\:\x00") || !strings.HasSuffix(name, ".dylib") || !strings.HasSuffix(h.Linkname, ".dylib") {
				return errUnsafePath
			}
			e.link = path.Join(path.Dir(name), h.Linkname)
		default:
			return errUnsafePath
		}
		entries = append(entries, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	byName := map[string]tarEntry{}
	for _, e := range entries {
		byName[e.name] = e
	}
	for i, e := range entries {
		if e.link == "" {
			continue
		}
		target := e
		chain := map[string]bool{e.name: true}
		for target.link != "" {
			var ok bool
			target, ok = byName[target.link]
			if !ok || chain[target.name] {
				return nil, errUnsafePath
			}
			chain[target.name] = true
		}
		if target.size > int64(maxExpandedBytes)-total {
			return nil, errIntegrity
		}
		total += target.size
		entries[i] = tarEntry{name: e.name, link: target.name, digest: target.digest, size: target.size, mode: target.mode}
	}
	return entries, nil
}

func extractTarArchive(ctx context.Context, filename, dest string) error {
	entries, err := tarManifest(ctx, filename)
	if err != nil {
		return err
	}
	if err := safeRoot(dest); err != nil {
		return err
	}
	byName := map[string]tarEntry{}
	for _, e := range entries {
		byName[e.name] = e
	}
	write := func(e tarEntry, r io.Reader) error {
		target := filepath.Join(dest, filepath.FromSlash(e.name))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, e.mode)
		if err != nil {
			return err
		}
		n, err := io.Copy(f, contextReader{ctx, r})
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if n != e.size {
			return errIntegrity
		}
		return closeErr
	}
	err = walkTar(ctx, filename, func(h *tar.Header, r io.Reader) error {
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
			return nil
		}
		return write(byName[h.Name], r)
	})
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.link == "" {
			continue
		}
		f, err := os.Open(filepath.Join(dest, filepath.FromSlash(e.link)))
		if err != nil {
			return err
		}
		err = write(e, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	return verifyTarEntries(ctx, dest, entries)
}

func verifyTarEntries(ctx context.Context, base string, entries []tarEntry) error {
	for _, e := range entries {
		filename := filepath.Join(base, filepath.FromSlash(e.name))
		if err := verifyFile(ctx, filename, e.size, e.digest); err != nil {
			return err
		}
		st, err := os.Lstat(filename)
		if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != e.mode {
			return errIntegrity
		}
	}
	return ctx.Err()
}

func archiveFileNames(ctx context.Context, filename string) ([]string, error) {
	names := []string{}
	if gzipArchive(filename) {
		entries, err := tarManifest(ctx, filename)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			names = append(names, e.name)
		}
		return names, nil
	}
	z, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	for _, e := range z.File {
		if !archiveDirectory(e) {
			names = append(names, archiveName(e.Name))
		}
	}
	return names, nil
}
