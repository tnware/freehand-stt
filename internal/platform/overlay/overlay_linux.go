//go:build linux && cgo

package overlay

/*
#cgo pkg-config: xcb xcb-shape xcb-randr cairo-xcb pangocairo gio-2.0
#cgo LDFLAGS: -lm
#include "overlay_linux.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/tnware/freehand-stt/internal/audio"
	nativekeyboard "github.com/tnware/freehand-stt/internal/platform/keyboard"
)

const linuxOverlayCloseBound = time.Second

var linuxOverlayOpenSlot = make(chan struct{}, 1)

// StatusOverlay owns a dedicated XCB connection and native rendering thread.
// The latest-value mailbox is bounded; hidden surfaces have no animation timer.
// Neither this owner nor its C backend activates, grabs input or uses a clipboard.
type StatusOverlay struct {
	mu      sync.Mutex
	status  OverlayStatus
	options OverlayOptions
	source  LevelSource
	closed  bool
	failure error
	wake    chan struct{}
	stop    chan struct{}
	done    chan struct{}
}

func NewStatusOverlay() (*StatusOverlay, error) {
	if !nativekeyboard.X11Session() {
		return nil, errors.New("The native status overlay requires a local X11 session. Wayland and XWayland are unavailable.")
	}
	// A stalled local server must not accumulate pending connection attempts.
	select {
	case linuxOverlayOpenSlot <- struct{}{}:
	default:
		return nil, errors.New("An X11 status overlay connection is still starting.")
	}
	o := &StatusOverlay{options: DefaultOverlayOptions(), wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	ready := make(chan error, 1)
	go o.run(ready)
	select {
	case err := <-ready:
		if err != nil {
			return nil, err
		}
		return o, nil
	case <-time.After(1500 * time.Millisecond):
		_ = o.Close()
		return nil, errors.New("X11 status overlay connection timed out.")
	}
}

func (o *StatusOverlay) signal() {
	select {
	case o.wake <- struct{}{}:
	default:
	}
}

func (o *StatusOverlay) Configure(options OverlayOptions) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return errors.New("native status overlay is closed")
	}
	if o.failure != nil {
		return o.failure
	}
	o.options = linuxOverlayOptions(options)
	o.signal()
	return nil
}

func (o *StatusOverlay) Update(status OverlayStatus) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return errors.New("native status overlay is closed")
	}
	if o.failure != nil {
		return o.failure
	}
	status.Shortcut = linuxOverlayShortcut(status.Shortcut)
	status.Caption = BoundedOverlayCaption(status.Caption)
	if !status.CaptionEnabled || !resolveOverlayView(status).Visible {
		status.Caption = ""
	}
	o.status = status
	o.signal()
	return nil
}

func (o *StatusOverlay) SetLevelSource(source LevelSource) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed && o.failure == nil {
		o.source = source
		o.signal()
	}
}

// Close revokes new work and erases transient text immediately. A stalled local
// X server cannot hold application shutdown indefinitely; timeout is an error,
// not a claim that native resources have already been destroyed.
func (o *StatusOverlay) Close() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	if !o.closed {
		o.closed = true
		o.status = OverlayStatus{}
		o.source = nil
		close(o.stop)
	}
	o.mu.Unlock()
	select {
	case <-o.done:
		return nil
	case <-time.After(linuxOverlayCloseBound):
		return errors.New("X11 status overlay shutdown timed out")
	}
}

func linuxOverlayOptions(options OverlayOptions) OverlayOptions {
	defaults := DefaultOverlayOptions()
	if math.IsNaN(options.Scale) || math.IsInf(options.Scale, 0) {
		options.Scale = defaults.Scale
	}
	if math.IsNaN(options.Opacity) || math.IsInf(options.Opacity, 0) {
		options.Opacity = defaults.Opacity
	}
	if math.IsNaN(options.Glow) || math.IsInf(options.Glow, 0) {
		options.Glow = defaults.Glow
	}
	return normalizeOverlayOptions(options)
}

func linuxOverlayShortcut(text string) string {
	runes := []rune(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, text))
	return string(runes[:min(len(runes), 48)])
}

func linuxOverlayPhase(status OverlayStatus) (string, string) {
	finish := "Use the shortcut again to finish"
	if status.RecordingMode == OverlayRecordingHold {
		finish = "Release the shortcut to finish"
	}
	switch status.Kind {
	case OverlayRecording, OverlayRecordingSpeech:
		return "Listening", finish
	case OverlayRecordingSilence:
		return "Silence detected", finish
	case OverlayRecordingCountdown:
		return "Silence countdown", "Speak to keep recording"
	case OverlayTranscribing:
		return "Transcribing", "Turning speech into text"
	case OverlayPostProcessing:
		return "Cleaning up", "Applying your processing profile"
	case OverlayReady:
		return "Ready", "Open Freehand for the transcript"
	case OverlayCopyRequired:
		return "Copy required", "Open Freehand to copy the transcript"
	case OverlayFailed:
		return "Something went wrong", "Open Freehand for details"
	case OverlayCancelling:
		return "Cancelling", "Discarding this dictation"
	default:
		return "", ""
	}
}

func linuxOverlayText(dst []C.char, text string) {
	for i := range dst {
		dst[i] = 0
	}
	// All call sites supply bounded valid UTF-8. The fixed buffers accommodate
	// the complete rune bound, so captions cannot be split inside a code point.
	for i, b := range []byte(text) {
		if i >= len(dst)-1 {
			break
		}
		dst[i] = C.char(b)
	}
}

func projectLinuxOverlay(status OverlayStatus, options OverlayOptions, now time.Time) C.fh_xoverlay_frame {
	options = linuxOverlayOptions(options)
	view := resolveOverlayView(status)
	f := C.fh_xoverlay_frame{}
	if !view.Visible {
		return f
	}
	f.visible = 1
	f.layout, f.anchor, f.surface, f.visualizer = C.int(options.Layout), C.int(options.Anchor), C.int(options.Surface), C.int(options.Visualizer)
	f.scale, f.opacity, f.glow = C.double(options.Scale), C.double(options.Opacity), C.double(options.Glow)
	f.edge, f.accent, f.icon, f.stage = C.int(options.EdgeOffset), C.uint32_t(view.Accent), C.int(view.Icon), C.int(view.Stage)
	f.progress = 1
	if status.CountdownDuration > 0 && view.Stage == overlayStageCountdown {
		f.progress = C.double(clamp01(float64(status.CountdownDeadline.Sub(now)) / float64(status.CountdownDuration)))
	}
	phase, instruction := linuxOverlayPhase(status)
	linuxOverlayText(f.phase[:], phase)
	linuxOverlayText(f.instruction[:], instruction)
	linuxOverlayText(f.shortcut[:], linuxOverlayShortcut(status.Shortcut))
	if status.CaptionEnabled {
		linuxOverlayText(f.caption[:], BoundedOverlayCaption(status.Caption))
		f.opacity = C.double(max(options.Opacity, .9))
		f.surface = C.int(OverlaySurfaceSolid)
	}
	end := now
	if !status.FinishedAt.IsZero() && status.FinishedAt.Before(end) {
		end = status.FinishedAt
	}
	seconds := 0
	if !status.StartedAt.IsZero() {
		seconds = min(max(int(end.Sub(status.StartedAt)/time.Second), 0), 5999)
	}
	linuxOverlayText(f.elapsed[:], fmt.Sprintf("%d:%02d elapsed", seconds/60, seconds%60))
	linuxOverlayText(f.checkpoints[:], fmt.Sprintf("%d checkpoints", min(max(status.Checkpoints, 0), 999)))
	return f
}

func (o *StatusOverlay) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(o.done)
	p := C.fh_xoverlay_open()
	<-linuxOverlayOpenSlot
	if p == nil {
		ready <- errors.New("X11 overlay initialization failed; Shape and RandR are required.")
		return
	}
	defer C.fh_xoverlay_close(p)
	select {
	case <-o.stop:
		ready <- errors.New("native status overlay is closed")
		return
	default:
	}
	ready <- nil
	var timer *time.Timer
	var tick <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	var previous OverlayStatus
	var levels [overlayLevelBars]float64
	var sampled time.Time
	envelope := audio.NewEnvelope()
	for {
		select {
		case <-o.stop:
			return
		case <-o.wake:
		case <-tick:
		}
		o.mu.Lock()
		if o.closed {
			o.mu.Unlock()
			return
		}
		status, options := o.status, o.options
		now := time.Now()
		visible := resolveOverlayView(status).Visible
		reset := visible && (!resolveOverlayView(previous).Visible || status.Generation != previous.Generation || status.Preview != previous.Preview)
		if reset || !visible {
			levels = [overlayLevelBars]float64{}
			envelope.Reset()
			sampled = time.Time{}
		}
		f := projectLinuxOverlay(status, options, now)
		if reset {
			f.capture_anchor = 1
		}
		recording := status.Kind == OverlayRecording || status.Kind >= OverlayRecordingSpeech && status.Kind <= OverlayRecordingCountdown
		if o.source != nil && (sampled.IsZero() || now.Sub(sampled) >= overlayLevelMS*time.Millisecond) {
			sampled = now
			level := o.source.TakeLevel()
			if visible && recording {
				if math.IsNaN(level) || math.IsInf(level, 0) {
					level = 0
				}
				copy(levels[:], levels[1:])
				levels[len(levels)-1] = envelope.Push(audio.NormalizeLevel(clamp01(level)))
			}
		}
		for i, level := range levels {
			f.levels[i] = C.double(level)
		}
		// Reduced motion retains functional amplitude/countdown feedback, without
		// decorative travel. No synthetic waveform claims microphone activity.
		if options.Motion != OverlayMotionReduced {
			f.animation_ms = C.int(now.UnixMilli() % 60000)
		}
		o.mu.Unlock()
		if C.fh_xoverlay_draw(p, &f) == 0 {
			o.mu.Lock()
			o.failure = errors.New("X11 status overlay connection failed")
			o.status = OverlayStatus{}
			o.source = nil
			o.mu.Unlock()
			return
		}
		previous = status
		// Do not retain a hidden caption in the worker snapshot either.
		if !visible {
			previous = OverlayStatus{}
		}
		if timer != nil {
			timer.Stop()
			timer = nil
			tick = nil
		}
		if visible {
			timer = time.NewTimer(33 * time.Millisecond)
			tick = timer.C
		}
	}
}
