package app

import (
	"log/slog"

	traycontroller "github.com/tnware/freehand-stt/internal/tray"
)

// installTray builds the system tray entry. Tray Quit is the authoritative
// shutdown path for the whole application; closing either interactive window
// only hides that window.
func (a *App) installTray(logger *slog.Logger) {
	a.tray = traycontroller.New(a.wails, traycontroller.Icons{
		Light: a.opts.TrayIcon,
		Dark:  a.opts.TrayDarkModeIcon,
	}, traycontroller.Actions{
		ShowMain:     a.mainWindow.Reveal,
		HideMain:     a.mainWindow.Hide,
		ShowSettings: func() { a.showSettings("general") },
		ShowAbout:    a.showAbout,
		CancelVoice:  a.dictation.Cancel,
		CancelFile:   a.files.CancelFileTranscription,
		CopyVoice:    a.dictation.CopyPending,
		CopyFile:     a.files.CopyFileTranscript,
		Quit:         a.wails.Quit,
	}, logger)
}
