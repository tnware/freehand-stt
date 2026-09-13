package tray

import "github.com/wailsapp/wails/v3/pkg/application"

type iconTarget interface {
	SetIcon([]byte) *application.SystemTray
	SetDarkModeIcon([]byte) *application.SystemTray
	SetTemplateIcon([]byte) *application.SystemTray
}

func installIcons(target iconTarget, icons Icons, platform string) {
	if platform == "darwin" {
		target.SetTemplateIcon(icons.Light)
		return
	}
	target.SetIcon(icons.Light)
	target.SetDarkModeIcon(icons.Dark)
}
