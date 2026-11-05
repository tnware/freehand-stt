//go:build !linux || !cgo

package app

import "github.com/tnware/freehand-stt/internal/shortcut"

func (a *App) globalShortcutPlatform() shortcut.Global { return a.wails.GlobalShortcut }
