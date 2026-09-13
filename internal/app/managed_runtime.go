package app

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	settingsservice "github.com/tnware/freehand-stt/internal/settings"
)

const managedRuntimeStatusEvent = "managed-runtime:status"

// Composition only: constructors do not install, inspect or start processes.
func (a *App) managedRuntimeOptions(directory string, admission *activity.Coordinator) managedruntime.ManagerOptions {
	return managedruntime.ManagerOptions{
		Directory: directory,
		Instances: a.settings.ManagedRuntimes,
		Logger:    a.logger,
		SaveInstances: func(instances []managedruntime.Instance) error {
			if a.settingsService == nil {
				return errors.New("settings are not ready")
			}
			return settingsservice.SaveManagedInstances(a.settingsService, instances)
		},
		CheckIdle: func() error {
			if err := admission.CheckShortcutCapture(); err != nil {
				return errors.New("finish active transcription before changing managed speech")
			}
			return nil
		},
		Changed: func(status managedruntime.InstanceStatus) {
			if a.wails != nil {
				a.wails.Event.Emit(managedRuntimeStatusEvent, status)
			}
		},
	}
}
