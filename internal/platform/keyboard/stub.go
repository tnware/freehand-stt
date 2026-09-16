//go:build !windows && !darwin

package keyboard

import (
	"context"
	"errors"
	"github.com/tnware/freehand-stt/internal/hotkey"
)

var unavailable = errors.New("Windows native functionality is unavailable on this platform")

func HoldAvailability() (bool, string) {
	return false, "Hold-to-talk requires the Windows low-level keyboard hook and is unavailable in this build."
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
