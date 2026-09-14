package app

import (
	"runtime"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestOpeningMainOrSettingsDismissesOnlyPopover(t *testing.T) {
	a := &App{mainWindow: &windowController{}, trayPopover: &windowController{pending: true}, settingsWindow: &windowController{}, settingsShell: &shellNavigation{}}
	a.showMain()
	if a.trayPopover.pending || !a.mainWindow.pending {
		t.Fatal("main did not dismiss popover and queue itself")
	}
	a.mainWindow.pending = false
	a.trayPopover.pending = true
	a.revealSettings("general")
	if a.trayPopover.pending || a.mainWindow.pending || !a.settingsWindow.pending {
		t.Fatal("settings changed unrelated window or left popover")
	}
}

func TestTrayPopoverConstructorRetainsExistingWindow(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("popover construction is unavailable on this platform")
	}
	window := &application.WebviewWindow{}
	a := &App{trayPopover: &windowController{window: window}}
	// Repeated construction must return before touching the native factory.
	// This checks reuse only; close hooks and startup timing need native acceptance.
	a.newTrayPopoverWindow()
	a.newTrayPopoverWindow()
	if a.trayPopover.current() != window {
		t.Fatal("popover constructor replaced the retained window")
	}
}

func TestTrayPopoverWindowsOptions(t *testing.T) {
	o := trayPopoverWindowOptionsForPlatform("windows", config.AppearanceModeDark, true)
	if !o.Windows.HiddenOnTaskbar {
		t.Fatal("tray panel must not create a taskbar entry")
	}
	if o.Mac.WindowClass == application.MacWindowClassPanel || o.Mac.PanelPreferences.NonActivating {
		t.Fatal("Windows must not use Mac panel preferences")
	}
}

func TestTrayPopoverOptions(t *testing.T) {
	o := trayPopoverWindowOptionsForPlatform("darwin", config.AppearanceModeDark, true)
	if o.Name != "tray-popover" || o.URL != "/?window=tray-popover" {
		t.Fatalf("identity %q %q", o.Name, o.URL)
	}
	if o.Width != 360 || o.Height != 500 || !o.Hidden || !o.Frameless || !o.DisableResize {
		t.Fatalf("not compact retained options: %+v", o)
	}
	if !o.HideOnEscape || !o.HideOnFocusLost {
		t.Fatal("missing dismissal")
	}
	if o.Mac.WindowClass != application.MacWindowClassPanel || !o.Mac.PanelPreferences.NonActivating || !o.Mac.PanelPreferences.BecomesKeyOnlyIfNeeded || !o.Mac.PanelPreferences.FloatingPanel {
		t.Fatal("not nonactivating panel")
	}
	if o.Mac.TabbingMode != application.MacWindowTabbingModeDisallowed {
		t.Fatal("panel must not tab")
	}
	if o.Permissions[application.PermissionMicrophone] != application.PermissionDeny {
		t.Fatal("renderer must not own microphone")
	}
}
