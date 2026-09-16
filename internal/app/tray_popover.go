package app

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"runtime"
)

// The panel is independent of main and created only after Wails owns screens
// and the status item. Closing hides it; only application shutdown destroys it.
func (a *App) newTrayPopoverWindow() {
	if (runtime.GOOS != "darwin" && runtime.GOOS != "windows") || a.trayPopover.current() != nil {
		return
	}
	window := a.wails.Window.NewWithOptions(trayPopoverWindowOptions(a.settings.AppearanceMode, a.wails.Env.IsDarkMode()))
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) { event.Cancel(); window.Hide() })
	a.trayPopover.attach(window)
	a.tray.AttachPopover(window)
}

func (a *App) hideTrayPopover() {
	if a.trayPopover != nil {
		a.trayPopover.Hide()
	}
}

func (a *App) showMain() {
	a.hideTrayPopover()
	a.mainWindow.Reveal()
}

func trayPopoverWindowOptions(appearance config.AppearanceMode, systemDark bool) application.WebviewWindowOptions {
	return trayPopoverWindowOptionsForPlatform(runtime.GOOS, appearance, systemDark)
}

func trayPopoverWindowOptionsForPlatform(osName string, appearance config.AppearanceMode, systemDark bool) application.WebviewWindowOptions {
	o := baseWindowOptionsForPlatform(osName, "tray-popover", "Freehand", "/?window=tray-popover", 360, 500, 360, 500, true, false, appearance, systemDark)
	o.Frameless = true
	o.DisableResize = true
	o.HideOnEscape = true
	o.HideOnFocusLost = true
	if osName == "darwin" {
		o.Mac.WindowClass = application.MacWindowClassPanel
		o.Mac.PanelPreferences = application.MacPanelPreferences{NonActivating: true, BecomesKeyOnlyIfNeeded: true, FloatingPanel: true}
	}
	if osName == "windows" {
		// This is a keyboard-interactive popup, not the passive status overlay.
		// Keep native activation for status, explicit copy, and navigation.
		// Recording starts only from a destination-focused shortcut or main.
		o.Windows.HiddenOnTaskbar = true
	}
	return o
}
