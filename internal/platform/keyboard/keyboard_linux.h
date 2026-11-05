#ifndef FH_KEYBOARD_LINUX_H
#define FH_KEYBOARD_LINUX_H
#include <stdint.h>
typedef struct fh_xkeyboard fh_xkeyboard;
typedef struct { uint16_t code, state; int down, lost; } fh_xevent;
fh_xkeyboard *fh_xopen(void);
void fh_xclose(fh_xkeyboard *k);
int fh_xread(fh_xkeyboard *k, fh_xevent *e);
int fh_xgrab(fh_xkeyboard *k, unsigned code, unsigned mask);
int fh_xcapture(fh_xkeyboard *k);
int fh_xheld(fh_xkeyboard *k, uint32_t wire);
int fh_xreleased(fh_xkeyboard *k);
uint32_t fh_xwire(fh_xkeyboard *k, unsigned code);
unsigned fh_xmod(fh_xkeyboard *k, unsigned mod);
unsigned fh_xcode(fh_xkeyboard *k, uint32_t wire);
void fh_xungrab_active(fh_xkeyboard *k);
#endif
