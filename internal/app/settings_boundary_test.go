package app

import (
	"github.com/tnware/freehand-stt/internal/config"
	"os"
	"reflect"
	"strings"
	"testing"
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

func TestSettingsIsOnlyConfigurationNativeWindow(t *testing.T) {
	source, err := os.ReadFile("window.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, obsolete := range []string{"newConnectionManagerWindow", "connectionsWindow", `"Freehand — Connection Manager"`} {
		if strings.Contains(text, obsolete) {
			t.Fatalf("separate Connections native boundary: %s", obsolete)
		}
	}
	start := strings.Index(text, "func (a *App) revealSettings(")
	end := strings.Index(text[start:], "func (a *App) emitSettings(") + start
	if strings.Contains(text[start:end], "NewWithOptions") || strings.Contains(text[start:end], "mainWindow.Reveal") {
		t.Fatal("repeat open creates a window or reveals main")
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
