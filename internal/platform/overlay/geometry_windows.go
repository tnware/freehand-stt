//go:build windows

package overlay

import (
	"math"
	"unsafe"
)

func (o *StatusOverlay) resolveAnchor(hwnd uintptr) {
	foreground, _, _ := getForegroundWindow.Call()
	monitor, _, _ := monitorFromWindow.Call(foreground, monitorDefault)
	if monitor == 0 {
		monitor, _, _ = monitorFromWindow.Call(0, monitorPrimary)
	}
	if monitor == 0 {
		return
	}
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if result, _, _ := getMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); result == 0 {
		return
	}
	dpi := uint32(96)
	var dpiX, dpiY uint32
	if status, _, _ := getDPIForMonitor.Call(monitor, 0, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY))); status == 0 && dpiX != 0 {
		dpi = dpiX
	} else if value, _, _ := getDPIForWindow.Call(hwnd); value != 0 {
		dpi = uint32(value)
	}
	o.anim.dpi = dpi
	o.anim.workArea = info.Work
	o.anim.anchorValid = true
}

// overlayGeometry is fully resolved device-pixel layout for one frame.
type overlayGeometry struct {
	layout      OverlayLayout
	scale       float64
	pillX       float64
	pillY       float64
	pillWidth   float64
	pillHeight  float64
	radius      float64
	iconX       float64
	centerY     float64
	stageX      float64
	stageWidth  float64
	stageHeight float64
	windowWidth int32
	windowHigh  int32
}

const overlayLayoutCaptions OverlayLayout = 255

func overlayLayout(layout OverlayLayout, scale, offsetY float64) overlayGeometry {
	pillWidth, pillHeight, radius := overlayPillWidth, overlayPillHeight, overlayPillHeight/2
	iconX, stageLeft, stageRight, stageHigh := overlayIconX, overlayStageLeft, overlayStageRight, overlayStageHigh
	switch layout {
	case overlayLayoutCaptions:
		pillWidth, pillHeight, radius = 640, 52, 16
	case OverlayLayoutMinimal:
		pillWidth, pillHeight, radius = 56, 56, 28
		iconX, stageLeft, stageRight, stageHigh = 28, 28, 28, 0
	case OverlayLayoutMeter:
		pillWidth, pillHeight, radius = 304, 62, 31
		iconX, stageLeft, stageRight, stageHigh = 36, 78, 24, 30
	case OverlayLayoutDetailed:
		pillWidth, pillHeight, radius = 382, 154, 18
		iconX, stageLeft, stageRight, stageHigh = 24, 20, 20, 28
	}
	geometry := overlayGeometry{
		layout:      layout,
		scale:       scale,
		pillWidth:   pillWidth * scale,
		pillHeight:  pillHeight * scale,
		radius:      radius * scale,
		stageHeight: stageHigh * scale,
	}
	geometry.windowWidth = int32(math.Ceil((pillWidth + 2*overlayPadding) * scale))
	geometry.windowHigh = int32(math.Ceil((pillHeight + 2*overlayPadding + overlayRise) * scale))
	geometry.pillX = (float64(geometry.windowWidth) - geometry.pillWidth) / 2
	geometry.pillY = overlayPadding*scale + offsetY
	geometry.centerY = geometry.pillY + geometry.pillHeight/2
	geometry.iconX = geometry.pillX + iconX*scale
	geometry.stageX = geometry.pillX + stageLeft*scale
	geometry.stageWidth = geometry.pillWidth - (stageLeft+stageRight)*scale
	return geometry
}

func overlayDestination(work nativeRect, geometry overlayGeometry, options OverlayOptions) nativePoint {
	edge := float64(options.EdgeOffset) * geometry.scale / options.Scale
	x := float64(work.Left) + (float64(work.Right-work.Left)-float64(geometry.windowWidth))/2
	switch options.Anchor {
	case OverlayAnchorTopLeft, OverlayAnchorBottomLeft:
		x = float64(work.Left) + edge - geometry.pillX
	case OverlayAnchorTopRight, OverlayAnchorBottomRight:
		x = float64(work.Right) - edge - geometry.pillX - geometry.pillWidth
	}
	y := float64(work.Top) + edge - overlayPadding*geometry.scale
	switch options.Anchor {
	case OverlayAnchorBottomLeft, OverlayAnchorBottomCenter, OverlayAnchorBottomRight:
		y = float64(work.Bottom) - edge - overlayPadding*geometry.scale - geometry.pillHeight
	}
	return nativePoint{X: int32(math.Round(x)), Y: int32(math.Round(y))}
}
