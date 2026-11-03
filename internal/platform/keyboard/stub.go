//go:build !windows && !darwin && (!linux || !cgo)

package keyboard

import (
	"context"
	"errors"
	"github.com/tnware/freehand-stt/internal/hotkey"
)

var unavailable = errors.New("This native feature is unavailable on this platform")

func HoldAvailability() (bool, string) {
	return false, "Hold-to-talk is unavailable on this platform. Use the recording button or a supported toggle shortcut."
}

type HoldHook struct{}

func NewHoldHook(func(), func(), func()) *HoldHook { return &HoldHook{} }
func (*HoldHook) Start(string) error               { return unavailable }
func (*HoldHook) Configure(value string) error {
	if value == "" {
		return nil
	}
	return unavailable
}
func (*HoldHook) Available() (bool, string) { return HoldAvailability() }
func (*HoldHook) Close() error              { return nil }

type ShortcutCapturer struct{}

func (*ShortcutCapturer) Capture(context.Context, hotkey.ShortcutPolicy, func(hotkey.Chord)) (hotkey.Chord, bool, error) {
	return hotkey.Chord{}, false, unavailable
}
func (*ShortcutCapturer) Cancel()      {}
func (*ShortcutCapturer) Close() error { return nil }
