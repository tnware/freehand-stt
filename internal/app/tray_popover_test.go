package app

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"os"
	"strings"
	"testing"
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

func TestTrayPopoverRetainedNativeLifecycle(t *testing.T) {
	source, err := os.ReadFile("tray_popover.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range []string{`runtime.GOOS != "darwin"`, `a.trayPopover.current() != nil`, `a.trayPopover.attach(window)`, `a.tray.AttachPopover(window)`, `events.Common.WindowClosing`, `event.Cancel()`, `window.Hide()`} {
		if !strings.Contains(string(source), contract) {
			t.Errorf("missing lifecycle boundary %s", contract)
		}
	}
	startup, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	started := strings.Split(string(startup), "func (a *App) onStarted(")[1]
	if !strings.Contains(started, "a.newTrayPopoverWindow()") {
		t.Fatal("popover not created after native startup")
	}
}

func TestTrayPopoverOptions(t *testing.T) {
	o := trayPopoverWindowOptions(config.AppearanceModeDark, true)
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
