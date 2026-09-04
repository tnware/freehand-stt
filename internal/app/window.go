package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/windowstate"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	mainWindowName          = "main"
	mainWindowURL           = "/index.html#main"
	mainWindowWidth         = 1080
	mainWindowHeight        = 720
	mainWindowMinWidth      = 560
	mainWindowMinHeight     = 560
	settingsWindowName      = "settings"
	settingsWindowURL       = "/index.html#settings"
	settingsWindowWidth     = 880
	settingsWindowHeight    = 680
	settingsWindowMinWidth  = 560
	settingsWindowMinHeight = 520
	aboutWindowName         = "about"
	aboutWindowURL          = "/index.html#about"
	aboutWindowWidth        = 620
	aboutWindowHeight       = 440
	aboutWindowMinWidth     = 480
	aboutWindowMinHeight    = 360
)

func opaqueWindowTheme() application.ThemeSettings {
	return application.ThemeSettings{
		LightModeActive: &application.WindowTheme{
			TitleBarColour:  application.NewRGBPtr(247, 248, 250),
			TitleTextColour: application.NewRGBPtr(15, 17, 21),
			BorderColour:    application.NewRGBPtr(229, 231, 235),
		},
		LightModeInactive: &application.WindowTheme{
			TitleBarColour:  application.NewRGBPtr(247, 248, 250),
			TitleTextColour: application.NewRGBPtr(100, 116, 139),
			BorderColour:    application.NewRGBPtr(229, 231, 235),
		},
		DarkModeActive: &application.WindowTheme{
			TitleBarColour:  application.NewRGBPtr(15, 17, 21),
			TitleTextColour: application.NewRGBPtr(247, 248, 250),
			BorderColour:    application.NewRGBPtr(42, 48, 57),
		},
		DarkModeInactive: &application.WindowTheme{
			TitleBarColour:  application.NewRGBPtr(15, 17, 21),
			TitleTextColour: application.NewRGBPtr(148, 163, 184),
			BorderColour:    application.NewRGBPtr(42, 48, 57),
		},
	}
}

func windowStartsHidden(startupLaunch, showWindowOnLaunch bool) bool {
	return startupLaunch || !showWindowOnLaunch
}

func nativeAppearance(mode config.AppearanceMode) application.Theme {
	switch mode {
	case config.AppearanceModeLight:
		return application.Light
	case config.AppearanceModeDark:
		return application.Dark
	default:
		return application.SystemDefault
	}
}

func appearanceIsDark(mode config.AppearanceMode, systemDark bool) bool {
	switch mode {
	case config.AppearanceModeLight:
		return false
	case config.AppearanceModeDark:
		return true
	default:
		return systemDark
	}
}

func baseWindowOptions(name, title, url string, width, height, minWidth, minHeight int, hidden, useMica bool, appearanceMode config.AppearanceMode, systemDark bool) application.WebviewWindowOptions {
	effectiveAppearance := config.EffectiveAppearanceMode(useMica, appearanceMode)
	options := application.WebviewWindowOptions{
		Name:      name,
		Title:     title,
		Width:     width,
		Height:    height,
		MinWidth:  minWidth,
		MinHeight: minHeight,
		Hidden:    hidden,
		URL:       url,
		Permissions: map[application.PermissionType]application.Permission{
			application.PermissionMicrophone:    application.PermissionDeny,
			application.PermissionCamera:        application.PermissionDeny,
			application.PermissionGeolocation:   application.PermissionDeny,
			application.PermissionNotifications: application.PermissionDeny,
			application.PermissionClipboardRead: application.PermissionDeny,
		},
		BackgroundColour: application.NewRGB(247, 248, 250),
		BackgroundType:   application.BackgroundTypeSolid,
		Frameless:        false,
		Windows: application.WindowsWindow{
			BackdropType:                      application.None,
			DisableIcon:                       true,
			Theme:                             nativeAppearance(effectiveAppearance),
			CustomTheme:                       opaqueWindowTheme(),
			DisableFramelessWindowDecorations: false,
		},
	}
	if appearanceIsDark(effectiveAppearance, systemDark) {
		options.BackgroundColour = application.NewRGB(15, 17, 21)
	}
	if useMica {
		options.BackgroundColour = application.NewRGBA(0, 0, 0, 0)
		options.BackgroundType = application.BackgroundTypeTranslucent
		options.Windows.BackdropType = application.Mica
		// Mica is a dynamic DWM material rather than a fixed colour. Leave the
		// native caption under Windows control so a solid custom title bar does
		// not cover the backdrop or its active/inactive treatment.
		options.Windows.CustomTheme = application.ThemeSettings{}
	}
	return options
}

func mainWindowOptions(startupLaunch, showWindowOnLaunch, useMica bool, appearanceMode config.AppearanceMode, systemDark bool) application.WebviewWindowOptions {
	return baseWindowOptions(
		mainWindowName, "Freehand STT", mainWindowURL,
		mainWindowWidth, mainWindowHeight, mainWindowMinWidth, mainWindowMinHeight,
		windowStartsHidden(startupLaunch, showWindowOnLaunch), useMica, appearanceMode, systemDark,
	)
}

func settingsWindowOptions(useMica bool, appearanceMode config.AppearanceMode, systemDark bool) application.WebviewWindowOptions {
	return baseWindowOptions(
		settingsWindowName, "Freehand STT — Settings", settingsWindowURL,
		settingsWindowWidth, settingsWindowHeight, settingsWindowMinWidth, settingsWindowMinHeight,
		true, useMica, appearanceMode, systemDark,
	)
}

func aboutWindowOptions(useMica bool, appearanceMode config.AppearanceMode, systemDark bool) application.WebviewWindowOptions {
	return baseWindowOptions(
		aboutWindowName, "Freehand STT — About", aboutWindowURL,
		aboutWindowWidth, aboutWindowHeight, aboutWindowMinWidth, aboutWindowMinHeight,
		true, useMica, appearanceMode, systemDark,
	)
}

// chooseAudioFile is the only path from the interactive window to a stored
// audio capability. The full path stays inside Go; Service returns only the
// selected file's bounded metadata to the renderer.
func (a *App) chooseAudioFile() (string, error) {
	if a.wails == nil || a.wails.Dialog == nil {
		return "", nil
	}
	selection, err := a.wails.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                   "Choose an audio file",
		ButtonText:              "Choose audio",
		CanChooseFiles:          true,
		CanChooseDirectories:    false,
		AllowsMultipleSelection: false,
		AllowsOtherFileTypes:    false,
		Filters: []application.FileFilter{{
			DisplayName: "Audio files",
			Pattern:     "*.flac;*.mp3;*.mp4;*.mpeg;*.mpga;*.m4a;*.ogg;*.wav;*.webm",
		}},
		Window: a.mainWindow.current(),
	}).PromptForSingleSelection()
	return normalizeDialogSelection(selection, err)
}

// chooseSpeechSaveFile returns a user-authorized destination for the current
// in-memory TTS session. The service writes the canonical WAV only after this
// native dialog succeeds.
func (a *App) chooseSpeechSaveFile() (string, error) {
	if a.wails == nil || a.wails.Dialog == nil {
		return "", nil
	}
	selection, err := a.wails.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:                "Save generated speech",
		Message:              "Save the current generated speech as a WAV file.",
		Filename:             "freehand-speech-" + time.Now().Format("20060102-150405") + ".wav",
		ButtonText:           "Save audio",
		CanCreateDirectories: true,
		AllowOtherFileTypes:  false,
		Filters: []application.FileFilter{{
			DisplayName: "WAV audio",
			Pattern:     "*.wav",
		}},
		Window: a.mainWindow.current(),
	}).PromptForSingleSelection()
	return normalizeDialogSelection(selection, err)
}

// normalizeDialogSelection converts native user cancellation into the empty,
// successful selection contract consumed by feature services. Wails v3 beta's
// Windows dialog implementation returns an internal, unexported sentinel with
// this stable message, so applications cannot use errors.Is against it yet.
func normalizeDialogSelection(selection string, err error) (string, error) {
	if err == nil {
		return selection, nil
	}
	if errors.Is(err, context.Canceled) || strings.EqualFold(strings.TrimSpace(err.Error()), "cancelled by user") {
		return "", nil
	}
	return "", err
}

// windowController owns the main window and the reveal path used by the
// tray, the global shortcut and a second launch. Reveal requests can arrive
// before the window exists, so they are remembered rather than dropped.
type windowController struct {
	mu      sync.RWMutex
	window  *application.WebviewWindow
	pending bool
}

// settingsWindowController tracks the singleton settings window separately
// from the main shell. A section request may arrive before the hidden
// renderer's runtime is ready, so the most recent section is delivered once
// its event listener can receive it.
type settingsWindowController struct {
	mu             sync.RWMutex
	window         *application.WebviewWindow
	runtimeReady   bool
	pendingReveal  bool
	pendingSection string
}

func (w *settingsWindowController) request(section string) (*application.WebviewWindow, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.window == nil {
		w.pendingReveal = true
		w.pendingSection = section
		return nil, false
	}
	if !w.runtimeReady {
		w.pendingSection = section
	}
	return w.window, w.runtimeReady
}

func (w *settingsWindowController) attach(window *application.WebviewWindow) (bool, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.window = window
	pending, section := w.pendingReveal, w.pendingSection
	w.pendingReveal = false
	return pending, section
}

func (w *settingsWindowController) markRuntimeReady() (*application.WebviewWindow, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.runtimeReady = true
	section := w.pendingSection
	w.pendingSection = ""
	return w.window, section
}

func (w *settingsWindowController) hide() *application.WebviewWindow {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pendingReveal = false
	w.pendingSection = ""
	return w.window
}

func (w *settingsWindowController) visible() bool {
	w.mu.RLock()
	window := w.window
	w.mu.RUnlock()
	// A minimised Settings window still owns an editable draft. Treat it as
	// open so the main renderer cannot enable its competing mutation surface.
	return window != nil && window.IsVisible()
}

func (w *windowController) current() application.Window {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.window == nil {
		return nil
	}
	return w.window
}

// Reveal shows and focuses the main window. Closing the window hides it,
// so this is also the path back from a hidden window.
func (w *windowController) Reveal() {
	w.mu.RLock()
	window := w.window
	w.mu.RUnlock()
	if window != nil {
		window.Show()
		window.Restore()
		window.Focus()
		return
	}
	w.mu.Lock()
	w.pending = true
	w.mu.Unlock()
}

// visible reports whether the main window is on screen. A minimised window
// counts as hidden: nothing in it can be seen.
func (w *windowController) visible() bool {
	w.mu.RLock()
	window := w.window
	w.mu.RUnlock()
	return window != nil && window.IsVisible() && !window.IsMinimised()
}

// open reports whether a reusable auxiliary window is shown. Minimized still
// counts as open so another renderer can present an accurate focus action.
func (w *windowController) open() bool {
	w.mu.RLock()
	window := w.window
	w.mu.RUnlock()
	return window != nil && window.IsVisible()
}

func (w *windowController) Hide() {
	w.mu.RLock()
	window := w.window
	w.mu.RUnlock()
	if window != nil {
		window.Hide()
	}
}

// attach installs the window and drains a reveal that arrived before it
// existed.
func (w *windowController) attach(window *application.WebviewWindow) {
	w.mu.Lock()
	w.window = window
	pending := w.pending
	w.pending = false
	w.mu.Unlock()
	if pending {
		go w.Reveal()
	}
}

// newMainWindow creates the normal interactive shell. Tray Quit is the
// authoritative shutdown path, so closing this window only hides it.
func (a *App) newMainWindow() {
	options := mainWindowOptions(
		a.opts.StartupLaunch,
		a.settings.ShowWindowOnLaunch,
		a.settings.UseMica,
		a.settings.AppearanceMode,
		a.wails.Env.IsDarkMode(),
	)
	a.applyMainWindowPlacement(&options)
	window := a.wails.Window.NewWithOptions(options)
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		a.persistMainWindowPlacement(window)
		window.Hide()
		event.Cancel()
	})
	window.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) {
		a.persistMainWindowPlacement(window)
	})
	window.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		a.tray.SetMainWindowVisible(true)
	})
	window.OnWindowEvent(events.Common.WindowHide, func(*application.WindowEvent) {
		a.tray.SetMainWindowVisible(false)
	})
	window.OnWindowEvent(events.Common.WindowMinimise, func(*application.WindowEvent) {
		a.tray.SetMainWindowVisible(false)
	})
	window.OnWindowEvent(events.Common.WindowRestore, func(*application.WindowEvent) {
		a.tray.SetMainWindowVisible(true)
	})
	window.OnWindowEvent(events.Common.WindowUnMinimise, func(*application.WindowEvent) {
		a.tray.SetMainWindowVisible(true)
	})
	window.OnWindowEvent(events.Windows.WindowEndMove, func(*application.WindowEvent) {
		a.persistMainWindowPlacement(window)
	})
	window.OnWindowEvent(events.Windows.WindowEndResize, func(*application.WindowEvent) {
		a.persistMainWindowPlacement(window)
	})
	a.mainWindow.attach(window)
	a.tray.SetMainWindowVisible(a.mainWindow.visible())
}

// newSettingsWindow creates one hidden renderer and reuses it for every
// settings request. Native close requests are handed to the renderer so its
// existing unsaved-draft confirmation remains authoritative.
func (a *App) newSettingsWindow() {
	window := a.wails.Window.NewWithOptions(settingsWindowOptions(
		a.settings.UseMica,
		a.settings.AppearanceMode,
		a.wails.Env.IsDarkMode(),
	))
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		window.EmitEvent(settingsCloseRequestedEvent)
	})
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		readyWindow, section := a.settingsWindow.markRuntimeReady()
		if readyWindow != nil && section != "" {
			readyWindow.EmitEvent(settingsOpenEvent, section)
		}
	})
	pending, section := a.settingsWindow.attach(window)
	if pending {
		go a.showSettings(section)
	}
}

func (a *App) showSettings(section string) {
	window, ready := a.settingsWindow.request(section)
	if window == nil {
		return
	}
	if !window.IsVisible() {
		a.centerAuxiliaryWindow(window)
	}
	window.Show()
	window.Restore()
	window.Focus()
	if a.wails != nil {
		a.wails.Event.Emit(settingsVisibilityEvent, true)
	}
	if ready {
		window.EmitEvent(settingsOpenEvent, section)
	}
}

func (a *App) hideSettings() {
	a.capture.Cancel()
	window := a.settingsWindow.hide()
	if window != nil {
		window.Hide()
	}
	if a.wails != nil {
		a.wails.Event.Emit(settingsVisibilityEvent, false)
	}
}

// newAboutWindow creates a compact, hidden informational renderer. It has no
// editable draft, so its native close action can hide it immediately.
func (a *App) newAboutWindow() {
	window := a.wails.Window.NewWithOptions(aboutWindowOptions(
		a.settings.UseMica,
		a.settings.AppearanceMode,
		a.wails.Env.IsDarkMode(),
	))
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		a.hideAbout()
		event.Cancel()
	})
	a.aboutWindow.attach(window)
}

func (a *App) showAbout() {
	window := a.aboutWindow.current()
	if window != nil && !window.IsVisible() {
		a.centerAuxiliaryWindow(window)
	}
	a.aboutWindow.Reveal()
	if a.wails != nil {
		a.wails.Event.Emit(aboutVisibilityEvent, true)
	}
}

func (a *App) applyMainWindowPlacement(options *application.WebviewWindowOptions) {
	if options == nil || a.mainPlacement == nil || a.wails == nil || a.wails.Screen == nil {
		return
	}
	bounds, screen, ok := windowstate.Resolve(
		*a.mainPlacement,
		a.wails.Screen.GetAll(),
		options.Width,
		options.Height,
		options.MinWidth,
		options.MinHeight,
	)
	if !ok || screen == nil {
		return
	}
	options.Width = bounds.Width
	options.Height = bounds.Height
	options.InitialPosition = application.WindowXY
	options.Screen = screen
	options.X = bounds.X - screen.WorkArea.X
	options.Y = bounds.Y - screen.WorkArea.Y
}

func (a *App) persistMainWindowPlacement(window application.Window) {
	if window == nil || a.windowState == nil || window.IsMinimised() || window.IsMaximised() || window.IsFullscreen() {
		return
	}
	screen, err := window.GetScreen()
	if err != nil || screen == nil {
		if err != nil {
			a.logger.Warn("window placement screen lookup failed", "error_kind", diagnostics.ErrorKind(err))
		}
		return
	}
	placement, ok := windowstate.Capture(window.Bounds(), screen)
	if !ok {
		return
	}
	if err := a.windowState.Save(placement); err != nil {
		a.logger.Warn("window placement save failed", "error_kind", diagnostics.ErrorKind(err))
	}
}

func (a *App) centerAuxiliaryWindow(window application.Window) {
	owner := a.mainWindow.current()
	if owner == nil || window == nil || owner.IsMinimised() {
		return
	}
	screen, err := owner.GetScreen()
	if err != nil || screen == nil {
		return
	}
	width, height := window.Size()
	window.SetBounds(windowstate.CenterOver(owner.Bounds(), width, height, screen.WorkArea))
}

func (a *App) hideAbout() {
	a.aboutWindow.Hide()
	if a.wails != nil {
		a.wails.Event.Emit(aboutVisibilityEvent, false)
	}
}
