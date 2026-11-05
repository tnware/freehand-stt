//go:build linux && cgo

#include "overlay_linux.h"
#include <xcb/xcb.h>
#include <xcb/shape.h>
#include <xcb/randr.h>
#include <cairo/cairo-xcb.h>
#include <pango/pangocairo.h>
#include <gio/gio.h>
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

struct fh_xoverlay {
    xcb_connection_t *c;
    xcb_screen_t *screen;
    xcb_window_t window;
    xcb_colormap_t colormap;
    cairo_surface_t *surface;
    xcb_atom_t workarea, desktop;
    int mapped, alpha, randr15, anchor_x, anchor_y;
    int width, height, area_dirty, randr_event;
    int work_x, work_y, work_w, work_h;
    double radius;
    int animations, high_contrast;
};

typedef struct { int x, y, w, h; } rectangle;

static int checked(fh_xoverlay *o, xcb_void_cookie_t cookie) {
    xcb_generic_error_t *error = xcb_request_check(o->c, cookie);
    if (error) { free(error); return 0; }
    return !xcb_connection_has_error(o->c);
}
static xcb_atom_t atom(fh_xoverlay *o, const char *name) {
    xcb_intern_atom_reply_t *reply = xcb_intern_atom_reply(o->c,
        xcb_intern_atom(o->c, 0, strlen(name), name), NULL);
    if (!reply) return XCB_ATOM_NONE;
    xcb_atom_t a = reply->atom;
    free(reply);
    return a;
}
static xcb_visualtype_t *visual(fh_xoverlay *o, int argb) {
    xcb_depth_iterator_t depth = xcb_screen_allowed_depths_iterator(o->screen);
    for (; depth.rem; xcb_depth_next(&depth)) {
        xcb_visualtype_iterator_t v = xcb_depth_visuals_iterator(depth.data);
        for (; v.rem; xcb_visualtype_next(&v)) {
            if ((!argb && v.data->visual_id == o->screen->root_visual) ||
                (argb && depth.data->depth == 32 && v.data->_class == XCB_VISUAL_CLASS_TRUE_COLOR &&
                 v.data->red_mask == 0xff0000 && v.data->green_mask == 0xff00 && v.data->blue_mask == 0xff)) return v.data;
        }
    }
    return NULL;
}

fh_xoverlay *fh_xoverlay_open(void) {
    fh_xoverlay *o = calloc(1, sizeof(*o));
    if (!o) return NULL;
    int screen_number = 0;
    o->c = xcb_connect(NULL, &screen_number);
    if (!o->c || xcb_connection_has_error(o->c)) goto failed;
    xcb_screen_iterator_t screens = xcb_setup_roots_iterator(xcb_get_setup(o->c));
    for (int i = 0; i < screen_number && screens.rem; i++) xcb_screen_next(&screens);
    if (!screens.rem) goto failed;
    o->screen = screens.data;
    const xcb_query_extension_reply_t *shape = xcb_get_extension_data(o->c, &xcb_shape_id);
    const xcb_query_extension_reply_t *randr = xcb_get_extension_data(o->c, &xcb_randr_id);
    if (!shape || !shape->present || !randr || !randr->present) goto failed;
    o->randr_event = randr->first_event;
    o->area_dirty = 1;
    xcb_shape_query_version_reply_t *sv = xcb_shape_query_version_reply(o->c, xcb_shape_query_version(o->c), NULL);
    int input_shape = sv && (sv->major_version > 1 || (sv->major_version == 1 && sv->minor_version >= 1));
    free(sv);
    if (!input_shape) goto failed;
    xcb_randr_query_version_reply_t *rv = xcb_randr_query_version_reply(o->c, xcb_randr_query_version(o->c, 1, 5), NULL);
    if (!rv) goto failed;
    o->randr15 = rv->major_version > 1 || (rv->major_version == 1 && rv->minor_version >= 5);
    free(rv);

    char selection[48];
    snprintf(selection, sizeof(selection), "_NET_WM_CM_S%d", screen_number);
    xcb_get_selection_owner_reply_t *owner = xcb_get_selection_owner_reply(o->c,
        xcb_get_selection_owner(o->c, atom(o, selection)), NULL);
    o->alpha = owner && owner->owner != XCB_WINDOW_NONE;
    free(owner);
    xcb_visualtype_t *v = o->alpha ? visual(o, 1) : NULL;
    if (!v) { o->alpha = 0; v = visual(o, 0); }
    if (!v) goto failed;
    o->colormap = xcb_generate_id(o->c);
    if (!checked(o, xcb_create_colormap_checked(o->c, XCB_COLORMAP_ALLOC_NONE, o->colormap, o->screen->root, v->visual_id))) goto failed;
    o->window = xcb_generate_id(o->c);
    uint32_t values[] = {0, 0, 1, XCB_EVENT_MASK_EXPOSURE | XCB_EVENT_MASK_STRUCTURE_NOTIFY, o->colormap};
    if (!checked(o, xcb_create_window_checked(o->c, o->alpha ? 32 : o->screen->root_depth,
        o->window, o->screen->root, 0, 0, 1, 1, 0, XCB_WINDOW_CLASS_INPUT_OUTPUT, v->visual_id,
        XCB_CW_BACK_PIXEL | XCB_CW_BORDER_PIXEL | XCB_CW_OVERRIDE_REDIRECT | XCB_CW_EVENT_MASK | XCB_CW_COLORMAP, values))) goto failed;
    // Input shape is empty BEFORE the window can ever be mapped. No keyboard,
    // input-selection, focus-changing, grab or clipboard-changing requests exist.
    if (!checked(o, xcb_shape_rectangles_checked(o->c, XCB_SHAPE_SO_SET, XCB_SHAPE_SK_INPUT,
        XCB_CLIP_ORDERING_UNSORTED, o->window, 0, 0, 0, NULL))) goto failed;
    uint32_t hints[9] = {1, 0}; // ICCCM InputHint, input=False.
    if (!checked(o, xcb_change_property_checked(o->c, XCB_PROP_MODE_REPLACE, o->window,
        XCB_ATOM_WM_HINTS, XCB_ATOM_WM_HINTS, 32, 9, hints))) goto failed;
    const char name[] = "Freehand status";
    xcb_change_property(o->c, XCB_PROP_MODE_REPLACE, o->window, XCB_ATOM_WM_NAME, XCB_ATOM_STRING, 8, sizeof(name)-1, name);
    const char klass[] = "freehand-status\0Freehand\0";
    xcb_change_property(o->c, XCB_PROP_MODE_REPLACE, o->window, XCB_ATOM_WM_CLASS, XCB_ATOM_STRING, 8, sizeof(klass)-1, klass);
    xcb_atom_t type = atom(o, "_NET_WM_WINDOW_TYPE_NOTIFICATION");
    xcb_change_property(o->c, XCB_PROP_MODE_REPLACE, o->window, atom(o, "_NET_WM_WINDOW_TYPE"), XCB_ATOM_ATOM, 32, 1, &type);
    xcb_atom_t states[] = {atom(o, "_NET_WM_STATE_ABOVE"), atom(o, "_NET_WM_STATE_SKIP_TASKBAR"), atom(o, "_NET_WM_STATE_SKIP_PAGER")};
    xcb_change_property(o->c, XCB_PROP_MODE_REPLACE, o->window, atom(o, "_NET_WM_STATE"), XCB_ATOM_ATOM, 32, 3, states);
    o->workarea = atom(o, "_NET_WORKAREA");
    o->desktop = atom(o, "_NET_CURRENT_DESKTOP");
    uint32_t root_events = XCB_EVENT_MASK_PROPERTY_CHANGE;
    if (!checked(o, xcb_change_window_attributes_checked(o->c, o->screen->root, XCB_CW_EVENT_MASK, &root_events))) goto failed;
    if (!checked(o, xcb_randr_select_input_checked(o->c, o->screen->root,
        XCB_RANDR_NOTIFY_MASK_SCREEN_CHANGE | XCB_RANDR_NOTIFY_MASK_CRTC_CHANGE | XCB_RANDR_NOTIFY_MASK_OUTPUT_CHANGE))) goto failed;
    // Optional GNOME desktop preferences, read-only. Missing schemas default
    // to reduced decoration. Never create schemas or alter desktop settings.
    int gnome = 0;
    const char *desktop = g_getenv("XDG_CURRENT_DESKTOP");
    char **desktops = g_strsplit(desktop ? desktop : "", ":", -1);
    for (char **d = desktops; *d; d++) if (g_ascii_strcasecmp(*d, "GNOME") == 0) gnome = 1;
    g_strfreev(desktops);
    GSettingsSchemaSource *schemas = gnome ? g_settings_schema_source_get_default() : NULL;
    GSettingsSchema *schema = schemas ? g_settings_schema_source_lookup(schemas, "org.gnome.desktop.interface", TRUE) : NULL;
    if (schema) {
        GSettings *settings = g_settings_new_full(schema, NULL, NULL);
        if (g_settings_schema_has_key(schema, "enable-animations")) o->animations = g_settings_get_boolean(settings, "enable-animations");
        if (g_settings_schema_has_key(schema, "gtk-theme")) {
            char *theme = g_settings_get_string(settings, "gtk-theme");
            o->high_contrast = g_str_has_prefix(theme, "HighContrast");
            g_free(theme);
        }
        g_object_unref(settings); g_settings_schema_unref(schema);
    }
    o->surface = cairo_xcb_surface_create(o->c, o->window, v, 1, 1);
    if (cairo_surface_status(o->surface) != CAIRO_STATUS_SUCCESS) goto failed;
    if (xcb_flush(o->c) <= 0) goto failed;
    return o;
failed:
    fh_xoverlay_close(o);
    return NULL;
}

void fh_xoverlay_close(fh_xoverlay *o) {
    if (!o) return;
    if (o->surface) cairo_surface_destroy(o->surface);
    if (o->c) {
        if (o->window) xcb_destroy_window(o->c, o->window);
        if (o->colormap) xcb_free_colormap(o->c, o->colormap);
        xcb_flush(o->c);
        xcb_disconnect(o->c);
    }
    free(o);
}
uint32_t fh_xoverlay_window(fh_xoverlay *o) { return o ? o->window : 0; }

static void capture_anchor(fh_xoverlay *o) {
    o->anchor_x = o->screen->width_in_pixels / 2;
    o->anchor_y = o->screen->height_in_pixels / 2;
    xcb_get_input_focus_reply_t *focus = xcb_get_input_focus_reply(o->c, xcb_get_input_focus(o->c), NULL);
    xcb_window_t window = focus ? focus->focus : 0;
    free(focus);
    if (window > XCB_INPUT_FOCUS_POINTER_ROOT && window != o->window) {
        xcb_get_geometry_reply_t *g = xcb_get_geometry_reply(o->c, xcb_get_geometry(o->c, window), NULL);
        xcb_translate_coordinates_reply_t *xy = g ? xcb_translate_coordinates_reply(o->c,
            xcb_translate_coordinates(o->c, window, o->screen->root, g->width/2, g->height/2), NULL) : NULL;
        if (xy) { o->anchor_x = xy->dst_x; o->anchor_y = xy->dst_y; free(xy); free(g); return; }
        free(g);
    }
    xcb_query_pointer_reply_t *pointer = xcb_query_pointer_reply(o->c, xcb_query_pointer(o->c, o->screen->root), NULL);
    if (pointer) { if (pointer->same_screen) { o->anchor_x = pointer->root_x; o->anchor_y = pointer->root_y; } free(pointer); }
}
static rectangle work_area(fh_xoverlay *o) {
    rectangle work = {0, 0, o->screen->width_in_pixels, o->screen->height_in_pixels};
    // Read root geometry again: a monitor reconfiguration can change it.
    xcb_get_geometry_reply_t *root = xcb_get_geometry_reply(o->c, xcb_get_geometry(o->c, o->screen->root), NULL);
    if (root) { work.w = root->width; work.h = root->height; free(root); }
    if (o->randr15) {
        xcb_randr_get_monitors_reply_t *reply = xcb_randr_get_monitors_reply(o->c,
            xcb_randr_get_monitors(o->c, o->screen->root, 1), NULL);
        if (reply) {
            xcb_randr_monitor_info_iterator_t it = xcb_randr_get_monitors_monitors_iterator(reply);
            int found = 0;
            for (int i = 0; it.rem && i < 128; i++, xcb_randr_monitor_info_next(&it)) {
                xcb_randr_monitor_info_t *m = it.data;
                if (!m->width || !m->height) continue;
                rectangle r = {m->x, m->y, m->width, m->height};
                if (o->anchor_x >= r.x && o->anchor_x < r.x+r.w && o->anchor_y >= r.y && o->anchor_y < r.y+r.h) { work = r; found = 1; break; }
                if (m->primary || (!found && i == 0)) work = r;
            }
            free(reply);
        }
    }
    uint32_t desktop = 0;
    xcb_get_property_reply_t *d = xcb_get_property_reply(o->c,
        xcb_get_property(o->c, 0, o->screen->root, o->desktop, XCB_ATOM_CARDINAL, 0, 1), NULL);
    if (d && d->type == XCB_ATOM_CARDINAL && d->format == 32 && xcb_get_property_value_length(d) == 4) desktop = *(uint32_t *)xcb_get_property_value(d);
    free(d);
    if (desktop > 1023) return work;
    xcb_get_property_reply_t *a = xcb_get_property_reply(o->c,
        xcb_get_property(o->c, 0, o->screen->root, o->workarea, XCB_ATOM_CARDINAL, desktop*4, 4), NULL);
    if (a && a->type == XCB_ATOM_CARDINAL && a->format == 32 && xcb_get_property_value_length(a) == 16) {
        uint32_t *r = xcb_get_property_value(a);
        // Reject malformed geometry before signed arithmetic. Normal X11
        // work-area coordinates are bounded by the signed protocol range.
        if (r[0] <= 32767 && r[1] <= 32767 && r[2] > 0 && r[2] <= 65535 && r[3] > 0 && r[3] <= 65535) {
            int x = work.x > (int)r[0] ? work.x : (int)r[0];
            int y = work.y > (int)r[1] ? work.y : (int)r[1];
            int right = work.x+work.w < (int)(r[0]+r[2]) ? work.x+work.w : (int)(r[0]+r[2]);
            int bottom = work.y+work.h < (int)(r[1]+r[3]) ? work.y+work.h : (int)(r[1]+r[3]);
            if (right > x && bottom > y) work = (rectangle){x, y, right-x, bottom-y};
        }
    }
    free(a);
    return work;
}
static void color(cairo_t *cr, uint32_t rgb, double alpha) {
    cairo_set_source_rgba(cr, (rgb>>16&255)/255., (rgb>>8&255)/255., (rgb&255)/255., alpha);
}
static void pill(cairo_t *cr, double x, double y, double w, double h, double r) {
    r = fmin(r, fmin(w,h)/2);
    cairo_new_sub_path(cr);
    cairo_arc(cr, x+w-r, y+r, r, -M_PI/2, 0);
    cairo_arc(cr, x+w-r, y+h-r, r, 0, M_PI/2);
    cairo_arc(cr, x+r, y+h-r, r, M_PI/2, M_PI);
    cairo_arc(cr, x+r, y+r, r, M_PI, M_PI*1.5);
    cairo_close_path(cr);
}
static void text(cairo_t *cr, const char *value, double x, double y, double width, double size, int bold, int tail) {
    PangoLayout *layout = pango_cairo_create_layout(cr);
    PangoFontDescription *font = pango_font_description_new();
    pango_font_description_set_family(font, "Sans");
    pango_font_description_set_absolute_size(font, size*PANGO_SCALE);
    pango_font_description_set_weight(font, bold ? PANGO_WEIGHT_SEMIBOLD : PANGO_WEIGHT_NORMAL);
    pango_layout_set_font_description(layout, font);
    pango_layout_set_text(layout, value, -1); // Never interpret caption markup.
    pango_layout_set_single_paragraph_mode(layout, TRUE);
    pango_layout_set_width(layout, fmax(1,width)*PANGO_SCALE);
    pango_layout_set_ellipsize(layout, tail ? PANGO_ELLIPSIZE_START : PANGO_ELLIPSIZE_END);
    cairo_move_to(cr, x, y);
    pango_cairo_show_layout(cr, layout);
    pango_font_description_free(font);
    g_object_unref(layout);
}
static int shape_window(fh_xoverlay *o, int w, int h, double radius) {
    xcb_rectangle_t *rows = calloc(h, sizeof(*rows));
    if (!rows) return 0;
    for (int y = 0; y < h; y++) {
        double dy = y < radius ? radius-y-.5 : y >= h-radius ? y-(h-radius)+.5 : 0;
        int inset = dy ? (int)ceil(radius-sqrt(fmax(0,radius*radius-dy*dy))) : 0;
        rows[y] = (xcb_rectangle_t){inset, y, w-2*inset, 1};
    }
    int ok = checked(o, xcb_shape_rectangles_checked(o->c, XCB_SHAPE_SO_SET, XCB_SHAPE_SK_BOUNDING,
        XCB_CLIP_ORDERING_YX_BANDED, o->window, 0, 0, h, rows));
    free(rows);
    return ok;
}

int fh_xoverlay_draw(fh_xoverlay *o, const fh_xoverlay_frame *f) {
    if (!o || xcb_connection_has_error(o->c)) return 0;
    fh_xoverlay_frame frame = *f;
    if (!o->animations) frame.animation_ms = 0;
    if (o->high_contrast) { frame.opacity=1; frame.glow=0; frame.surface=1; }
    f = &frame;
    // Bound event draining. Events never trigger input or application actions.
    for (int i=0; i<64; i++) {
        xcb_generic_event_t *event = xcb_poll_for_event(o->c);
        if (!event) break;
        int kind = event->response_type & 0x7f;
        int error = kind == 0;
        if (kind == o->randr_event + XCB_RANDR_SCREEN_CHANGE_NOTIFY || kind == o->randr_event + XCB_RANDR_NOTIFY) o->area_dirty = 1;
        if (kind == XCB_PROPERTY_NOTIFY) {
            xcb_property_notify_event_t *property = (xcb_property_notify_event_t *)event;
            if (property->window == o->screen->root && (property->atom == o->workarea || property->atom == o->desktop)) o->area_dirty = 1;
        }
        free(event);
        if (error) return 0;
    }
    if (!f->visible) {
        // Erase the surface before hiding; no transcript survives in a hidden
        // backing surface. This window is the only drawable ever modified.
        cairo_t *cr = cairo_create(o->surface);
        cairo_set_operator(cr, CAIRO_OPERATOR_CLEAR); cairo_paint(cr); cairo_destroy(cr);
        cairo_surface_flush(o->surface);
        if (o->mapped && !checked(o, xcb_unmap_window_checked(o->c, o->window))) return 0;
        o->mapped = 0;
        return xcb_flush(o->c) > 0;
    }
    if (f->capture_anchor || !o->mapped) { capture_anchor(o); o->area_dirty = 1; }
    if (o->area_dirty) {
        rectangle a = work_area(o);
        o->work_x=a.x; o->work_y=a.y; o->work_w=a.w; o->work_h=a.h;
        o->area_dirty=0;
    }
    rectangle work = {o->work_x, o->work_y, o->work_w, o->work_h};
    if (work.w <= 0 || work.h <= 0) return 0;
    double scale = f->scale;
    double w = 260, h = 48;
    if (f->layout == 0) { w=56; h=56; }
    if (f->layout == 2) { w=304; h=72; }
    if (f->layout == 3) { w=400; h=154; }
    if (f->caption[0]) { w=640; h=84; }
    scale = fmin(scale, fmin(work.w/w, work.h/h));
    int width = fmax(1, floor(w*scale)), height = fmax(1, floor(h*scale));
    int edge = f->edge;
    int x = work.x+(work.w-width)/2;
    if (f->anchor == 0 || f->anchor == 3) x=work.x+edge;
    if (f->anchor == 2 || f->anchor == 5) x=work.x+work.w-width-edge;
    int y = f->anchor <= 2 ? work.y+edge : work.y+work.h-height-edge;
    x=fmax(work.x,fmin(x,work.x+work.w-width));
    y=fmax(work.y,fmin(y,work.y+work.h-height));
    uint32_t geometry[] = {(uint32_t)x,(uint32_t)y,width,height,XCB_STACK_MODE_ABOVE};
    if (!checked(o, xcb_configure_window_checked(o->c, o->window,
        XCB_CONFIG_WINDOW_X | XCB_CONFIG_WINDOW_Y | XCB_CONFIG_WINDOW_WIDTH | XCB_CONFIG_WINDOW_HEIGHT | XCB_CONFIG_WINDOW_STACK_MODE, geometry))) return 0;
    double radius = f->layout == 3 || f->caption[0] ? 18 : h/2;
    if (width != o->width || height != o->height || radius*scale != o->radius) {
        if (!shape_window(o, width, height, fmin(radius*scale,fmin(width,height)/2.))) return 0;
        cairo_xcb_surface_set_size(o->surface, width, height);
        o->width=width; o->height=height; o->radius=radius*scale;
    }
    if (!o->mapped) {
        if (!checked(o, xcb_map_window_checked(o->c, o->window))) return 0;
        o->mapped=1;
    }
    cairo_t *cr=cairo_create(o->surface);
    cairo_set_operator(cr, CAIRO_OPERATOR_SOURCE);
    cairo_set_source_rgba(cr,0,0,0,0); cairo_paint(cr);
    cairo_set_operator(cr, CAIRO_OPERATOR_OVER);
    cairo_scale(cr,scale,scale);
    double opacity = o->alpha ? f->opacity : 1;
    pill(cr,.5,.5,w-1,h-1,radius);
    color(cr,0x1b1b1b,opacity*(f->surface == 0 ? .87 : f->surface == 2 ? .62 : 1)); cairo_fill_preserve(cr);
    color(cr,f->accent,opacity*(.20+.15*f->glow)); cairo_set_line_width(cr,1); cairo_stroke(cr);
    double iconx=26, icony=24;
    if (f->layout == 0 && !f->caption[0]) { iconx=28; icony=28; }
    if (f->glow > 0) {
        cairo_pattern_t *glow=cairo_pattern_create_radial(iconx,icony,0,iconx,icony,28);
        cairo_pattern_add_color_stop_rgba(glow,0,(f->accent>>16&255)/255.,(f->accent>>8&255)/255.,(f->accent&255)/255.,.12*f->glow*opacity);
        cairo_pattern_add_color_stop_rgba(glow,1,0,0,0,0);
        cairo_set_source(cr,glow); cairo_paint(cr); cairo_pattern_destroy(glow);
    }
    color(cr,f->accent,opacity);
    if (f->icon == 1 && f->stage == 4) {
        // Cleanup stays distinct from transcription even in reduced-motion
        // Minimal layout, where no phase label accompanies the glyph.
        cairo_set_line_width(cr,2);
        cairo_arc(cr,iconx-3,icony,5,-M_PI*.75,M_PI*.75); cairo_stroke(cr);
        cairo_arc(cr,iconx+3,icony,5,M_PI*.25,M_PI*1.75); cairo_stroke(cr);
    } else if (f->icon == 1 || f->icon == 6) {
        double start = f->animation_ms ? fmod(f->animation_ms/1400.,1)*2*M_PI : -M_PI/2;
        cairo_arc(cr,iconx,icony,8,start,start+2*M_PI*(f->stage == 6 ? f->progress : .72));
        cairo_set_line_width(cr,2.5); cairo_set_line_cap(cr,CAIRO_LINE_CAP_ROUND); cairo_stroke(cr);
        if (f->icon == 6) { cairo_move_to(cr,iconx,icony-4); cairo_line_to(cr,iconx,icony); cairo_line_to(cr,iconx+3,icony+2); cairo_stroke(cr); }
    } else if (f->icon == 2) {
        cairo_set_line_width(cr,2.5); cairo_move_to(cr,iconx-6,icony); cairo_line_to(cr,iconx-1,icony+5); cairo_line_to(cr,iconx+7,icony-5); cairo_stroke(cr);
    } else if (f->icon == 3) {
        pill(cr,iconx-6,icony-7,12,15,2); cairo_set_line_width(cr,2); cairo_stroke(cr);
        cairo_rectangle(cr,iconx-3,icony-9,6,4); cairo_fill(cr);
    } else if (f->icon == 4) {
        cairo_move_to(cr,iconx,icony-8); cairo_line_to(cr,iconx+8,icony+7); cairo_line_to(cr,iconx-8,icony+7); cairo_close_path(cr);
        cairo_set_line_width(cr,1.8); cairo_stroke(cr);
        cairo_move_to(cr,iconx,icony-3); cairo_line_to(cr,iconx,icony+1); cairo_stroke(cr); cairo_arc(cr,iconx,icony+4,1,0,2*M_PI); cairo_fill(cr);
    } else if (f->icon == 5) {
        cairo_set_line_width(cr,2.5); cairo_move_to(cr,iconx-5,icony-5); cairo_line_to(cr,iconx+5,icony+5); cairo_move_to(cr,iconx+5,icony-5); cairo_line_to(cr,iconx-5,icony+5); cairo_stroke(cr);
    } else {
        pill(cr,iconx-3,icony-8,6,12,3); cairo_fill(cr);
        cairo_arc(cr,iconx,icony+1,6,0,M_PI); cairo_set_line_width(cr,1.8); cairo_stroke(cr);
        cairo_move_to(cr,iconx,icony+7); cairo_line_to(cr,iconx,icony+10); cairo_stroke(cr);
    }
    if (f->layout != 0 || f->caption[0]) {
        color(cr,0xf2f2f2,opacity); text(cr,f->phase,45,14,w-61,14,1,0);
        if (f->caption[0]) { text(cr,f->caption,20,47,w-40,15,0,1); }
        else if (f->layout == 3) {
            color(cr,0xc4c4c4,opacity); text(cr,f->instruction,20,42,w-40,12,0,0);
            char detail[264]; snprintf(detail,sizeof(detail),"%.195s  %.63s",f->shortcut,f->elapsed);
            text(cr,detail,20,65,w-40,11,0,0); text(cr,f->checkpoints,20,84,w-40,11,0,0);
        }
    }
    if (!f->caption[0] && (f->layout == 2 || f->layout == 3)) {
        double bar_y=f->layout == 3 ? 113 : 44, max_h=f->layout == 3 ? 22 : 16;
        double bar_w=(w-40)/17;
        color(cr,f->accent,opacity);
        if (f->stage == 0 || f->stage == 5 || f->stage == 6) {
            if (f->visualizer == 1) {
                double level=fmax(0,fmin(f->levels[16],1));
                cairo_arc(cr,w/2,bar_y+max_h/2,3+level*max_h/2,0,2*M_PI); cairo_fill(cr);
            } else if (f->visualizer == 2) {
                cairo_move_to(cr,20,bar_y+max_h);
                for (int i=0;i<17;i++) cairo_line_to(cr,20+i*(w-40)/16,bar_y+max_h-fmax(1,fmin(f->levels[i],1)*max_h));
                cairo_line_to(cr,w-20,bar_y+max_h); cairo_close_path(cr); cairo_fill(cr);
            } else for (int i=0;i<17;i++) {
                double level=fmax(0,fmin(f->levels[i],1));
                double high=fmax(2,level*max_h);
                if (f->visualizer == 3) high=i < (int)(f->levels[16]*17) ? max_h : 2;
                pill(cr,20+i*bar_w,bar_y+(max_h-high)/2,bar_w-3,high,2); cairo_fill(cr);
            }
        } else {
            double center=f->animation_ms ? .5+.4*sin(f->animation_ms/700.) : .5;
            pill(cr,20+(w-72)*center,bar_y+max_h/2-2,32,4,2); cairo_fill(cr);
        }
    }
    cairo_status_t status=cairo_status(cr);
    cairo_destroy(cr); cairo_surface_flush(o->surface);
    return status == CAIRO_STATUS_SUCCESS && cairo_surface_status(o->surface) == CAIRO_STATUS_SUCCESS && xcb_flush(o->c) > 0;
}
