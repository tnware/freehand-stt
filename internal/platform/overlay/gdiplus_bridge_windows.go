//go:build windows

package overlay

/*
#include <stdint.h>
#include <windows.h>

// GDI+ flat API signatures keep REAL arguments typed, including on ARM64.
// Addresses come only from Go's System32-only loader, never the default DLL search.
typedef int (WINAPI *gp_arc_fn)(void*, float, float, float, float, float, float);
typedef int (WINAPI *gp_rect_fn)(void*, float, float, float, float);
typedef int (WINAPI *gp_focus_fn)(void*, float, float);
typedef int (WINAPI *gp_pen_fn)(uint32_t, float, int, void**);
typedef int (WINAPI *gp_brush_pen_fn)(void*, float, int, void**);
typedef int (WINAPI *gp_draw_arc_fn)(void*, void*, float, float, float, float, float, float);
typedef int (WINAPI *gp_font_fn)(void*, float, int, int, void**);
typedef int (WINAPI *gp_translate_fn)(void*, float, float, int);

static int gp_arc(uintptr_t fn, uintptr_t path, float x, float y, float w, float h, float start, float sweep) {
    return ((gp_arc_fn)fn)((void*)path, x, y, w, h, start, sweep);
}
static int gp_line(uintptr_t fn, uintptr_t path, float x1, float y1, float x2, float y2) {
    return ((gp_rect_fn)fn)((void*)path, x1, y1, x2, y2);
}
static int gp_ellipse(uintptr_t fn, uintptr_t path, float x, float y, float w, float h) {
    return ((gp_rect_fn)fn)((void*)path, x, y, w, h);
}
static int gp_focus(uintptr_t fn, uintptr_t brush, float focus) {
    return ((gp_focus_fn)fn)((void*)brush, focus, focus);
}
static int gp_pen(uintptr_t fn, uint32_t color, float width, int unit, uintptr_t* result) {
    void* pen = NULL;
    int status = ((gp_pen_fn)fn)(color, width, unit, &pen);
    *result = (uintptr_t)pen;
    return status;
}
static int gp_brush_pen(uintptr_t fn, uintptr_t brush, float width, int unit, uintptr_t* result) {
    void* pen = NULL;
    int status = ((gp_brush_pen_fn)fn)((void*)brush, width, unit, &pen);
    *result = (uintptr_t)pen;
    return status;
}
static int gp_draw_arc(uintptr_t fn, uintptr_t graphics, uintptr_t pen, float x, float y, float w, float h, float start, float sweep) {
    return ((gp_draw_arc_fn)fn)((void*)graphics, (void*)pen, x, y, w, h, start, sweep);
}
static int gp_font(uintptr_t fn, uintptr_t family, float size, int style, int unit, uintptr_t* result) {
    void* font = NULL;
    int status = ((gp_font_fn)fn)((void*)family, size, style, unit, &font);
    *result = (uintptr_t)font;
    return status;
}
static int gp_translate(uintptr_t fn, uintptr_t graphics, float dx, float dy, int order) {
    return ((gp_translate_fn)fn)((void*)graphics, dx, dy, order);
}
*/
import "C"

import "unsafe"

func gpNativeArc(path uintptr, x, y, w, h, start, sweep float64) {
	C.gp_arc(C.uintptr_t(gdipAddPathArc.Addr()), C.uintptr_t(path), C.float(x), C.float(y), C.float(w), C.float(h), C.float(start), C.float(sweep))
}

func gpNativeLine(path uintptr, x1, y1, x2, y2 float64) {
	C.gp_line(C.uintptr_t(gdipAddPathLine.Addr()), C.uintptr_t(path), C.float(x1), C.float(y1), C.float(x2), C.float(y2))
}

func gpNativeEllipse(path uintptr, x, y, w, h float64) {
	C.gp_ellipse(C.uintptr_t(gdipAddPathEllipse.Addr()), C.uintptr_t(path), C.float(x), C.float(y), C.float(w), C.float(h))
}

func gpNativeFocus(brush uintptr, focus float64) {
	C.gp_focus(C.uintptr_t(gdipSetPathGradientFocusScales.Addr()), C.uintptr_t(brush), C.float(focus))
}

func gpNativePen(color uint32, width float64, pen *uintptr) int {
	return int(C.gp_pen(C.uintptr_t(gdipCreatePen1.Addr()), C.uint32_t(color), C.float(width), unitPixel, (*C.uintptr_t)(unsafe.Pointer(pen))))
}

func gpNativeBrushPen(brush uintptr, width float64, pen *uintptr) int {
	return int(C.gp_brush_pen(C.uintptr_t(gdipCreatePen2.Addr()), C.uintptr_t(brush), C.float(width), unitPixel, (*C.uintptr_t)(unsafe.Pointer(pen))))
}

func gpNativeDrawArc(graphics, pen uintptr, x, y, w, h, start, sweep float64) {
	C.gp_draw_arc(C.uintptr_t(gdipDrawArc.Addr()), C.uintptr_t(graphics), C.uintptr_t(pen), C.float(x), C.float(y), C.float(w), C.float(h), C.float(start), C.float(sweep))
}

func gpNativeFont(family uintptr, size float64, style int, font *uintptr) int {
	return int(C.gp_font(C.uintptr_t(gdipCreateFont.Addr()), C.uintptr_t(family), C.float(size), C.int(style), unitPixel, (*C.uintptr_t)(unsafe.Pointer(font))))
}

func gpNativeTranslate(graphics uintptr, dx, dy float64) {
	C.gp_translate(C.uintptr_t(gdipTranslateWorldTransform.Addr()), C.uintptr_t(graphics), C.float(dx), C.float(dy), matrixOrderPrepend)
}
