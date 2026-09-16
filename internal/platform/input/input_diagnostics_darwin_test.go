//go:build darwin && cgo

package input

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDarwinNativeInsertionDiagnostics(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "input-diagnostics")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "clang", "-fblocks", "-mmacosx-version-min=13.0", "-framework", "AppKit", "-framework", "ApplicationServices", "-framework", "Carbon", "-I", wd, filepath.Join(wd, "testdata/input_diagnostics.m"), "-o", exe)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, exe).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}
