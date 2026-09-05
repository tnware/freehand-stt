// Package app is the composition root. It owns wiring and lifecycle only:
// every product decision lives in a cohesive feature package under internal.
package app

import (
	"errors"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/buildinfo"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/connection"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/filetranscription"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	inputservice "github.com/tnware/freehand-stt/internal/input"
	overlayservice "github.com/tnware/freehand-stt/internal/overlay"
	"github.com/tnware/freehand-stt/internal/platform"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/releaseinfo"
	settingsservice "github.com/tnware/freehand-stt/internal/settings"
	"github.com/tnware/freehand-stt/internal/shortcut"
	traycontroller "github.com/tnware/freehand-stt/internal/tray"
	"github.com/tnware/freehand-stt/internal/tts"
	"github.com/tnware/freehand-stt/internal/updates"
	"github.com/tnware/freehand-stt/internal/windowing"
	"github.com/tnware/freehand-stt/internal/windowstate"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/updater"
	wailsgithub "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	applicationName             = "Freehand"
	statusEvent                 = "dictation:status"
	fileStatusEvent             = "file-transcription:status"
	ttsStatusEvent              = "tts:status"
	secondInstanceEvent         = "app:second-instance-revealed"
	settingsChangedEvent        = "settings:changed"
	settingsOpenEvent           = "settings:open"
	settingsCloseRequestedEvent = "settings:close-requested"
	settingsVisibilityEvent     = "settings:visibility"
	aboutVisibilityEvent        = "about:visibility"
)

// Options carries what only the main package can supply: the embedded assets
// and icons, which must be embedded from the module root.
type Options struct {
	Assets           fs.FS
	TrayIcon         []byte
	TrayDarkModeIcon []byte
	InstanceKey      [32]byte
	StartupLaunch    bool
	Release          releaseinfo.Info
	Development      bool
}

// App holds the assembled application. Construction is ordered so that nothing
// observable exists before the thing that publishes to it.
type App struct {
	opts            Options
	settings        config.Settings
	settingsService *settingsservice.Service
	buildInfo       *buildinfo.Service
	connection      *connection.Service
	inputService    *inputservice.Service
	dictation       *dictation.Service
	history         *history.Service
	files           *filetranscription.Service
	tts             *tts.Service
	updates         *updates.Service
	windowing       *windowing.Service
	services        []application.Service
	audio           *platform.Capture
	playback        *platform.Playback
	hold            *platform.HoldHook
	shortcuts       *shortcut.Controller
	capture         *platform.ShortcutCapturer
	wails           *application.App
	mainWindow      *windowController
	settingsWindow  *settingsWindowController
	aboutWindow     *windowController
	windowState     *windowstate.Store
	mainPlacement   *windowstate.Placement
	levels          *levelPump
	overlay         *overlayservice.Service
	tray            *traycontroller.Controller
	logger          *slog.Logger
	wailsLog        *slog.Logger
}

// New assembles the application without starting it.
func New(opts Options) (*App, error) {
	rootLogger := application.DefaultLogger(diagnostics.ApplicationLogLevel)
	logger := rootLogger.With("component", "app")
	store, err := config.NewStore()
	if err != nil {
		return nil, err
	}
	settings, err := store.Load()
	var settingsFailure *config.LoadFailure
	if err != nil {
		failure := config.LoadFailureFor(err)
		settingsFailure = &failure
		logger.Warn("settings load failed; recovery required", "error_kind", failure.Kind)
		settings = config.Default()
	} else if report := store.LoadReport(); report.PreservedFieldCount > 0 {
		logger.Warn("newer settings preserved", "unknown_field_count", report.PreservedFieldCount)
	}
	windowState, err := windowstate.NewStore()
	if err != nil {
		logger.Warn("window placement store unavailable; default placement used", "error_kind", diagnostics.ErrorKind(err))
	}
	var mainPlacement *windowstate.Placement
	if windowState != nil {
		placement, found, loadErr := windowState.Load()
		if loadErr != nil {
			logger.Warn("window placement load failed; default placement used", "error_kind", diagnostics.ErrorKind(loadErr))
		} else if found {
			mainPlacement = &placement
		}
	}

	a := &App{
		opts:           opts,
		settings:       settings,
		mainWindow:     &windowController{},
		settingsWindow: &settingsWindowController{},
		aboutWindow:    &windowController{},
		windowState:    windowState,
		mainPlacement:  mainPlacement,
		logger:         logger,
		wailsLog:       rootLogger.With("component", "wails"),
	}

	// The hold hook is referenced by the service before it exists, so
	// availability is reported through a closure that tolerates the gap.
	holdAvailability := func() (bool, string) {
		if a.hold == nil {
			return false, "Hold-to-talk is not initialized."
		}
		return a.hold.Available()
	}

	a.audio = &platform.Capture{}
	a.overlay = overlayservice.NewService(settings, func() platform.LevelSource {
		return a.audio.NewLevelTap()
	}, rootLogger)
	a.capture = &platform.ShortcutCapturer{}
	keys := credential.Keyring{}
	processingKeys := credential.Keyring{Account: credential.PostProcessingAccount}
	ttsKeys := credential.Keyring{Account: credential.TextToSpeechAccount}
	client := inference.New()
	processor := postprocess.New(client, processingKeys, rootLogger.With("component", "postprocess"))
	nativeInput := platform.NewInput(rootLogger.With("component", "insertion"))
	admission := activity.New(activity.Sources{
		DictationActive: func() bool { return dictation.Active(a.dictation) },
		FileActive:      func() bool { return filetranscription.Active(a.files) },
		StopPlayback: func() error {
			if a.tts == nil {
				return nil
			}
			return a.tts.Stop()
		},
	})
	a.updates = updates.NewService(opts.Release.Version, settings.CheckForUpdates, opts.Development, a.publishUpdateStatus, rootLogger)
	transcripts := history.NewStore(settings.HistoryEnabled, nativeInput)
	a.settingsService = settingsservice.NewService(
		store, settings, keys, processingKeys, platform.Startup{}, holdAvailability,
		a.applyShortcuts, a.applyOverlaySettings,
		transcripts.SetEnabled,
		func(next config.Settings) { filetranscription.ApplySettings(a.files, next) },
		a.publishSettings,
		rootLogger,
		settingsservice.WithConfigurationLoad(store, settingsFailure, store.LoadReport()),
		settingsservice.WithTextToSpeechCredential(ttsKeys),
		settingsservice.WithUpdateChecks(func(enabled bool) { updates.ApplyEnabled(a.updates, enabled) }),
	)
	settingsSource := settingsservice.CurrentSource(a.settingsService)
	profileSource := settingsservice.RequestProfiles(a.settingsService)
	a.dictation = dictation.NewService(a.audio, nativeInput, client, processor, settingsSource, profileSource, transcripts, admission, a.publishStatus, rootLogger.With("component", "dictation"))
	a.files = filetranscription.NewService(settingsSource, profileSource, client, processor, transcripts, nativeInput, a.chooseAudioFile, a.publishFileStatus, a.publishFileDelta, admission, rootLogger)
	a.playback = &platform.Playback{}
	a.tts = tts.NewService(
		settingsservice.TextToSpeechProfiles(a.settingsService),
		client,
		a.playback,
		transcripts,
		func() (string, error) { return filetranscription.PlaybackTranscript(a.files) },
		a.chooseSpeechSaveFile,
		admission,
		a.publishTTSStatus,
		rootLogger,
	)
	a.history = history.NewService(transcripts)
	a.connection = connection.NewService(keys, processingKeys, ttsKeys, client, rootLogger)
	a.inputService = inputservice.NewService(a.audio, a.capture, a, admission, settingsSource, a.publishShortcutCapture, rootLogger)
	a.buildInfo = buildinfo.NewService(
		opts.Release.ProductName,
		opts.Release.Version,
		opts.Release.WindowsVersion,
		opts.Development,
	)
	a.windowing = windowing.NewService(
		a.showSettings,
		a.hideSettings,
		a.settingsWindow.visible,
		a.showAbout,
		a.hideAbout,
		a.aboutWindow.open,
	)
	a.services = []application.Service{
		application.NewService(a.buildInfo),
		application.NewService(a.history),
		application.NewService(a.settingsService),
		application.NewService(a.connection),
		application.NewService(a.dictation),
		application.NewService(a.inputService),
		application.NewService(a.files),
		application.NewService(a.tts),
		application.NewService(a.updates),
		application.NewService(a.windowing),
		application.NewService(a.overlay),
	}
	a.hold = platform.NewHoldHook(
		func() { _ = a.dictation.StartRecording(dictation.RecordingHold) },
		func() { _ = a.dictation.StopRecording() },
		func() { _ = a.dictation.Cancel() },
	)

	a.newWailsApp()
	if err := a.configureUpdater(); err != nil {
		return nil, err
	}
	a.wails.OnShutdown(func() {
		admission.Close()
		a.persistMainWindowPlacement(a.mainWindow.current())
		a.tray.Close()
	})
	a.shortcuts = shortcut.New(a.wails.GlobalShortcut, a.hold, a.toggleRecording, a.mainWindow.Reveal)
	a.wails.Event.OnApplicationEvent(events.Common.ApplicationStarted, a.onStarted)
	a.installTray(rootLogger.With("component", "tray"))

	return a, nil
}

func (a *App) configureUpdater() error {
	if a.opts.Development {
		return nil
	}
	provider, err := wailsgithub.New(wailsgithub.Config{
		Repository:    "tnware/freehand-stt",
		Prerelease:    strings.Contains(a.opts.Release.Version, "-"),
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher:  freehandReleaseAsset,
	})
	if err != nil {
		return err
	}
	if err := a.wails.Updater.Init(updater.Config{
		CurrentVersion: a.opts.Release.Version,
		Providers:      []updater.Provider{provider},
		Window: &updater.BuiltinWindow{
			CSS: updaterWindowCSS,
			Options: updater.WindowOptions{
				Title: "Freehand STT — Software Update",
			},
		},
	}); err != nil {
		return err
	}
	updates.Configure(a.updates, a.wails.Updater)
	return nil
}

func freehandReleaseAsset(request updater.CheckRequest, assets []wailsgithub.ReleaseAsset) int {
	expected := "freehand-" + strings.ToLower(request.Platform) + "-" + strings.ToLower(request.Arch) + ".exe"
	for index, asset := range assets {
		if strings.EqualFold(asset.Name, expected) {
			return index
		}
	}
	return -1
}

const updaterWindowCSS = `
:root {
  --bg: #f7f8fa;
  --surface: #ffffff;
  --surface-2: #f0f2f5;
  --fg: #0f1115;
  --fg-dim: #64748b;
  --fg-faint: #94a3b8;
  --border: #e5e7eb;
  --accent: #2563eb;
  --accent-dim: rgba(37, 99, 235, 0.14);
  --radius: 12px;
  --radius-sm: 8px;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0f1115;
    --surface: #171a20;
    --surface-2: #20242c;
    --fg: #f7f8fa;
    --fg-dim: #94a3b8;
    --fg-faint: #64748b;
    --border: #2a3039;
    --accent: #8ab4ff;
    --accent-dim: rgba(138, 180, 255, 0.16);
  }
}
`

// Run starts the application and blocks until it quits, then unwinds the
// resources that outlive the Wails run loop.
func (a *App) Run() error {
	started := time.Now()
	a.logger.Info("application run started")
	runErr := a.wails.Run()
	if a.levels != nil {
		a.levels.stop()
	}
	if err := a.hold.Close(); err != nil {
		a.logger.Warn("hold-to-talk shutdown failed", "error_kind", diagnostics.ErrorKind(err))
	}
	if runErr != nil {
		a.logger.Error("application run failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(runErr))
	} else {
		a.logger.Info("application run completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "stopped")
	}
	return runErr
}

func (a *App) newWailsApp() {
	name := a.opts.Release.ProductName
	if name == "" {
		name = applicationName
	}
	a.wails = application.New(application.Options{
		Name:        name,
		Description: "Speech to text, anywhere you type.",
		Logger:      a.wailsLog,
		LogLevel:    diagnostics.ApplicationLogLevel,
		Services:    a.services,
		Assets: application.AssetOptions{
			Handler:        application.AssetFileServerFS(a.opts.Assets),
			DisableLogging: true,
		},
		Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:      a.opts.Release.ProductIdentifier,
			EncryptionKey: a.opts.InstanceKey,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				// A second launch reveals the main shell instead of starting another
				// recorder, unless Windows started it at sign-in.
				if !StartupRequested(data.Args) {
					a.mainWindow.Reveal()
					a.wails.Event.Emit(secondInstanceEvent)
				}
			},
		},
	})
}

// onStarted brings up the surfaces that must not touch the Windows message
// loop before Wails owns it. Each failure is degraded, never fatal.
func (a *App) onStarted(*application.ApplicationEvent) {
	a.logger.Info("application started")
	a.tray.Start()
	// Wails populates the screen manager as part of Run, before this event.
	// Creating windows here lets their initial options target a real display
	// without post-creation movement or a visible placement correction.
	a.newMainWindow()
	a.newSettingsWindow()
	a.newAboutWindow()
	a.tray.ApplyDictation(dictation.Snapshot(a.dictation))
	a.tray.ApplyFile(a.files.CurrentFileTranscription())
	overlayservice.Start(a.overlay)
	a.levels = newLevelPump(a.audio.NewLevelTap(), a.emitLevel, a.levelsWanted)
	go a.levels.run()
	if err := a.hold.Start(a.settings.HoldShortcut); err != nil {
		a.logger.Warn("hold-to-talk unavailable", "error_kind", diagnostics.ErrorKind(err))
	}
	if err := a.shortcuts.Configure(a.settings); err != nil {
		a.logger.Warn("global shortcut unavailable", "error_kind", diagnostics.ErrorKind(err))
	}
}

func (a *App) applyOverlaySettings(settings config.Settings) {
	overlayservice.ApplySettings(a.overlay, settings)
}

// publishStatus fans dictation state out to the renderer and passive overlay.
func (a *App) publishStatus(status dictation.Status) {
	if a.tray != nil {
		a.tray.ApplyDictation(status)
	}
	if a.wails != nil {
		a.wails.Event.Emit(statusEvent, status)
	}
	overlayservice.ApplyStatus(a.overlay, status)
}

func (a *App) publishFileStatus(status filetranscription.FileTranscriptionStatus) {
	if a.tray != nil {
		a.tray.ApplyFile(status)
	}
	if a.wails != nil {
		a.wails.Event.Emit(fileStatusEvent, status)
	}
}

func (a *App) publishTTSStatus(status tts.Status) {
	if a.wails != nil {
		a.wails.Event.Emit(ttsStatusEvent, status)
	}
}

func (a *App) publishUpdateStatus(status updates.Status) {
	if a.wails != nil {
		a.wails.Event.Emit(updates.StatusEvent, status)
	}
}

func (a *App) publishFileDelta(delta filetranscription.FileTranscriptionDelta) {
	if a.wails != nil {
		a.wails.Event.Emit(filetranscription.DeltaEvent, delta)
	}
}

func (a *App) publishShortcutCapture(progress inputservice.ShortcutCaptureProgress) {
	if a.wails != nil {
		a.wails.Event.Emit(inputservice.ShortcutCaptureProgressEvent, progress)
	}
}

func (a *App) publishSettings(settings settingsservice.SettingsDTO) {
	if a.wails != nil {
		a.wails.Event.Emit(settingsChangedEvent, settings)
	}
}

// applyShortcuts re-registers global shortcuts after a settings save.
func (a *App) applyShortcuts(settings config.Settings) error {
	if a.shortcuts == nil {
		return nil
	}
	return a.shortcuts.Configure(settings)
}

// Suspend and Resume implement input.ShortcutCaptureGuard. Recording a new
// chord requires the global shortcuts to be released first, or the shortcut
// being recorded would fire instead of being captured. The controller is built
// after the service, so both tolerate being called before it exists.
func (a *App) Suspend() error {
	if a.shortcuts == nil {
		return errors.New("shortcut controller is not ready")
	}
	return a.shortcuts.Suspend()
}

func (a *App) Resume() error {
	if a.shortcuts == nil {
		return errors.New("shortcut controller is not ready")
	}
	return a.shortcuts.Resume()
}

// toggleRecording is the global toggle shortcut. It never cancels an in-flight
// transcription; only idle and failed states may start a new recording.
func (a *App) toggleRecording() {
	switch dictation.Snapshot(a.dictation).State {
	case dictation.Recording:
		_ = a.dictation.StopRecording()
	case dictation.Idle, dictation.Failed:
		_ = a.dictation.StartRecording(dictation.RecordingToggle)
	}
}
