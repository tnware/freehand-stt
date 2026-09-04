package app

import (
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	wailsgithub "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestFreehandReleaseAssetSelectsOnlyTheBarePlatformBinary(t *testing.T) {
	assets := []wailsgithub.ReleaseAsset{
		{Name: "freehand-windows-amd64-installer.exe"},
		{Name: "freehand-windows-amd64.exe"},
		{Name: "SHA256SUMS"},
	}
	got := freehandReleaseAsset(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets)
	if got != 1 {
		t.Fatalf("asset index = %d, want 1", got)
	}
	if got := freehandReleaseAsset(updater.CheckRequest{Platform: "linux", Arch: "amd64"}, assets); got != -1 {
		t.Fatalf("unsupported platform selected asset %d", got)
	}
}

func TestNativeDialogCancellationIsNotAnApplicationError(t *testing.T) {
	selection, err := normalizeDialogSelection("", errors.New("cancelled by user"))
	if err != nil || selection != "" {
		t.Fatalf("cancel result = %q, %v", selection, err)
	}

	want := errors.New("dialog initialization failed")
	if _, err := normalizeDialogSelection("", want); !errors.Is(err, want) {
		t.Fatalf("genuine dialog failure was swallowed: %v", err)
	}
}

func TestStartupRequested(t *testing.T) {
	if !StartupRequested([]string{"dictation.exe", "--startup"}) {
		t.Fatal("the exact startup flag was not recognized")
	}
	// A value-carrying form must not match, so a crafted argument cannot
	// suppress the settings window on an ordinary launch.
	if StartupRequested([]string{"dictation.exe", "--startup=true"}) {
		t.Fatal("startup argument parsing is not exact")
	}
	if StartupRequested(nil) || StartupRequested([]string{"dictation.exe"}) {
		t.Fatal("an absent startup flag was treated as present")
	}
}

func TestMainWindowLaunchVisibility(t *testing.T) {
	for _, test := range []struct {
		name     string
		startup  bool
		show     bool
		wantHide bool
	}{
		{name: "manual default shows", show: true},
		{name: "manual preference hides", show: false, wantHide: true},
		{name: "Windows startup stays quiet", startup: true, show: true, wantHide: true},
		{name: "Windows startup ignores manual-show preference", startup: true, show: false, wantHide: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := windowStartsHidden(test.startup, test.show); got != test.wantHide {
				t.Fatalf("windowStartsHidden(%v, %v) = %v, want %v", test.startup, test.show, got, test.wantHide)
			}
		})
	}
}

func TestMainWindowUsesNativeChrome(t *testing.T) {
	options := mainWindowOptions(false, true, false, config.AppearanceModeSystem, false)
	if options.MinWidth != 560 {
		t.Fatalf("main window minimum width = %d, want 560", options.MinWidth)
	}
	if options.Frameless {
		t.Fatal("main window replaces the native frame")
	}
	if options.Title != "Freehand STT" {
		t.Fatalf("main window caption = %q, want Freehand STT", options.Title)
	}
	if !options.Windows.DisableIcon {
		t.Fatal("main window leaves the app icon in the native title bar")
	}
	if options.Windows.DisableFramelessWindowDecorations {
		t.Fatal("main window disables native Windows decorations")
	}
	if options.DisableResize {
		t.Fatal("native window is not resizable")
	}
	if options.BackgroundType != application.BackgroundTypeSolid {
		t.Fatal("main window does not use a solid backdrop by default")
	}
	if options.BackgroundColour != application.NewRGB(247, 248, 250) {
		t.Fatalf("default light background = %#v, want off white", options.BackgroundColour)
	}
	if options.Windows.BackdropType != application.None {
		t.Fatal("main window enables a system backdrop without opt-in")
	}
	if options.Windows.Theme != application.SystemDefault {
		t.Fatal("main window does not follow the Windows theme")
	}
	assertWindowThemeColour(t, options.Windows.CustomTheme.LightModeActive, "light active", [3]uint8{247, 248, 250}, [3]uint8{15, 17, 21}, [3]uint8{229, 231, 235})
	assertWindowThemeColour(t, options.Windows.CustomTheme.LightModeInactive, "light inactive", [3]uint8{247, 248, 250}, [3]uint8{100, 116, 139}, [3]uint8{229, 231, 235})
	assertWindowThemeColour(t, options.Windows.CustomTheme.DarkModeActive, "dark active", [3]uint8{15, 17, 21}, [3]uint8{247, 248, 250}, [3]uint8{42, 48, 57})
	assertWindowThemeColour(t, options.Windows.CustomTheme.DarkModeInactive, "dark inactive", [3]uint8{15, 17, 21}, [3]uint8{148, 163, 184}, [3]uint8{42, 48, 57})
}

func assertWindowThemeColour(t *testing.T, theme *application.WindowTheme, name string, titleBar, titleText, border [3]uint8) {
	t.Helper()
	if theme == nil {
		t.Fatalf("%s native window theme is missing", name)
	}
	for label, test := range map[string]struct {
		got  *uint32
		want *uint32
	}{
		"title bar":  {got: theme.TitleBarColour, want: application.NewRGBPtr(titleBar[0], titleBar[1], titleBar[2])},
		"title text": {got: theme.TitleTextColour, want: application.NewRGBPtr(titleText[0], titleText[1], titleText[2])},
		"border":     {got: theme.BorderColour, want: application.NewRGBPtr(border[0], border[1], border[2])},
	} {
		if test.got == nil || *test.got != *test.want {
			t.Errorf("%s %s colour = %v, want %#08x", name, label, test.got, *test.want)
		}
	}
}

func TestMainWindowDeniesUnusedWebViewCapabilities(t *testing.T) {
	options := mainWindowOptions(false, true, false, config.AppearanceModeSystem, false)
	for _, capability := range []application.PermissionType{
		application.PermissionMicrophone,
		application.PermissionCamera,
		application.PermissionGeolocation,
		application.PermissionNotifications,
		application.PermissionClipboardRead,
	} {
		if got := options.Permissions[capability]; got != application.PermissionDeny {
			t.Fatalf("permission %d = %d, want deny", capability, got)
		}
	}
	if options.AllowSimpleEventEmit {
		t.Fatal("simple renderer event emission is enabled")
	}
	if options.EnableFileDrop {
		t.Fatal("renderer file drop is enabled")
	}
}

func TestMainWindowUsesDarkSolidBackground(t *testing.T) {
	options := mainWindowOptions(false, true, false, config.AppearanceModeSystem, true)
	if options.BackgroundColour != application.NewRGB(15, 17, 21) {
		t.Fatalf("default dark background = %#v, want charcoal", options.BackgroundColour)
	}
}

func TestMainWindowCanOverrideSystemAppearanceWithoutMica(t *testing.T) {
	dark := mainWindowOptions(false, true, false, config.AppearanceModeDark, false)
	if dark.Windows.Theme != application.Dark || dark.BackgroundColour != application.NewRGB(15, 17, 21) {
		t.Fatalf("forced dark appearance = theme %d background %#v", dark.Windows.Theme, dark.BackgroundColour)
	}

	light := mainWindowOptions(false, true, false, config.AppearanceModeLight, true)
	if light.Windows.Theme != application.Light || light.BackgroundColour != application.NewRGB(247, 248, 250) {
		t.Fatalf("forced light appearance = theme %d background %#v", light.Windows.Theme, light.BackgroundColour)
	}
}

func TestMainWindowUsesMicaOnlyAfterOptIn(t *testing.T) {
	options := mainWindowOptions(false, true, true, config.AppearanceModeDark, true)
	if options.BackgroundType != application.BackgroundTypeTranslucent {
		t.Fatal("opted-in main window does not use a translucent backdrop")
	}
	if options.BackgroundColour.Alpha != 0 {
		t.Fatal("webview background covers the opted-in system backdrop")
	}
	if options.Windows.BackdropType != application.Mica {
		t.Fatal("opted-in main window does not use Windows Mica")
	}
	if options.Windows.Theme != application.SystemDefault {
		t.Fatal("Mica did not return native appearance control to Windows")
	}
	if options.Windows.CustomTheme.LightModeActive != nil ||
		options.Windows.CustomTheme.LightModeInactive != nil ||
		options.Windows.CustomTheme.DarkModeActive != nil ||
		options.Windows.CustomTheme.DarkModeInactive != nil {
		t.Fatal("a solid custom title-bar theme covers the opted-in Mica backdrop")
	}
}

func TestSettingsWindowIsAHiddenReusableRenderer(t *testing.T) {
	options := settingsWindowOptions(false, config.AppearanceModeSystem, false)
	if options.Name != "settings" || options.URL != settingsWindowURL {
		t.Fatalf("settings identity = name %q URL %q", options.Name, options.URL)
	}
	if options.Title != "Freehand STT — Settings" {
		t.Fatalf("settings caption = %q", options.Title)
	}
	if !options.Hidden {
		t.Fatal("secondary settings renderer is visible at startup")
	}
	if options.Width != 880 || options.Height != 680 {
		t.Fatalf("settings size = %dx%d", options.Width, options.Height)
	}
	if options.MinWidth != 560 || options.MinHeight != 520 {
		t.Fatalf("settings minimum size = %dx%d", options.MinWidth, options.MinHeight)
	}
	if options.Frameless || options.Windows.DisableFramelessWindowDecorations {
		t.Fatal("settings window replaces native Windows chrome")
	}
}

func TestAboutWindowIsACompactHiddenReusableRenderer(t *testing.T) {
	options := aboutWindowOptions(false, config.AppearanceModeSystem, false)
	if options.Name != "about" || options.URL != aboutWindowURL {
		t.Fatalf("About identity = name %q URL %q", options.Name, options.URL)
	}
	if options.Title != "Freehand STT — About" || !options.Hidden {
		t.Fatalf("About caption/visibility = %q hidden=%v", options.Title, options.Hidden)
	}
	if options.Width != 620 || options.Height != 440 {
		t.Fatalf("About size = %dx%d", options.Width, options.Height)
	}
	if options.MinWidth != 480 || options.MinHeight != 360 {
		t.Fatalf("About minimum size = %dx%d", options.MinWidth, options.MinHeight)
	}
	if options.Frameless || options.Windows.DisableFramelessWindowDecorations {
		t.Fatal("About window replaces native Windows chrome")
	}
}

// Reveal requests arrive from the tray, the global shortcut and a second
// launch, any of which can land before the window exists.
func TestWindowControllerRemembersARevealBeforeAttach(t *testing.T) {
	controller := &windowController{}
	controller.Reveal()
	controller.mu.RLock()
	pending := controller.pending
	controller.mu.RUnlock()
	if !pending {
		t.Fatal("a reveal that arrived before the window existed was dropped")
	}
}

func TestSettingsWindowControllerRemembersSectionUntilRuntimeReady(t *testing.T) {
	controller := &settingsWindowController{}
	window, ready := controller.request("processing")
	if window != nil || ready {
		t.Fatal("settings request reported an unattached renderer as ready")
	}
	controller.mu.RLock()
	pending, section := controller.pendingReveal, controller.pendingSection
	controller.mu.RUnlock()
	if !pending || section != "processing" {
		t.Fatalf("pending settings request = %v %q", pending, section)
	}
}
