package app

import (
	"runtime"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func macAppearance(mode config.AppearanceMode) application.MacAppearanceType {
	switch mode {
	case config.AppearanceModeDark:
		return application.NSAppearanceNameDarkAqua
	case config.AppearanceModeLight:
		return application.NSAppearanceNameAqua
	default:
		return application.DefaultAppearance
	}
}

// Cancel Wails' default reopen handler: it otherwise shows all five hidden
// auxiliary renderers, including credential editing and transcript details.
func reopenMainWindow(event *application.ApplicationEvent, reveal func()) {
	event.Cancel()
	reveal()
}

func (a *App) configureDesktop() {
	if runtime.GOOS != "darwin" {
		return
	}
	menu := application.NewMenu()
	appMenu := menu.AddSubmenu("Freehand")
	appMenu.Add("About Freehand…").OnClick(func(*application.Context) { a.showAbout() })
	appMenu.Add("Settings…").SetAccelerator("Cmd+,").OnClick(func(*application.Context) { a.showSettings("general") })
	appMenu.AddSeparator()
	appMenu.AddRole(application.ServicesMenu)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Hide)
	appMenu.AddRole(application.HideOthers)
	appMenu.AddRole(application.ShowAll)
	appMenu.AddSeparator()
	appMenu.Add("Quit Freehand").SetAccelerator("Cmd+Q").OnClick(func(*application.Context) { a.wails.Quit() })
	menu.AddRole(application.FileMenu)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.WindowMenu)
	a.wails.Menu.SetApplicationMenu(menu)
	a.wails.Event.RegisterApplicationEventHook(events.Mac.ApplicationShouldHandleReopen, func(event *application.ApplicationEvent) { reopenMainWindow(event, a.mainWindow.Reveal) })
}
