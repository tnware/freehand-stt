package app

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestSettingsTaskReturnWaitsForMainListeners(t *testing.T) {
	a := &App{mainWindow: &windowController{}, settingsWindow: &windowController{}, shell: &shellNavigation{}, settingsShell: &shellNavigation{}}
	a.settingsReady()
	a.finishSettings("file")
	if !a.settingsShell.runtimeReady || a.shell.runtimeReady || a.shell.event != "workspace:select-task" || a.shell.value != "file" {
		t.Fatal("task return crossed renderer readiness boundaries")
	}
	var got []string
	a.shell.ready(func(event, value string) { got = append(got, event+":"+value) })
	a.shell.ready(func(event, value string) { t.Fatal("task return replayed") })
	if !reflect.DeepEqual(got, []string{"workspace:select-task:file"}) {
		t.Fatalf("return event=%v", got)
	}
}

func TestSettingsConstructorRetainsExistingWindow(t *testing.T) {
	window := &application.WebviewWindow{}
	a := &App{settingsWindow: &windowController{window: window}}
	// No native factory is configured: an existing Settings window must suffice.
	a.newSettingsWindow()
	a.newSettingsWindow()
	if a.settingsWindow.current() != window {
		t.Fatal("settings constructor replaced the retained window")
	}
}

func TestDedicatedSettingsNativeReadinessAndClose(t *testing.T) {
	a := &App{mainWindow: &windowController{}, settingsWindow: &windowController{}, shell: &shellNavigation{}, settingsShell: &shellNavigation{}}
	a.requestSettingsClose()
	a.revealSettings("speech")
	a.revealSettings("audio")
	if a.mainWindow.pending || !a.settingsWindow.pending {
		t.Fatal("Settings revealed unrelated main or lost reveal")
	}
	var got []string
	a.shellReady()
	if a.settingsShell.runtimeReady {
		t.Fatal("main readiness leaked to Settings")
	}
	a.settingsShell.ready(func(e, v string) { got = append(got, e) })
	if !reflect.DeepEqual(got, []string{"settings:open", "settings:close-requested"}) {
		t.Fatalf("events=%v", got)
	}
	a.finishSettings("")
	if a.mainWindow.pending || a.settingsWindow.pending {
		t.Fatal("general finish revealed main or retained pending reveal")
	}
	a.finishSettings("tts")
	if !a.mainWindow.pending {
		t.Fatal("task finish did not reveal main")
	}
	options := settingsWindowOptions(false, config.AppearanceModeSystem, false)
	if options.URL != "/?window=settings" || options.Name != "settings" || options.MinWidth != 560 || options.MinHeight != 520 || !options.Hidden {
		t.Fatalf("settings options=%+v", options)
	}
}
