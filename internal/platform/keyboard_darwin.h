//go:build darwin

#ifndef FH_KEYBOARD_DARWIN_H
#define FH_KEYBOARD_DARWIN_H
#include <stdint.h>
typedef struct fh_keyboard fh_keyboard;
typedef struct { uint16_t code; uint8_t down; uint8_t lost; } fh_keyboard_event;
int fh_keyboard_authorized(void);
int fh_keyboard_capture_authorized(void);
fh_keyboard *fh_keyboard_new(int capture);
void fh_keyboard_retain(fh_keyboard *k);
void fh_keyboard_release(fh_keyboard *k);
void fh_keyboard_run(fh_keyboard *k);
int fh_keyboard_ready(fh_keyboard *k);
void fh_keyboard_stop(fh_keyboard *k);
int fh_keyboard_read(fh_keyboard *k, fh_keyboard_event *event);
// Pure ingress seam shared with the native callback; does not post input.
int fh_keyboard_feed(fh_keyboard *k, uint16_t code, int down, int injected, int repeat);
#endif
