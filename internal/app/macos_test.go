package app

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	wailsgithub "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
	"testing"
)

func TestMacUpdaterSelectsOnlyMatchingAppArchive(t *testing.T) {
	assets := []wailsgithub.ReleaseAsset{{Name: "freehand-darwin-arm64.exe"}, {Name: "freehand-darwin-amd64.zip"}, {Name: "freehand-darwin-arm64.zip"}, {Name: "freehand-darwin-arm64.dmg"}}
	if got := freehandReleaseAsset(updater.CheckRequest{Platform: "darwin", Arch: "arm64"}, assets); got != 2 {
		t.Fatalf("index=%d want 2", got)
	}
	if got := freehandReleaseAsset(updater.CheckRequest{Platform: "linux", Arch: "arm64"}, []wailsgithub.ReleaseAsset{{Name: "freehand-linux-arm64.exe"}}); got != -1 {
		t.Fatalf("unsupported platform index=%d", got)
	}
}

func TestMacShellUsesNativeAppearanceAndIgnoresMica(t *testing.T) {
	options := baseWindowOptionsForPlatform("darwin", "test", "Test", "/index.html", 800, 600, 560, 520, false, true, config.AppearanceModeDark, false)
	if options.BackgroundType != application.BackgroundTypeSolid || options.BackgroundColour != application.NewRGB(18, 18, 18) {
		t.Fatalf("Mac background=%#v", options)
	}
	if options.Mac.Appearance != application.NSAppearanceNameDarkAqua {
		t.Fatal("Mac native caption does not follow app appearance")
	}
	if options.Frameless || options.Mac.TitleBar.Hide {
		t.Fatal("Mac native chrome replaced")
	}
	if options.Mac.TabbingMode != application.MacWindowTabbingModeDisallowed {
		t.Fatal("auxiliary windows can be tabbed into other surfaces")
	}
}

func TestMacMainWindowKeepsNativeTrafficLightsWithFullSizeContent(t *testing.T) {
	options := mainWindowOptionsForPlatform("darwin", false, true, true, config.AppearanceModeDark, false)
	if options.Frameless || options.Mac.TitleBar != application.MacTitleBarHiddenInset {
		t.Fatal("main workspace does not preserve the native inset traffic lights")
	}
	if options.Mac.InvisibleTitleBarHeight != 0 {
		t.Fatal("a native drag strip would intercept the workspace title-bar controls")
	}
	if options.DisableResize || options.Mac.TabbingMode != application.MacWindowTabbingModeDisallowed {
		t.Fatal("main workspace lost its resizing or independent-window policy")
	}
	if options.Windows.NonClientRegionSupport || options.Windows.WebView2CompositionHosting {
		t.Fatal("Windows custom caption settings leaked into the macOS window")
	}
	if options.BackgroundType != application.BackgroundTypeSolid || options.Mac.Appearance != application.NSAppearanceNameDarkAqua {
		t.Fatal("macOS full-size content lost its native appearance or ignored Mica policy")
	}
}

func TestDockReopenRevealsOnlyMainWindow(t *testing.T) {
	event := &application.ApplicationEvent{}
	calls := 0
	reopenMainWindow(event, func() { calls++ })
	if !event.IsCancelled() {
		t.Fatal("Wails default would reveal every hidden auxiliary renderer")
	}
	if calls != 1 {
		t.Fatalf("main reveals=%d", calls)
	}
}
