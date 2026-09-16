//go:build windows

package overlay

import (
	"sync"
	"syscall"

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

var user32 = windows.NewLazySystemDLL("user32.dll")
var kernel32 = windows.NewLazySystemDLL("kernel32.dll")
var getForegroundWindow = user32.NewProc("GetForegroundWindow")
var getMessage = user32.NewProc("GetMessageW")
var postThreadMessage = user32.NewProc("PostThreadMessageW")
var getCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")

type nativeMessage struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   struct{ X, Y int32 }
	Private uint32
}

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
