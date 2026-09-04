//go:build windows

package platform

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/audio"
	"golang.org/x/sys/windows"
)

const (
	wmClose                   = 0x0010
	wmDestroy                 = 0x0002
	wmPaint                   = 0x000F
	wmEraseBkgnd              = 0x0014
	wmSettingChange           = 0x001A
	wmNCHitTest               = 0x0084
	wmMouseActivate           = 0x0021
	wmTimer                   = 0x0113
	wmDPIChanged              = 0x02E0
	wmOverlayApply            = 0x8000 + 41
	wmOverlayOptions          = 0x8000 + 42
	overlayTimerID            = 1
	overlayFrameMS            = 16
	maNoActivate              = 3
	swHide                    = 0
	swShowNoActivate          = 4
	wsPopup                   = 0x80000000
	swpNoSize                 = 0x0001
	swpNoMove                 = 0x0002
	swpNoActivate             = 0x0010
	swpShowWindow             = 0x0040
	monitorDefault            = 2
	monitorPrimary            = 1
	ulwAlpha                  = 0x00000002
	acSrcOver                 = 0x00
	acSrcAlpha                = 0x01
	biRGB                     = 0
	dibRGBColors              = 0
	spiGetClientAreaAnimation = 0x1042
	spiGetHighContrast        = 0x0042
	hcfHighContrastOn         = 0x00000001
	colorWindow               = 5
	colorWindowText           = 8
	colorHighlight            = 13
	colorHighlightText        = 14
)

// Overlay layout, in logical pixels at 96 dpi and scaled to device pixels at
// paint time. The window is deliberately larger than the capsule so the drop
// shadow and the accent bloom have somewhere to fall off to; everything outside
// the capsule is transparent, and WS_EX_TRANSPARENT keeps the padding from
// affecting hit testing.
const (
	overlayPillWidth  = 208.0
	overlayPillHeight = 52.0
	overlayPadding    = 24.0
	overlayRise       = 10.0

	overlayIconX      = 33.0
	overlayIconBox    = 20.0
	overlayDividerX   = 57.0
	overlayStageLeft  = 71.0
	overlayStageRight = 22.0
	overlayStageHigh  = 26.0
)

var (
	registerClassEx      = user32.NewProc("RegisterClassExW")
	unregisterClass      = user32.NewProc("UnregisterClassW")
	createWindowEx       = user32.NewProc("CreateWindowExW")
	defWindowProc        = user32.NewProc("DefWindowProcW")
	destroyWindow        = user32.NewProc("DestroyWindow")
	postMessage          = user32.NewProc("PostMessageW")
	translateMessage     = user32.NewProc("TranslateMessage")
	dispatchMessage      = user32.NewProc("DispatchMessageW")
	postQuitMessage      = user32.NewProc("PostQuitMessage")
	showWindow           = user32.NewProc("ShowWindow")
	setWindowPos         = user32.NewProc("SetWindowPos")
	updateLayeredWindow  = user32.NewProc("UpdateLayeredWindow")
	monitorFromWindow    = user32.NewProc("MonitorFromWindow")
	getMonitorInfo       = user32.NewProc("GetMonitorInfoW")
	getDPIForWindow      = user32.NewProc("GetDpiForWindow")
	beginPaint           = user32.NewProc("BeginPaint")
	endPaint             = user32.NewProc("EndPaint")
	setTimer             = user32.NewProc("SetTimer")
	killTimer            = user32.NewProc("KillTimer")
	systemParametersInfo = user32.NewProc("SystemParametersInfoW")
	getSysColor          = user32.NewProc("GetSysColor")

	getModuleHandle = kernel32.NewProc("GetModuleHandleW")

	gdi32              = windows.NewLazySystemDLL("gdi32.dll")
	createCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	createDIBSection   = gdi32.NewProc("CreateDIBSection")
	selectObject       = gdi32.NewProc("SelectObject")
	deleteObject       = gdi32.NewProc("DeleteObject")
	deleteDC           = gdi32.NewProc("DeleteDC")

	shcore           = windows.NewLazySystemDLL("shcore.dll")
	getDPIForMonitor = shcore.NewProc("GetDpiForMonitor")
)

var overlayClassName = windows.StringToUTF16Ptr("Freehand.NativeStatusOverlay.v1")
var overlayWindowProc = syscall.NewCallback(statusOverlayWindowProc)
var overlayWindows sync.Map

type overlayWindowClass struct {
	Size       uint32
	Style      uint32
	WindowProc uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	SmallIcon  uintptr
}

type nativeRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type nativePoint struct {
	X int32
	Y int32
}

type nativeSize struct {
	CX int32
	CY int32
}

type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type monitorInfo struct {
	Size    uint32
	Monitor nativeRect
	Work    nativeRect
	Flags   uint32
}

type paintStruct struct {
	DC        uintptr
	Erase     int32
	Paint     nativeRect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

type highContrastInfo struct {
	Size          uint32
	Flags         uint32
	DefaultScheme *uint16
}

type overlayFontKey struct {
	size  int
	style int
}

// overlaySurface is the premultiplied-ARGB layered surface. GDI+ draws straight
// into the DIB bits and UpdateLayeredWindow presents them, so there is no
// intermediate copy and no window region clip: the rounded corners, the shadow
// and the bloom are all real per-pixel alpha.
type overlaySurface struct {
	dc       uintptr
	bitmap   uintptr
	bits     unsafe.Pointer
	gpBitmap uintptr
	graphics uintptr
	width    int32
	height   int32
}

func (s *overlaySurface) ensure(width, height int32) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	if s.graphics != 0 && s.width == width && s.height == height {
		return true
	}
	s.release()

	dc, _, _ := createCompatibleDC.Call(0)
	if dc == 0 {
		return false
	}
	header := bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       width,
		Height:      -height, // top-down, so GDI+ and the DIB agree on row order
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}
	var bits unsafe.Pointer
	bitmap, _, _ := createDIBSection.Call(
		dc, uintptr(unsafe.Pointer(&header)), dibRGBColors,
		uintptr(unsafe.Pointer(&bits)), 0, 0,
	)
	if bitmap == 0 || bits == nil {
		deleteDC.Call(dc)
		return false
	}
	selectObject.Call(dc, bitmap)

	gpBitmap := gpBitmapOverMemory(width, height, bits)
	if gpBitmap == 0 {
		deleteObject.Call(bitmap)
		deleteDC.Call(dc)
		return false
	}
	graphics := gpGraphicsFromImage(gpBitmap)
	if graphics == 0 {
		gdipDisposeImage.Call(gpBitmap)
		deleteObject.Call(bitmap)
		deleteDC.Call(dc)
		return false
	}

	s.dc, s.bitmap, s.bits = dc, bitmap, bits
	s.gpBitmap, s.graphics = gpBitmap, graphics
	s.width, s.height = width, height
	return true
}

func (s *overlaySurface) release() {
	if s.graphics != 0 {
		gdipDeleteGraphics.Call(s.graphics)
	}
	if s.gpBitmap != 0 {
		gdipDisposeImage.Call(s.gpBitmap)
	}
	if s.bitmap != 0 {
		deleteObject.Call(s.bitmap)
	}
	if s.dc != 0 {
		deleteDC.Call(s.dc)
	}
	*s = overlaySurface{}
}

// overlayAnimation is owned exclusively by the message-loop thread.
type overlayAnimation struct {
	shown            bool
	hiding           bool
	showAt           time.Time
	hideAt           time.Time
	morphAt          time.Time
	target           overlayView
	from             overlayView
	lastView         overlayView
	workArea         nativeRect
	anchorValid      bool
	anchorGeneration uint64
	dpi              uint32
}

// StatusOverlay owns one native Win32 tool window on a dedicated message-loop
// thread. The window is presentation-only and never owns focus or insertion.
type StatusOverlay struct {
	mu        sync.RWMutex
	view      overlayView
	options   OverlayOptions
	hwnd      atomic.Uintptr
	threadID  atomic.Uint32
	closed    atomic.Bool
	closeOnce sync.Once
	done      chan struct{}

	surface           overlaySurface
	anim              overlayAnimation
	animationsEnabled bool
	highContrast      bool
	fontFamily        uintptr
	fonts             map[overlayFontKey]uintptr

	// Live capture amplitude. The source is set once during composition; the
	// ring, envelope and scratch buffer belong to the message-loop thread.
	source   atomic.Value
	levels   *audio.LevelRing
	envelope audio.Envelope
	scratch  []float64
	sampled  time.Time
}

func (o *StatusOverlay) font(size float64, style int) uintptr {
	if o.fontFamily == 0 {
		o.fontFamily = gpFontFamily("Segoe UI")
		if o.fontFamily == 0 {
			return 0
		}
	}
	key := overlayFontKey{size: int(math.Round(size * 10)), style: style}
	if font := o.fonts[key]; font != 0 {
		return font
	}
	font := gpFont(o.fontFamily, float64(key.size)/10, style)
	if font != 0 {
		o.fonts[key] = font
	}
	return font
}

func (o *StatusOverlay) releaseFonts() {
	for key, font := range o.fonts {
		gpDeleteFont(font)
		delete(o.fonts, key)
	}
	gpDeleteFontFamily(o.fontFamily)
	o.fontFamily = 0
}

// SetLevelSource attaches capture amplitude to the recording meter. Without
// one the meter animates from the clock, so the overlay degrades to its
// previous behaviour rather than going flat.
func (o *StatusOverlay) SetLevelSource(source LevelSource) {
	if o == nil || source == nil {
		return
	}
	o.source.Store(source)
}

func (o *StatusOverlay) levelSource() LevelSource {
	if o == nil {
		return nil
	}
	source, _ := o.source.Load().(LevelSource)
	return source
}

// sampleLevels advances the meter. Readings are taken on their own cadence
// rather than once per frame, so the history scrolls at a fixed rate whatever
// the paint rate is.
func (o *StatusOverlay) sampleLevels(now time.Time, recording bool) {
	source := o.levelSource()
	if source == nil {
		return
	}
	if o.levels == nil {
		o.levels = audio.NewLevelRing(overlayLevelBars)
		o.envelope = audio.NewEnvelope()
	}
	if !recording {
		// Draining keeps a peak captured mid-transition from appearing at the
		// head of the next recording.
		source.TakeLevel()
		o.levels.Reset()
		o.envelope.Reset()
		o.sampled = time.Time{}
		return
	}
	if !o.sampled.IsZero() && now.Sub(o.sampled) < overlayLevelMS*time.Millisecond {
		return
	}
	o.sampled = now
	// Decibels first, so the envelope smooths what the eye will see rather
	// than raw amplitude.
	o.levels.Push(o.envelope.Push(audio.NormalizeLevel(source.TakeLevel())))
}

// liveLevels returns the meter history, or nil when the meter is not driving
// this frame.
func (o *StatusOverlay) liveLevels(recording bool) []float64 {
	if !recording || o.levels == nil || o.levelSource() == nil {
		return nil
	}
	o.scratch = o.levels.Snapshot(o.scratch)
	return o.scratch
}

func NewStatusOverlay() (*StatusOverlay, error) {
	o := &StatusOverlay{
		done:              make(chan struct{}),
		options:           DefaultOverlayOptions(),
		animationsEnabled: clientAreaAnimationsEnabled(),
		highContrast:      highContrastEnabled(),
		fonts:             make(map[overlayFontKey]uintptr),
	}
	ready := make(chan error, 1)
	go o.loop(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return o, nil
}

// clientAreaAnimationsEnabled follows the Windows Animation Effects setting.
// If the preference cannot be read, retain the existing animated behaviour.
func clientAreaAnimationsEnabled() bool {
	enabled := int32(1)
	result, _, _ := systemParametersInfo.Call(
		spiGetClientAreaAnimation,
		0,
		uintptr(unsafe.Pointer(&enabled)),
		0,
	)
	return result == 0 || enabled != 0
}

func highContrastEnabled() bool {
	info := highContrastInfo{Size: uint32(unsafe.Sizeof(highContrastInfo{}))}
	result, _, _ := systemParametersInfo.Call(
		spiGetHighContrast,
		uintptr(info.Size),
		uintptr(unsafe.Pointer(&info)),
		0,
	)
	return result != 0 && info.Flags&hcfHighContrastOn != 0
}

// Configure updates independent presentation options on the native message
// loop. It never recreates or directly manipulates the HWND from the caller's
// thread.
func (o *StatusOverlay) Configure(options OverlayOptions) error {
	if o == nil || o.closed.Load() {
		return errors.New("native status overlay is closed")
	}
	o.mu.Lock()
	o.options = normalizeOverlayOptions(options)
	o.mu.Unlock()
	hwnd := o.hwnd.Load()
	if hwnd == 0 {
		return errors.New("native status overlay window is unavailable")
	}
	result, _, callErr := postMessage.Call(hwnd, wmOverlayOptions, 0, 0)
	if result == 0 {
		return callFailure("native status overlay options could not be queued", callErr)
	}
	return nil
}

func (o *StatusOverlay) Update(status OverlayStatus) error {
	if o == nil || o.closed.Load() {
		return errors.New("native status overlay is closed")
	}
	o.mu.Lock()
	o.view = resolveOverlayView(status)
	o.mu.Unlock()
	hwnd := o.hwnd.Load()
	if hwnd == 0 {
		return errors.New("native status overlay window is unavailable")
	}
	result, _, callErr := postMessage.Call(hwnd, wmOverlayApply, 0, 0)
	if result == 0 {
		return callFailure("native status overlay update could not be queued", callErr)
	}
	return nil
}

func (o *StatusOverlay) Close() error {
	if o == nil {
		return nil
	}
	var closeErr error
	o.closeOnce.Do(func() {
		o.closed.Store(true)
		hwnd := o.hwnd.Load()
		if hwnd != 0 {
			if result, _, callErr := postMessage.Call(hwnd, wmClose, 0, 0); result == 0 {
				closeErr = callFailure("native status overlay close could not be queued", callErr)
				if threadID := o.threadID.Load(); threadID != 0 {
					postThreadMessage.Call(uintptr(threadID), 0x0012, 0, 0)
				}
			}
		}
		<-o.done
	})
	return closeErr
}

func (o *StatusOverlay) snapshot() overlayView {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.view
}

func (o *StatusOverlay) optionSnapshot() OverlayOptions {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.options.Scale == 0 {
		return DefaultOverlayOptions()
	}
	return o.options
}

func (o *StatusOverlay) loop(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := getCurrentThreadID.Call()
	o.threadID.Store(uint32(threadID))
	if err := gdiplusInit(); err != nil {
		ready <- err
		close(o.done)
		return
	}
	instance, _, instanceErr := getModuleHandle.Call(0)
	if instance == 0 {
		ready <- callFailure("native status overlay module handle is unavailable", instanceErr)
		close(o.done)
		return
	}
	class := overlayWindowClass{
		Size:       uint32(unsafe.Sizeof(overlayWindowClass{})),
		WindowProc: overlayWindowProc,
		Instance:   instance,
		ClassName:  overlayClassName,
	}
	if result, _, classErr := registerClassEx.Call(uintptr(unsafe.Pointer(&class))); result == 0 {
		ready <- callFailure("native status overlay window class could not be registered", classErr)
		close(o.done)
		return
	}

	title := windows.StringToUTF16Ptr("Freehand status")
	hwnd, _, createErr := createWindowEx.Call(
		nativeOverlayExtendedStyle(),
		uintptr(unsafe.Pointer(overlayClassName)),
		uintptr(unsafe.Pointer(title)),
		wsPopup,
		0, 0, 1, 1,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		unregisterClass.Call(uintptr(unsafe.Pointer(overlayClassName)), instance)
		ready <- callFailure("native status overlay window could not be created", createErr)
		close(o.done)
		return
	}
	o.hwnd.Store(hwnd)
	overlayWindows.Store(hwnd, o)
	ready <- nil

	var message nativeMessage
	for {
		result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
	overlayWindows.Delete(hwnd)
	o.hwnd.Store(0)
	unregisterClass.Call(uintptr(unsafe.Pointer(overlayClassName)), instance)
	close(o.done)
}

func statusOverlayWindowProc(hwnd uintptr, message uint32, wparam, lparam uintptr) uintptr {
	switch message {
	case wmOverlayApply:
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).apply(hwnd)
		}
		return 0
	case wmOverlayOptions:
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).reconfigure(hwnd)
		}
		return 0
	case wmTimer:
		if wparam == overlayTimerID {
			if value, ok := overlayWindows.Load(hwnd); ok {
				value.(*StatusOverlay).tick(hwnd)
			}
		}
		return 0
	case wmPaint:
		// A layered window presented through UpdateLayeredWindow is not painted
		// from the update region, but the region still has to be validated or
		// Windows keeps resending WM_PAINT.
		var paint paintStruct
		beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
		endPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).render(hwnd)
		}
		return 0
	case wmDPIChanged:
		if value, ok := overlayWindows.Load(hwnd); ok {
			overlay := value.(*StatusOverlay)
			overlay.resolveAnchor(hwnd)
			overlay.render(hwnd)
		}
		return 0
	case wmSettingChange:
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).refreshSystemPreferences(hwnd)
		}
		return 0
	case wmEraseBkgnd:
		return 1
	case wmNCHitTest:
		return ^uintptr(0) // HTTRANSPARENT
	case wmMouseActivate:
		return maNoActivate
	case wmClose:
		destroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		killTimer.Call(hwnd, overlayTimerID)
		if value, ok := overlayWindows.Load(hwnd); ok {
			overlay := value.(*StatusOverlay)
			overlay.surface.release()
			overlay.releaseFonts()
		}
		gdiplusRelease()
		postQuitMessage.Call(0)
		return 0
	}
	result, _, _ := defWindowProc.Call(hwnd, uintptr(message), wparam, lparam)
	return result
}

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

func overlayLayout(layout OverlayLayout, scale, offsetY float64) overlayGeometry {
	pillWidth, pillHeight, radius := overlayPillWidth, overlayPillHeight, overlayPillHeight/2
	iconX, stageLeft, stageRight, stageHigh := overlayIconX, overlayStageLeft, overlayStageRight, overlayStageHigh
	switch layout {
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

func callFailure(prefix string, err error) error {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return errors.New(prefix)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
