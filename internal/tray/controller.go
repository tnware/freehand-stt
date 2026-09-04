package tray

import (
	"log/slog"
	"sync"
	"time"

	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/filetranscription"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Actions are app-owned capabilities exposed through the native tray. The tray
// deliberately has no recording-start action because opening a menu destroys
// the insertion target that a new dictation would need to capture honestly.
type Actions struct {
	ShowMain     func()
	HideMain     func()
	ShowSettings func()
	ShowAbout    func()
	CancelVoice  func() error
	CancelFile   func() error
	CopyVoice    func() error
	CopyFile     func() error
	Quit         func()
}

// Icons contains purpose-drawn multi-resolution Windows tray resources. Wails
// selects the closest ICO frame to the current system small-icon metric and
// swaps the two resources when the Windows system theme changes.
type Icons struct {
	Light []byte
	Dark  []byte
}

type menuItems struct {
	status *application.MenuItem
	detail *application.MenuItem
	cancel *application.MenuItem
	copy   *application.MenuItem
	window *application.MenuItem
}

// Controller owns the native tray surface. It is not a Wails binding service:
// no renderer calls it, and its lifecycle is owned by the app composition root.
type Controller struct {
	mu               sync.Mutex
	tray             *application.SystemTray
	items            menuItems
	actions          Actions
	state            snapshot
	voiceStarted     time.Time
	fileStarted      time.Time
	revision         uint64
	started          bool
	closed           bool
	lastPresentation presentation
	logger           *slog.Logger
}

// New installs an initially idle tray and its stable menu structure. Runtime
// mutations start only after Start, once Wails owns the native message loop.
func New(app *application.App, icons Icons, actions Actions, logger *slog.Logger) *Controller {
	if logger == nil {
		logger = slog.Default()
	}
	controller := &Controller{
		actions: actions,
		logger:  logger,
		state: snapshot{
			dictation: dictation.Status{State: dictation.Idle},
			file:      filetranscription.FileTranscriptionStatus{Phase: filetranscription.FileTranscriptionEmpty},
		},
	}
	controller.tray = app.SystemTray.New()
	controller.tray.SetIcon(icons.Light)
	controller.tray.SetDarkModeIcon(icons.Dark)
	controller.installMenu(app)
	initial := present(controller.state)
	controller.lastPresentation = initial
	controller.apply(initial)
	controller.tray.OnClick(controller.invokeShowMain)
	return controller
}

func (c *Controller) installMenu(app *application.App) {
	menu := app.NewMenu()
	c.items.status = menu.Add("Ready").SetEnabled(false)
	c.items.detail = menu.Add("Last activity will appear here").SetEnabled(false).SetHidden(true)
	menu.AddSeparator()
	c.items.cancel = menu.Add("Cancel current operation").SetHidden(true).OnClick(func(*application.Context) {
		c.cancel()
	})
	c.items.copy = menu.Add("Copy transcript").SetHidden(true).OnClick(func(*application.Context) {
		c.copyTranscript()
	})
	c.items.window = menu.Add("Show Freehand").OnClick(func(*application.Context) {
		c.toggleMainWindow()
	})
	menu.Add("Settings…").OnClick(func(*application.Context) {
		c.runAction("open_settings", c.actions.ShowSettings)
	})
	menu.Add("About Freehand…").OnClick(func(*application.Context) {
		c.runAction("open_about", c.actions.ShowAbout)
	})
	menu.AddSeparator()
	menu.Add("Quit Freehand").OnClick(func(*application.Context) {
		c.runAction("quit", c.actions.Quit)
	})
	c.tray.SetMenu(menu)
}

// Start enables main-thread menu refreshes after Wails has started.
func (c *Controller) Start() {
	c.mu.Lock()
	if c.closed || c.started {
		c.mu.Unlock()
		return
	}
	c.started = true
	c.revision++
	revision := c.revision
	view := present(c.state)
	c.lastPresentation = view
	c.mu.Unlock()
	c.schedule(revision, view)
}

// Close prevents late domain publications from mutating native objects during
// Wails shutdown. Wails remains responsible for destroying the tray itself.
func (c *Controller) Close() {
	c.mu.Lock()
	c.closed = true
	c.revision++
	c.mu.Unlock()
}

func (c *Controller) ApplyDictation(status dictation.Status) {
	now := time.Now()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	previous := c.state.dictation
	if status.State == dictation.Recording && (previous.Generation != status.Generation || c.voiceStarted.IsZero()) {
		c.voiceStarted = status.StartedAt
		if c.voiceStarted.IsZero() {
			c.voiceStarted = now
		}
	}
	if dictationSettled(previous, status) {
		c.state.last = dictationActivity(previous, status, c.voiceStarted, now)
	}
	c.state.dictation = status
	c.state.latest = sourceDictation
	c.refreshLocked()
	c.mu.Unlock()
}

func (c *Controller) ApplyFile(status filetranscription.FileTranscriptionStatus) {
	now := time.Now()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	previous := c.state.file
	if status.Phase == filetranscription.FileTranscriptionUploading && (previous.Generation != status.Generation || c.fileStarted.IsZero()) {
		c.fileStarted = now
	}
	if fileSettled(previous, status) {
		c.state.last = fileActivity(previous, status, c.fileStarted, now)
	}
	c.state.file = status
	c.state.latest = sourceFile
	c.refreshLocked()
	c.mu.Unlock()
}

func (c *Controller) SetMainWindowVisible(visible bool) {
	c.mu.Lock()
	if c.closed || c.state.mainWindow == visible {
		c.mu.Unlock()
		return
	}
	c.state.mainWindow = visible
	c.refreshLocked()
	c.mu.Unlock()
}

func (c *Controller) refreshLocked() {
	view := present(c.state)
	if view == c.lastPresentation {
		return
	}
	c.lastPresentation = view
	c.revision++
	revision := c.revision
	if !c.started {
		c.apply(view)
		return
	}
	c.schedule(revision, view)
}

func (c *Controller) schedule(revision uint64, view presentation) {
	application.InvokeAsync(func() {
		c.mu.Lock()
		current := !c.closed && c.revision == revision
		c.mu.Unlock()
		if current {
			c.apply(view)
		}
	})
}

func (c *Controller) apply(view presentation) {
	c.tray.SetTooltip(view.tooltip)
	c.items.status.SetLabel(view.status)
	c.items.detail.SetLabel(view.detail).SetHidden(view.detail == "")
	c.items.cancel.SetLabel(view.cancelLabel).SetHidden(view.cancel == sourceNone)
	c.items.copy.SetHidden(view.copy == sourceNone)
	c.items.window.SetLabel(view.windowLabel)
}

func (c *Controller) toggleMainWindow() {
	c.mu.Lock()
	visible := c.state.mainWindow
	c.mu.Unlock()
	if visible {
		c.runAction("hide_main", c.actions.HideMain)
		return
	}
	c.runAction("show_main", c.actions.ShowMain)
}

func (c *Controller) invokeShowMain() {
	c.runAction("show_main", c.actions.ShowMain)
}

func (c *Controller) cancel() {
	c.mu.Lock()
	source := present(c.state).cancel
	c.mu.Unlock()
	var action func() error
	switch source {
	case sourceDictation:
		action = c.actions.CancelVoice
	case sourceFile:
		action = c.actions.CancelFile
	default:
		return
	}
	c.runErrorAction("cancel", action)
}

func (c *Controller) copyTranscript() {
	c.mu.Lock()
	source := present(c.state).copy
	c.mu.Unlock()
	var action func() error
	switch source {
	case sourceDictation:
		action = c.actions.CopyVoice
	case sourceFile:
		action = c.actions.CopyFile
	default:
		return
	}
	if !c.runErrorAction("copy_transcript", action) || source != sourceFile {
		return
	}
	c.mu.Lock()
	if !c.closed {
		c.state.last = "Audio transcript copied"
		c.refreshLocked()
	}
	c.mu.Unlock()
}

func (c *Controller) runAction(name string, action func()) {
	if action == nil {
		return
	}
	c.logger.Debug("tray action requested", "action", name)
	action()
}

func (c *Controller) runErrorAction(name string, action func() error) bool {
	if action == nil {
		return false
	}
	c.logger.Info("tray action requested", "action", name)
	if err := action(); err != nil {
		c.logger.Warn("tray action failed", "action", name, "error_kind", diagnostics.ErrorKind(err))
		return false
	}
	c.logger.Info("tray action completed", "action", name, "outcome", "succeeded")
	return true
}

func dictationSettled(previous, current dictation.Status) bool {
	if previous.Generation == 0 || previous.Generation != current.Generation {
		return false
	}
	if activeDictation(previous.State) && (current.State == dictation.Idle || current.State == dictation.Failed) {
		return true
	}
	return previous.State == dictation.Failed && previous.CanCopy && current.State == dictation.Idle
}

func dictationActivity(previous, current dictation.Status, started, completed time.Time) string {
	if previous.State == dictation.Cancelling {
		return withElapsed("Last dictation cancelled", started, completed)
	}
	if previous.State == dictation.Failed && previous.CanCopy && current.State == dictation.Idle {
		return "Last dictation copied"
	}
	if current.State == dictation.Failed {
		if current.CanCopy {
			return withElapsed("Last dictation needs attention", started, completed)
		}
		return withElapsed("Last dictation failed", started, completed)
	}
	switch current.Message {
	case "No speech detected":
		return withElapsed("Last dictation · no speech", started, completed)
	case "Post-processing removed filler or noise":
		return withElapsed("Last dictation · no text produced", started, completed)
	case "Post-processing failed; raw transcript used":
		return withElapsed("Last dictation completed · raw fallback", started, completed)
	default:
		return withElapsed("Last dictation completed", started, completed)
	}
}

func fileSettled(previous, current filetranscription.FileTranscriptionStatus) bool {
	if previous.Generation == 0 || previous.Generation != current.Generation || !activeFile(previous.Phase) {
		return false
	}
	return !activeFile(current.Phase)
}

func fileActivity(previous, current filetranscription.FileTranscriptionStatus, started, completed time.Time) string {
	if previous.Phase == filetranscription.FileTranscriptionCancelling && current.Phase == filetranscription.FileTranscriptionSelected {
		return withElapsed("Last audio transcription cancelled", started, completed)
	}
	switch current.Phase {
	case filetranscription.FileTranscriptionCompleted:
		return withElapsed("Last audio transcription completed", started, completed)
	case filetranscription.FileTranscriptionFailed:
		return withElapsed("Last audio transcription failed", started, completed)
	default:
		return ""
	}
}
