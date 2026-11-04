#ifndef FREEHAND_OVERLAY_LINUX_H
#define FREEHAND_OVERLAY_LINUX_H
#include <stdint.h>

typedef struct fh_xoverlay fh_xoverlay;
typedef struct {
    int visible, capture_anchor, layout, anchor, surface, visualizer, icon, stage;
    double scale, opacity, glow, progress;
    int edge, animation_ms;
    uint32_t accent;
    double levels[17];
    char phase[64], instruction[128], caption[768], shortcut[196], elapsed[64], checkpoints[64];
} fh_xoverlay_frame;

fh_xoverlay *fh_xoverlay_open(void);
int fh_xoverlay_draw(fh_xoverlay *, const fh_xoverlay_frame *);
void fh_xoverlay_close(fh_xoverlay *);
uint32_t fh_xoverlay_window(fh_xoverlay *);
#endif
