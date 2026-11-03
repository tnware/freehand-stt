//go:build !linux

package app

import nativeinput "github.com/tnware/freehand-stt/internal/platform/input"

func (*App) configureNativeInput(nativeinput.Input) {}
