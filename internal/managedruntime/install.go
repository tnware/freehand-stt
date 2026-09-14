package managedruntime

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

type asset struct {
	backend, url, sha256 string
	size                 int64
}

var assets = map[string]asset{
	"cpu":    {"cpu", "https://github.com/NVIDIA/NeMo-Speech.cpp/releases/download/v0.1.0/nemo-speech-0.1.0-windows-x86_64-cpu.zip", "5e4ea81046012edcd77fd8848de8eefb5a4ba38cc26f52eb544ab184695a75d6", 4730421},
	"cuda":   {"cuda", "https://github.com/NVIDIA/NeMo-Speech.cpp/releases/download/v0.1.0/nemo-speech-0.1.0-windows-x86_64-cuda.zip", "ba024204e76ca2fa4eefa8787506c3c49e418147f627f60cf9206a582b60089c", 106044768},
	"vulkan": {"vulkan", "https://github.com/NVIDIA/NeMo-Speech.cpp/releases/download/v0.1.0/nemo-speech-0.1.0-windows-x86_64-vulkan.zip", "b5e7b04a637da4eb25a60253e2db65774998e8dfb48c08b4db763009b82ac7ac", 21967184},
}
var errIntegrity = errors.New("Runtime integrity verification failed. Remove and reinstall the runtime.")

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
func verifyFile(ctx context.Context, path string, size int64, digest string) error {
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
		return errIntegrity
	}
	h := sha256.New()
	if _, e = io.Copy(h, contextReader{ctx, f}); e != nil {
		return e
	}
	if hex.EncodeToString(h.Sum(nil)) != digest {
		return errIntegrity
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
func installAsset(ctx context.Context, root string, a asset, client *http.Client, progress func(float64)) error {
	return installBinaryAsset(ctx, root, a, "bin/nemo-speech.exe", client, progress)
}

// expectedExecutable is a private, built-in recipe path, never renderer input.
func installBinaryAsset(ctx context.Context, root string, a asset, expectedExecutable string, client *http.Client, progress func(float64)) error {
	if e := safeRoot(root); e != nil {
		return e
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		return e
	}
	stage, e := os.MkdirTemp(root, ".install-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if e != nil {
		return e
	}
	res, e := client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || res.ContentLength > 0 && res.ContentLength != a.size {
		return errIntegrity
	}
	archive := filepath.Join(stage, ".release.zip")
	f, e := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	n, e := io.Copy(&progressWriter{w: f, total: a.size, changed: progress}, io.LimitReader(res.Body, a.size+1))
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	if n != a.size {
		return errIntegrity
	}
	if e = verifyFile(ctx, archive, a.size, a.sha256); e != nil {
		return e
	}
	if e = extractArchive(ctx, archive, stage); e != nil {
		return e
	}
	if st, statErr := os.Stat(filepath.Join(stage, filepath.FromSlash(expectedExecutable))); statErr != nil || !st.Mode().IsRegular() {
		return errIntegrity
	}
	if e = os.WriteFile(filepath.Join(stage, ".backend"), []byte(a.backend), 0600); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	return os.Rename(stage, filepath.Join(root, "runtime"))
}
func extractArchive(ctx context.Context, archive, dest string) error {
	z, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	defer z.Close()
	if e = safeArchive(z); e != nil {
		return e
	}
	if e = safeRoot(dest); e != nil {
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
		_, e = io.Copy(f, contextReader{ctx, r})
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
func verifyRuntime(ctx context.Context, root string, a asset) error {
	if e := safeRoot(root); e != nil {
		return e
	}
	base := filepath.Join(root, "runtime")
	archive := filepath.Join(base, ".release.zip")
	return verifyRuntimeArchive(ctx, base, archive, a)
}

func verifyRuntimeArchive(ctx context.Context, base, archive string, a asset) error {
	if e := verifyFile(ctx, archive, a.size, a.sha256); e != nil {
		return e
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
		_, e = io.Copy(h, contextReader{ctx, r})
		r.Close()
		if e != nil {
			return e
		}
		if e = verifyFile(ctx, filepath.Join(base, filepath.FromSlash(archiveName(entry.Name))), int64(entry.UncompressedSize64), hex.EncodeToString(h.Sum(nil))); e != nil {
			return e
		}
	}
	return nil
}
