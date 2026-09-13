package app

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	settingsservice "github.com/tnware/freehand-stt/internal/settings"
	"path/filepath"
)

const managedRuntimeStatusEvent = "managed-runtime:status"

// Composition only: constructors do not install, inspect or start processes.
func (a *App) managedRuntimeOptions(directory string, admission *activity.Coordinator) managedruntime.Options {
	return managedruntime.Options{
		Directory:   filepath.Join(directory, "managed-runtime"),
		Preferences: a.settings.ManagedRuntime,
		Logger:      a.logger,
		SavePreferences: func(p managedruntime.Preferences) error {
			if a.settingsService == nil {
				return errors.New("settings are not ready")
			}
			return settingsservice.SaveManagedPreferences(a.settingsService, p)
		},
		CheckIdle: func() error {
			if err := admission.CheckShortcutCapture(); err != nil {
				return errors.New("finish active transcription before changing managed speech")
			}
			return nil
		},
		Changed: func(status managedruntime.Status) {
			if a.wails != nil {
				a.wails.Event.Emit(managedRuntimeStatusEvent, status)
			}
		},
	}
}
