package platform

import (
	"math"
	"testing"
)

func TestNativeOverlayExtendedStyleIsNonActivatingAndHiddenFromShell(t *testing.T) {
	style := nativeOverlayExtendedStyle()
	for name, required := range map[string]uintptr{
		"WS_EX_NOACTIVATE":  wsExNoActivate,
		"WS_EX_TOOLWINDOW":  wsExToolWindow,
		"WS_EX_TOPMOST":     wsExTopmost,
		"WS_EX_TRANSPARENT": wsExTransparent,
		"WS_EX_LAYERED":     wsExLayered,
	} {
		if style&required == 0 {
			t.Fatalf("native overlay style is missing %s", name)
		}
	}
}

var overlayVisibleKinds = []OverlayKind{
	OverlayRecording, OverlayRecordingSpeech, OverlayRecordingSilence, OverlayRecordingCountdown,
	OverlayTranscribing, OverlayPostProcessing, OverlayReady,
	OverlayCopyRequired, OverlayFailed, OverlayCancelling,
}

func viewForKind(kind OverlayKind) overlayView {
	return resolveOverlayView(OverlayStatus{Kind: kind})
}

func TestResolveOverlayView(t *testing.T) {
	if view := viewForKind(OverlayHidden); view.Visible {
		t.Fatal("hidden coordinator state must hide the native overlay")
	}
	recording := viewForKind(OverlayRecording)
	if recording.Icon != overlayIconMicrophone || recording.Stage != overlayStageWaveform || !recording.Animated {
		t.Fatalf("recording state must show an animated microphone and waveform: %#v", recording)
	}
	transcribing := viewForKind(OverlayTranscribing)
	if transcribing.Icon != overlayIconSpinner || !transcribing.Animated {
		t.Fatalf("transcribing state must show an animated spinner: %#v", transcribing)
	}
	processing := viewForKind(OverlayPostProcessing)
	if processing.Icon != overlayIconSpinner || processing.Stage != overlayStageBreath || !processing.Animated {
		t.Fatalf("post-processing state must show a distinct animated spinner: %#v", processing)
	}
	if viewForKind(OverlayFailed).Animated {
		t.Fatal("failure state should remain visually steady")
	}
	if silence := viewForKind(OverlayRecordingSilence); silence.Icon != overlayIconMicrophone || silence.Stage != overlayStageFlatline || silence.Animated {
		t.Fatalf("detected silence must show a steady microphone and flatline: %#v", silence)
	}
	countdown := viewForKind(OverlayRecordingCountdown)
	if countdown.Icon != overlayIconTimer || countdown.Stage != overlayStageCountdown || !countdown.Animated {
		t.Fatalf("automatic stop countdown must show an animated timer: %#v", countdown)
	}
}

func TestReducedMotionPreservesOverlayMeaningWithoutDecorativeFrames(t *testing.T) {
	transcribing := viewForKind(OverlayTranscribing)
	reduced := overlayViewForMotion(transcribing, false)
	if reduced.Animated {
		t.Fatal("reduced motion left the transcribing animation enabled")
	}
	if reduced.Icon != transcribing.Icon || reduced.Stage != transcribing.Stage || reduced.Accent != transcribing.Accent {
		t.Fatalf("reduced motion changed semantic overlay identity: got %#v, want %#v", reduced, transcribing)
	}
	if overlayNeedsContinuousFrames(transcribing, false) {
		t.Fatal("decorative transcribing state kept a frame timer under reduced motion")
	}
	countdown := viewForKind(OverlayRecordingCountdown)
	if !overlayNeedsContinuousFrames(countdown, false) {
		t.Fatal("functional automatic-stop countdown stopped updating under reduced motion")
	}
}

// Glyph and stage identity must remain sufficient even though Detailed adds a
// fixed caption; the other layouts and assistive colour modes cannot rely on it.
func TestEveryVisibleOverlayStateIsDistinguishable(t *testing.T) {
	type identity struct {
		icon   overlayIcon
		stage  overlayStage
		accent uint32
	}
	seen := map[identity]OverlayKind{}
	for _, kind := range overlayVisibleKinds {
		view := viewForKind(kind)
		if !view.Visible {
			t.Fatalf("state %d should be visible", kind)
		}
		key := identity{icon: view.Icon, stage: view.Stage, accent: view.Accent}
		if other, ok := seen[key]; ok {
			t.Fatalf("states %d and %d share their entire visual identity", other, kind)
		}
		seen[key] = kind
	}
}

func TestEveryVisibleOverlayStateHasAReadablePalette(t *testing.T) {
	luminance := func(rgb uint32) float64 {
		return 0.2126*float64(rgb>>16&0xFF) + 0.7152*float64(rgb>>8&0xFF) + 0.0722*float64(rgb&0xFF)
	}
	for _, kind := range overlayVisibleKinds {
		view := viewForKind(kind)
		if view.Accent == 0 || view.AccentSoft == 0 {
			t.Fatalf("state %d is missing an accent pair", kind)
		}
		if luminance(view.AccentSoft) <= luminance(view.Accent) {
			t.Fatalf("state %d has an accent highlight darker than its accent", kind)
		}
		if luminance(view.Accent) <= luminance(view.Background)+40 {
			t.Fatalf("state %d does not separate its accent from its background", kind)
		}
		if view.Glow < 0 || view.Glow > 1 {
			t.Fatalf("state %d has an out-of-range glow: %v", kind, view.Glow)
		}
	}
}

// Bloom is the one thing most likely to be dialled up until it stops looking
// like glass, so the master scale is pinned to a restrained range.
func TestGlowStaysRestrained(t *testing.T) {
	if overlayGlowScale <= 0 || overlayGlowScale > 0.5 {
		t.Fatalf("overlayGlowScale = %v, outside a restrained 0..0.5", overlayGlowScale)
	}
	for _, kind := range overlayVisibleKinds {
		if effective := overlayGlowScale * viewForKind(kind).Glow; effective > 0.35 {
			t.Fatalf("state %d has an effective glow of %v", kind, effective)
		}
	}
}

func TestOverlayOptionsPreserveDefaultsAndClampBounds(t *testing.T) {
	defaults := DefaultOverlayOptions()
	if defaults.Layout != OverlayLayoutCapsule || defaults.Anchor != OverlayAnchorTopCenter ||
		defaults.Motion != OverlayMotionSystem || defaults.Surface != OverlaySurfaceGlass || defaults.Visualizer != OverlayVisualizerBars ||
		defaults.Scale != 1 || defaults.Opacity != 1 || defaults.EdgeOffset != 18 || defaults.Glow != 1 {
		t.Fatalf("default overlay options = %#v", defaults)
	}
	if got := normalizeOverlayOptions(OverlayOptions{Scale: 0.1, Opacity: 0.1, EdgeOffset: -5, Glow: -1}); got != (OverlayOptions{Scale: 0.75, Opacity: 0.4, EdgeOffset: 0, Glow: 0}) {
		t.Fatalf("lower-clamped overlay options = %#v", got)
	}
	if got := normalizeOverlayOptions(OverlayOptions{Scale: 4, Opacity: 4, EdgeOffset: 900, Glow: 4}); got != (OverlayOptions{Scale: 1.5, Opacity: 1, EdgeOffset: 240, Glow: 1}) {
		t.Fatalf("upper-clamped overlay options = %#v", got)
	}
}

func TestOverlayOptionsComposeWithDPIAlphaAndGlow(t *testing.T) {
	options := OverlayOptions{Scale: 1.25, Opacity: 0.6, EdgeOffset: 18, Glow: 0.5}
	if got := overlayScaleForDPI(144, options); math.Abs(got-1.875) > 1e-9 {
		t.Fatalf("overlay scale = %v, want 1.875", got)
	}
	if got := overlayAlpha(0.5, options); math.Abs(got-0.3) > 1e-9 {
		t.Fatalf("overlay alpha = %v, want 0.3", got)
	}
	if got := overlayGlowStrength(options); math.Abs(got-0.15) > 1e-9 {
		t.Fatalf("overlay glow = %v, want 0.15", got)
	}
}

func TestScaleForDPI(t *testing.T) {
	cases := []struct {
		value int32
		dpi   uint32
		want  int32
	}{{360, 96, 360}, {360, 144, 540}, {64, 192, 128}, {20, 0, 20}}
	for _, tc := range cases {
		if got := scaleForDPI(tc.value, tc.dpi); got != tc.want {
			t.Fatalf("scaleForDPI(%d, %d) = %d, want %d", tc.value, tc.dpi, got, tc.want)
		}
	}
	for dpi, want := range map[uint32]float64{0: 1, 96: 1, 144: 1.5, 192: 2} {
		if got := dpiScale(dpi); math.Abs(got-want) > 1e-9 {
			t.Fatalf("dpiScale(%d) = %v, want %v", dpi, got, want)
		}
	}
}

func TestEasingCurvesStayAnchoredAtTheirEndpoints(t *testing.T) {
	for name, ease := range map[string]func(float64) float64{
		"easeOutCubic":  easeOutCubic,
		"easeInOutSine": easeInOutSine,
		"easeOutBack":   easeOutBack,
	} {
		if got := ease(0); math.Abs(got) > 1e-9 {
			t.Fatalf("%s(0) = %v, want 0", name, got)
		}
		if got := ease(1); math.Abs(got-1) > 1e-9 {
			t.Fatalf("%s(1) = %v, want 1", name, got)
		}
		// Inputs outside 0..1 arrive whenever a transition is interrupted, so
		// every curve has to clamp rather than diverge.
		if got := ease(-3); math.Abs(got) > 1e-9 {
			t.Fatalf("%s(-3) = %v, want a clamp to 0", name, got)
		}
		if got := ease(7); math.Abs(got-1) > 1e-9 {
			t.Fatalf("%s(7) = %v, want a clamp to 1", name, got)
		}
	}
	// easeOutBack is the only curve allowed to leave the unit range, and only
	// by overshooting its target before settling.
	if peak := easeOutBack(0.55); peak <= 1 {
		t.Fatalf("easeOutBack did not overshoot: %v", peak)
	}
	if easeInOutSine(0.25) >= easeInOutSine(0.75) {
		t.Fatal("easeInOutSine is not monotonic")
	}
}

func TestOverlayStageMotionIsBoundedAndSmooth(t *testing.T) {
	const bars = 17
	previous := map[int]float64{}
	for elapsed := uint32(0); elapsed < 6000; elapsed += 16 {
		if breath := overlayBreath(elapsed, overlayBreathMS, 0); breath < 0 || breath > 1 {
			t.Fatalf("overlayBreath(%d) = %v, outside 0..1", elapsed, breath)
		}
		for index := 0; index < bars; index++ {
			level := overlayWaveLevel(elapsed, index, bars)
			if level < 0 || level > 1 {
				t.Fatalf("overlayWaveLevel(%d, %d) = %v, outside 0..1", elapsed, index, level)
			}
			// A frame-to-frame jump would read as a stutter rather than motion.
			if last, ok := previous[index]; ok && math.Abs(level-last) > 0.09 {
				t.Fatalf("overlayWaveLevel(%d, %d) jumped by %v", elapsed, index, math.Abs(level-last))
			}
			previous[index] = level
		}
		for index := 0; index < 13; index++ {
			for _, reverse := range []bool{false, true} {
				if got := overlayCometIntensity(elapsed, index, 13, reverse); got < 0 || got > 1 {
					t.Fatalf("overlayCometIntensity(%d, %d, %v) = %v, outside 0..1", elapsed, index, reverse, got)
				}
			}
		}
		if center := overlayShuttleCenter(elapsed); center < 0 || center > 1 {
			t.Fatalf("overlayShuttleCenter(%d) = %v, outside 0..1", elapsed, center)
		}
		if angle := overlaySpinnerAngle(elapsed); angle < 0 || angle >= 360 {
			t.Fatalf("overlaySpinnerAngle(%d) = %v, outside 0..360", elapsed, angle)
		}
		if sweep := overlaySpinnerSweep(elapsed); sweep < 38 || sweep > 260 {
			t.Fatalf("overlaySpinnerSweep(%d) = %v, outside 38..260", elapsed, sweep)
		}
	}
	if math.Abs(overlayBreath(0, 1000, 0)-overlayBreath(1000, 1000, 0)) > 1e-9 {
		t.Fatal("overlayBreath is not periodic")
	}
	// Neighbouring bars must not move in lockstep, or the row reads as one
	// block rather than a travelling wave.
	if math.Abs(overlayWaveLevel(500, 4, bars)-overlayWaveLevel(500, 5, bars)) < 1e-6 {
		t.Fatal("waveform bars are not phase-offset from one another")
	}
	// The centre envelope has to make the middle of the row taller on average.
	edge, middle := 0.0, 0.0
	for elapsed := uint32(0); elapsed < 4000; elapsed += 20 {
		edge += overlayWaveLevel(elapsed, 0, bars)
		middle += overlayWaveLevel(elapsed, bars/2, bars)
	}
	if middle <= edge {
		t.Fatalf("waveform envelope is not centre-weighted: edge %v, middle %v", edge, middle)
	}
	// Forward and reverse sweeps must be mirror images, which is what tells
	// cancelling apart from transcribing.
	if math.Abs(overlayCometIntensity(700, 1, 13, false)-overlayCometIntensity(700, 11, 13, true)) > 1e-9 {
		t.Fatal("reversed comet sweep is not the mirror of the forward sweep")
	}
}

func TestOverlayTransitionProgressIsClamped(t *testing.T) {
	if overlayEntrance(0) != 0 || overlayEntrance(overlayEnterMS) != 1 || overlayEntrance(99999) != 1 {
		t.Fatal("overlayEntrance did not span 0..1 across its duration")
	}
	if overlayExit(0) != 0 || overlayExit(overlayExitMS) != 1 || overlayExit(99999) != 1 {
		t.Fatal("overlayExit did not span 0..1 across its duration")
	}
}

func TestColourHelpers(t *testing.T) {
	if got := blendRGB(0x112233, 0xAABBCC, 0); got != 0x112233 {
		t.Fatalf("zero blend changed the base color: %#x", got)
	}
	if got := blendRGB(0x112233, 0xAABBCC, 255); got != 0xAABBCC {
		t.Fatalf("full blend did not produce the accent color: %#x", got)
	}
	if got := mixRGB(0x000000, 0xFFFFFF, 0.5); got != 0x808080 {
		t.Fatalf("mixRGB(0.5) = %#x, want 0x808080", got)
	}
	if got := shadeRGB(0x808080, 1); got != 0xFFFFFF {
		t.Fatalf("full lighten = %#x, want 0xFFFFFF", got)
	}
	if got := shadeRGB(0x808080, -1); got != 0x000000 {
		t.Fatalf("full darken = %#x, want 0x000000", got)
	}
	if got := argbColor(0x336699, 1); got != 0xFF336699 {
		t.Fatalf("argbColor at full alpha = %#x, want 0xFF336699", got)
	}
	// Out-of-range alpha reaches this helper whenever a glow is scaled by an
	// interrupted morph, so it has to saturate instead of wrapping the byte.
	if got := argbColor(0x336699, 4.2); got != 0xFF336699 {
		t.Fatalf("argbColor did not clamp a high alpha: %#x", got)
	}
	if got := argbColor(0x336699, -2); got != 0x00336699 {
		t.Fatalf("argbColor did not clamp a negative alpha: %#x", got)
	}
}
