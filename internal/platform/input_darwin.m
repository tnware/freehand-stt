//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#include <libproc.h>
#include <stdatomic.h>
#include <stdlib.h>
#include <unistd.h>
#include "input_darwin.h"

#define FH_INPUT_MAX_BYTES (1024 * 1024)
#define FH_INPUT_EVENT_MARKER INT64_C(0x46524844494e5054)
#define FH_AX_TIMEOUT 0.15f

// Serialized by the Go owner. Only the captured app/window identity is retained.
struct fh_input_owner {
    AXUIElementRef window;
    uint32_t pid;
    uint64_t started;
    const char *rejection; // Static first-failure category; no retained AX data.
};
static int fh_check(fh_input_owner *o, int ok, const char *reason) {
    if (!ok && o && !o->rejection) o->rejection = reason;
    return ok;
}
const char *fh_input_rejection(fh_input_owner *o) {
    return o ? (o->rejection ? o->rejection : "") : "owner_invalid";
}
static void fh_input_clear(fh_input_owner *o) {
    if (!o) return;
    if (o->window) CFRelease(o->window);
    o->window = NULL; o->pid = 0; o->started = 0;
}
void fh_input_reset_target(fh_input_owner *o) {
    fh_input_clear(o);
    if (o) o->rejection = NULL;
}
fh_input_owner *fh_input_create(void) { return calloc(1, sizeof(fh_input_owner)); }
void fh_input_destroy(fh_input_owner *o) { fh_input_clear(o); free(o); }
int fh_input_accessibility_authorized(void) { return AXIsProcessTrusted(); }
void fh_input_request_accessibility(void) {
    const void *keys[] = { kAXTrustedCheckOptionPrompt };
    const void *values[] = { kCFBooleanTrue };
    CFDictionaryRef options = CFDictionaryCreate(NULL, keys, values, 1,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    if (options) { AXIsProcessTrustedWithOptions(options); CFRelease(options); }
}
static uint64_t fh_process_started(fh_input_owner *o, pid_t pid) {
    struct proc_bsdinfo info = {0};
    if (!fh_check(o, pid > 0, "pid_invalid") || !fh_check(o, pid != getpid(), "self_target") ||
        !fh_check(o, proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof(info)) == sizeof(info), "process_start_unavailable")) return 0;
    return info.pbi_start_tvsec * UINT64_C(1000000) + info.pbi_start_tvusec;
}
// Only static attribute/error categories cross the diagnostic boundary.
static const char *fh_read_failure(CFStringRef name, AXError error) {
#define FH_READ_FAILURE(attribute, label) \
    if (CFEqual(name, attribute)) { \
        switch (error) { \
        case kAXErrorCannotComplete: return label "__cannot_complete"; \
        case kAXErrorAttributeUnsupported: return label "__unsupported"; \
        case kAXErrorNoValue: return label "__no_value"; \
        case kAXErrorInvalidUIElement: return label "__invalid_element"; \
        case kAXErrorAPIDisabled: return label "__api_disabled"; \
        case kAXErrorNotImplemented: return label "__not_implemented"; \
        case kAXErrorFailure: return label "__failure"; \
        default: return label "__other"; \
        } \
    }
    FH_READ_FAILURE(kAXFocusedApplicationAttribute, "focused_app")
    FH_READ_FAILURE(kAXFocusedUIElementAttribute, "focused_element")
    FH_READ_FAILURE(kAXFocusedWindowAttribute, "focused_window")
    FH_READ_FAILURE(kAXWindowAttribute, "element_window")
    FH_READ_FAILURE(kAXRoleAttribute, "role")
    FH_READ_FAILURE(kAXEnabledAttribute, "enabled")
#undef FH_READ_FAILURE
    return "attribute_read_failed";
}
static CFTypeRef fh_attribute(fh_input_owner *o, AXUIElementRef e, CFStringRef name) {
    CFTypeRef value = NULL;
    AXError error = AXUIElementCopyAttributeValue(e, name, &value);
    if (!fh_check(o, error == kAXErrorSuccess, fh_read_failure(name, error))) {
        if (value) CFRelease(value);
        return NULL;
    }
    return value;
}
static AXUIElementRef fh_element_attribute(fh_input_owner *o, AXUIElementRef e, CFStringRef name) {
    CFTypeRef value = fh_attribute(o, e, name);
    if (value && !fh_check(o, CFGetTypeID(value) == AXUIElementGetTypeID(), "element_type_invalid")) {
        CFRelease(value); return NULL;
    }
    if (value && !fh_check(o, AXUIElementSetMessagingTimeout((AXUIElementRef)value, FH_AX_TIMEOUT) == kAXErrorSuccess, "element_timeout_failed")) {
        CFRelease(value); return NULL;
    }
    return (AXUIElementRef)value;
}
// NSWorkspace supplies the frontmost process even when the system AX root
// does not expose AXFocusedApplication. Never activate or retarget an app.
#ifndef FH_INPUT_FRONTMOST_PID
static pid_t fh_frontmost_pid(void) {
    @autoreleasepool {
        NSRunningApplication *app = NSWorkspace.sharedWorkspace.frontmostApplication;
        return app && !app.terminated ? app.processIdentifier : 0;
    }
}
#define FH_INPUT_FRONTMOST_PID() fh_frontmost_pid()
#endif
static int fh_snapshot(fh_input_owner *out) {
    if (!fh_check(out, fh_input_accessibility_authorized(), "ax_not_trusted") ||
        !fh_check(out, !IsSecureEventInputEnabled(), "secure_input")) return 0;
    pid_t pid = FH_INPUT_FRONTMOST_PID(), windowPID = 0;
    uint64_t started = fh_process_started(out, pid);
    if (!fh_check(out, started != 0, "process_start_zero")) return 0;
    AXUIElementRef app = AXUIElementCreateApplication(pid), window = NULL;
    int ok = 0;
    if (!fh_check(out, app != NULL, "focused_app_unavailable") ||
        !fh_check(out, AXUIElementSetMessagingTimeout(app, FH_AX_TIMEOUT) == kAXErrorSuccess, "element_timeout_failed")) goto cleanup;
    window = fh_element_attribute(out, app, kAXFocusedWindowAttribute);
    if (!fh_check(out, window != NULL, "focused_window_unavailable") ||
        !fh_check(out, AXUIElementGetPid(window, &windowPID) == kAXErrorSuccess, "window_pid_failed") ||
        !fh_check(out, windowPID == pid, "window_pid_mismatch") ||
        !fh_check(out, FH_INPUT_FRONTMOST_PID() == pid, "process_changed") ||
        !fh_check(out, started == fh_process_started(out, pid), "process_start_changed") ||
        !fh_check(out, !IsSecureEventInputEnabled(), "secure_input_late") ||
        !fh_check(out, AXIsProcessTrusted(), "ax_not_trusted_late")) goto cleanup;
    out->window = window; window = NULL;
    out->pid = (uint32_t)pid; out->started = started;
    ok = 1;
cleanup:
    if (window) CFRelease(window);
    if (app) CFRelease(app);
    return ok;
}
int fh_input_capture(fh_input_owner *o, uint32_t *pid, uint64_t *started) {
    if (o) o->rejection = NULL; // Explicit operation start, not cleanup.
    if (!fh_check(o, o && pid && started, "owner_invalid")) return 0;
    *pid = 0; *started = 0;
    fh_input_clear(o);
    if (!fh_snapshot(o)) return 0;
    *pid = o->pid; *started = o->started;
    return 1;
}
// Internal send rechecks share the caller's first-failure diagnostic.
static int fh_validate(fh_input_owner *o, uint32_t pid, uint64_t started) {
    if (!fh_check(o, o && o->window, "owner_invalid") ||
        !fh_check(o, pid && started && pid == o->pid && started == o->started, "target_invalid")) return 0;
    fh_input_owner current = {0};
    int ok = fh_snapshot(&current);
    if (!ok) fh_check(o, 0, current.rejection);
    ok = ok && fh_check(o, current.pid == pid, "process_changed") &&
        fh_check(o, current.started == started, "process_reused") &&
        fh_check(o, CFEqual(current.window, o->window), "focused_window_changed");
    fh_input_clear(&current);
    if (!ok) fh_input_clear(o);
    return ok;
}
int fh_input_validate(fh_input_owner *o, uint32_t pid, uint64_t started) {
    if (o) o->rejection = NULL;
    return fh_validate(o, pid, started);
}
int fh_input_modifiers_released(void) {
    CGEventFlags mask = kCGEventFlagMaskShift | kCGEventFlagMaskControl |
        kCGEventFlagMaskAlternate | kCGEventFlagMaskCommand | kCGEventFlagMaskSecondaryFn;
    return (CGEventSourceFlagsState(kCGEventSourceStateHIDSystemState) & mask) == 0;
}
int fh_input_send(fh_input_owner *o, const uint16_t *units, size_t count) {
    if (o) o->rejection = NULL;
    if (!fh_check(o, o != NULL, "owner_invalid") ||
        !fh_check(o, units && count != 0 && count <= 20, "input_invalid")) return 0;
    for (size_t i = 0; i < count; i++) {
        uint16_t u = units[i];
        if (!fh_check(o, u && !(u >= 0xDC00 && u <= 0xDFFF), "input_invalid")) return 0;
        if (u >= 0xD800 && u <= 0xDBFF) {
            if (!fh_check(o, ++i < count && units[i] >= 0xDC00 && units[i] <= 0xDFFF, "input_invalid")) return 0;
        }
    }
    CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStatePrivate);
    if (!fh_check(o, source != NULL, "event_source_unavailable")) return 0;
    CGEventSourceSetUserData(source, FH_INPUT_EVENT_MARKER);
    CGEventRef down = CGEventCreateKeyboardEvent(source, 0, true);
    CGEventRef up = CGEventCreateKeyboardEvent(source, 0, false);
    CFRelease(source);
    int ok = 0;
    if (!fh_check(o, down && up, "keyboard_event_unavailable")) goto cleanup;
    CGEventSetFlags(down, 0); CGEventSetFlags(up, 0);
    CGEventKeyboardSetUnicodeString(down, count, units);
    CGEventKeyboardSetUnicodeString(up, count, units);
    // Source PID remains our process: keyboard_darwin.m ignores synthetic input.
    if (!fh_check(o, fh_input_modifiers_released(), "modifiers_held") || !fh_validate(o, o->pid, o->started)) goto cleanup;
    CGEventPostToPid((pid_t)o->pid, down);
    if (!fh_check(o, fh_input_modifiers_released(), "modifiers_held") || !fh_validate(o, o->pid, o->started)) goto cleanup;
    CGEventPostToPid((pid_t)o->pid, up);
    ok = 1; // Posted only; Quartz has no application-delivery acknowledgement.
cleanup:
    if (down) CFRelease(down);
    if (up) CFRelease(up);
    return ok;
}

// Exactly one outstanding main-queue block process-wide. Cancellation does not
// free the slot until that block drains: a stalled AppKit queue cannot accumulate
// requests or transcript payloads. No worker or caller ever waits inside Cocoa.
static atomic_bool fh_copy_busy;
@interface FHInputCopyRequest : NSObject {
@public
    atomic_int state;
    NSString *text;
}
@end
@implementation FHInputCopyRequest
- (void)dealloc { [text release]; [super dealloc]; }
@end
#ifndef FH_INPUT_COPY_NOW
static int fh_copy_now(NSString *text) {
    @try {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        [pb clearContents];
        return [pb setString:text forType:NSPasteboardTypeString] ? 1 : 0;
    } @catch (NSException *exception) { return 0; }
}
#define FH_INPUT_COPY_NOW fh_copy_now
#endif
void *fh_input_copy_begin(const char *text, size_t length) {
    if (!text || length > FH_INPUT_MAX_BYTES || memchr(text, 0, length)) return NULL;
    bool expected = false;
    if (!atomic_compare_exchange_strong(&fh_copy_busy, &expected, true)) return NULL;
    @autoreleasepool {
        NSString *value = [[NSString alloc] initWithBytes:text length:length encoding:NSUTF8StringEncoding];
        if (!value) { atomic_store(&fh_copy_busy, false); return NULL; }
        FHInputCopyRequest *request = [[FHInputCopyRequest alloc] init];
        if (!request) { [value release]; atomic_store(&fh_copy_busy, false); return NULL; }
        atomic_init(&request->state, FH_COPY_QUEUED);
        request->text = value;
        // Always async, including main-thread callers. The Go wait is bounded;
        // an inline AppKit call cannot be bounded or safely interrupted.
        dispatch_async(dispatch_get_main_queue(), ^{
            @autoreleasepool {
                int expected = FH_COPY_QUEUED;
                if (atomic_compare_exchange_strong(&request->state, &expected, FH_COPY_RUNNING)) {
                    int result = FH_INPUT_COPY_NOW(request->text);
                    [request->text release]; request->text = nil;
                    atomic_store(&request->state, result ? FH_COPY_SUCCEEDED : FH_COPY_FAILED);
                }
                atomic_store(&fh_copy_busy, false);
            }
        });
        return request; // caller owns one reference, dispatch owns another
    }
}
int fh_input_copy_status(void *value) {
    FHInputCopyRequest *request = value;
    return atomic_load(&request->state);
}
int fh_input_copy_cancel(void *value) {
    FHInputCopyRequest *request = value;
    int expected = FH_COPY_QUEUED;
    if (atomic_compare_exchange_strong(&request->state, &expected, FH_COPY_CANCELLED)) {
        // Only the winning canceller accesses queued text; the block skips it.
        [request->text release]; request->text = nil;
        return FH_COPY_CANCELLED;
    }
    // RUNNING means may still complete. Never claim rollback or safe retry.
    return expected;
}
void fh_input_copy_release(void *value) { [(FHInputCopyRequest *)value release]; }
