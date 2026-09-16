//go:build windows

package overlay

import (
	"math"
	"unsafe"
)

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
