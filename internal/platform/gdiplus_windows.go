//go:build windows

package platform

import (
	"errors"
	"math"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GDI+ flat API. Only the surface the status overlay needs is bound here.
//
// REAL (C float) parameters are passed as uintptr(math.Float32bits(v)) through
// gpReal. On windows/amd64 the stdcall bridge mirrors the first four integer
// argument registers into XMM0-XMM3, and arguments past the fourth are passed
// on the stack in both classes, so a float lands correctly in either position.
var gdiplusDLL = windows.NewLazySystemDLL("gdiplus.dll")

var (
	gdiplusStartup  = gdiplusDLL.NewProc("GdiplusStartup")
	gdiplusShutdown = gdiplusDLL.NewProc("GdiplusShutdown")

	gdipCreateBitmapFromScan0   = gdiplusDLL.NewProc("GdipCreateBitmapFromScan0")
	gdipGetImageGraphicsContext = gdiplusDLL.NewProc("GdipGetImageGraphicsContext")
	gdipDisposeImage            = gdiplusDLL.NewProc("GdipDisposeImage")
	gdipDeleteGraphics          = gdiplusDLL.NewProc("GdipDeleteGraphics")
	gdipGraphicsClear           = gdiplusDLL.NewProc("GdipGraphicsClear")
	gdipFlush                   = gdiplusDLL.NewProc("GdipFlush")

	gdipSetSmoothingMode      = gdiplusDLL.NewProc("GdipSetSmoothingMode")
	gdipSetPixelOffsetMode    = gdiplusDLL.NewProc("GdipSetPixelOffsetMode")
	gdipSetCompositingQuality = gdiplusDLL.NewProc("GdipSetCompositingQuality")
	gdipSetInterpolationMode  = gdiplusDLL.NewProc("GdipSetInterpolationMode")
	gdipSetTextRenderingHint  = gdiplusDLL.NewProc("GdipSetTextRenderingHint")

	gdipSaveGraphics            = gdiplusDLL.NewProc("GdipSaveGraphics")
	gdipRestoreGraphics         = gdiplusDLL.NewProc("GdipRestoreGraphics")
	gdipTranslateWorldTransform = gdiplusDLL.NewProc("GdipTranslateWorldTransform")
	gdipSetClipPath             = gdiplusDLL.NewProc("GdipSetClipPath")
	gdipResetClip               = gdiplusDLL.NewProc("GdipResetClip")

	gdipCreatePath      = gdiplusDLL.NewProc("GdipCreatePath")
	gdipDeletePath      = gdiplusDLL.NewProc("GdipDeletePath")
	gdipAddPathArc      = gdiplusDLL.NewProc("GdipAddPathArc")
	gdipAddPathEllipse  = gdiplusDLL.NewProc("GdipAddPathEllipse")
	gdipAddPathLine     = gdiplusDLL.NewProc("GdipAddPathLine")
	gdipStartPathFigure = gdiplusDLL.NewProc("GdipStartPathFigure")
	gdipClosePathFigure = gdiplusDLL.NewProc("GdipClosePathFigure")

	gdipCreateSolidFill = gdiplusDLL.NewProc("GdipCreateSolidFill")
	gdipDeleteBrush     = gdiplusDLL.NewProc("GdipDeleteBrush")

	gdipCreateLineBrushI   = gdiplusDLL.NewProc("GdipCreateLineBrushI")
	gdipSetLinePresetBlend = gdiplusDLL.NewProc("GdipSetLinePresetBlend")

	gdipCreatePathGradientFromPath             = gdiplusDLL.NewProc("GdipCreatePathGradientFromPath")
	gdipSetPathGradientCenterColor             = gdiplusDLL.NewProc("GdipSetPathGradientCenterColor")
	gdipSetPathGradientSurroundColorsWithCount = gdiplusDLL.NewProc("GdipSetPathGradientSurroundColorsWithCount")
	gdipSetPathGradientFocusScales             = gdiplusDLL.NewProc("GdipSetPathGradientFocusScales")

	gdipCreatePen1     = gdiplusDLL.NewProc("GdipCreatePen1")
	gdipCreatePen2     = gdiplusDLL.NewProc("GdipCreatePen2")
	gdipDeletePen      = gdiplusDLL.NewProc("GdipDeletePen")
	gdipSetPenStartCap = gdiplusDLL.NewProc("GdipSetPenStartCap")
	gdipSetPenEndCap   = gdiplusDLL.NewProc("GdipSetPenEndCap")
	gdipSetPenLineJoin = gdiplusDLL.NewProc("GdipSetPenLineJoin")

	gdipFillPath = gdiplusDLL.NewProc("GdipFillPath")
	gdipDrawPath = gdiplusDLL.NewProc("GdipDrawPath")
	gdipDrawArc  = gdiplusDLL.NewProc("GdipDrawArc")

	gdipCreateFontFamilyFromName = gdiplusDLL.NewProc("GdipCreateFontFamilyFromName")
	gdipDeleteFontFamily         = gdiplusDLL.NewProc("GdipDeleteFontFamily")
	gdipCreateFont               = gdiplusDLL.NewProc("GdipCreateFont")
	gdipDeleteFont               = gdiplusDLL.NewProc("GdipDeleteFont")
	gdipCreateStringFormat       = gdiplusDLL.NewProc("GdipCreateStringFormat")
	gdipDeleteStringFormat       = gdiplusDLL.NewProc("GdipDeleteStringFormat")
	gdipSetStringFormatAlign     = gdiplusDLL.NewProc("GdipSetStringFormatAlign")
	gdipSetStringFormatLineAlign = gdiplusDLL.NewProc("GdipSetStringFormatLineAlign")
	gdipDrawString               = gdiplusDLL.NewProc("GdipDrawString")
	gdipMeasureString            = gdiplusDLL.NewProc("GdipMeasureString")
)

const (
	gpOk = 0

	// PixelFormat32bppPARGB. UpdateLayeredWindow consumes premultiplied alpha,
	// so GDI+ is pointed straight at the layered surface in that format and no
	// conversion pass is needed.
	pixelFormat32bppPARGB = 0x000E200B

	smoothingModeAntiAlias        = 4
	pixelOffsetModeHighQuality    = 2
	compositingQualityHighQuality = 2
	interpolationModeHighQuality  = 7
	textRenderingHintAntiAliasFit = 3

	unitPixel = 2

	wrapModeTile = 0

	combineModeReplace = 0

	lineCapRound  = 2
	lineJoinRound = 2

	matrixOrderPrepend = 0

	fontStyleRegular = 0
	fontStyleBold    = 1

	stringAlignmentNear   = 0
	stringAlignmentCenter = 1
	stringAlignmentFar    = 2
)

type gpPoint struct {
	X int32
	Y int32
}

type gpRectF struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

type gdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

var (
	gdiplusMu    sync.Mutex
	gdiplusToken uintptr
)

func gdiplusInit() error {
	gdiplusMu.Lock()
	defer gdiplusMu.Unlock()
	if gdiplusToken != 0 {
		return nil
	}
	if err := gdiplusDLL.Load(); err != nil {
		return err
	}
	input := gdiplusStartupInput{GdiplusVersion: 1}
	status, _, _ := gdiplusStartup.Call(
		uintptr(unsafe.Pointer(&gdiplusToken)),
		uintptr(unsafe.Pointer(&input)),
		0,
	)
	if status != gpOk {
		gdiplusToken = 0
		return errors.New("GDI+ could not be initialized for the native status overlay")
	}
	return nil
}

func gdiplusRelease() {
	gdiplusMu.Lock()
	defer gdiplusMu.Unlock()
	if gdiplusToken != 0 {
		gdiplusShutdown.Call(gdiplusToken)
		gdiplusToken = 0
	}
}

// gpReal encodes a C float argument for the stdcall bridge.
func gpReal(value float64) uintptr {
	return uintptr(math.Float32bits(float32(value)))
}

// --- surface -----------------------------------------------------------------

func gpBitmapOverMemory(width, height int32, bits unsafe.Pointer) uintptr {
	var bitmap uintptr
	status, _, _ := gdipCreateBitmapFromScan0.Call(
		uintptr(width), uintptr(height), uintptr(width*4),
		pixelFormat32bppPARGB, uintptr(bits), uintptr(unsafe.Pointer(&bitmap)),
	)
	if status != gpOk {
		return 0
	}
	return bitmap
}

func gpGraphicsFromImage(image uintptr) uintptr {
	var graphics uintptr
	status, _, _ := gdipGetImageGraphicsContext.Call(image, uintptr(unsafe.Pointer(&graphics)))
	if status != gpOk {
		return 0
	}
	gdipSetSmoothingMode.Call(graphics, smoothingModeAntiAlias)
	gdipSetPixelOffsetMode.Call(graphics, pixelOffsetModeHighQuality)
	gdipSetCompositingQuality.Call(graphics, compositingQualityHighQuality)
	gdipSetInterpolationMode.Call(graphics, interpolationModeHighQuality)
	gdipSetTextRenderingHint.Call(graphics, textRenderingHintAntiAliasFit)
	return graphics
}

func gpClear(graphics uintptr, color uint32) {
	gdipGraphicsClear.Call(graphics, uintptr(color))
}

// --- paths -------------------------------------------------------------------

func gpNewPath() uintptr {
	var path uintptr
	if status, _, _ := gdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path))); status != gpOk {
		return 0
	}
	return path
}

func gpDeletePath(path uintptr) {
	if path != 0 {
		gdipDeletePath.Call(path)
	}
}

func gpArc(path uintptr, x, y, w, h, start, sweep float64) {
	gdipAddPathArc.Call(path, gpReal(x), gpReal(y), gpReal(w), gpReal(h), gpReal(start), gpReal(sweep))
}

// gpCapsulePath builds a rounded rectangle, clamping the corner radius so that
// a fully round capsule and a soft-cornered card share one code path.
func gpCapsulePath(x, y, w, h, radius float64) uintptr {
	path := gpNewPath()
	if path == 0 {
		return 0
	}
	if radius > w/2 {
		radius = w / 2
	}
	if radius > h/2 {
		radius = h / 2
	}
	if radius <= 0.5 {
		radius = 0.5
	}
	diameter := radius * 2
	gpArc(path, x, y, diameter, diameter, 180, 90)
	gpArc(path, x+w-diameter, y, diameter, diameter, 270, 90)
	gpArc(path, x+w-diameter, y+h-diameter, diameter, diameter, 0, 90)
	gpArc(path, x, y+h-diameter, diameter, diameter, 90, 90)
	gdipClosePathFigure.Call(path)
	return path
}

// gpPolylinePath chains open segments through the supplied points. Icon glyphs
// are built from these and stroked with round caps and joins.
func gpPolylinePath(points [][2]float64) uintptr {
	if len(points) < 2 {
		return 0
	}
	path := gpNewPath()
	if path == 0 {
		return 0
	}
	gdipStartPathFigure.Call(path)
	for index := 1; index < len(points); index++ {
		from, to := points[index-1], points[index]
		gdipAddPathLine.Call(path, gpReal(from[0]), gpReal(from[1]), gpReal(to[0]), gpReal(to[1]))
	}
	return path
}

func gpEllipsePath(x, y, w, h float64) uintptr {
	path := gpNewPath()
	if path == 0 {
		return 0
	}
	gdipAddPathEllipse.Call(path, gpReal(x), gpReal(y), gpReal(w), gpReal(h))
	return path
}

// --- brushes and pens --------------------------------------------------------

func gpSolidBrush(color uint32) uintptr {
	var brush uintptr
	if status, _, _ := gdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush))); status != gpOk {
		return 0
	}
	return brush
}

// gpGradientBrush spans exactly the two supplied points. Callers extend the
// span one pixel past the shape they fill, because a tiled GDI+ linear gradient
// wraps at its own boundary.
func gpGradientBrush(x1, y1, x2, y2 int32, colors []uint32, positions []float32) uintptr {
	if len(colors) < 2 || len(colors) != len(positions) {
		return 0
	}
	start := gpPoint{X: x1, Y: y1}
	end := gpPoint{X: x2, Y: y2}
	var brush uintptr
	status, _, _ := gdipCreateLineBrushI.Call(
		uintptr(unsafe.Pointer(&start)), uintptr(unsafe.Pointer(&end)),
		uintptr(colors[0]), uintptr(colors[len(colors)-1]),
		wrapModeTile, uintptr(unsafe.Pointer(&brush)),
	)
	if status != gpOk {
		return 0
	}
	gdipSetLinePresetBlend.Call(
		brush,
		uintptr(unsafe.Pointer(&colors[0])),
		uintptr(unsafe.Pointer(&positions[0])),
		uintptr(len(colors)),
	)
	return brush
}

// gpRadialBrush fades from center to fully transparent at the path boundary.
// focus keeps the core solid before the falloff begins.
func gpRadialBrush(path uintptr, center, edge uint32, focus float64) uintptr {
	var brush uintptr
	if status, _, _ := gdipCreatePathGradientFromPath.Call(path, uintptr(unsafe.Pointer(&brush))); status != gpOk {
		return 0
	}
	gdipSetPathGradientCenterColor.Call(brush, uintptr(center))
	surround := []uint32{edge}
	count := int32(1)
	gdipSetPathGradientSurroundColorsWithCount.Call(
		brush,
		uintptr(unsafe.Pointer(&surround[0])),
		uintptr(unsafe.Pointer(&count)),
	)
	if focus > 0 {
		gdipSetPathGradientFocusScales.Call(brush, gpReal(focus), gpReal(focus))
	}
	return brush
}

func gpDeleteBrush(brush uintptr) {
	if brush != 0 {
		gdipDeleteBrush.Call(brush)
	}
}

func gpPen(color uint32, width float64) uintptr {
	var pen uintptr
	if status, _, _ := gdipCreatePen1.Call(uintptr(color), gpReal(width), unitPixel, uintptr(unsafe.Pointer(&pen))); status != gpOk {
		return 0
	}
	gdipSetPenStartCap.Call(pen, lineCapRound)
	gdipSetPenEndCap.Call(pen, lineCapRound)
	gdipSetPenLineJoin.Call(pen, lineJoinRound)
	return pen
}

func gpBrushPen(brush uintptr, width float64) uintptr {
	var pen uintptr
	if status, _, _ := gdipCreatePen2.Call(brush, gpReal(width), unitPixel, uintptr(unsafe.Pointer(&pen))); status != gpOk {
		return 0
	}
	gdipSetPenStartCap.Call(pen, lineCapRound)
	gdipSetPenEndCap.Call(pen, lineCapRound)
	gdipSetPenLineJoin.Call(pen, lineJoinRound)
	return pen
}

func gpDeletePen(pen uintptr) {
	if pen != 0 {
		gdipDeletePen.Call(pen)
	}
}

// --- drawing -----------------------------------------------------------------

func gpFillPath(graphics, brush, path uintptr) {
	if graphics != 0 && brush != 0 && path != 0 {
		gdipFillPath.Call(graphics, brush, path)
	}
}

func gpFillPathColor(graphics, path uintptr, color uint32) {
	brush := gpSolidBrush(color)
	gpFillPath(graphics, brush, path)
	gpDeleteBrush(brush)
}

func gpStrokePath(graphics, pen, path uintptr) {
	if graphics != 0 && pen != 0 && path != 0 {
		gdipDrawPath.Call(graphics, pen, path)
	}
}

func gpStrokePathColor(graphics, path uintptr, color uint32, width float64) {
	pen := gpPen(color, width)
	gpStrokePath(graphics, pen, path)
	gpDeletePen(pen)
}

func gpDrawArc(graphics, pen uintptr, x, y, w, h, start, sweep float64) {
	if graphics == 0 || pen == 0 {
		return
	}
	gdipDrawArc.Call(graphics, pen, gpReal(x), gpReal(y), gpReal(w), gpReal(h), gpReal(start), gpReal(sweep))
}

// --- text --------------------------------------------------------------------

func gpFontFamily(name string) uintptr {
	value, err := windows.UTF16FromString(name)
	if err != nil || len(value) == 0 {
		return 0
	}
	var family uintptr
	if status, _, _ := gdipCreateFontFamilyFromName.Call(
		uintptr(unsafe.Pointer(&value[0])), 0, uintptr(unsafe.Pointer(&family)),
	); status != gpOk {
		return 0
	}
	return family
}

func gpDeleteFontFamily(family uintptr) {
	if family != 0 {
		gdipDeleteFontFamily.Call(family)
	}
}

func gpFont(family uintptr, size float64, style int) uintptr {
	if family == 0 || size <= 0 {
		return 0
	}
	var font uintptr
	if status, _, _ := gdipCreateFont.Call(
		family, gpReal(size), uintptr(style), unitPixel, uintptr(unsafe.Pointer(&font)),
	); status != gpOk {
		return 0
	}
	return font
}

func gpDeleteFont(font uintptr) {
	if font != 0 {
		gdipDeleteFont.Call(font)
	}
}

func gpStringFormat(horizontal, vertical int) uintptr {
	var format uintptr
	if status, _, _ := gdipCreateStringFormat.Call(0, 0, uintptr(unsafe.Pointer(&format))); status != gpOk {
		return 0
	}
	gdipSetStringFormatAlign.Call(format, uintptr(horizontal))
	gdipSetStringFormatLineAlign.Call(format, uintptr(vertical))
	return format
}

func gpDeleteStringFormat(format uintptr) {
	if format != 0 {
		gdipDeleteStringFormat.Call(format)
	}
}

func gpDrawText(graphics uintptr, text string, font uintptr, rect gpRectF, color uint32, horizontal, vertical int) {
	if graphics == 0 || font == 0 || text == "" {
		return
	}
	value, err := windows.UTF16FromString(text)
	if err != nil || len(value) <= 1 {
		return
	}
	format := gpStringFormat(horizontal, vertical)
	brush := gpSolidBrush(color)
	if format != 0 && brush != 0 {
		gdipDrawString.Call(
			graphics, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), font,
			uintptr(unsafe.Pointer(&rect)), format, brush,
		)
	}
	gpDeleteBrush(brush)
	gpDeleteStringFormat(format)
}

func gpMeasureText(graphics uintptr, text string, font uintptr) (float64, float64) {
	if graphics == 0 || font == 0 || text == "" {
		return 0, 0
	}
	value, err := windows.UTF16FromString(text)
	if err != nil || len(value) <= 1 {
		return 0, 0
	}
	format := gpStringFormat(stringAlignmentNear, stringAlignmentNear)
	if format == 0 {
		return 0, 0
	}
	defer gpDeleteStringFormat(format)
	layout := gpRectF{Width: 4096, Height: 512}
	var bounds gpRectF
	var fitted, lines int32
	status, _, _ := gdipMeasureString.Call(
		graphics, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), font,
		uintptr(unsafe.Pointer(&layout)), format, uintptr(unsafe.Pointer(&bounds)),
		uintptr(unsafe.Pointer(&fitted)), uintptr(unsafe.Pointer(&lines)),
	)
	if status != gpOk {
		return 0, 0
	}
	return float64(bounds.Width), float64(bounds.Height)
}

// gpGlow approximates a Gaussian bloom by stacking concentric strokes of one
// path. Coverage falls off with distance because progressively fewer strokes
// reach outward, and the per-layer alpha is solved so the innermost
// accumulation lands on peak instead of saturating.
func gpGlow(graphics, path uintptr, rgb uint32, maxWidth float64, layers int, peak float64) {
	peak = clamp01(peak)
	if graphics == 0 || path == 0 || layers < 1 || maxWidth <= 0 || peak <= 0 {
		return
	}
	layerAlpha := 1 - math.Pow(1-peak, 1/float64(layers))
	color := argbColor(rgb, layerAlpha)
	for i := layers; i >= 1; i-- {
		width := maxWidth * float64(i) / float64(layers)
		pen := gpPen(color, width)
		gpStrokePath(graphics, pen, path)
		gpDeletePen(pen)
	}
}

func gpTranslated(graphics uintptr, dx, dy float64, draw func()) {
	var state uint32
	gdipSaveGraphics.Call(graphics, uintptr(unsafe.Pointer(&state)))
	gdipTranslateWorldTransform.Call(graphics, gpReal(dx), gpReal(dy), matrixOrderPrepend)
	draw()
	gdipRestoreGraphics.Call(graphics, uintptr(state))
}

func gpClipped(graphics, path uintptr, draw func()) {
	gdipSetClipPath.Call(graphics, path, combineModeReplace)
	draw()
	gdipResetClip.Call(graphics)
}
