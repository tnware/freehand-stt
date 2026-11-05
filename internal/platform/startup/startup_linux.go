//go:build linux

package startup

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"

	"errors"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// Startup owns only Freehand's per-user XDG autostart entry. Registration does
// not start a process and is enabled only by the user's settings toggle.
type Startup struct{}

const startupLabel = "io.github.tnware.freehand.startup"

type linuxStartup struct{ config, executable string }

func startupForCurrentUser() (linuxStartup, error) {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" || !filepath.IsAbs(config) {
		home, err := os.UserHomeDir()
		if err != nil {
			return linuxStartup{}, errors.New("startup home is unavailable")
		}
		config = filepath.Join(home, ".config")
	}
	exe, err := os.Executable()
	if err != nil {
		return linuxStartup{}, errors.New("startup executable is unavailable")
	}
	return linuxStartup{config: config, executable: exe}, nil
}
func (Startup) Set(on bool) error {
	s, err := startupForCurrentUser()
	if err != nil {
		return err
	}
	return s.Set(on)
}
func (Startup) Enabled() (bool, error) {
	s, err := startupForCurrentUser()
	if err != nil {
		return false, err
	}
	return s.Enabled()
}
func (s linuxStartup) path() string {
	return filepath.Join(s.config, "autostart", startupLabel+".desktop")
}

// execArgument applies Exec quoting followed by desktop string escaping,
// without a shell. The spec forbids "=" in an executable path.
func execArgument(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 ||
		!utf8.ValidString(path) || strings.Contains(path, "=") ||
		strings.ContainsFunc(path, func(r rune) bool { return r < 32 || r == 127 }) {
		return "", errors.New("startup requires a safe absolute executable path")
	}
	quoted := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "`", "\\`", "$", "\\$", "%", "%%").Replace(path)
	return "\"" + strings.ReplaceAll(quoted, "\\", "\\\\") + "\"", nil
}
func (s linuxStartup) content() ([]byte, error) {
	arg, err := execArgument(s.executable)
	if err != nil {
		return nil, err
	}
	data := []byte("[Desktop Entry]\nType=Application\nVersion=1.0\nName=Freehand\nExec=" + arg +
		" --startup\nTerminal=false\nX-Freehand-Autostart=true\n")
	if len(data) > 16384 {
		return nil, errors.New("startup entry is too long")
	}
	return data, nil
}
func (s linuxStartup) executableAvailable() bool {
	info, err := os.Lstat(s.executable)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}

// The mutex serializes app-owned settings transactions; directory-relative
// operations keep symlink redirects out of read/replace/delete paths.
var startupMu sync.Mutex
var errStartupOwnership = errors.New("startup registration is not safely owned by this app")

func ownedStartupStat(st *unix.Stat_t, directory bool) bool {
	kind := uint32(unix.S_IFREG)
	if directory {
		kind = unix.S_IFDIR
	}
	return st.Mode&unix.S_IFMT == kind && st.Uid == uint32(os.Getuid()) && st.Mode&0022 == 0 && (directory || st.Nlink == 1)
}

// Walk with directory-relative O_NOFOLLOW operations. Shared writable
// ancestors require the sticky bit (e.g. /tmp); config and autostart must
// be owned by this user and must not be group/world writable.
func (s linuxStartup) openDirectory(create bool) (int, error) {
	if !filepath.IsAbs(s.config) || filepath.Clean(s.config) != s.config || s.config == "/" {
		return -1, errStartupOwnership
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, errStartupOwnership
	}
	parts := strings.Split(strings.TrimPrefix(s.config, "/"), "/")
	configIndex := len(parts) - 1
	parts = append(parts, "autostart")
	for i, part := range parts {
		next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if errors.Is(openErr, unix.ENOENT) && create {
			if mkErr := unix.Mkdirat(fd, part, 0700); mkErr != nil && !errors.Is(mkErr, unix.EEXIST) {
				unix.Close(fd)
				return -1, errStartupOwnership
			}
			next, openErr = unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		}
		unix.Close(fd)
		if openErr != nil {
			if errors.Is(openErr, unix.ENOENT) {
				return -1, os.ErrNotExist
			}
			return -1, errStartupOwnership
		}
		fd = next
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil ||
			(i >= configIndex && !ownedStartupStat(&st, true)) ||
			(i < configIndex && (st.Uid != 0 && st.Uid != uint32(os.Getuid()) ||
				st.Mode&0022 != 0 && st.Mode&unix.S_ISVTX == 0)) {
			unix.Close(fd)
			return -1, errStartupOwnership
		}
	}
	return fd, nil
}
func (s linuxStartup) readOwned(fd int) (bool, error) {
	fileFD, err := unix.Openat(fd, startupLabel+".desktop", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return false, nil
	}
	if err != nil {
		return false, errStartupOwnership
	}
	f := os.NewFile(uintptr(fileFD), "startup registration")
	defer f.Close()
	var st unix.Stat_t
	if unix.Fstat(fileFD, &st) != nil || !ownedStartupStat(&st, false) || st.Size > 16384 {
		return false, errStartupOwnership
	}
	data, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil || len(data) > 16384 {
		return false, errStartupOwnership
	}
	want, err := s.content()
	if err != nil {
		return false, err
	}
	if !bytes.Equal(data, want) {
		return false, errStartupOwnership
	}
	return true, nil
}
func (s linuxStartup) Enabled() (bool, error) {
	startupMu.Lock()
	defer startupMu.Unlock()
	fd, err := s.openDirectory(false)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer unix.Close(fd)
	owned, err := s.readOwned(fd)
	return owned && s.executableAvailable(), err
}
func (s linuxStartup) Set(on bool) error {
	startupMu.Lock()
	defer startupMu.Unlock()
	var data []byte
	if on {
		if !s.executableAvailable() {
			return errors.New("startup executable is unavailable")
		}
		var err error
		data, err = s.content()
		if err != nil {
			return err
		}
	}
	fd, err := s.openDirectory(on)
	if !on && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	exists, err := s.readOwned(fd)
	if err != nil {
		return err
	}
	if !on {
		if !exists {
			return nil
		}
		if err := unix.Unlinkat(fd, startupLabel+".desktop", 0); err != nil && !errors.Is(err, unix.ENOENT) {
			return errors.New("startup registration cannot be removed")
		}
		return nil
	}
	if exists {
		return nil
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errors.New("startup temporary name unavailable")
	}
	name := ".freehand-startup-" + hex.EncodeToString(nonce[:])
	tempFD, err := unix.Openat(fd, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return errors.New("startup registration cannot be created")
	}
	defer unix.Unlinkat(fd, name, 0)
	f := os.NewFile(uintptr(tempFD), "startup registration")
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return errors.New("startup registration cannot be written")
	}
	if err := f.Sync(); err != nil {
		return errors.New("startup registration cannot be synced")
	}
	if err := f.Close(); err != nil {
		return errors.New("startup registration cannot be closed")
	}
	// RENAME_NOREPLACE prevents a concurrent process from replacing a newly
	// installed foreign registration. Existing app-owned files need no rewrite.
	if err := unix.Renameat2(fd, name, fd, startupLabel+".desktop", unix.RENAME_NOREPLACE); err != nil {
		return errors.New("startup registration cannot be installed")
	}
	return nil
}
