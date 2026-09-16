package artifact

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Asset is a pinned public release archive selected by the owning provider.
// Its URLs and integrity values are built-in recipe data, never renderer input.
type Asset struct {
	Backend, URL, SHA256 string
	Size                 int64
}

var ErrIntegrity = errors.New("Runtime integrity verification failed. Remove and reinstall the runtime.")

// ContextReader stops synchronous file and response reads after cancellation.
type ContextReader struct {
	Context context.Context
	Reader  io.Reader
}

func (r ContextReader) Read(p []byte) (int, error) {
	if err := r.Context.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}

// VerifyFile checks exact size and SHA256 without executing or loading content.
func VerifyFile(ctx context.Context, path string, size int64, digest string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() || st.Size() != size {
		return ErrIntegrity
	}
	h := sha256.New()
	if _, e = io.Copy(h, ContextReader{ctx, f}); e != nil {
		return e
	}
	if hex.EncodeToString(h.Sum(nil)) != digest {
		return ErrIntegrity
	}
	return nil
}

type progressWriter struct {
	w        io.Writer
	total, n int64
	changed  func(float64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, e := p.w.Write(b)
	p.n += int64(n)
	if p.changed != nil {
		p.changed(float64(p.n) / float64(p.total))
	}
	return n, e
}

// The caller owns a fresh staging directory and removes it on failure. Both
// archives in a bundle share the same pinned download checks.
func downloadRuntimeArchive(ctx context.Context, path string, a Asset, client *http.Client, progress func(float64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || res.ContentLength > 0 && res.ContentLength != a.Size {
		return ErrIntegrity
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	n, err := io.Copy(&progressWriter{w: f, total: a.Size, changed: progress}, io.LimitReader(ContextReader{ctx, res.Body}, a.Size+1))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n != a.Size {
		return ErrIntegrity
	}
	return VerifyFile(ctx, path, a.Size, a.SHA256)
}

func extractArchive(ctx context.Context, archive, dest string) error {
	if gzipArchive(archive) {
		return extractTarArchive(ctx, archive, dest)
	}
	z, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	defer z.Close()
	if e = safeArchive(z); e != nil {
		return e
	}
	if e = CheckRoot(dest); e != nil {
		return e
	}
	for _, entry := range z.File {
		target := filepath.Join(dest, filepath.FromSlash(archiveName(entry.Name)))
		if archiveDirectory(entry) {
			if e = os.MkdirAll(target, 0700); e != nil {
				return e
			}
			continue
		}
		if e = os.MkdirAll(filepath.Dir(target), 0700); e != nil {
			return e
		}
		r, e := entry.Open()
		if e != nil {
			return e
		}
		f, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			r.Close()
			return e
		}
		_, e = io.Copy(f, ContextReader{ctx, r})
		r.Close()
		ce := f.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
	}
	return nil
}
func verifyRuntimeArchive(ctx context.Context, base, archive string, a Asset) error {
	if e := VerifyFile(ctx, archive, a.Size, a.SHA256); e != nil {
		return e
	}
	if gzipArchive(archive) {
		entries, err := tarManifest(ctx, archive)
		if err != nil {
			return err
		}
		return verifyTarEntries(ctx, base, entries)
	}
	z, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	defer z.Close()
	if e = safeArchive(z); e != nil {
		return e
	}
	for _, entry := range z.File {
		if archiveDirectory(entry) {
			continue
		}
		r, e := entry.Open()
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, ContextReader{ctx, r})
		r.Close()
		if e != nil {
			return e
		}
		if e = VerifyFile(ctx, filepath.Join(base, filepath.FromSlash(archiveName(entry.Name))), int64(entry.UncompressedSize64), hex.EncodeToString(h.Sum(nil))); e != nil {
			return e
		}
	}
	return nil
}
