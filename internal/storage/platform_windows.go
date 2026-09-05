//go:build windows

package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func localDataDir() (string, error) {
	p := os.Getenv("LOCALAPPDATA")
	if p == "" {
		return "", errors.New("local application data directory unavailable")
	}
	return p, nil
}
func secureDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	token := windows.GetCurrentProcessToken()
	user, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}
func lockDatabase(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
