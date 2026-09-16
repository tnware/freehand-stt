//go:build windows

package overlay

import (
	"math"
	"time"
	"unsafe"
)

func (o *StatusOverlay) reconfigure(hwnd uintptr) {
	if !o.anim.shown {
		return
	}
	o.render(hwnd)
	o.updateFrameTimer(hwnd, o.anim.target)
}

func (o *StatusOverlay) refreshSystemPreferences(hwnd uintptr) {
	enabled := clientAreaAnimationsEnabled()
	highContrast := highContrastEnabled()
	if enabled == o.animationsEnabled && highContrast == o.highContrast {
		return
	}
	o.animationsEnabled = enabled
	o.highContrast = highContrast

	anim := &o.anim
	if !anim.shown {
		return
	}
	if !o.motionEnabled() && anim.hiding {
		killTimer.Call(hwnd, overlayTimerID)
		showWindow.Call(hwnd, swHide)
		anim.shown = false
		anim.hiding = false
		return
	}

	now := time.Now()
	anim.showAt = now.Add(-overlayEnterMS * time.Millisecond)
	anim.morphAt = now.Add(-overlayMorphMS * time.Millisecond)
	o.render(hwnd)
	o.updateFrameTimer(hwnd, anim.target)
}

func (o *StatusOverlay) motionEnabled() bool {
	return o.animationsEnabled && o.optionSnapshot().Motion != OverlayMotionReduced
}

func (o *StatusOverlay) frameInterval(view overlayView) uintptr {
	if overlayNeedsContinuousFrames(view, o.motionEnabled()) {
		return overlayFrameMS
	}
	options := o.optionSnapshot()
	if options.Layout == OverlayLayoutDetailed && !view.StartedAt.IsZero() && view.FinishedAt.IsZero() {
		return 250
	}
	return 0
}

func (o *StatusOverlay) updateFrameTimer(hwnd uintptr, view overlayView) {
	interval := o.frameInterval(view)
	if interval == 0 {
		killTimer.Call(hwnd, overlayTimerID)
		return
	}
	setTimer.Call(hwnd, overlayTimerID, interval, 0)
}

// apply reconciles the window with the coordinator state that Update stored. It
// starts the entrance, the exit or a colour morph, and it resolves placement
// once per operation so a focus change mid-operation cannot move the surface
// between monitors.
func (o *StatusOverlay) apply(hwnd uintptr) {
	view := o.snapshot()
	anim := &o.anim
	now := time.Now()

	if !view.Visible {
		if !o.motionEnabled() {
			killTimer.Call(hwnd, overlayTimerID)
			showWindow.Call(hwnd, swHide)
			anim.shown = false
			anim.hiding = false
			return
		}
		if anim.shown && !anim.hiding {
			anim.hiding = true
			anim.hideAt = now
			setTimer.Call(hwnd, overlayTimerID, overlayFrameMS, 0)
		}
		return
	}

	wasShown := anim.shown && !anim.hiding
	if wasShown {
		anim.from = anim.lastView
	} else {
		anim.from = view
		anim.showAt = now
	}
	anim.hiding = false
	anim.shown = true
	anim.target = view
	anim.morphAt = now
	if !anim.anchorValid || !wasShown || anim.anchorGeneration != view.Generation {
		o.resolveAnchor(hwnd)
		anim.anchorGeneration = view.Generation
	}
	if o.motionEnabled() {
		setTimer.Call(hwnd, overlayTimerID, overlayFrameMS, 0)
	} else {
		o.updateFrameTimer(hwnd, view)
	}
	o.render(hwnd)
	setWindowPos.Call(hwnd, ^uintptr(0), 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)
}

func (o *StatusOverlay) tick(hwnd uintptr) {
	anim := &o.anim
	if !anim.shown {
		killTimer.Call(hwnd, overlayTimerID)
		return
	}
	if !o.motionEnabled() {
		if anim.hiding {
			killTimer.Call(hwnd, overlayTimerID)
			showWindow.Call(hwnd, swHide)
			anim.shown = false
			anim.hiding = false
			return
		}
		o.render(hwnd)
		o.updateFrameTimer(hwnd, anim.target)
		return
	}
	if anim.hiding && time.Since(anim.hideAt) >= overlayExitMS*time.Millisecond {
		killTimer.Call(hwnd, overlayTimerID)
		showWindow.Call(hwnd, swHide)
		anim.shown = false
		anim.hiding = false
		return
	}
	o.render(hwnd)
	settled := !anim.target.Animated &&
		!anim.hiding &&
		time.Since(anim.showAt) >= overlayEnterMS*time.Millisecond &&
		time.Since(anim.morphAt) >= overlayMorphMS*time.Millisecond
	if settled {
		o.updateFrameTimer(hwnd, anim.target)
	}
}

// resolveAnchor captures the work area of the foreground monitor once at the
// start of an operation. State changes do not chase focus across monitors.
func (o *StatusOverlay) render(hwnd uintptr) {
	anim := &o.anim
	if !anim.shown || !anim.target.Visible {
		return
	}
	if anim.dpi == 0 {
		o.resolveAnchor(hwnd)
	}
	now := time.Now()
	options := o.optionSnapshot()
	if o.highContrast {
		options.Opacity = 1
		options.Glow = 0
	}
	scale := overlayScaleForDPI(anim.dpi, options)
	motionEnabled := o.motionEnabled()

	// Colour morphs between consecutive coordinator states so that a transition
	// reads as one object changing rather than two objects swapping.
	morph := 1.0
	if motionEnabled {
		morph = easeInOutSine(float64(now.Sub(anim.morphAt).Milliseconds()) / overlayMorphMS)
	}
	view := anim.target
	view.Background = mixRGB(anim.from.Background, anim.target.Background, morph)
	view.Accent = mixRGB(anim.from.Accent, anim.target.Accent, morph)
	view.AccentSoft = mixRGB(anim.from.AccentSoft, anim.target.AccentSoft, morph)
	view.Glow = lerp(anim.from.Glow, anim.target.Glow, morph)
	if view.Stage == overlayStageCountdown && view.CountdownDuration > 0 {
		view.CountdownProgress = clamp01(float64(view.CountdownDeadline.Sub(now)) / float64(view.CountdownDuration))
	}
	view = overlayViewForMotion(view, motionEnabled)
	anim.lastView = view

	// The entrance rises and settles with a slight overshoot; the exit sinks.
	alpha := 1.0
	offsetY := 0.0
	if motionEnabled {
		progress := overlayEntrance(uint32(now.Sub(anim.showAt).Milliseconds()))
		alpha = easeOutCubic(progress)
		offsetY = (1 - easeOutBack(progress)) * overlayRise * scale
	}
	if anim.hiding {
		exit := easeInOutSine(overlayExit(uint32(now.Sub(anim.hideAt).Milliseconds())))
		alpha = math.Min(alpha, 1-exit)
		offsetY = exit * overlayRise * 0.5 * scale
	}

	if view.CaptionEnabled {
		options.Layout = overlayLayoutCaptions
		options.Surface = OverlaySurfaceSolid
		options.Opacity = max(options.Opacity, 0.9)
	}
	geometry := overlayLayout(options.Layout, scale, offsetY)
	if !o.surface.ensure(geometry.windowWidth, geometry.windowHigh) {
		return
	}
	graphics := o.surface.graphics
	elapsed := uint32(0)
	if motionEnabled {
		elapsed = uint32(now.Sub(anim.showAt).Milliseconds())
	}

	gpClear(graphics, 0)
	recording := view.Stage == overlayStageWaveform && view.Animated && !view.Preview
	o.sampleLevels(now, recording)
	levels := o.liveLevels(recording)
	if view.CaptionEnabled {
		caption := view.Caption
		if caption == "" {
			caption = "Listening…"
		}
		view.Caption = o.captionCache.fit(caption, geometry.pillWidth-100*scale, scale, func(text string) float64 {
			width, _ := gpMeasureText(graphics, text, o.font(16*scale, fontStyleRegular))
			return width
		})
	} else {
		o.captionCache = overlayCaptionCache{}
	}
	drawOverlay(graphics, view, geometry, elapsed, levels, options, o.highContrast, o.font)
	gdipFlush.Call(graphics, 1)

	size := nativeSize{CX: geometry.windowWidth, CY: geometry.windowHigh}
	destination := overlayDestination(anim.workArea, geometry, options)
	source := nativePoint{}
	blend := blendFunction{
		BlendOp:             acSrcOver,
		SourceConstantAlpha: byte(math.Round(overlayAlpha(alpha, options) * 255)),
		AlphaFormat:         acSrcAlpha,
	}
	updateLayeredWindow.Call(
		hwnd, 0,
		uintptr(unsafe.Pointer(&destination)), uintptr(unsafe.Pointer(&size)),
		o.surface.dc, uintptr(unsafe.Pointer(&source)), 0,
		uintptr(unsafe.Pointer(&blend)), ulwAlpha,
	)
	if !anim.hiding {
		showWindow.Call(hwnd, swShowNoActivate)
	}
}

// drawOverlayStage paints the wide animated area. Every coordinator state gets
// its own motion so the phase is legible without relying on colour alone.
func drawOverlayStage(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, breath float64, levels []float64, glowScale float64, visualizer OverlayVisualizer) {
	switch view.Stage {
	case overlayStageComet:
		drawStageComet(graphics, view, geometry, elapsed, false, glowScale)
	case overlayStageCometReverse:
		drawStageComet(graphics, view, geometry, elapsed, true, glowScale)
	case overlayStageShuttle:
		drawStageShuttle(graphics, view, geometry, elapsed, glowScale)
	case overlayStageBreath:
		drawStageBreath(graphics, view, geometry, breath, glowScale)
	case overlayStageFlatline:
		drawStageFlatline(graphics, view, geometry)
	case overlayStageCountdown:
		drawStageCountdown(graphics, view, geometry, glowScale)
	default:
		switch visualizer {
		case OverlayVisualizerPulse:
			drawStagePulse(graphics, view, geometry, elapsed, levels, glowScale)
		case OverlayVisualizerEnvelope:
			drawStageEnvelope(graphics, view, geometry, elapsed, levels, glowScale)
		case OverlayVisualizerMeter:
			drawStageMeter(graphics, view, geometry, elapsed, levels, glowScale)
		default:
			drawStageWaveform(graphics, view, geometry, elapsed, levels, glowScale)
		}
	}
}

func drawStageCountdown(graphics uintptr, view overlayView, geometry overlayGeometry, glowScale float64) {
	scale := geometry.scale
	trackHigh := 4.8 * scale
	track := gpCapsulePath(geometry.stageX, geometry.centerY-trackHigh/2, geometry.stageWidth, trackHigh, trackHigh/2)
	if track != 0 {
		gpFillPathColor(graphics, track, argbColor(view.Accent, 0.18))
		gpDeletePath(track)
	}
	width := geometry.stageWidth * clamp01(view.CountdownProgress)
	if width <= 0 {
		return
	}
	fill := gpCapsulePath(geometry.stageX, geometry.centerY-trackHigh/2, math.Max(trackHigh, width), trackHigh, trackHigh/2)
	if fill == 0 {
		return
	}
	gpGlow(graphics, fill, view.Accent, 6*scale, 3, 0.72*glowScale)
	fillBrush := gpGradientBrush(
		int32(geometry.stageX)-1, 0, int32(geometry.stageX+width)+1, 0,
		[]uint32{argbColor(view.AccentSoft, 1), argbColor(view.Accent, 0.8)},
		[]float32{0, 1},
	)
	gpFillPath(graphics, fillBrush, fill)
	gpDeleteBrush(fillBrush)
	gpDeletePath(fill)
}

// drawStageWaveform paints the amplitude history. Each bar is one reading
// rather than a slice of the signal, so this is a level meter drawn as a
// waveform, not the waveform itself. With no live levels it falls back to the
// clock-driven animation.
func drawStageWaveform(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, levels []float64, glowScale float64) {
	// Bar count matches the meter history so a live reading maps one to one.
	const count = overlayLevelBars
	pitch := geometry.stageWidth / count
	barWidth := pitch * 0.5
	minHeight := 2.6 * geometry.scale
	maxHeight := geometry.stageHeight

	for index := 0; index < count; index++ {
		level := 0.45
		if index < len(levels) {
			// Floored just above zero so silence reads as a quiet meter
			// rather than as a broken one.
			level = 0.05 + 0.95*levels[index]
		} else if view.Animated {
			level = overlayWaveLevel(elapsed, index, count)
		}
		height := minHeight + level*(maxHeight-minHeight)
		left := geometry.stageX + float64(index)*pitch + (pitch-barWidth)/2
		bar := gpCapsulePath(left, geometry.centerY-height/2, barWidth, height, barWidth/2)
		if bar == 0 {
			continue
		}
		gpGlow(graphics, bar, view.Accent, 4*geometry.scale, 3, (0.22+0.48*level)*glowScale)
		gpFillPathColor(graphics, bar, argbColor(mixRGB(view.Accent, view.AccentSoft, level*0.75), lerp(0.6, 1, level)))
		gpDeletePath(bar)
	}
}

func stageLevel(view overlayView, elapsed uint32, levels []float64) float64 {
	if len(levels) > 0 {
		return clamp01(levels[len(levels)-1])
	}
	if view.Animated {
		return overlayWaveLevel(elapsed, overlayLevelBars/2, overlayLevelBars)
	}
	return 0.08
}

func drawStagePulse(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, levels []float64, glowScale float64) {
	level := stageLevel(view, elapsed, levels)
	radius := 4*geometry.scale + level*geometry.stageHeight*0.34
	x := geometry.stageX + geometry.stageWidth/2
	if halo := gpEllipsePath(x-radius*1.8, geometry.centerY-radius*1.8, radius*3.6, radius*3.6); halo != 0 {
		gpFillPathColor(graphics, halo, argbColor(view.Accent, (0.06+0.15*level)*glowScale))
		gpDeletePath(halo)
	}
	if pulse := gpEllipsePath(x-radius, geometry.centerY-radius, radius*2, radius*2); pulse != 0 {
		gpFillPathColor(graphics, pulse, argbColor(mixRGB(view.Accent, view.AccentSoft, level), 0.75+0.25*level))
		gpDeletePath(pulse)
	}
}

func drawStageEnvelope(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, levels []float64, glowScale float64) {
	const count = overlayLevelBars
	points := make([][2]float64, count)
	for index := 0; index < count; index++ {
		level := 0.08
		if index < len(levels) {
			level = levels[index]
		} else if view.Animated {
			level = overlayWaveLevel(elapsed, index, count)
		}
		points[index] = [2]float64{
			geometry.stageX + float64(index)*geometry.stageWidth/float64(count-1),
			geometry.centerY + (0.5-level)*geometry.stageHeight*0.82,
		}
	}
	path := gpPolylinePath(points)
	if path == 0 {
		return
	}
	gpGlow(graphics, path, view.Accent, 5*geometry.scale, 3, 0.4*glowScale)
	gpStrokePathColor(graphics, path, argbColor(view.AccentSoft, 0.94), 2*geometry.scale)
	gpDeletePath(path)
}

func drawStageMeter(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, levels []float64, glowScale float64) {
	level := stageLevel(view, elapsed, levels)
	height := 4.2 * geometry.scale
	track := gpCapsulePath(geometry.stageX, geometry.centerY-height/2, geometry.stageWidth, height, height/2)
	if track != 0 {
		gpFillPathColor(graphics, track, argbColor(view.Accent, 0.17))
		gpDeletePath(track)
	}
	width := max(height, geometry.stageWidth*level)
	fill := gpCapsulePath(geometry.stageX, geometry.centerY-height/2, width, height, height/2)
	if fill != 0 {
		gpGlow(graphics, fill, view.Accent, 5*geometry.scale, 3, (0.28+0.4*level)*glowScale)
		gpFillPathColor(graphics, fill, argbColor(mixRGB(view.Accent, view.AccentSoft, level), 0.9))
		gpDeletePath(fill)
	}
}

func drawStageComet(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, reverse bool, glowScale float64) {
	const count = 13
	pitch := geometry.stageWidth / count
	base := pitch * 0.44

	for index := 0; index < count; index++ {
		intensity := 0.35
		if view.Animated {
			intensity = overlayCometIntensity(elapsed, index, count, reverse)
		}
		diameter := base * lerp(0.62, 1.5, intensity)
		centerX := geometry.stageX + (float64(index)+0.5)*pitch
		dot := gpEllipsePath(centerX-diameter/2, geometry.centerY-diameter/2, diameter, diameter)
		if dot == 0 {
			continue
		}
		if intensity > 0.25 {
			gpGlow(graphics, dot, view.Accent, 5*geometry.scale, 3, 0.70*intensity*glowScale)
		}
		gpFillPathColor(graphics, dot,
			argbColor(mixRGB(view.Accent, view.AccentSoft, intensity), lerp(0.22, 1, intensity)))
		gpDeletePath(dot)
	}
}

func drawStageShuttle(graphics uintptr, view overlayView, geometry overlayGeometry, elapsed uint32, glowScale float64) {
	scale := geometry.scale
	trackHigh := 4.6 * scale
	track := gpCapsulePath(geometry.stageX, geometry.centerY-trackHigh/2, geometry.stageWidth, trackHigh, trackHigh/2)
	if track != 0 {
		gpFillPathColor(graphics, track, argbColor(view.Accent, 0.16))
		gpDeletePath(track)
	}
	segment := geometry.stageWidth * 0.36
	center := 0.5
	if view.Animated {
		center = overlayShuttleCenter(elapsed)
	}
	left := geometry.stageX + center*(geometry.stageWidth-segment)
	fill := gpCapsulePath(left, geometry.centerY-trackHigh/2, segment, trackHigh, trackHigh/2)
	if fill == 0 {
		return
	}
	gpGlow(graphics, fill, view.Accent, 6*scale, 3, 0.72*glowScale)
	fillBrush := gpGradientBrush(
		int32(left)-1, 0, int32(left+segment)+1, 0,
		[]uint32{argbColor(view.Accent, 0.55), argbColor(view.AccentSoft, 1), argbColor(view.Accent, 0.55)},
		[]float32{0, 0.5, 1},
	)
	gpFillPath(graphics, fillBrush, fill)
	gpDeleteBrush(fillBrush)
	gpDeletePath(fill)
}

func drawStageBreath(graphics uintptr, view overlayView, geometry overlayGeometry, breath float64, glowScale float64) {
	scale := geometry.scale
	trackHigh := 4.6 * scale
	track := gpCapsulePath(geometry.stageX, geometry.centerY-trackHigh/2, geometry.stageWidth, trackHigh, trackHigh/2)
	if track == 0 {
		return
	}
	gpGlow(graphics, track, view.Accent, 6*scale, 3, 0.70*breath*glowScale)
	gpFillPathColor(graphics, track, argbColor(mixRGB(view.Accent, view.AccentSoft, breath), lerp(0.32, 0.95, breath)))
	gpDeletePath(track)
}

func drawStageFlatline(graphics uintptr, view overlayView, geometry overlayGeometry) {
	scale := geometry.scale
	lineHigh := 2.4 * scale
	line := gpCapsulePath(geometry.stageX, geometry.centerY-lineHigh/2, geometry.stageWidth, lineHigh, lineHigh/2)
	if line != 0 {
		gpFillPathColor(graphics, line, argbColor(view.Accent, 0.5))
		gpDeletePath(line)
	}
	// A single stationary blip keeps the flat line from reading as a divider.
	blip := 4.4 * scale
	dot := gpEllipsePath(geometry.stageX+geometry.stageWidth/2-blip/2, geometry.centerY-blip/2, blip, blip)
	if dot != 0 {
		gpFillPathColor(graphics, dot, argbColor(view.AccentSoft, 1))
		gpDeletePath(dot)
	}
}
