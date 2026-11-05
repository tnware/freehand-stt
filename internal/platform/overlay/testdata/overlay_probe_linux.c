// Isolated Xvfb-only fixture: all focus and selection ownership belong to its
// own synthetic window. No physical input or existing clipboard is accessed.
#include "../overlay_linux.h"
#include <xcb/xcb.h>
#include <xcb/shape.h>
#include <cairo/cairo.h>
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static xcb_atom_t atom(xcb_connection_t *c, const char *name) {
    xcb_intern_atom_reply_t *r=xcb_intern_atom_reply(c,xcb_intern_atom(c,0,strlen(name),name),NULL);
    assert(r); xcb_atom_t a=r->atom; free(r); return a;
}
static void passive(xcb_connection_t *c, xcb_window_t window, xcb_window_t focus, xcb_atom_t clipboard) {
    xcb_get_input_focus_reply_t *f=xcb_get_input_focus_reply(c,xcb_get_input_focus(c),NULL);
    assert(f && f->focus==focus); free(f);
    xcb_get_selection_owner_reply_t *s=xcb_get_selection_owner_reply(c,xcb_get_selection_owner(c,clipboard),NULL);
    assert(s && s->owner==focus); free(s);
    xcb_get_window_attributes_reply_t *a=xcb_get_window_attributes_reply(c,xcb_get_window_attributes(c,window),NULL);
    assert(a && a->override_redirect && a->map_state==XCB_MAP_STATE_VIEWABLE); free(a);
    xcb_shape_get_rectangles_reply_t *r=xcb_shape_get_rectangles_reply(c,xcb_shape_get_rectangles(c,window,XCB_SHAPE_SK_INPUT),NULL);
    assert(r && xcb_shape_get_rectangles_rectangles_length(r)==0); free(r);
    xcb_get_property_reply_t *h=xcb_get_property_reply(c,xcb_get_property(c,0,window,XCB_ATOM_WM_HINTS,XCB_ATOM_WM_HINTS,0,9),NULL);
    assert(h && xcb_get_property_value_length(h)==36);
    uint32_t *hints=xcb_get_property_value(h); assert((hints[0]&1) && hints[1]==0); free(h);
}
static void snapshot(xcb_connection_t *c, xcb_window_t window, const char *path) {
    xcb_get_geometry_reply_t *g=xcb_get_geometry_reply(c,xcb_get_geometry(c,window),NULL);
    assert(g);
    xcb_get_image_reply_t *image=xcb_get_image_reply(c,xcb_get_image(c,XCB_IMAGE_FORMAT_Z_PIXMAP,window,0,0,g->width,g->height,~0u),NULL);
    assert(image && (image->depth==24 || image->depth==32));
    int stride=cairo_format_stride_for_width(CAIRO_FORMAT_RGB24,g->width);
    assert(xcb_get_image_data_length(image)==stride*g->height);
    cairo_surface_t *surface=cairo_image_surface_create_for_data(xcb_get_image_data(image),CAIRO_FORMAT_RGB24,g->width,g->height,stride);
    assert(cairo_surface_write_to_png(surface,path)==CAIRO_STATUS_SUCCESS);
    cairo_surface_destroy(surface); free(image); free(g);
}
int main(int argc, char **argv) {
    int screen=0;
    xcb_connection_t *c=xcb_connect(NULL,&screen); assert(!xcb_connection_has_error(c));
    xcb_screen_iterator_t screens=xcb_setup_roots_iterator(xcb_get_setup(c));
    for(int i=0;i<screen;i++) xcb_screen_next(&screens);
    xcb_screen_t *s=screens.data;
    xcb_window_t target=xcb_generate_id(c);
    uint32_t color=0x345678;
    xcb_create_window(c,s->root_depth,target,s->root,400,300,240,100,0,XCB_WINDOW_CLASS_INPUT_OUTPUT,s->root_visual,XCB_CW_BACK_PIXEL,&color);
    xcb_map_window(c,target);
    xcb_set_input_focus(c,XCB_INPUT_FOCUS_POINTER_ROOT,target,XCB_CURRENT_TIME);
    xcb_atom_t clipboard=atom(c,"CLIPBOARD");
    xcb_set_selection_owner(c,target,clipboard,XCB_CURRENT_TIME);
    uint32_t area[]={100,100,800,500}, desktop=0;
    xcb_change_property(c,XCB_PROP_MODE_REPLACE,s->root,atom(c,"_NET_CURRENT_DESKTOP"),XCB_ATOM_CARDINAL,32,1,&desktop);
    xcb_change_property(c,XCB_PROP_MODE_REPLACE,s->root,atom(c,"_NET_WORKAREA"),XCB_ATOM_CARDINAL,32,4,area);
    xcb_flush(c);
    // Synchronize fixture setup on its own connection.
    xcb_get_input_focus_reply_t *barrier=xcb_get_input_focus_reply(c,xcb_get_input_focus(c),NULL); assert(barrier); free(barrier);
    fh_xoverlay *o=fh_xoverlay_open(); assert(o);
    xcb_window_t window=fh_xoverlay_window(o);
    fh_xoverlay_frame hidden={0};
    assert(fh_xoverlay_draw(o,&hidden));
    xcb_get_window_attributes_reply_t *initial=xcb_get_window_attributes_reply(c,xcb_get_window_attributes(c,window),NULL);
    assert(initial && initial->map_state==XCB_MAP_STATE_UNMAPPED); free(initial);
    fh_xoverlay_frame f={.visible=1,.capture_anchor=1,.scale=1,.opacity=.85,.glow=.7,.accent=0x4d8dff,.progress=.7};
    strcpy(f.phase,"Listening"); strcpy(f.instruction,"Release the shortcut to finish");
    strcpy(f.shortcut,"Ctrl+Shift+F8"); strcpy(f.elapsed,"0:08 elapsed"); strcpy(f.checkpoints,"2 checkpoints");
    for(int i=0;i<17;i++) f.levels[i]=(i%5+1)/5.;
    for(int layout=0;layout<4;layout++) for(int anchor=0;anchor<6;anchor++) for(int v=0;v<4;v++) {
        f.visualizer=v; f.layout=layout; f.anchor=anchor; assert(fh_xoverlay_draw(o,&f));
        passive(c,window,target,clipboard);
        xcb_get_geometry_reply_t *g=xcb_get_geometry_reply(c,xcb_get_geometry(c,window),NULL);
        assert(g && g->x>=100 && g->y>=100 && g->x+g->width<=900 && g->y+g->height<=600); free(g);
        f.capture_anchor=0;
    }
    f.visualizer=0; f.layout=3; f.anchor=4; assert(fh_xoverlay_draw(o,&f));
    if(argc==2) snapshot(c,window,argv[1]);
    strcpy(f.caption,"Freehand Linux captions: ✓ 漢字 😀");
    for(int icon=0;icon<7;icon++) for(int stage=0;stage<7;stage++) { f.icon=icon; f.stage=stage; assert(fh_xoverlay_draw(o,&f)); passive(c,window,target,clipboard); }
    // Live work-area changes must invalidate placement without resetting focus.
    uint32_t small[]={8,12,90,42};
    xcb_change_property(c,XCB_PROP_MODE_REPLACE,s->root,atom(c,"_NET_WORKAREA"),XCB_ATOM_CARDINAL,32,4,small);
    barrier=xcb_get_input_focus_reply(c,xcb_get_input_focus(c),NULL); assert(barrier); free(barrier);
    assert(fh_xoverlay_draw(o,&f));
    xcb_get_geometry_reply_t *g=xcb_get_geometry_reply(c,xcb_get_geometry(c,window),NULL);
    assert(g && g->x>=8 && g->y>=12 && g->x+g->width<=98 && g->y+g->height<=54); free(g);
    passive(c,window,target,clipboard);
    f.visible=0; assert(fh_xoverlay_draw(o,&f));
    xcb_get_window_attributes_reply_t *a=xcb_get_window_attributes_reply(c,xcb_get_window_attributes(c,window),NULL);
    assert(a && a->map_state==XCB_MAP_STATE_UNMAPPED); free(a);
    fh_xoverlay_close(o);
    xcb_generic_error_t *error=NULL;
    a=xcb_get_window_attributes_reply(c,xcb_get_window_attributes(c,window),&error);
    assert(!a && error && error->error_code==XCB_WINDOW); free(error);
    xcb_destroy_window(c,target); xcb_flush(c); xcb_disconnect(c);
    puts("PASS: native overlay preserves focus/clipboard, has empty input shape, stays in work area, hides and destroys");
    return 0;
}
