//go:build windows

package overlay

import (
	"fmt"
	"strings"
	"time"
)

func drawOverlay(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, levels []float64, options OverlayOptions, highContrast bool, font func(float64, int) uintptr) {
	if highContrast {
		view.Background = systemColorRGB(colorWindow)
		view.Accent = systemColorRGB(colorHighlight)
		view.AccentSoft = systemColorRGB(colorWindowText)
		options.Surface = OverlaySurfaceSolid
		options.Glow = 0
	}
	breath := 0.85
	if view.Animated {
		breath = lerp(0.7, 1.0, overlayBreath(elapsed, overlayBreathMS, 0))
	}
	glowScale := overlayGlowStrength(options)
	glow := glowScale * view.Glow * breath

	drawOverlayBody(graphics, view, geometry, breath, glow, options.Surface)

	switch geometry.layout {
	case overlayLayoutCaptions:
		drawCaptionOverlay(graphics, view, geometry, font)
	case OverlayLayoutMinimal:
		drawMinimalOverlay(graphics, view, geometry, elapsed, levels, glowScale)
	case OverlayLayoutDetailed:
		drawDetailedOverlay(graphics, view, geometry, elapsed, levels, glowScale, options.Visualizer, font)
	default:
		drawOverlayIcon(graphics, view, geometry, elapsed, glow)
		drawOverlayDivider(graphics, view, geometry)
		drawOverlayStage(graphics, view, geometry, elapsed, breath, levels, glowScale, options.Visualizer)
		if geometry.layout == OverlayLayoutMeter {
			drawMeterIndicators(graphics, view, geometry)
		}
	}
}

func drawOverlayBody(graphics uintptr, view overlayView, geometry overlayGeometry, breath, glow float64, surface OverlaySurface) {
	scale := geometry.scale
	capsule := gpCapsulePath(geometry.pillX, geometry.pillY, geometry.pillWidth, geometry.pillHeight, geometry.radius)
	if capsule == 0 {
		return
	}
	defer gpDeletePath(capsule)

	if surface != OverlaySurfaceMinimal {
		gpTranslated(graphics, 0, 6*scale, func() {
			gpFillPathColor(graphics, capsule, argbColor(0x000000, 0.32))
			gpGlow(graphics, capsule, 0x000000, 20*scale, 8, 0.30)
		})
	}
	gpGlow(graphics, capsule, view.Accent, 13*scale, 5, 0.46*glow)

	if surface == OverlaySurfaceGlass {
		top := mixRGB(shadeRGB(view.Background, 0.13), view.Accent, 0.09)
		bottom := shadeRGB(view.Background, -0.30)
		bodyBrush := gpGradientBrush(
			0, int32(geometry.pillY)-1, 0, int32(geometry.pillY+geometry.pillHeight)+1,
			[]uint32{argbColor(top, 0.985), argbColor(mixRGB(top, bottom, 0.55), 0.975), argbColor(bottom, 0.962)},
			[]float32{0, 0.55, 1},
		)
		gpFillPath(graphics, bodyBrush, capsule)
		gpDeleteBrush(bodyBrush)
	} else {
		alpha := 0.985
		if surface == OverlaySurfaceMinimal {
			alpha = 0.88
		}
		gpFillPathColor(graphics, capsule, argbColor(view.Background, alpha))
	}

	if surface != OverlaySurfaceMinimal {
		washBrush := gpGradientBrush(
			int32(geometry.pillX)-1, 0, int32(geometry.pillX+geometry.pillWidth)+1, 0,
			[]uint32{argbColor(view.Accent, 0.11*breath), argbColor(view.Accent, 0.025), argbColor(view.Accent, 0)},
			[]float32{0, 0.4, 1},
		)
		if washBrush != 0 {
			gpClipped(graphics, capsule, func() { gpFillPath(graphics, washBrush, capsule) })
			gpDeleteBrush(washBrush)
		}
	}

	if surface != OverlaySurfaceGlass {
		borderAlpha := 0.24
		if surface == OverlaySurfaceSolid {
			borderAlpha = 0.42
		}
		gpStrokePathColor(graphics, capsule, argbColor(view.AccentSoft, borderAlpha), 1.2*scale)
		return
	}

	// Preserve the original Capsule glass edge exactly as the compatibility
	// default while the other surfaces deliberately flatten it.
	borderBrush := gpGradientBrush(
		0, int32(geometry.pillY)-1, 0, int32(geometry.pillY+geometry.pillHeight)+1,
		[]uint32{argbColor(view.AccentSoft, 0.52), argbColor(view.Accent, 0.30), argbColor(view.Accent, 0.18)},
		[]float32{0, 0.5, 1},
	)
	if borderBrush != 0 {
		pen := gpBrushPen(borderBrush, 1.25*scale)
		gpStrokePath(graphics, pen, capsule)
		gpDeletePen(pen)
		gpDeleteBrush(borderBrush)
	}

	inset := 2.0 * scale
	rim := gpCapsulePath(geometry.pillX+inset, geometry.pillY+inset,
		geometry.pillWidth-2*inset, geometry.pillHeight-2*inset, geometry.radius-inset)
	if rim == 0 {
		return
	}
	defer gpDeletePath(rim)
	rimBrush := gpGradientBrush(
		0, int32(geometry.pillY)-1, 0, int32(geometry.pillY+geometry.pillHeight)+1,
		[]uint32{argbColor(0xFFFFFF, 0.26), argbColor(0xFFFFFF, 0.03), argbColor(0xFFFFFF, 0.08)},
		[]float32{0, 0.5, 1},
	)
	if rimBrush != 0 {
		pen := gpBrushPen(rimBrush, 1.0*scale)
		gpStrokePath(graphics, pen, rim)
		gpDeletePen(pen)
		gpDeleteBrush(rimBrush)
	}
}

func systemColorRGB(index int) uint32 {
	value, _, _ := getSysColor.Call(uintptr(index))
	return uint32(value&0xFF)<<16 | uint32(value&0xFF00) | uint32(value>>16&0xFF)
}

func drawOverlayDivider(graphics uintptr, view overlayView, geometry overlayGeometry) {
	scale := geometry.scale
	x := geometry.stageX - 14*scale
	half := 10.0 * scale
	divider := gpCapsulePath(x, geometry.centerY-half, 1.2*scale, half*2, 0.6*scale)
	if divider == 0 {
		return
	}
	gpFillPathColor(graphics, divider, argbColor(view.AccentSoft, 0.16))
	gpDeletePath(divider)
}

// drawOverlayIcon paints a faint halo and the phase glyph. The glyphs are drawn
// as vector paths rather than font characters so their stroke weight stays
// exact at any DPI and the overlay carries no font dependency.
func drawOverlayIcon(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, glow float64) {
	scale := geometry.scale
	halo := 15.0 * scale
	if haloPath := gpEllipsePath(geometry.iconX-halo, geometry.centerY-halo, halo*2, halo*2); haloPath != 0 {
		haloBrush := gpRadialBrush(haloPath, argbColor(view.Accent, 0.55*glow), argbColor(view.Accent, 0), 0.2)
		gpFillPath(graphics, haloBrush, haloPath)
		gpDeleteBrush(haloBrush)
		gpDeletePath(haloPath)
	}

	unit := overlayIconBox * scale / 20 // glyphs are authored on a 20-unit grid
	stroke := 1.9 * unit
	x, y := geometry.iconX, geometry.centerY

	switch view.Icon {
	case overlayIconSpinner:
		radius := 8.0 * unit
		if track := gpEllipsePath(x-radius, y-radius, radius*2, radius*2); track != 0 {
			gpStrokePathColor(graphics, track, argbColor(view.Accent, 0.22), stroke)
			gpDeletePath(track)
		}
		angle, sweep := -90.0, 270.0
		if view.Animated {
			angle, sweep = overlaySpinnerAngle(elapsed), overlaySpinnerSweep(elapsed)
		}
		if pen := gpPen(argbColor(view.AccentSoft, 1), stroke); pen != 0 {
			gpDrawArc(graphics, pen, x-radius, y-radius, radius*2, radius*2, angle, sweep)
			gpDeletePen(pen)
		}
	case overlayIconCheck:
		check := gpPolylinePath([][2]float64{
			{x - 6.2*unit, y + 0.2*unit},
			{x - 1.8*unit, y + 4.6*unit},
			{x + 6.4*unit, y - 4.8*unit},
		})
		if check != 0 {
			gpStrokePathColor(graphics, check, argbColor(view.AccentSoft, 1), 2.4*unit)
			gpDeletePath(check)
		}
	case overlayIconClipboard:
		back := gpCapsulePath(x-6.8*unit, y-7.6*unit, 9.6*unit, 12.4*unit, 2*unit)
		if back != 0 {
			gpStrokePathColor(graphics, back, argbColor(view.Accent, 0.55), stroke*0.85)
			gpDeletePath(back)
		}
		front := gpCapsulePath(x-2.8*unit, y-4.2*unit, 9.6*unit, 12.4*unit, 2*unit)
		if front != 0 {
			gpFillPathColor(graphics, front, argbColor(shadeRGB(view.Background, -0.05), 0.96))
			gpStrokePathColor(graphics, front, argbColor(view.AccentSoft, 1), stroke*0.85)
			gpDeletePath(front)
		}
	case overlayIconWarning:
		radius := 8.2 * unit
		if ring := gpEllipsePath(x-radius, y-radius, radius*2, radius*2); ring != 0 {
			gpStrokePathColor(graphics, ring, argbColor(view.Accent, 0.85), stroke*0.9)
			gpDeletePath(ring)
		}
		if stem := gpPolylinePath([][2]float64{{x, y - 4.4*unit}, {x, y + 1.3*unit}}); stem != 0 {
			gpStrokePathColor(graphics, stem, argbColor(view.AccentSoft, 1), 2.1*unit)
			gpDeletePath(stem)
		}
		if dot := gpEllipsePath(x-1.15*unit, y+3.3*unit, 2.3*unit, 2.3*unit); dot != 0 {
			gpFillPathColor(graphics, dot, argbColor(view.AccentSoft, 1))
			gpDeletePath(dot)
		}
	case overlayIconCancel:
		arm := 5.6 * unit
		if first := gpPolylinePath([][2]float64{{x - arm, y - arm}, {x + arm, y + arm}}); first != 0 {
			gpStrokePathColor(graphics, first, argbColor(view.AccentSoft, 1), 2.4*unit)
			gpDeletePath(first)
		}
		if second := gpPolylinePath([][2]float64{{x + arm, y - arm}, {x - arm, y + arm}}); second != 0 {
			gpStrokePathColor(graphics, second, argbColor(view.AccentSoft, 1), 2.4*unit)
			gpDeletePath(second)
		}
	case overlayIconTimer:
		radius := 7.3 * unit
		if ring := gpEllipsePath(x-radius, y-radius+1.2*unit, radius*2, radius*2); ring != 0 {
			gpStrokePathColor(graphics, ring, argbColor(view.AccentSoft, 1), stroke)
			gpDeletePath(ring)
		}
		if crown := gpPolylinePath([][2]float64{{x - 2.4*unit, y - 8.6*unit}, {x + 2.4*unit, y - 8.6*unit}}); crown != 0 {
			gpStrokePathColor(graphics, crown, argbColor(view.AccentSoft, 0.9), stroke)
			gpDeletePath(crown)
		}
		if hand := gpPolylinePath([][2]float64{{x, y + 1.2*unit}, {x, y - 3.4*unit}, {x + 3.5*unit, y - 0.8*unit}}); hand != 0 {
			gpStrokePathColor(graphics, hand, argbColor(view.AccentSoft, 1), stroke)
			gpDeletePath(hand)
		}
	default:
		// Microphone: a filled capsule in its cradle, on a short stand.
		body := gpCapsulePath(x-3.4*unit, y-9*unit, 6.8*unit, 10.4*unit, 3.4*unit)
		if body != 0 {
			bodyBrush := gpGradientBrush(
				0, int32(y-9*unit)-1, 0, int32(y+1.4*unit)+1,
				[]uint32{argbColor(view.AccentSoft, 1), argbColor(view.Accent, 1), argbColor(mixRGB(view.Accent, 0x000000, 0.2), 1)},
				[]float32{0, 0.55, 1},
			)
			gpFillPath(graphics, bodyBrush, body)
			gpDeleteBrush(bodyBrush)
			gpDeletePath(body)
		}
		cradle := 6.6 * unit
		if pen := gpPen(argbColor(view.AccentSoft, 0.95), stroke*0.95); pen != 0 {
			gpDrawArc(graphics, pen, x-cradle, y-cradle+1.2*unit, cradle*2, cradle*2, 0, 180)
			gpDeletePen(pen)
		}
		if stand := gpPolylinePath([][2]float64{{x, y + 7.8*unit}, {x, y + 9.4*unit}}); stand != 0 {
			gpStrokePathColor(graphics, stand, argbColor(view.AccentSoft, 0.95), stroke*0.95)
			gpDeletePath(stand)
		}
	}
}

func drawMinimalOverlay(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, levels []float64, glowScale float64) {
	if view.Stage == overlayStageWaveform {
		level := 0.45
		if len(levels) > 0 {
			level = levels[len(levels)-1]
		} else if view.Animated {
			level = overlayWaveLevel(elapsed, overlayLevelBars/2, overlayLevelBars)
		}
		radius := (17 + 5*level) * geometry.scale
		if pulse := gpEllipsePath(geometry.iconX-radius, geometry.centerY-radius, radius*2, radius*2); pulse != 0 {
			gpStrokePathColor(graphics, pulse, argbColor(view.Accent, 0.22+0.3*level), 1.5*geometry.scale)
			gpDeletePath(pulse)
		}
	}
	drawOverlayIcon(graphics, view, geometry, elapsed, glowScale*view.Glow)
}

func drawMeterIndicators(graphics uintptr, view overlayView, geometry overlayGeometry) {
	count := min(max(view.Checkpoints, 0), 5)
	if count == 0 {
		return
	}
	scale := geometry.scale
	for index := 0; index < count; index++ {
		diameter := 3.2 * scale
		x := geometry.stageX + float64(index)*7*scale
		y := geometry.pillY + geometry.pillHeight - 9*scale
		if dot := gpEllipsePath(x, y, diameter, diameter); dot != 0 {
			gpFillPathColor(graphics, dot, argbColor(view.AccentSoft, 0.72))
			gpDeletePath(dot)
		}
	}
}

func drawDetailedOverlay(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, levels []float64, glowScale float64, visualizer OverlayVisualizer, font func(float64, int) uintptr) {
	scale := geometry.scale
	left := geometry.pillX + 20*scale
	right := geometry.pillX + geometry.pillWidth - 20*scale
	top := geometry.pillY

	// The Freehand mark is drawn from the same varied-bar motif as the product
	// icon, without loading an image or exposing any renderer-controlled text.
	for index, high := range []float64{7, 13, 19, 12, 8} {
		width := 3 * scale
		x := left + float64(index)*5*scale
		bar := gpCapsulePath(x, top+(27-high/2)*scale, width, high*scale, width/2)
		if bar != 0 {
			gpFillPathColor(graphics, bar, argbColor(view.Accent, 0.94))
			gpDeletePath(bar)
		}
	}
	gpDrawText(graphics, "Freehand", font(15*scale, fontStyleBold), gpRectF{
		X: float32(left + 30*scale), Y: float32(top + 13*scale), Width: float32(120 * scale), Height: float32(26 * scale),
	}, argbColor(view.AccentSoft, 0.96), stringAlignmentNear, stringAlignmentCenter)

	shortcut := strings.ReplaceAll(view.Shortcut, "Super", "Win")
	if shortcut != "" {
		shortcutFont := font(10.5*scale, fontStyleRegular)
		width, _ := gpMeasureText(graphics, shortcut, shortcutFont)
		width = min(max(width+16*scale, 54*scale), 154*scale)
		keycap := gpCapsulePath(right-width, top+14*scale, width, 25*scale, 6*scale)
		if keycap != 0 {
			gpFillPathColor(graphics, keycap, argbColor(view.Accent, 0.12))
			gpStrokePathColor(graphics, keycap, argbColor(view.AccentSoft, 0.24), scale)
			gpDeletePath(keycap)
		}
		gpDrawText(graphics, shortcut, shortcutFont, gpRectF{
			X: float32(right - width), Y: float32(top + 14*scale), Width: float32(width), Height: float32(25 * scale),
		}, argbColor(view.AccentSoft, 0.9), stringAlignmentCenter, stringAlignmentCenter)
	}

	phase, instruction := overlayPhaseText(view)
	gpDrawText(graphics, phase, font(17*scale, fontStyleBold), gpRectF{
		X: float32(left), Y: float32(top + 45*scale), Width: float32(210 * scale), Height: float32(25 * scale),
	}, argbColor(view.AccentSoft, 1), stringAlignmentNear, stringAlignmentCenter)
	gpDrawText(graphics, instruction, font(10.5*scale, fontStyleRegular), gpRectF{
		X: float32(left), Y: float32(top + 67*scale), Width: float32(342 * scale), Height: float32(18 * scale),
	}, argbColor(view.AccentSoft, 0.62), stringAlignmentNear, stringAlignmentCenter)

	stage := geometry
	stage.stageX = left
	stage.stageWidth = right - left
	stage.centerY = top + 105*scale
	stage.stageHeight = 24 * scale
	drawOverlayStage(graphics, view, stage, elapsed, 0.9, levels, glowScale, visualizer)

	elapsedText := formatOverlayElapsed(view, time.Now())
	checkpointText := fmt.Sprintf("%d checkpoints", min(max(view.Checkpoints, 0), 999))
	if view.Checkpoints == 1 {
		checkpointText = "1 checkpoint"
	}
	gpDrawText(graphics, elapsedText, font(10*scale, fontStyleRegular), gpRectF{
		X: float32(left), Y: float32(top + 126*scale), Width: float32(120 * scale), Height: float32(18 * scale),
	}, argbColor(view.AccentSoft, 0.7), stringAlignmentNear, stringAlignmentCenter)
	gpDrawText(graphics, checkpointText, font(10*scale, fontStyleRegular), gpRectF{
		X: float32(right - 130*scale), Y: float32(top + 126*scale), Width: float32(130 * scale), Height: float32(18 * scale),
	}, argbColor(view.AccentSoft, 0.7), stringAlignmentFar, stringAlignmentCenter)
}

func overlayPhaseText(view overlayView) (string, string) {
	switch view.Kind {
	case OverlayRecordingSpeech:
		return "Listening", recordingInstruction(view.RecordingMode)
	case OverlayRecordingSilence:
		return "Silence detected", recordingInstruction(view.RecordingMode)
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
		return "Listening", recordingInstruction(view.RecordingMode)
	}
}

func recordingInstruction(mode OverlayRecordingMode) string {
	if mode == OverlayRecordingHold {
		return "Release the shortcut to finish"
	}
	return "Use the shortcut again to finish"
}

func formatOverlayElapsed(view overlayView, now time.Time) string {
	if view.StartedAt.IsZero() || now.Before(view.StartedAt) {
		return "0:00 elapsed"
	}
	if !view.FinishedAt.IsZero() && view.FinishedAt.Before(now) {
		now = view.FinishedAt
	}
	total := min(int(now.Sub(view.StartedAt)/time.Second), 99*60+59)
	return fmt.Sprintf("%d:%02d elapsed", total/60, total%60)
}

func drawCaptionOverlay(graphics uintptr, view overlayView, geometry overlayGeometry, font func(float64, int) uintptr) {
	scale := geometry.scale
	left, top := geometry.pillX+16*scale, geometry.pillY
	label := "Live"
	if view.Kind == OverlayTranscribing {
		label = "Finishing"
	}
	gpDrawText(graphics, label, font(11*scale, fontStyleBold), gpRectF{X: float32(left), Y: float32(top), Width: float32(62 * scale), Height: float32(geometry.pillHeight)}, argbColor(view.AccentSoft, 0.65), stringAlignmentNear, stringAlignmentCenter)
	width := geometry.pillWidth - 100*scale
	gpDrawSingleLineText(graphics, view.Caption, font(16*scale, fontStyleRegular), gpRectF{X: float32(left + 68*scale), Y: float32(top), Width: float32(width), Height: float32(geometry.pillHeight)}, argbColor(view.AccentSoft, 1), stringAlignmentNear, stringAlignmentCenter)
}
