package app

import "github.com/tnware/freehand-stt/internal/diagnostics"

func (a *App) stopNativeSources() {
	a.sourcesOnce.Do(func() {
		if a.levels != nil {
			a.levels.stop()
		}
		if a.hold != nil {
			if err := a.hold.Close(); err != nil && a.logger != nil {
				a.logger.Warn("hold-to-talk shutdown failed", "error_kind", diagnostics.ErrorKind(err))
			}
		}
	})
}

// macOS terminates inside Cocoa's run loop; Run's defer is not the normal
// shutdown path. Wails invokes this after feature ServiceShutdown handlers,
// so SQLite and AX target state outlive every operation that can use them.
func (a *App) closeResources() {
	a.shutdownOnce.Do(func() {
		if a.nativeInput != nil {
			if err := a.nativeInput.Close(); err != nil && a.logger != nil {
				a.logger.Warn("native input shutdown failed", "error_kind", diagnostics.ErrorKind(err))
			}
		}
		if a.storage != nil {
			if err := a.storage.Close(); err != nil && a.logger != nil {
				a.logger.Warn("storage shutdown failed", "error_kind", diagnostics.ErrorKind(err))
			}
		}
	})
}
