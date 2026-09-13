package managedruntime

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func zipFixture(t *testing.T, names map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for n, v := range names {
		w, e := z.Create(n)
		if e != nil {
			t.Fatal(e)
		}
		w.Write([]byte(v))
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestArchiveRejectsUnsafePaths(t *testing.T) {
	for _, name := range []string{"../escape", "bin/../../escape", "/absolute", "C:/outside", "bin/evil:stream", "bin/CON", "bin/trailing. ", "bin\\..\\evil", "bin/../escape"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			z := filepath.Join(root, "test.zip")
			os.WriteFile(z, zipFixture(t, map[string]string{name: "bad"}), 0600)
			dest := filepath.Join(root, "stage")
			os.Mkdir(dest, 0700)
			if extractArchive(context.Background(), z, dest) == nil {
				t.Fatalf("accepted %q", name)
			}
		})
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	h := &zip.FileHeader{Name: "bin/link"}
	h.SetMode(os.ModeSymlink | 0777)
	w, _ := z.CreateHeader(h)
	w.Write([]byte("../../escape"))
	z.Close()
	root := t.TempDir()
	path := filepath.Join(root, "symlink.zip")
	os.WriteFile(path, b.Bytes(), 0600)
	if extractArchive(context.Background(), path, root) == nil {
		t.Fatal("accepted symlink")
	}
}
func TestArchiveRejectsBomb(t *testing.T) {
	b := zipFixture(t, map[string]string{"bin/bomb": string(bytes.Repeat([]byte{0}, 2<<20))})
	root := t.TempDir()
	p := filepath.Join(root, "bomb.zip")
	os.WriteFile(p, b, 0600)
	if extractArchive(context.Background(), p, root) == nil {
		t.Fatal("accepted excessive compression ratio")
	}
}
func TestVerifiedAtomicInstall(t *testing.T) {
	b := zipFixture(t, map[string]string{"bin/nemo-speech.exe": "binary", "share/nemo-speech/model-index.json": "{}"})
	h := sha256.Sum256(b)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(b) }))
	defer srv.Close()
	a := asset{backend: "cpu", url: srv.URL, size: int64(len(b)), sha256: fmt.Sprintf("%x", h)}
	root := t.TempDir()
	if err := installAsset(context.Background(), root, a, srv.Client(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "runtime", "bin", "nemo-speech.exe")); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntime(context.Background(), root, a); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "runtime", "bin", "nemo-speech.exe"), []byte("tampered"), 0600)
	if verifyRuntime(context.Background(), root, a) == nil {
		t.Fatal("accepted changed executable")
	}
	a.sha256 = "bad"
	other := t.TempDir()
	if installAsset(context.Background(), other, a, srv.Client(), nil) == nil {
		t.Fatal("accepted bad hash")
	}
	if _, err := os.Stat(filepath.Join(other, "runtime")); !os.IsNotExist(err) {
		t.Fatal("published failed install")
	}
}
