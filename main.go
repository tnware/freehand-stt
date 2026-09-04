// Command freehand is the Freehand Windows application entry point. It embeds
// the frontend and the tray icon, which must be embedded from
// the module root, and hands everything else to internal/app.
package main

import (
	"embed"
	"log"
	"os"

	"github.com/tnware/freehand-stt/internal/app"
	"github.com/tnware/freehand-stt/internal/buildinfo"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/releaseinfo"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/tray-light.ico
var trayIcon []byte

//go:embed build/windows/tray-dark.ico
var trayDarkModeIcon []byte

//go:embed build/config.yml
var releaseConfig []byte

var instanceKey = [32]byte{
	0x41, 0x19, 0x77, 0x2a, 0xb3, 0x8c, 0x15, 0xef,
	0x01, 0x49, 0x68, 0x91, 0xba, 0x36, 0x2c, 0xdd,
	0x17, 0x7e, 0x55, 0xa0, 0xce, 0x24, 0x8b, 0x63,
	0x09, 0xfa, 0x43, 0x88, 0xd1, 0x5b, 0x32, 0x6e,
}

func main() {
	release, err := releaseinfo.Parse(releaseConfig)
	if err != nil {
		// The embedded release source contains no user data. Keep the failure
		// bounded anyway: a malformed build must be corrected, not diagnosed by
		// echoing arbitrary configuration content.
		log.Printf("application bootstrap failed: error_kind=invalid_release_identity")
		os.Exit(1)
	}
	application, err := app.New(app.Options{
		Assets:           assets,
		TrayIcon:         trayIcon,
		TrayDarkModeIcon: trayDarkModeIcon,
		InstanceKey:      instanceKey,
		StartupLaunch:    app.StartupRequested(os.Args),
		Release:          release,
		Development:      buildinfo.Development,
	})
	if err != nil {
		// The Wails logger does not exist when construction fails. Keep this one
		// content-free bootstrap logger isolated from runtime diagnostics.
		log.Printf("application bootstrap failed: error_kind=%s", diagnostics.ErrorKind(err))
		os.Exit(1)
	}
	if err := application.Run(); err != nil {
		// App.Run owns the structured terminal record after initialization.
		os.Exit(1)
	}
}
