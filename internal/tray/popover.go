package tray

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"runtime"
)

type popoverTray interface {
	AttachWindow(application.Window) *application.SystemTray
	OnClick(func()) *application.SystemTray
	OnRightClick(func()) *application.SystemTray
	ToggleWindow()
}

// AttachPopover is called after ApplicationStarted, when the status item and
// panel exist. Other platforms retain their conventional main-window action.
func (c *Controller) AttachPopover(window application.Window) {
	attachPopover(runtime.GOOS, c.tray, window)
}

func attachPopover(osName string, tray popoverTray, window application.Window) {
	if osName != "darwin" || window == nil {
		return
	}
	tray.AttachWindow(window)
	// AttachWindow does not replace the pre-existing ShowMain click callback.
	tray.OnClick(tray.ToggleWindow)
	// Run's smart defaults install ShowMenu. Clear it to use AppKit's native
	// pre-click tracking, which also hides the attached panel before the menu.
	tray.OnRightClick(nil)
}
