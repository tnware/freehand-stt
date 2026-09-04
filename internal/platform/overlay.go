package platform

import (
	"math"
	"time"
)

// OverlayKind is presentation-only state. It never owns coordinator or
// insertion authority.
type OverlayKind uint8

const (
	OverlayHidden OverlayKind = iota
	OverlayRecording
	OverlayTranscribing
	OverlayPostProcessing
	OverlayReady
	OverlayCopyRequired
	OverlayFailed
	OverlayCancelling
	OverlayRecordingSpeech
	OverlayRecordingSilence
	OverlayRecordingCountdown
)

// OverlayStatus carries presentation state only. Countdown timing is supplied
// by the coordinator so the native overlay and renderer describe the same VAD
// deadline without either becoming a second recording authority.
type OverlayStatus struct {
	Kind              OverlayKind
	CountdownDeadline time.Time
	CountdownDuration time.Duration
	Generation        uint64
	Preview           bool
	StartedAt         time.Time
	FinishedAt        time.Time
	RecordingMode     OverlayRecordingMode
	Shortcut          string
	Checkpoints       int
}

type OverlayRecordingMode uint8

const (
	OverlayRecordingToggle OverlayRecordingMode = iota
	OverlayRecordingHold
)

type OverlayLayout uint8

const (
	OverlayLayoutMinimal OverlayLayout = iota
	OverlayLayoutCapsule
	OverlayLayoutMeter
	OverlayLayoutDetailed
)

type OverlayAnchor uint8

const (
	OverlayAnchorTopLeft OverlayAnchor = iota
	OverlayAnchorTopCenter
	OverlayAnchorTopRight
	OverlayAnchorBottomLeft
	OverlayAnchorBottomCenter
	OverlayAnchorBottomRight
)

type OverlayMotion uint8

const (
	OverlayMotionSystem OverlayMotion = iota
	OverlayMotionReduced
)

type OverlaySurface uint8

const (
	OverlaySurfaceGlass OverlaySurface = iota
	OverlaySurfaceSolid
	OverlaySurfaceMinimal
)

type OverlayVisualizer uint8

const (
	OverlayVisualizerBars OverlayVisualizer = iota
	OverlayVisualizerPulse
	OverlayVisualizerEnvelope
	OverlayVisualizerMeter
)

const (
	wsExTopmost     = uintptr(0x00000008)
	wsExTransparent = uintptr(0x00000020)
	wsExToolWindow  = uintptr(0x00000080)
	wsExLayered     = uintptr(0x00080000)
	wsExNoActivate  = uintptr(0x08000000)
)

// Every coordinator state has a glyph and distinct stage treatment. Detailed
// may add fixed product/phase copy plus the bounded operational fields carried
// by OverlayStatus; transcript and provider/user content never enter this type.
type overlayIcon uint8

const (
	overlayIconMicrophone overlayIcon = iota
	overlayIconSpinner
	overlayIconCheck
	overlayIconClipboard
	overlayIconWarning
	overlayIconCancel
	overlayIconTimer
)

type overlayStage uint8

const (
	overlayStageWaveform overlayStage = iota
	overlayStageComet
	overlayStageCometReverse
	overlayStageShuttle
	overlayStageBreath
	overlayStageFlatline
	overlayStageCountdown
)

// Overlay motion is driven by wall-clock milliseconds rather than a frame
// counter so that coalesced or dropped timer messages change smoothness only,
// never animation speed.
// overlayLevelBars is how many amplitude readings the recording meter shows,
// and overlayLevelMS how often one is taken. Together they are the length of
// history on screen: roughly two thirds of a second.
const (
	overlayLevelBars = 17
	overlayLevelMS   = 33
)

const (
	overlayEnterMS  = 340
	overlayExitMS   = 200
	overlayMorphMS  = 280
	overlayBreathMS = 1700
)

// overlayGlowScale is the single master dial for every accent bloom in the
// overlay. Per-state Glow weights are relative to it, so the whole surface can
// be made more or less luminous from one place.
const overlayGlowScale = 0.30

// OverlayOptions contains only the independent presentation dials that can be
// changed without altering the overlay's focus, monitor, state, or geometry
// ownership. Scale composes with Windows DPI; the other fractional values are
// in the inclusive range 0..1.
type OverlayOptions struct {
	Layout     OverlayLayout
	Anchor     OverlayAnchor
	Motion     OverlayMotion
	Surface    OverlaySurface
	Visualizer OverlayVisualizer
	Scale      float64
	Opacity    float64
	EdgeOffset int32
	Glow       float64
}

func DefaultOverlayOptions() OverlayOptions {
	return OverlayOptions{
		Layout: OverlayLayoutCapsule, Anchor: OverlayAnchorTopCenter,
		Motion: OverlayMotionSystem, Surface: OverlaySurfaceGlass, Visualizer: OverlayVisualizerBars,
		Scale: 1, Opacity: 1, EdgeOffset: 18, Glow: 1,
	}
}

func normalizeOverlayOptions(options OverlayOptions) OverlayOptions {
	defaults := DefaultOverlayOptions()
	if options.Layout > OverlayLayoutDetailed {
		options.Layout = defaults.Layout
	}
	if options.Anchor > OverlayAnchorBottomRight {
		options.Anchor = defaults.Anchor
	}
	if options.Motion > OverlayMotionReduced {
		options.Motion = defaults.Motion
	}
	if options.Surface > OverlaySurfaceMinimal {
		options.Surface = defaults.Surface
	}
	if options.Visualizer > OverlayVisualizerMeter {
		options.Visualizer = defaults.Visualizer
	}
	options.Scale = min(max(options.Scale, 0.75), 1.5)
	options.Opacity = min(max(options.Opacity, 0.4), 1)
	options.EdgeOffset = min(max(options.EdgeOffset, 0), 240)
	options.Glow = clamp01(options.Glow)
	return options
}

func overlayScaleForDPI(dpi uint32, options OverlayOptions) float64 {
	return dpiScale(dpi) * options.Scale
}

func overlayAlpha(animationAlpha float64, options OverlayOptions) float64 {
	return clamp01(animationAlpha * options.Opacity)
}

func overlayGlowStrength(options OverlayOptions) float64 {
	return overlayGlowScale * options.Glow
}

// overlayViewForMotion preserves the semantic glyph, stage, colour and
// countdown while removing decorative motion when Windows asks applications
// to reduce animation. Countdown progress remains live because it communicates
// when recording will stop rather than merely decorating the surface.
func overlayViewForMotion(view overlayView, animationsEnabled bool) overlayView {
	if !animationsEnabled {
		view.Animated = false
	}
	return view
}

func overlayNeedsContinuousFrames(view overlayView, animationsEnabled bool) bool {
	return view.Stage == overlayStageCountdown || (animationsEnabled && view.Animated)
}

type overlayView struct {
	Kind              OverlayKind
	Visible           bool
	Background        uint32
	Accent            uint32
	AccentSoft        uint32
	Icon              overlayIcon
	Stage             overlayStage
	Animated          bool
	CountdownDeadline time.Time
	CountdownDuration time.Duration
	CountdownProgress float64
	Generation        uint64
	Preview           bool
	StartedAt         time.Time
	FinishedAt        time.Time
	RecordingMode     OverlayRecordingMode
	Shortcut          string
	Checkpoints       int
	// Glow is the relative bloom weight for this state, 0..1.
	Glow float64
}

func nativeOverlayExtendedStyle() uintptr {
	return wsExTopmost | wsExTransparent | wsExToolWindow | wsExLayered | wsExNoActivate
}

// The overlay shares the window's palette rather than carrying one of its own.
// It floats over whatever the user is typing into, so it uses a single ground
// and a single accent hue; the status colours stay reserved for status, and the
// waveform gradients mix the accent toward overlayInk rather than toward a
// second hue.
const (
	overlayGround = 0x12161D
	overlayInk    = 0xF2F5F9
	overlayAccent = 0x3B82F6
	// The record colour, shared with the window's record control. Capture being
	// open is red; the accent is reserved for speech actually being heard, so
	// the two recording states stay distinguishable without a second hue.
	overlayRecord = 0xD13438
	overlayQuiet  = 0x7B8697
	overlayOK     = 0x22C55E
	overlayWarn   = 0xFBBF24
	overlayBad    = 0xF87171
)

func resolveOverlayView(status OverlayStatus) overlayView {
	view := overlayView{
		Kind:              status.Kind,
		Visible:           true,
		CountdownDeadline: status.CountdownDeadline,
		CountdownDuration: status.CountdownDuration,
		CountdownProgress: 1,
		Generation:        status.Generation,
		Preview:           status.Preview,
		StartedAt:         status.StartedAt,
		FinishedAt:        status.FinishedAt,
		RecordingMode:     status.RecordingMode,
		Shortcut:          status.Shortcut,
		Checkpoints:       status.Checkpoints,
	}
	switch status.Kind {
	case OverlayRecording:
		view.Background = overlayGround
		view.Accent = overlayRecord
		view.AccentSoft = overlayInk
		view.Icon = overlayIconMicrophone
		view.Stage = overlayStageWaveform
		view.Animated = true
		view.Glow = 1.0
	case OverlayRecordingSpeech:
		view.Background = overlayGround
		view.Accent = overlayAccent
		view.AccentSoft = overlayInk
		view.Icon = overlayIconMicrophone
		view.Stage = overlayStageWaveform
		view.Animated = true
		view.Glow = 0.95
	case OverlayRecordingSilence:
		view.Background = overlayGround
		view.Accent = overlayQuiet
		view.AccentSoft = overlayInk
		view.Icon = overlayIconMicrophone
		view.Stage = overlayStageFlatline
		view.Glow = 0.42
	case OverlayRecordingCountdown:
		view.Background = overlayGround
		view.Accent = overlayWarn
		view.AccentSoft = overlayInk
		view.Icon = overlayIconTimer
		view.Stage = overlayStageCountdown
		view.Animated = true
		view.Glow = 0.9
	case OverlayTranscribing:
		view.Background = overlayGround
		view.Accent = overlayAccent
		view.AccentSoft = overlayInk
		view.Icon = overlayIconSpinner
		view.Stage = overlayStageComet
		view.Animated = true
		view.Glow = 0.85
	case OverlayPostProcessing:
		view.Background = overlayGround
		view.Accent = overlayAccent
		view.AccentSoft = overlayInk
		view.Icon = overlayIconSpinner
		view.Stage = overlayStageBreath
		view.Animated = true
		view.Glow = 0.8
	case OverlayReady:
		view.Background = overlayGround
		view.Accent = overlayOK
		view.AccentSoft = overlayInk
		view.Icon = overlayIconCheck
		view.Stage = overlayStageShuttle
		view.Animated = true
		view.Glow = 0.8
	case OverlayCopyRequired:
		view.Background = overlayGround
		view.Accent = overlayAccent
		view.AccentSoft = overlayInk
		view.Icon = overlayIconClipboard
		view.Stage = overlayStageBreath
		view.Animated = true
		view.Glow = 0.9
	case OverlayFailed:
		view.Background = overlayGround
		view.Accent = overlayBad
		view.AccentSoft = overlayInk
		view.Icon = overlayIconWarning
		view.Stage = overlayStageFlatline
		view.Glow = 0.55
	case OverlayCancelling:
		view.Background = overlayGround
		view.Accent = overlayQuiet
		view.AccentSoft = overlayInk
		view.Icon = overlayIconCancel
		view.Stage = overlayStageCometReverse
		view.Animated = true
		view.Glow = 0.7
	default:
		return overlayView{}
	}
	return view
}

func scaleForDPI(value int32, dpi uint32) int32 {
	if dpi == 0 {
		dpi = 96
	}
	return int32((int64(value)*int64(dpi) + 48) / 96)
}

// dpiScale is the fractional companion to scaleForDPI. Geometry is laid out in
// float device pixels so that antialiased edges land where the design says,
// instead of being rounded to whole logical units first.
func dpiScale(dpi uint32) float64 {
	if dpi == 0 {
		return 1
	}
	return float64(dpi) / 96
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func lerp(from, to, amount float64) float64 {
	return from + (to-from)*amount
}

// easeOutCubic decelerates into its target and is used for entrances.
func easeOutCubic(t float64) float64 {
	t = clamp01(t)
	inverse := 1 - t
	return 1 - inverse*inverse*inverse
}

// easeInOutSine has no velocity discontinuity at its endpoints, which keeps
// colour and width morphs from visibly snapping.
func easeInOutSine(t float64) float64 {
	return 0.5 - 0.5*math.Cos(math.Pi*clamp01(t))
}

// easeOutBack overshoots slightly before settling, which reads as a physical
// pop rather than a fade.
func easeOutBack(t float64) float64 {
	t = clamp01(t)
	const overshoot = 1.34
	inverse := t - 1
	return 1 + (overshoot+1)*inverse*inverse*inverse + overshoot*inverse*inverse
}

// overlayBreath is a seamless 0..1 sinusoid. phase is expressed in turns.
func overlayBreath(elapsedMS uint32, periodMS uint32, phase float64) float64 {
	if periodMS == 0 {
		return 0
	}
	turns := float64(elapsedMS)/float64(periodMS) + phase
	return 0.5 - 0.5*math.Cos(2*math.Pi*turns)
}

// overlayWaveLevel returns a 0..1 height for one bar of the recording stage.
// Three incommensurable sine terms travel across the row so the bars never
// march in step, and a centre-weighted envelope makes the group read as a
// waveform rather than a bar chart. It is paint-only motion and does not
// represent microphone amplitude.
func overlayWaveLevel(elapsedMS uint32, index, count int) float64 {
	if count <= 0 {
		return 0
	}
	seconds := float64(elapsedMS) / 1000
	position := (float64(index) + 0.5) / float64(count)
	phase := position * 2.6
	primary := math.Sin(2 * math.Pi * (seconds*1.35 - phase))
	detail := math.Sin(2*math.Pi*(seconds*2.31-phase*1.7) + 0.7)
	drift := math.Sin(2*math.Pi*(seconds*0.67-phase*0.5) + 1.9)
	level := 0.5 + 0.5*(primary*0.46+detail*0.3+drift*0.24)
	envelope := 0.42 + 0.58*math.Sin(math.Pi*position)
	return clamp01(level * envelope)
}

// overlayCometIntensity returns a 0..1 brightness for one element of a sweeping
// stage: a tight leading edge at the head and a longer decaying trail behind
// it. Reversing the sweep is what separates cancelling from transcribing.
func overlayCometIntensity(elapsedMS uint32, index, count int, reverse bool) float64 {
	if count <= 0 {
		return 0
	}
	const periodMS = 1500
	position := (float64(index) + 0.5) / float64(count)
	if reverse {
		position = 1 - position
	}
	// The head starts before the row and finishes past it, so the sweep enters
	// and leaves cleanly instead of appearing mid-row.
	head := math.Mod(float64(elapsedMS)/periodMS, 1)*1.45 - 0.3
	distance := head - position
	if distance < 0 {
		leading := distance / 0.055
		return clamp01(math.Exp(-leading * leading))
	}
	return clamp01(math.Exp(-distance / 0.24))
}

// overlayShuttleCenter is the 0..1 centre of the indeterminate progress segment.
func overlayShuttleCenter(elapsedMS uint32) float64 {
	return overlayBreath(elapsedMS, 1600, 0)
}

// overlaySpinnerAngle and overlaySpinnerSweep describe an arc whose tail
// stretches and contracts as it rotates, the way a determinate-less progress
// ring should.
func overlaySpinnerAngle(elapsedMS uint32) float64 {
	turns := float64(elapsedMS)/1400 + overlayBreath(elapsedMS, 2100, 0)*0.22
	return math.Mod(turns*360, 360)
}

func overlaySpinnerSweep(elapsedMS uint32) float64 {
	return lerp(38, 260, overlayBreath(elapsedMS, 2100, 0))
}

// overlayEntrance and overlayExit return 0..1 progress for the show and hide
// transitions.
func overlayEntrance(elapsedMS uint32) float64 {
	return clamp01(float64(elapsedMS) / overlayEnterMS)
}

func overlayExit(elapsedMS uint32) float64 {
	return clamp01(float64(elapsedMS) / overlayExitMS)
}

func blendRGB(base, accent uint32, amount uint8) uint32 {
	mix := func(a, b uint32) uint32 {
		return (a*uint32(255-amount) + b*uint32(amount) + 127) / 255
	}
	return mix(base>>16&0xFF, accent>>16&0xFF)<<16 |
		mix(base>>8&0xFF, accent>>8&0xFF)<<8 |
		mix(base&0xFF, accent&0xFF)
}

// mixRGB is the fractional form of blendRGB, for animated colour morphs.
func mixRGB(base, accent uint32, amount float64) uint32 {
	return blendRGB(base, accent, uint8(math.Round(clamp01(amount)*255)))
}

// shadeRGB lightens for amount > 0 and darkens for amount < 0, staying inside
// the 0..255 channel range at every step.
func shadeRGB(rgb uint32, amount float64) uint32 {
	if amount >= 0 {
		return mixRGB(rgb, 0xFFFFFF, amount)
	}
	return mixRGB(rgb, 0x000000, -amount)
}

// argbColor packs an 0xRRGGBB value and a 0..1 alpha into the 0xAARRGGBB word
// GDI+ expects.
func argbColor(rgb uint32, alpha float64) uint32 {
	return uint32(math.Round(clamp01(alpha)*255))<<24 | rgb&0xFFFFFF
}
