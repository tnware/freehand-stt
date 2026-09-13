//go:build darwin && cgo

#ifndef FH_OVERLAY_DARWIN_H
#define FH_OVERLAY_DARWIN_H
#include <stdint.h>

typedef struct { double x,y,w,h; } fh_overlay_rect;
typedef struct {
 int visible,layout,surface,visualizer,icon,stage,animated,captions,timer_ms;
 int checkpoint_count; // Bounded to 0..5; detailed text retains the full count.
 uint32_t accent,background,animation_ms;
 double x,y,width,height,scale,opacity,glow,progress;
 double levels[17];
 char caption[721],shortcut[193],phase[96],instruction[160],elapsed[32],checkpoints[32];
} fh_overlay_frame;

void *fh_overlay_create(uint64_t identifier);
void fh_overlay_wake(void *bridge);
int fh_overlay_close(void *bridge);
uint32_t fh_overlay_capture_display(void);
void fh_overlay_work_area(uint32_t display,fh_overlay_rect *rect);
void fh_overlay_snapshot(uint64_t identifier,int reduced,int contrast,fh_overlay_frame *frame);
#endif
