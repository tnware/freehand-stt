package managedruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadDiagnosticsCleanupPreservesPartialWeights(t *testing.T) {
	root := t.TempDir()
	a := newAdapter(root)
	for _, spec := range modelSpecs {
		file := spec.path(root)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		for _, suffix := range []string{".partial", ".partial.curl-errors"} {
			if err := os.WriteFile(file+suffix, []byte("fixture"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := a.clearDownloadDiagnostics(); err != nil {
		t.Fatal(err)
	}
	for _, spec := range modelSpecs {
		if _, err := os.Stat(spec.path(root) + ".partial.curl-errors"); !os.IsNotExist(err) {
			t.Fatalf("diagnostic retained: %v", err)
		}
		if _, err := os.Stat(spec.path(root) + ".partial"); err != nil {
			t.Fatalf("resumable weights removed: %v", err)
		}
	}
	if err := a.clearDownloadDiagnostics(); err != nil {
		t.Fatal("cleanup not idempotent", err)
	}
}
