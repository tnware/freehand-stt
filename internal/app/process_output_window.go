package app

import (
	"errors"

	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/windowing"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const processOutputChangedEvent = "process-output:changed"

// Only navigation crosses this window event. Process text is opt-in pull data.
func (a *App) configureProcessOutput() {
	windowing.ConfigureProcessOutput(a.windowing, windowing.ProcessOutputNavigation{
		Exists: func(id string) bool {
			for _, instance := range a.managedRuntime.GetInstances() {
				if instance.Instance.ID == id {
					return true
				}
			}
			return false
		},
		Open:  a.showProcessOutput,
		Close: a.hideProcessOutput,
	})
}

func (a *App) newProcessOutputWindow() {
	window := a.wails.Window.NewWithOptions(baseWindowOptions(
		"process-output", "Freehand — Runtime output", "/?window=process-output",
		920, 580, 560, 360, true, a.settings.UseMica,
		a.settings.AppearanceMode, a.wails.Env.IsDarkMode(),
	))
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		a.windowing.CloseProcessOutput()
	})
	a.outputWindow.attach(window)
}

func (a *App) showProcessOutput(id string) error {
	window := a.outputWindow.current()
	if window == nil {
		return errors.New("runtime output window is unavailable")
	}
	if !window.IsVisible() {
		// Opening an existing private tail never authorizes it without fresh consent.
		if err := a.managedRuntime.DisableProcessOutput(managedruntime.InstanceRequest{InstanceID: id}); err != nil {
			return err
		}
		a.centerAuxiliaryWindow(window)
		window.EmitEvent(processOutputChangedEvent)
	}
	a.outputWindow.Reveal()
	return nil
}

func (a *App) hideProcessOutput(id string) {
	_ = a.managedRuntime.DisableProcessOutput(managedruntime.InstanceRequest{InstanceID: id})
	a.outputWindow.Hide()
	if window := a.outputWindow.current(); window != nil {
		window.EmitEvent(processOutputChangedEvent)
	}
}
