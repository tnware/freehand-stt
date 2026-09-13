//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices
#include "overlay_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/audio"
)

var darwinOverlays sync.Map
var darwinOverlayNextID atomic.Uint64

// StatusOverlay is lazy: construction/configuration creates neither a panel,
// goroutine nor timer. The application owns the Cocoa main run loop. Each
// overlay has one latest-value mailbox, at most one queued main-thread wake,
// and at most one timer (only while visible).
type StatusOverlay struct {
	mu               sync.Mutex
	native           unsafe.Pointer
	id               uint64
	closed           bool
	status           OverlayStatus
	options          OverlayOptions
	source           LevelSource
	placement        darwinOverlayPlacement
	display          uint32
	shownAt, sampled time.Time
	levels           [overlayLevelBars]float64
	envelope         audio.Envelope
}

func NewStatusOverlay() (*StatusOverlay, error) {
	return &StatusOverlay{options: DefaultOverlayOptions(), envelope: audio.NewEnvelope()}, nil
}

func (o *StatusOverlay) Configure(options OverlayOptions) error {
	if o == nil {
		return errors.New("native status overlay is closed")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return errors.New("native status overlay is closed")
	}
	o.options = darwinOverlayOptions(options)
	if o.native != nil {
		C.fh_overlay_wake(o.native)
	}
	return nil
}

func (o *StatusOverlay) Update(status OverlayStatus) error {
	if o == nil {
		return errors.New("native status overlay is closed")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return errors.New("native status overlay is closed")
	}
	visible := resolveOverlayView(status).Visible
	if visible && (!resolveOverlayView(o.status).Visible || status.Generation != o.status.Generation || status.Preview != o.status.Preview) {
		// Read window geometry now, not when a delayed Cocoa paint arrives. This is
		// metadata-only and does not request Screen Recording or Accessibility.
		o.display = uint32(C.fh_overlay_capture_display())
		o.shownAt = time.Now()
		o.placement.valid = false
		o.levels = [overlayLevelBars]float64{}
		o.envelope.Reset()
		o.sampled = time.Time{}
		if o.source != nil {
			o.source.TakeLevel()
		}
	}
	status.Shortcut = darwinOverlayShortcut(status.Shortcut)
	status.Caption = BoundedOverlayCaption(status.Caption)
	if !status.CaptionEnabled || !visible {
		status.Caption = ""
	}
	o.status = status
	if !visible {
		o.placement.valid = false
	}
	if o.native == nil && visible {
		o.id = darwinOverlayNextID.Add(1)
		o.native = C.fh_overlay_create(C.uint64_t(o.id))
		if o.native == nil {
			return errors.New("native status overlay allocation failed")
		}
		darwinOverlays.Store(o.id, o)
	}
	if o.native != nil {
		C.fh_overlay_wake(o.native)
	}
	return nil
}

func (o *StatusOverlay) SetLevelSource(source LevelSource) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed {
		o.source = source
		if o.native != nil {
			C.fh_overlay_wake(o.native)
		}
	}
}

// Close rejects new work immediately and erases transient Go text. Native
// cleanup runs on the Cocoa main thread; a background caller waits at most
// 500ms. If the host run loop is stopped, return an error rather than claiming
// destruction. At most one cleanup block remains, and it cannot call back into
// this object. Never wait for the main queue from the main thread itself.
func (o *StatusOverlay) Close() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	o.closed = true
	native := o.native
	o.native = nil
	o.status = OverlayStatus{}
	o.source = nil
	o.levels = [overlayLevelBars]float64{}
	darwinOverlays.Delete(o.id)
	o.mu.Unlock()
	if native != nil && C.fh_overlay_close(native) == 0 {
		return errors.New("native status overlay cleanup awaits Cocoa main run loop")
	}
	return nil
}

type darwinOverlayRect struct{ X, Y, W, H float64 }
type darwinOverlayPlacement struct {
	valid      bool
	generation uint64
	preview    bool
	work       darwinOverlayRect
}

func (p *darwinOverlayPlacement) resolve(status OverlayStatus, candidate darwinOverlayRect) darwinOverlayRect {
	if !resolveOverlayView(status).Visible {
		p.valid = false
		return p.work
	}
	if !p.valid || p.generation != status.Generation || p.preview != status.Preview {
		p.valid = true
		p.generation = status.Generation
		p.preview = status.Preview
		p.work = candidate
	}
	return p.work
}

// Cocoa coordinates are bottom-left-origin logical points, including negative
// origins. AppKit handles the backing scale; do not multiply Retina DPI again.
func darwinOverlayDestination(work darwinOverlayRect, width, height float64, options OverlayOptions) darwinOverlayRect {
	width = min(width, work.W)
	height = min(height, work.H)
	edge := float64(options.EdgeOffset)
	x := work.X + (work.W-width)/2
	switch options.Anchor {
	case OverlayAnchorTopLeft, OverlayAnchorBottomLeft:
		x = work.X + edge
	case OverlayAnchorTopRight, OverlayAnchorBottomRight:
		x = work.X + work.W - width - edge
	}
	y := work.Y + edge
	if options.Anchor <= OverlayAnchorTopRight {
		y = work.Y + work.H - height - edge
	}
	return darwinOverlayRect{X: min(max(x, work.X), work.X+work.W-width), Y: min(max(y, work.Y), work.Y+work.H-height), W: width, H: height}
}

type darwinOverlayFrame struct {
	Visible                                                     bool
	Options                                                     OverlayOptions
	View                                                        overlayView
	Caption, Shortcut, Phase, Instruction, Elapsed, Checkpoints string
	Progress                                                    float64
	TimerMS                                                     int
	CheckpointCount                                             int
	Levels                                                      [overlayLevelBars]float64
	AnimationMS                                                 uint32
}

func darwinOverlayOptions(options OverlayOptions) OverlayOptions {
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

func darwinOverlayShortcut(text string) string {
	runes := []rune(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, text))
	return string(runes[:min(len(runes), 48)])
}

func projectDarwinOverlay(status OverlayStatus, options OverlayOptions, now time.Time, reducedMotion, highContrast bool) darwinOverlayFrame {
	options = darwinOverlayOptions(options)
	view := resolveOverlayView(status)
	frame := darwinOverlayFrame{Visible: view.Visible, Options: options, View: view, Progress: 1}
	if !view.Visible {
		frame.View = overlayView{}
		return frame
	}
	if status.CaptionEnabled {
		frame.Caption = BoundedOverlayCaption(status.Caption)
		frame.Options.Surface = OverlaySurfaceSolid
		frame.Options.Opacity = max(frame.Options.Opacity, .9)
	}
	// Do not retain a disabled caption in the view snapshot either.
	frame.View.Caption = frame.Caption
	frame.Shortcut = darwinOverlayShortcut(status.Shortcut)
	frame.View.Shortcut = frame.Shortcut
	end := now
	if !status.FinishedAt.IsZero() && status.FinishedAt.Before(end) {
		end = status.FinishedAt
	}
	seconds := 0
	if !status.StartedAt.IsZero() {
		seconds = min(max(int(end.Sub(status.StartedAt)/time.Second), 0), 5999)
	}
	frame.Elapsed = fmt.Sprintf("%d:%02d elapsed", seconds/60, seconds%60)
	count := min(max(status.Checkpoints, 0), 999)
	frame.CheckpointCount = min(count, 5)
	frame.Checkpoints = fmt.Sprintf("%d checkpoints", count)
	if count == 1 {
		frame.Checkpoints = "1 checkpoint"
	}
	if view.Stage == overlayStageCountdown && status.CountdownDuration > 0 {
		frame.Progress = clamp01(float64(status.CountdownDeadline.Sub(now)) / float64(status.CountdownDuration))
	}
	frame.View = overlayViewForMotion(frame.View, !reducedMotion && options.Motion != OverlayMotionReduced)
	if highContrast {
		frame.Options.Opacity = 1
		frame.Options.Glow = 0
		frame.Options.Surface = OverlaySurfaceSolid
	}
	if overlayNeedsContinuousFrames(frame.View, frame.View.Animated) {
		frame.TimerMS = 33
	}
	if options.Layout == OverlayLayoutDetailed && !status.StartedAt.IsZero() && status.FinishedAt.IsZero() && frame.TimerMS == 0 {
		frame.TimerMS = 1000
	}
	frame.Phase, frame.Instruction = darwinOverlayPhase(view)
	return frame
}

// snapshot is called under mu by the sole native timer/wake callback. LevelSource
// is the existing nonblocking scalar peak tap, never microphone/audio work.
func (o *StatusOverlay) snapshot(now time.Time, reducedMotion, highContrast bool) darwinOverlayFrame {
	frame := projectDarwinOverlay(o.status, o.options, now, reducedMotion, highContrast)
	recording := o.status.Kind == OverlayRecording || o.status.Kind >= OverlayRecordingSpeech && o.status.Kind <= OverlayRecordingCountdown
	if !frame.Visible || !recording {
		o.levels = [overlayLevelBars]float64{}
		o.envelope.Reset()
		o.sampled = time.Time{}
		if o.source != nil {
			o.source.TakeLevel()
		}
	} else if o.source != nil {
		if o.sampled.IsZero() || now.Sub(o.sampled) >= overlayLevelMS*time.Millisecond {
			level := o.source.TakeLevel()
			if math.IsNaN(level) || math.IsInf(level, 0) {
				level = 0
			}
			copy(o.levels[:], o.levels[1:])
			o.levels[overlayLevelBars-1] = o.envelope.Push(audio.NormalizeLevel(clamp01(level)))
			o.sampled = now
		}
		frame.Levels = o.levels
		// Audio amplitude is functional information, not decorative animation.
		frame.TimerMS = overlayLevelMS
	}
	if frame.View.Animated && !o.shownAt.IsZero() {
		frame.AnimationMS = uint32(max(now.Sub(o.shownAt).Milliseconds(), 0))
	}
	if frame.Visible && recording && o.source == nil {
		for i := range frame.Levels {
			frame.Levels[i] = .08
			if frame.View.Animated {
				frame.Levels[i] = overlayWaveLevel(frame.AnimationMS, i, overlayLevelBars)
			}
		}
	}
	return frame
}

func darwinOverlayPhase(view overlayView) (string, string) {
	instruction := "Use the shortcut again to finish"
	if view.RecordingMode == OverlayRecordingHold {
		instruction = "Release the shortcut to finish"
	}
	switch view.Kind {
	case OverlayRecordingSpeech, OverlayRecording:
		return "Listening", instruction
	case OverlayRecordingSilence:
		return "Silence detected", instruction
	case OverlayRecordingCountdown:
		return "Silence countdown", "Speak to keep recording"
	case OverlayTranscribing:
		return "Transcribing", "Turning speech into text"
	case OverlayPostProcessing:
		return "Cleaning up", "Applying your processing profile"
	case OverlayReady:
		return "Ready", "Transcript delivered"
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

func darwinOverlayCopy(dst []C.char, text string) {
	for i := range dst {
		dst[i] = 0
	}
	for i, b := range []byte(text) {
		if i >= len(dst)-1 {
			break
		}
		dst[i] = C.char(b)
	}
}

//export fh_overlay_snapshot
func fh_overlay_snapshot(id C.uint64_t, reduced C.int, contrast C.int, out *C.fh_overlay_frame) {
	*out = C.fh_overlay_frame{}
	value, ok := darwinOverlays.Load(uint64(id))
	if !ok {
		return
	}
	o := value.(*StatusOverlay)
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	frame := o.snapshot(time.Now(), reduced != 0, contrast != 0)
	if !frame.Visible {
		return
	}
	var candidate C.fh_overlay_rect
	C.fh_overlay_work_area(C.uint32_t(o.display), &candidate)
	work := o.placement.resolve(o.status, darwinOverlayRect{X: float64(candidate.x), Y: float64(candidate.y), W: float64(candidate.w), H: float64(candidate.h)})
	width, height := 208.0, 52.0
	switch frame.Options.Layout {
	case OverlayLayoutMinimal:
		width, height = 56, 56
	case OverlayLayoutMeter:
		width, height = 304, 62
	case OverlayLayoutDetailed:
		width, height = 382, 154
	}
	if frame.View.CaptionEnabled {
		width, height = 640, 52
	}
	rect := darwinOverlayDestination(work, width*frame.Options.Scale, height*frame.Options.Scale, frame.Options)
	out.x = C.double(rect.X)
	out.y = C.double(rect.Y)
	out.width = C.double(rect.W)
	out.height = C.double(rect.H)
	out.visible = 1
	out.layout = C.int(frame.Options.Layout)
	out.surface = C.int(frame.Options.Surface)
	out.visualizer = C.int(frame.Options.Visualizer)
	out.scale = C.double(frame.Options.Scale)
	out.opacity = C.double(frame.Options.Opacity)
	out.glow = C.double(overlayGlowStrength(frame.Options) * frame.View.Glow)
	out.accent = C.uint32_t(frame.View.Accent)
	out.background = C.uint32_t(frame.View.Background)
	out.icon = C.int(frame.View.Icon)
	out.stage = C.int(frame.View.Stage)
	out.progress = C.double(frame.Progress)
	out.timer_ms = C.int(frame.TimerMS)
	out.animation_ms = C.uint32_t(frame.AnimationMS)
	out.checkpoint_count = C.int(frame.CheckpointCount)
	if frame.View.Animated {
		out.animated = 1
	}
	if frame.View.CaptionEnabled {
		out.captions = 1
	}
	for i, v := range frame.Levels {
		out.levels[i] = C.double(v)
	}
	darwinOverlayCopy(out.caption[:], frame.Caption)
	darwinOverlayCopy(out.shortcut[:], frame.Shortcut)
	darwinOverlayCopy(out.phase[:], frame.Phase)
	darwinOverlayCopy(out.instruction[:], frame.Instruction)
	darwinOverlayCopy(out.elapsed[:], frame.Elapsed)
	darwinOverlayCopy(out.checkpoints[:], frame.Checkpoints)
}
