//go:build darwin

package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

// macOS exposes /var as an OS symlink. Use its physical temporary directory
// for fixtures; production managed storage still rejects linked ancestors.
func TestMain(m *testing.M) {
	dir, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil || os.Setenv("TMPDIR", dir) != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}
