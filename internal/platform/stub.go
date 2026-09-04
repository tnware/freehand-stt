//go:build !windows

package platform

import (
	"context"
	"errors"
	"log/slog"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"github.com/tnware/freehand-stt/internal/insertion"
)

var unavailable = errors.New("Windows native functionality is unavailable on this platform")

type Capture struct{}

type Playback struct{}

func (*Playback) Load([]byte, uint32, uint32) error { return unavailable }
func (*Playback) Play() error                       { return unavailable }
func (*Playback) Pause() error                      { return unavailable }
func (*Playback) Restart() error                    { return unavailable }
func (*Playback) Position() (int64, int64, bool)    { return 0, 0, false }
func (*Playback) OutputName() string                { return "System default" }
func (*Playback) Save(string) error                 { return unavailable }
func (*Playback) Stop() error                       { return nil }
func (*Playback) Unload() error                     { return nil }
func (*Playback) Close() error                      { return nil }

func (*Capture) List(context.Context) ([]audio.Device, error) {
	return []audio.Device{{ID: "", Name: "System default microphone", Default: true}}, nil
}
func (*Capture) Start(context.Context, string, int) (<-chan error, error) {
	return nil, unavailable
}
func (*Capture) StartStream(context.Context, string, int, audio.PCMStreamSink) (<-chan error, error) {
	return nil, unavailable
}
func (*Capture) Stop(context.Context) (audio.Result, error) { return audio.Result{}, unavailable }
func (*Capture) Cancel(context.Context) error               { return nil }
func (*Capture) Close() error                               { return nil }
func (*Capture) NewLevelTap() *LevelTap                     { return &LevelTap{} }
func (*Capture) DeviceName() string                         { return "System default" }
func (*Capture) Prepare(context.Context, string) error      { return unavailable }

type Input struct{}

func NewInput(*slog.Logger) Input { return Input{} }

func (Input) CaptureTarget() (insertion.Target, error)                      { return insertion.Target{}, unavailable }
func (Input) Foreground() (insertion.Target, error)                         { return insertion.Target{}, unavailable }
func (Input) InsertUnicode(context.Context, insertion.Target, string) error { return unavailable }
func (Input) Copy(context.Context, string) error                            { return unavailable }

type Startup struct{}

func (Startup) Set(bool) error         { return unavailable }
func (Startup) Enabled() (bool, error) { return false, unavailable }
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

type StatusOverlay struct{}

func NewStatusOverlay() (*StatusOverlay, error)       { return nil, unavailable }
func (*StatusOverlay) Update(OverlayStatus) error     { return unavailable }
func (*StatusOverlay) Configure(OverlayOptions) error { return unavailable }
func (*StatusOverlay) Close() error                   { return nil }
func (*StatusOverlay) SetLevelSource(LevelSource)     {}
