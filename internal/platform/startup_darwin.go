//go:build darwin

package platform

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// Startup registers only this user's app at their next login. It deliberately
// does not bootstrap a job now (which would start a second recorder).
type Startup struct{}

const startupLabel = "io.github.tnware.freehand.startup"

type darwinStartup struct{ home, executable string }

func startupForCurrentUser() (darwinStartup, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return darwinStartup{}, errors.New("startup home is unavailable")
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return darwinStartup{}, errors.New("startup home is unavailable")
	}
	exe, err := os.Executable()
	if err != nil {
		return darwinStartup{}, errors.New("startup executable is unavailable")
	}
	return darwinStartup{home: home, executable: exe}, nil
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
func (s darwinStartup) path() string {
	return filepath.Join(s.home, "Library", "LaunchAgents", startupLabel+".plist")
}
func (s darwinStartup) content() ([]byte, error) {
	if !filepath.IsAbs(s.executable) || filepath.Clean(s.executable) != s.executable || len(s.executable) > 4096 || !utf8.ValidString(s.executable) || strings.ContainsFunc(s.executable, func(r rune) bool { return r < 32 || r == 127 }) || filepath.Base(s.executable) != "freehand" || filepath.Base(filepath.Dir(s.executable)) != "MacOS" || filepath.Base(filepath.Dir(filepath.Dir(s.executable))) != "Contents" || !strings.HasSuffix(filepath.Dir(filepath.Dir(filepath.Dir(s.executable))), ".app") {
		return nil, errors.New("startup requires an absolute Freehand app bundle executable")
	}
	info, err := os.Lstat(s.executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return nil, errors.New("startup executable is unavailable")
	}

	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(s.executable)); err != nil {
		return nil, errors.New("startup executable is invalid")
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>` + startupLabel + `</string>
<key>ProgramArguments</key><array><string>` + escaped.String() + `</string><string>--startup</string></array>
<key>RunAtLoad</key><true/>
<key>ProcessType</key><string>Interactive</string>
</dict></plist>
`), nil
}

// The mutex serializes app-owned settings transactions; directory-relative
// operations keep symlink redirects out of read/replace/delete paths.
var startupMu sync.Mutex
var errStartupOwnership = errors.New("startup registration is not safely owned by this app")

func ownedStartupStat(st *unix.Stat_t, directory bool) bool {
	kind := uint16(unix.S_IFREG)
	if directory {
		kind = unix.S_IFDIR
	}
	return st.Mode&unix.S_IFMT == kind && st.Uid == uint32(os.Getuid()) && st.Mode&0022 == 0 && (directory || st.Nlink == 1)
}

// Walk from / with O_NOFOLLOW; never follow a symlink in the home or agent path.
// Ancestors may be owned by root, but home/Library/LaunchAgents must be ours.
func (s darwinStartup) openDirectory(create bool) (int, error) {
	if !filepath.IsAbs(s.home) || filepath.Clean(s.home) != s.home {
		return -1, errStartupOwnership
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, errStartupOwnership
	}
	parts := strings.Split(strings.TrimPrefix(s.home, "/"), "/")
	homeIndex := len(parts) - 1
	parts = append(parts, "Library", "LaunchAgents")
	for i, part := range parts {
		next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if errors.Is(openErr, unix.ENOENT) && create && i > homeIndex {
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
		if i >= homeIndex {
			var st unix.Stat_t
			if unix.Fstat(fd, &st) != nil || !ownedStartupStat(&st, true) {
				unix.Close(fd)
				return -1, errStartupOwnership
			}
		}
	}
	return fd, nil
}
func (s darwinStartup) readOwned(fd int) (bool, error) {
	fileFD, err := unix.Openat(fd, startupLabel+".plist", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
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
func (s darwinStartup) Enabled() (bool, error) {
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
	return s.readOwned(fd)
}
func (s darwinStartup) Set(on bool) error {
	startupMu.Lock()
	defer startupMu.Unlock()
	var data []byte
	if on {
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
		if err := unix.Unlinkat(fd, startupLabel+".plist", 0); err != nil && !errors.Is(err, unix.ENOENT) {
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
	// RENAME_EXCL prevents even a concurrent process from replacing a newly
	// installed foreign registration. Existing app-owned files need no rewrite.
	if err := unix.RenameatxNp(fd, name, fd, startupLabel+".plist", unix.RENAME_EXCL); err != nil {
		return errors.New("startup registration cannot be installed")
	}
	return nil
}
