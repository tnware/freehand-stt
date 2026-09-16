//go:build darwin

#include "keyboard_darwin.h"
#include <ApplicationServices/ApplicationServices.h>
#include <Carbon/Carbon.h>
#include <stdatomic.h>
#include <stdlib.h>

#define FH_KEYBOARD_CAPACITY 256
struct fh_keyboard {
    _Atomic unsigned refs, head, tail;
    _Atomic int stop, ready, failed, loss;
    int capture, primary;
    unsigned char owned[128]; // callback-thread-only capture ownership
    fh_keyboard_event queue[FH_KEYBOARD_CAPACITY];
};
int fh_keyboard_authorized(void) { return CGPreflightListenEventAccess(); }
int fh_keyboard_capture_authorized(void) { return fh_keyboard_authorized() && AXIsProcessTrusted(); }
fh_keyboard *fh_keyboard_new(int capture) {
    fh_keyboard *k = calloc(1, sizeof(*k));
    if (k) { atomic_init(&k->refs, 1); k->capture = capture; k->primary = -1; }
    return k;
}
void fh_keyboard_retain(fh_keyboard *k) { atomic_fetch_add(&k->refs, 1); }
void fh_keyboard_release(fh_keyboard *k) { if (atomic_fetch_sub(&k->refs, 1) == 1) free(k); }
void fh_keyboard_stop(fh_keyboard *k) { atomic_store(&k->stop, 1); }
int fh_keyboard_ready(fh_keyboard *k) { return atomic_load(&k->ready); }
static void fh_keyboard_fail(fh_keyboard *k) {
    atomic_store(&k->failed, 1);
    atomic_store(&k->loss, 1);
    atomic_store(&k->stop, 1);
}
static int fh_keyboard_modifier(uint16_t code) { return code >= 54 && code <= 62 && code != 57; }
int fh_keyboard_feed(fh_keyboard *k, uint16_t code, int down, int injected, int repeat) {
    if (injected || code >= 128 || atomic_load(&k->stop)) return 0;
    int suppress = 0;
    if (k->capture) {
        suppress = k->owned[code];
        if (down && !repeat && (k->primary < 0 || suppress || code == 53)) { // Escape always cancels an in-progress chord
            k->owned[code] = 1;
            suppress = 1;
            if (!fh_keyboard_modifier(code)) k->primary = code;
        }
        if (!down) k->owned[code] = 0;
        if (!suppress) return 0;
    }
    if (repeat) return suppress;
    unsigned head = atomic_load_explicit(&k->head, memory_order_relaxed);
    unsigned tail = atomic_load_explicit(&k->tail, memory_order_acquire);
    if (head - tail >= FH_KEYBOARD_CAPACITY) { fh_keyboard_fail(k); return suppress; }
    k->queue[head % FH_KEYBOARD_CAPACITY] = (fh_keyboard_event){code, !!down, 0};
    atomic_store_explicit(&k->head, head + 1, memory_order_release);
    return suppress;
}
int fh_keyboard_read(fh_keyboard *k, fh_keyboard_event *event) {
    if (atomic_load(&k->failed)) {
        if (!atomic_exchange(&k->loss, 0)) return 0;
        *event = (fh_keyboard_event){0,0,1};
        return 1; // never replay queued presses/releases after losing edges
    }
    unsigned tail = atomic_load_explicit(&k->tail, memory_order_relaxed);
    unsigned head = atomic_load_explicit(&k->head, memory_order_acquire);
    if (tail == head) return 0;
    *event = k->queue[tail % FH_KEYBOARD_CAPACITY];
    atomic_store_explicit(&k->tail, tail + 1, memory_order_release);
    return 1;
}
static CGEventRef fh_keyboard_callback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *ref) {
    fh_keyboard *k = ref;
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        fh_keyboard_fail(k); // never re-enable and continue a potentially stuck hold
        return event;
    }
    if (!event) return event;
    uint16_t code = (uint16_t)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
    int down = type == kCGEventKeyDown;
    if (type == kCGEventFlagsChanged) {
        // NX device-dependent masks distinguish left/right even when their
        // shared NSEvent modifier flag stays set as one side is released.
        uint64_t mask = 0;
        switch(code) {
            case 59: mask = 0x00000001; break; // left control
            case 62: mask = 0x00002000; break; // right control
            case 56: mask = 0x00000002; break; // left shift
            case 60: mask = 0x00000004; break; // right shift
            case 55: mask = 0x00000008; break; // left command
            case 54: mask = 0x00000010; break; // right command
            case 58: mask = 0x00000020; break; // left option
            case 61: mask = 0x00000040; break; // right option
            default: return event; // Caps Lock/Fn are not supported modifiers
        }
        down = (CGEventGetFlags(event) & mask) != 0;
    } else if (type != kCGEventKeyDown && type != kCGEventKeyUp) return event;
    int injected = CGEventGetIntegerValueField(event, kCGEventSourceUnixProcessID) != 0;
    int repeat = CGEventGetIntegerValueField(event, kCGKeyboardEventAutorepeat) != 0;
    return fh_keyboard_feed(k, code, down, injected, repeat) ? NULL : event;
}
static int fh_keyboard_session_active(void) {
    CFDictionaryRef session = CGSessionCopyCurrentDictionary();
    if (!session) return 0;
    int active = CFDictionaryGetValue(session, kCGSessionOnConsoleKey) == kCFBooleanTrue &&
                 CFDictionaryGetValue(session, kCGSessionLoginDoneKey) == kCFBooleanTrue;
    // Locking and Secure Input can remove key-up delivery. Fail closed rather
    // than hold a recorder open until the user's next unrelated keystroke.
    CFTypeRef locked = CFDictionaryGetValue(session, CFSTR("CGSSessionScreenIsLocked"));
    if (locked == kCFBooleanTrue) active = 0;
    CFRelease(session);
    return active && !IsSecureEventInputEnabled();
}
void fh_keyboard_run(fh_keyboard *k) {
    if (atomic_load(&k->stop) || !fh_keyboard_authorized() ||
        (k->capture && !fh_keyboard_capture_authorized()) || !fh_keyboard_session_active()) {
        atomic_store(&k->ready, -1); return;
    }
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp) | CGEventMaskBit(kCGEventFlagsChanged);
    CFMachPortRef tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap,
        k->capture ? kCGEventTapOptionDefault : kCGEventTapOptionListenOnly,
        mask, fh_keyboard_callback, k);
    if (!tap) { atomic_store(&k->ready, -1); return; }
    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
    if (!source) { CFRelease(tap); atomic_store(&k->ready, -1); return; }
    CFRunLoopRef loop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(loop, source, kCFRunLoopDefaultMode);
    CGEventTapEnable(tap, true);
    atomic_store(&k->ready, 1);
    CFAbsoluteTime check = 0;
    while (!atomic_load(&k->stop)) {
        CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.02, true);
        CFAbsoluteTime now = CFAbsoluteTimeGetCurrent();
        if (now >= check) {
            check = now + 0.1;
            if (!CGEventTapIsEnabled(tap) || !fh_keyboard_authorized() || !fh_keyboard_session_active() ||
                (k->capture && !fh_keyboard_capture_authorized())) fh_keyboard_fail(k);
        }
    }
    CGEventTapEnable(tap, false);
    CFRunLoopRemoveSource(loop, source, kCFRunLoopDefaultMode);
    CFMachPortInvalidate(tap);
    CFRelease(source);
    CFRelease(tap);
}
