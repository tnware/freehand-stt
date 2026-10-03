//go:build windows

package overlay

import (
	"math"
	"testing"
	"unsafe"
)

func TestGDIPlusCanRestartAfterOverlayShutdown(t *testing.T) {
	if err := gdiplusInit(); err != nil {
		t.Fatal(err)
	}
	if gdiplusToken == 0 {
		t.Fatal("first GDI+ startup returned no token")
	}
	gdiplusRelease()
	if gdiplusToken != 0 {
		t.Fatal("GDI+ shutdown retained its token")
	}

	if err := gdiplusInit(); err != nil {
		t.Fatal(err)
	}
	if gdiplusToken == 0 {
		t.Fatal("GDI+ did not restart after overlay shutdown")
	}
	t.Cleanup(gdiplusRelease)
}

// These are offscreen native API checks, not interactive overlay acceptance.
// Fractional arguments expose incorrect integer/float register allocation.
func TestGDIPlusFloatArgumentsPreserveGeometryAndStyle(t *testing.T) {
	if err := gdiplusInit(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gdiplusRelease)
	for _, test := range []struct {
		name string
		path func() uintptr
		want gpRectF
	}{
		{"ellipse", func() uintptr { return gpEllipsePath(1.25, 2.5, 12.75, 8.25) }, gpRectF{X: 1.25, Y: 2.5, Width: 12.75, Height: 8.25}},
		{"line", func() uintptr { return gpPolylinePath([][2]float64{{1.25, 2.5}, {14, 10.75}}) }, gpRectF{X: 1.25, Y: 2.5, Width: 12.75, Height: 8.25}},
		{"arc", func() uintptr { path := gpNewPath(); gpArc(path, 10, 20, 20, 20, 0, 90); return path }, gpRectF{X: 20, Y: 30, Width: 10, Height: 10}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := test.path()
			if path == 0 {
				t.Fatal("GDI+ returned no path")
			}
			defer gpDeletePath(path)
			var bounds gpRectF
			status, _, _ := gdiplusDLL.NewProc("GdipGetPathWorldBounds").Call(path, uintptr(unsafe.Pointer(&bounds)), 0, 0)
			if status != gpOk {
				t.Fatalf("path bounds failed: %d", status)
			}
			if math.Abs(float64(bounds.X-test.want.X))+math.Abs(float64(bounds.Y-test.want.Y))+math.Abs(float64(bounds.Width-test.want.Width))+math.Abs(float64(bounds.Height-test.want.Height)) > 0.001 {
				t.Fatalf("bounds = %+v, want %+v", bounds, test.want)
			}
		})
	}
	brush := gpSolidBrush(0xffffffff)
	if brush == 0 {
		t.Fatal("GDI+ returned no brush")
	}
	defer gpDeleteBrush(brush)
	for _, pen := range []uintptr{gpPen(0xffffffff, 1.25), gpBrushPen(brush, 1.25)} {
		if pen == 0 {
			t.Fatal("GDI+ returned no pen")
		}
		var width float32
		status, _, _ := gdiplusDLL.NewProc("GdipGetPenWidth").Call(pen, uintptr(unsafe.Pointer(&width)))
		gpDeletePen(pen)
		if status != gpOk || width != 1.25 {
			t.Fatalf("pen width = %v, status = %d", width, status)
		}
	}
	family := gpFontFamily("Arial")
	if family == 0 {
		t.Fatal("GDI+ returned no font family")
	}
	defer gpDeleteFontFamily(family)
	font := gpFont(family, 13.5, fontStyleRegular)
	if font == 0 {
		t.Fatal("GDI+ returned no font")
	}
	defer gpDeleteFont(font)
	var size float32
	status, _, _ := gdiplusDLL.NewProc("GdipGetFontSize").Call(font, uintptr(unsafe.Pointer(&size)))
	if status != gpOk || size != 13.5 {
		t.Fatalf("font size = %v, status = %d", size, status)
	}
}
