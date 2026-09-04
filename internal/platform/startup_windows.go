//go:build windows

package platform

import (
	"errors"
	"golang.org/x/sys/windows/registry"
	"os"
	"strings"
)

const runPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const runName = "Freehand"

type Startup struct{}

func command() (string, error) {
	p, e := os.Executable()
	if e != nil {
		return "", e
	}
	if strings.Contains(p, "\"") {
		return "", errors.New("executable path contains a quote")
	}
	v := "\"" + p + "\" --startup"
	if len(v) > 2048 {
		return "", errors.New("startup command is too long")
	}
	return v, nil
}
func (Startup) Set(on bool) error {
	k, _, e := registry.CreateKey(registry.CURRENT_USER, runPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	if !on {
		e = k.DeleteValue(runName)
		if errors.Is(e, registry.ErrNotExist) {
			return nil
		}
		return e
	}
	v, e := command()
	if e != nil {
		return e
	}
	return k.SetStringValue(runName, v)
}
func (Startup) Enabled() (bool, error) {
	k, e := registry.OpenKey(registry.CURRENT_USER, runPath, registry.QUERY_VALUE)
	if errors.Is(e, registry.ErrNotExist) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	defer k.Close()
	v, _, e := k.GetStringValue(runName)
	if errors.Is(e, registry.ErrNotExist) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	want, e := command()
	return v == want, e
}
