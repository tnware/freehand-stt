package storage

import (
	"path/filepath"
	"testing"
)

func TestDirectoryUsesTheSettingsDatabaseRoot(t *testing.T) {
	root := t.TempDir()
	s := newStore(filepath.Join(root, "freehand.db"), nil)
	defer s.Close()
	if Directory(s) != root {
		t.Fatal("runtime storage must share settings data root")
	}
}
