package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReleaseCheckAcceptsCRLFWithoutHidingStaleMetadata(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"build/config.yml", "build/darwin/Info.plist", "build/darwin/Info.dev.plist",
		"build/windows/info.json", "build/windows/wails.exe.manifest",
		"build/windows/nsis/project.nsi", "build/windows/nsis/wails_tools.nsh",
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.ReplaceAll(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\n"), []byte("\r\n"))
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	check := func() ([]byte, error) {
		return exec.Command("go", "run", ".", "-root", root, "check").CombinedOutput()
	}
	if output, err := check(); err != nil {
		t.Fatalf("CRLF checkout must pass release check: %v\n%s", err, output)
	}
	path := filepath.Join(root, "build", "darwin", "Info.plist")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("io.github.tnware.freehand"), []byte("io.github.tnware.stale"), 1)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := check(); err == nil || !bytes.Contains(output, []byte("Info.plist")) {
		t.Fatalf("stale identity must still fail: %v\n%s", err, output)
	}
}
