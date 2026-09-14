package app

import (
	"reflect"
	"testing"
)

// Settings is a pane of the main window. Revealing it must therefore be main
// window navigation, delivered through the one shell readiness handshake
// rather than a second renderer's.
func TestSettingsRevealNavigatesTheMainShell(t *testing.T) {
	a := &App{mainWindow: &windowController{}, shell: &shellNavigation{}}
	a.revealSettings("speech")
	if !a.mainWindow.pending {
		t.Fatal("revealing settings did not reveal the main window")
	}
	if a.shell.event != settingsOpenEvent || a.shell.value != "speech" {
		t.Fatalf("queued event=%q value=%q", a.shell.event, a.shell.value)
	}

	var got []string
	a.shell.ready(func(event, value string) { got = append(got, event+":"+value) })
	a.shell.ready(func(string, string) { t.Fatal("settings request replayed") })
	if !reflect.DeepEqual(got, []string{"settings:open:speech"}) {
		t.Fatalf("delivered=%v", got)
	}
}

// A tray request without a section still has to land somewhere specific.
func TestSettingsRevealDefaultsToGeneral(t *testing.T) {
	a := &App{mainWindow: &windowController{}, shell: &shellNavigation{}}
	a.revealSettings("")
	if a.shell.value != "general" {
		t.Fatalf("section=%q, want general", a.shell.value)
	}
}

// Leaving configuration returns to whichever workflow opened it, and only when
// one did: a general visit has nowhere in particular to go back to.
func TestSettingsFinishReturnsToTheOriginWorkflow(t *testing.T) {
	a := &App{mainWindow: &windowController{}, shell: &shellNavigation{}}
	a.finishSettings("file")
	if a.shell.event != "workspace:select-task" || a.shell.value != "file" {
		t.Fatalf("queued event=%q value=%q", a.shell.event, a.shell.value)
	}

	var got []string
	a.shell.ready(func(event, value string) { got = append(got, event+":"+value) })
	if !reflect.DeepEqual(got, []string{"workspace:select-task:file"}) {
		t.Fatalf("delivered=%v", got)
	}
}

func TestSettingsFinishWithoutOriginNavigatesNowhere(t *testing.T) {
	a := &App{mainWindow: &windowController{}, shell: &shellNavigation{}}
	a.finishSettings("")
	if a.mainWindow.pending || a.shell.event != "" {
		t.Fatalf("general finish navigated: pending=%v event=%q", a.mainWindow.pending, a.shell.event)
	}
}
