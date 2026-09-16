//go:build !windows && !darwin

package audio

import (
	"context"
	"errors"
	"github.com/tnware/freehand-stt/internal/audio"
)

var unavailable = errors.New("Windows native functionality is unavailable on this platform")

type Capture struct{}

type Playback struct{}

func (*Playback) Load([]byte, uint32, uint32) error { return unavailable }
func (*Playback) Play() error                       { return unavailable }
func (*Playback) Pause() error                      { return unavailable }
func (*Playback) SeekTo(int64) error                { return unavailable }
func (*Playback) Rewind() error                     { return unavailable }
func (*Playback) Position() (int64, int64, bool)    { return 0, 0, false }
func (*Playback) OutputName() string                { return "System default" }
func (*Playback) Snapshot() ([]byte, error)         { return nil, unavailable }
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
