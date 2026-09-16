//go:build darwin

package startup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func startupFixture(t *testing.T) darwinStartup {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(home, "Free & Hand.app", "Contents", "MacOS", "freehand")
	if err := os.MkdirAll(filepath.Dir(executable), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("test executable"), 0700); err != nil {
		t.Fatal(err)
	}
	return darwinStartup{home: home, executable: executable}
}
func TestDarwinStartupRefusesUnsafeExecutable(t *testing.T) {
	for _, path := range []string{"relative.app/Contents/MacOS/freehand", "/tmp/freehand", "/tmp/X.app/Contents/MacOS/other", "/tmp/X.app/Contents/MacOS/freehand\n"} {
		s := startupFixture(t)
		s.executable = path
		if err := s.Set(true); err == nil {
			t.Errorf("accepted unsafe executable %q", path)
		}
	}
}
func TestDarwinStartupRefusesUnownedRegistration(t *testing.T) {
	for _, kind := range []string{"foreign", "symlink", "hardlink", "writable", "directory", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			s := startupFixture(t)
			if err := s.Set(true); err != nil {
				t.Fatal(err)
			}
			path := s.path()
			switch kind {
			case "foreign":
				if err := os.WriteFile(path, []byte("not owned by Freehand"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink", "hardlink":
				target := filepath.Join(s.home, "untouched")
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
				var err error
				if kind == "symlink" {
					err = os.Symlink(target, path)
				} else {
					err = os.Link(target, path)
				}
				if err != nil {
					t.Fatal(err)
				}
			case "writable":
				if err := os.Chmod(path, 0666); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				if err := os.WriteFile(path, []byte(strings.Repeat("x", 17000)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if enabled, err := s.Enabled(); err == nil || enabled {
				t.Fatal("unsafe registration reported without error")
			}
			if err := s.Set(true); err == nil {
				t.Fatal("unsafe registration overwritten")
			}
			if err := s.Set(false); err == nil {
				t.Fatal("unsafe registration removed")
			}
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("unsafe registration changed")
			}
		})
	}
}
func TestDarwinStartupRefusesRedirectedOrWritableDirectory(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		s := startupFixture(t)
		library := filepath.Join(s.home, "Library")
		if symlink {
			target := t.TempDir()
			if err := os.Symlink(target, library); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.Mkdir(library, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(library, 0777); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.Set(true); err == nil {
			t.Fatal("unsafe directory accepted")
		}
	}
}
func TestDarwinStartupRoundTrip(t *testing.T) {
	s := startupFixture(t)
	if enabled, err := s.Enabled(); err != nil || enabled {
		t.Fatalf("absent=%v, %v", enabled, err)
	}
	if err := s.Set(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.home, "Library")); !os.IsNotExist(err) {
		t.Fatal("disable created directory")
	}
	if err := s.Set(true); err != nil {
		t.Fatal(err)
	}
	if enabled, err := s.Enabled(); err != nil || !enabled {
		t.Fatalf("enabled=%v, %v", enabled, err)
	}
	data, err := os.ReadFile(filepath.Join(s.home, "Library", "LaunchAgents", startupLabel+".plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Free &amp; Hand.app/Contents/MacOS/freehand</string>") || !strings.Contains(string(data), "<string>--startup</string>") {
		t.Fatal("incorrect escaped executable/argument")
	}
	if strings.Contains(string(data), "/bin/sh") {
		t.Fatal("must not run a shell")
	}
	if err := s.Set(true); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(false); err != nil {
		t.Fatal(err)
	}
	if enabled, err := s.Enabled(); err != nil || enabled {
		t.Fatalf("disabled=%v, %v", enabled, err)
	}
}
