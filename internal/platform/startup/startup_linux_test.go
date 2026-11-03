//go:build linux

package startup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func startupFixture(t *testing.T) linuxStartup {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(home, "Free & Hand", "freehand")
	if err := os.MkdirAll(filepath.Dir(executable), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("test executable"), 0700); err != nil {
		t.Fatal(err)
	}
	return linuxStartup{config: filepath.Join(home, ".config"), executable: executable}
}
func TestLinuxStartupRefusesUnsafeExecutable(t *testing.T) {
	for _, path := range []string{"relative/freehand", "/tmp/../freehand", "/tmp/free=hand", "/tmp/freehand\n"} {
		s := startupFixture(t)
		s.executable = path
		if err := s.Set(true); err == nil {
			t.Errorf("accepted unsafe executable %q", path)
		}
	}
}
func TestLinuxStartupRefusesUnownedRegistration(t *testing.T) {
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
				target := filepath.Join(filepath.Dir(s.config), "untouched")
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
func TestLinuxStartupRefusesRedirectedOrWritableDirectory(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		s := startupFixture(t)
		library := filepath.Join(filepath.Dir(s.config), ".config")
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
func TestLinuxStartupRoundTrip(t *testing.T) {
	s := startupFixture(t)
	if enabled, err := s.Enabled(); err != nil || enabled {
		t.Fatalf("absent=%v, %v", enabled, err)
	}
	if err := s.Set(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.config), ".config")); !os.IsNotExist(err) {
		t.Fatal("disable created directory")
	}
	if err := s.Set(true); err != nil {
		t.Fatal(err)
	}
	if enabled, err := s.Enabled(); err != nil || !enabled {
		t.Fatalf("enabled=%v, %v", enabled, err)
	}
	data, err := os.ReadFile(s.path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Free & Hand/freehand\"") || !strings.Contains(string(data), " --startup\n") {
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

func TestLinuxExecArgument(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"/opt/free hand/freehand", `"/opt/free hand/freehand"`},
		{"/opt/a\\b/freehand", `"/opt/a\\\\b/freehand"`},
		{"/opt/a\"b/freehand", `"/opt/a\\"b/freehand"`},
		{"/opt/$HOME/`x`/freehand", `"/opt/\\$HOME/\\` + "`" + `x\\` + "`" + `/freehand"`},
		{"/opt/100%/freehand", `"/opt/100%%/freehand"`},
		{"/opt/日本語/freehand", `"/opt/日本語/freehand"`},
	} {
		got, err := execArgument(tc.path)
		if err != nil || got != tc.want {
			t.Errorf("%q: got %q, %v; want %q", tc.path, got, err, tc.want)
		}
	}
}

func TestLinuxStartupRecoveryAndOtherEntries(t *testing.T) {
	s := startupFixture(t)
	if err := s.Set(true); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(filepath.Dir(s.path()), "unrelated.desktop")
	if err := os.WriteFile(other, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(s.path())
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("registration not private", err)
	}
	if err := os.Remove(s.executable); err != nil {
		t.Fatal(err)
	}
	if enabled, err := s.Enabled(); enabled || err != nil {
		t.Fatal("missing executable enabled", enabled, err)
	}
	// Recovery can remove our exact entry even if its executable disappeared.
	if err := s.Set(false); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(other); err != nil || string(got) != "untouched" {
		t.Fatal("unrelated entry changed", err)
	}
	if err := s.Set(false); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(s.path()))
	if err != nil || len(entries) != 1 {
		t.Fatal("temporary files leaked", err, entries)
	}
}

func TestLinuxStartupRejectsDifferentExecutable(t *testing.T) {
	s := startupFixture(t)
	if err := s.Set(true); err != nil {
		t.Fatal(err)
	}
	s.executable = filepath.Join(filepath.Dir(s.executable), "another")
	if err := os.WriteFile(s.executable, []byte("other"), 0700); err != nil {
		t.Fatal(err)
	}
	if enabled, err := s.Enabled(); enabled || err == nil {
		t.Fatal("other executable accepted")
	}
	if err := s.Set(false); err == nil {
		t.Fatal("other executable entry deleted")
	}
	if _, err := os.Stat(s.path()); err != nil {
		t.Fatal(err)
	}
}

func TestLinuxStartupXDGPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, tc := range []struct{ xdg, want string }{
		{"", filepath.Join(home, ".config")},
		{"relative", filepath.Join(home, ".config")},
		{filepath.Join(home, "custom"), filepath.Join(home, "custom")},
	} {
		t.Setenv("XDG_CONFIG_HOME", tc.xdg)
		s, err := startupForCurrentUser()
		if err != nil || s.config != tc.want {
			t.Fatal(s, err, tc.want)
		}
		if err := (Startup{}).Set(false); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(tc.want); !os.IsNotExist(err) {
			t.Fatal("false created directory")
		}
	}
}

func TestLinuxStartupMissingExecutableDoesNotCreateDirectory(t *testing.T) {
	s := startupFixture(t)
	if err := os.Remove(s.executable); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(true); err == nil {
		t.Fatal("missing executable accepted")
	}
	if _, err := os.Stat(s.config); !os.IsNotExist(err) {
		t.Fatal("directory created")
	}
}

// Validate syntax with the desktop environment's parser when available.
// This never installs or launches a desktop entry.
func TestLinuxDesktopEntryValidation(t *testing.T) {
	validator, err := exec.LookPath("desktop-file-validate")
	if err != nil {
		t.Skip("desktop-file-validate unavailable")
	}
	for _, path := range []string{"/opt/free hand/freehand", "/opt/a\\b/freehand", "/opt/a\"b/freehand", "/opt/$HOME/`x`/freehand", "/opt/100%/freehand", "/opt/日本語/freehand"} {
		s := linuxStartup{executable: path}
		data, err := s.content()
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(t.TempDir(), "freehand.desktop")
		if err := os.WriteFile(file, data, 0600); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command(validator, file).CombinedOutput(); err != nil {
			t.Errorf("%q: %s: %v", path, output, err)
		}
	}
}

func TestLinuxStartupRejectsOversizedEscapedEntry(t *testing.T) {
	s := linuxStartup{executable: "/" + strings.Repeat("\\", 4095)}
	if _, err := s.content(); err == nil {
		t.Fatal("oversized escaped entry accepted")
	}
}

func TestLinuxStartupRejectsAutostartDirectorySymlink(t *testing.T) {
	s := startupFixture(t)
	if err := os.Mkdir(s.config, 0700); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Dir(s.path())); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(true); err == nil {
		t.Fatal("redirected autostart accepted")
	}
	if err := s.Set(false); err == nil {
		t.Fatal("redirected autostart accepted")
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 0 {
		t.Fatal("redirect target changed", err)
	}
}
