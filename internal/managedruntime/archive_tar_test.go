package managedruntime

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func tarFixture(t *testing.T, headers []*tar.Header, bodies []string) string {
	t.Helper()
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	w := tar.NewWriter(z)
	for i, h := range headers {
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(bodies[i])); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "release.tar.gz")
	if err := os.WriteFile(filename, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return filename
}

func TestTarArchiveMaterializesOnlyVerifiedDylibAliases(t *testing.T) {
	archive := tarFixture(t, []*tar.Header{
		{Name: "runtime/libalias.dylib", Typeflag: tar.TypeSymlink, Linkname: "libshort.dylib"},
		{Name: "runtime/libshort.dylib", Typeflag: tar.TypeSymlink, Linkname: "libversion.dylib"},
		{Name: "runtime/libversion.dylib", Typeflag: tar.TypeReg, Mode: 0755, Size: 7},
	}, []string{"", "", "library"})
	dest := t.TempDir()
	if err := extractArchive(t.Context(), archive, dest); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"libalias.dylib", "libshort.dylib", "libversion.dylib"} {
		st, err := os.Lstat(filepath.Join(dest, "runtime", name))
		if err != nil || !st.Mode().IsRegular() {
			t.Fatal("archive link reached disk", err)
		}
	}
	entries, err := tarManifest(t.Context(), archive)
	if err != nil {
		t.Fatal(err)
	}
	if err = verifyTarEntries(t.Context(), dest, entries); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dest, "runtime/libalias.dylib"), []byte("changed"), 0700); err != nil {
		t.Fatal(err)
	}
	if verifyTarEntries(t.Context(), dest, entries) == nil {
		t.Fatal("modified alias accepted")
	}
}

func TestTarArchiveRejectsUnsafeEntries(t *testing.T) {
	for _, h := range []*tar.Header{
		{Name: "../escape", Typeflag: tar.TypeReg},
		{Name: "/absolute", Typeflag: tar.TypeReg},
		{Name: "a/../escape", Typeflag: tar.TypeReg},
		{Name: ".backend", Typeflag: tar.TypeReg},
		{Name: ".release-1.zip", Typeflag: tar.TypeReg},
		{Name: "a/link.dylib", Typeflag: tar.TypeSymlink, Linkname: "../../outside.dylib"},
		{Name: "a/link.dylib", Typeflag: tar.TypeSymlink, Linkname: "/outside.dylib"},
		{Name: "a/link.dylib", Typeflag: tar.TypeSymlink, Linkname: "link.dylib"},
		{Name: "a/link.dylib", Typeflag: tar.TypeSymlink, Linkname: "missing.dylib"},
		{Name: "hardlink", Typeflag: tar.TypeLink, Linkname: "somewhere"},
		{Name: "pipe", Typeflag: tar.TypeFifo},
	} {
		t.Run(h.Name+"-"+h.Linkname, func(t *testing.T) {
			archive := tarFixture(t, []*tar.Header{h}, []string{""})
			dest := t.TempDir()
			if extractArchive(t.Context(), archive, dest) == nil {
				t.Fatal("unsafe archive accepted")
			}
			entries, _ := os.ReadDir(dest)
			if len(entries) != 0 {
				t.Fatal("invalid archive wrote files")
			}
		})
	}
	archive := tarFixture(t, []*tar.Header{{Name: "a", Typeflag: tar.TypeReg}, {Name: "A", Typeflag: tar.TypeReg}}, []string{"", ""})
	if extractArchive(t.Context(), archive, t.TempDir()) == nil {
		t.Fatal("case collision accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if extractArchive(ctx, archive, t.TempDir()) == nil {
		t.Fatal("cancellation ignored")
	}
}
