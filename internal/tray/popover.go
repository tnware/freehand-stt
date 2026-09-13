package tray

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"runtime"
)

type popoverTray interface {
	AttachWindow(application.Window) *application.SystemTray
	WindowOffset(int) *application.SystemTray
	OnClick(func()) *application.SystemTray
	OnRightClick(func()) *application.SystemTray
	ToggleWindow()
	HideWindow()
	ShowMenu()
}

// AttachPopover is called after ApplicationStarted, when the tray icon and
// panel exist. Unsupported platforms retain their main-window action.
func (c *Controller) AttachPopover(window application.Window) {
	attachPopover(runtime.GOOS, c.tray, window)
}

func attachPopover(osName string, tray popoverTray, window application.Window) {
	if (osName != "darwin" && osName != "windows") || window == nil {
		return
	}
	tray.AttachWindow(window)
	// AttachWindow does not replace the pre-existing ShowMain click callback.
	tray.OnClick(tray.ToggleWindow)
	if osName == "windows" {
		// Wails positions in DIPs within the taskbar monitor's work area.
		tray.WindowOffset(8)
		// Windows has no AppKit pre-click tracking. Dismiss explicitly before
		// opening the native menu, so it does not overlap the attached panel.
		tray.OnRightClick(func() {
			tray.HideWindow()
			tray.ShowMenu()
		})
		return
	}
	// Run's smart defaults install ShowMenu. Clear it to use AppKit's native
	// pre-click tracking, which also hides the attached panel before the menu.
	tray.OnRightClick(nil)
}
