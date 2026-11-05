//go:build linux && cgo

package app

import (
	"errors"
	"os"
	"strings"

	nativekeyboard "github.com/tnware/freehand-stt/internal/platform/keyboard"
	"github.com/tnware/freehand-stt/internal/shortcut"
)

func (a *App) globalShortcutPlatform() shortcut.Global {
	if nativekeyboard.X11Session() {
		owner := nativekeyboard.NewGlobal()
		a.nativeGlobal = owner
		return owner
	}
	// A Wayland compositor owns shortcut approval; never substitute XWayland grabs.
	if strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") ||
		(os.Getenv("XDG_SESSION_TYPE") == "" && os.Getenv("WAYLAND_DISPLAY") != "") {
		return a.wails.GlobalShortcut
	}
	return unavailableLinuxShortcuts{}
}

type unavailableLinuxShortcuts struct{}

func (unavailableLinuxShortcuts) Register(string, func()) error {
	return errors.New("Global shortcuts require a recognized local X11 or Wayland desktop session.")
}
func (unavailableLinuxShortcuts) Unregister(string) error { return nil }
