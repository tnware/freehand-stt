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

// Serialized by the Go owner. Only one element/window pair survives a call.
struct fh_input_owner {
    AXUIElementRef element;
    AXUIElementRef window;
    uint32_t pid;
    uint64_t started;
};
static void fh_input_clear(fh_input_owner *o) {
    if (!o) return;
    if (o->element) CFRelease(o->element);
    if (o->window) CFRelease(o->window);
    o->element = NULL; o->window = NULL; o->pid = 0; o->started = 0;
}
void fh_input_reset_target(fh_input_owner *o) { fh_input_clear(o); }
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
static uint64_t fh_process_started(pid_t pid) {
    struct proc_bsdinfo info = {0};
    if (pid <= 0 || pid == getpid() ||
        proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof(info)) != sizeof(info)) return 0;
    return info.pbi_start_tvsec * UINT64_C(1000000) + info.pbi_start_tvusec;
}
static CFTypeRef fh_attribute(AXUIElementRef e, CFStringRef name) {
    CFTypeRef value = NULL;
    if (AXUIElementCopyAttributeValue(e, name, &value) != kAXErrorSuccess) {
        if (value) CFRelease(value);
        return NULL;
    }
    return value;
}
static AXUIElementRef fh_element_attribute(AXUIElementRef e, CFStringRef name) {
    CFTypeRef value = fh_attribute(e, name);
    if (value && CFGetTypeID(value) != AXUIElementGetTypeID()) {
        CFRelease(value); return NULL;
    }
    if (value && AXUIElementSetMessagingTimeout((AXUIElementRef)value, FH_AX_TIMEOUT) != kAXErrorSuccess) {
        CFRelease(value); return NULL;
    }
    return (AXUIElementRef)value;
}
static int fh_editable(AXUIElementRef e) {
    CFTypeRef role = fh_attribute(e, kAXRoleAttribute);
    int ok = role && CFGetTypeID(role) == CFStringGetTypeID() &&
        (CFEqual(role, kAXTextFieldRole) || CFEqual(role, kAXTextAreaRole) || CFEqual(role, kAXComboBoxRole));
    if (role) CFRelease(role);
    if (!ok) return 0;
    CFTypeRef subrole = NULL;
    AXError error = AXUIElementCopyAttributeValue(e, kAXSubroleAttribute, &subrole);
    ok = error == kAXErrorAttributeUnsupported ||
        (error == kAXErrorSuccess && subrole && CFGetTypeID(subrole) == CFStringGetTypeID() &&
         !CFEqual(subrole, kAXSecureTextFieldSubrole));
    if (subrole) CFRelease(subrole);
    if (!ok) return 0;
    CFTypeRef enabled = fh_attribute(e, kAXEnabledAttribute);
    ok = enabled && CFEqual(enabled, kCFBooleanTrue);
    if (enabled) CFRelease(enabled);
    if (!ok) return 0;
    CFTypeRef protected = NULL;
    error = AXUIElementCopyAttributeValue(e, CFSTR("AXProtectedContent"), &protected);
    ok = error == kAXErrorAttributeUnsupported ||
        (error == kAXErrorSuccess && protected && CFEqual(protected, kCFBooleanFalse));
    if (protected) CFRelease(protected);
    Boolean settable = false;
    return ok && AXUIElementIsAttributeSettable(e, kAXValueAttribute, &settable) == kAXErrorSuccess && settable;
}
static int fh_snapshot(fh_input_owner *out) {
    if (!fh_input_accessibility_authorized() || IsSecureEventInputEnabled()) return 0;
    AXUIElementRef system = AXUIElementCreateSystemWide();
    if (!system) return 0;
    AXUIElementRef app = NULL, element = NULL, window = NULL, elementWindow = NULL;
    pid_t pid = 0, elementPID = 0, windowPID = 0;
    uint64_t started = 0;
    int ok = 0;
    if (AXUIElementSetMessagingTimeout(system, FH_AX_TIMEOUT) != kAXErrorSuccess) goto cleanup;
    app = fh_element_attribute(system, kAXFocusedApplicationAttribute);
    if (!app || AXUIElementGetPid(app, &pid) != kAXErrorSuccess || !(started = fh_process_started(pid))) goto cleanup;
    element = fh_element_attribute(app, kAXFocusedUIElementAttribute);
    window = fh_element_attribute(app, kAXFocusedWindowAttribute);
    if (!element || !window || AXUIElementGetPid(element, &elementPID) != kAXErrorSuccess ||
        AXUIElementGetPid(window, &windowPID) != kAXErrorSuccess || elementPID != pid || windowPID != pid) goto cleanup;
    elementWindow = fh_element_attribute(element, kAXWindowAttribute);
    if (!elementWindow || !CFEqual(window, elementWindow) || !fh_editable(element) ||
        started != fh_process_started(pid) || IsSecureEventInputEnabled() || !AXIsProcessTrusted()) goto cleanup;
    out->element = element; element = NULL;
    out->window = window; window = NULL;
    out->pid = (uint32_t)pid; out->started = started;
    ok = 1;
cleanup:
    if (elementWindow) CFRelease(elementWindow);
    if (window) CFRelease(window);
    if (element) CFRelease(element);
    if (app) CFRelease(app);
    CFRelease(system);
    return ok;
}
int fh_input_capture(fh_input_owner *o, uint32_t *pid, uint64_t *started) {
    if (!o || !pid || !started) return 0;
    *pid = 0; *started = 0;
    fh_input_clear(o);
    if (!fh_snapshot(o)) return 0;
    *pid = o->pid; *started = o->started;
    return 1;
}
int fh_input_validate(fh_input_owner *o, uint32_t pid, uint64_t started) {
    if (!o || !o->element || !o->window || !pid || !started || pid != o->pid || started != o->started) return 0;
    fh_input_owner current = {0};
    int ok = fh_snapshot(&current) && current.pid == pid && current.started == started &&
        CFEqual(current.element, o->element) && CFEqual(current.window, o->window);
    fh_input_clear(&current);
    if (!ok) fh_input_clear(o);
    return ok;
}
int fh_input_modifiers_released(void) {
    CGEventFlags mask = kCGEventFlagMaskShift | kCGEventFlagMaskControl |
        kCGEventFlagMaskAlternate | kCGEventFlagMaskCommand | kCGEventFlagMaskSecondaryFn;
    return (CGEventSourceFlagsState(kCGEventSourceStateHIDSystemState) & mask) == 0;
}
int fh_input_send(fh_input_owner *o, const uint16_t *units, size_t count) {
    if (!o || !units || count == 0 || count > 20) return 0;
    for (size_t i = 0; i < count; i++) {
        uint16_t u = units[i];
        if (!u || (u >= 0xDC00 && u <= 0xDFFF)) return 0;
        if (u >= 0xD800 && u <= 0xDBFF) {
            if (++i >= count || units[i] < 0xDC00 || units[i] > 0xDFFF) return 0;
        }
    }
    CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStatePrivate);
    if (!source) return 0;
    CGEventSourceSetUserData(source, FH_INPUT_EVENT_MARKER);
    CGEventRef down = CGEventCreateKeyboardEvent(source, 0, true);
    CGEventRef up = CGEventCreateKeyboardEvent(source, 0, false);
    CFRelease(source);
    int ok = 0;
    if (!down || !up) goto cleanup;
    CGEventSetFlags(down, 0); CGEventSetFlags(up, 0);
    CGEventKeyboardSetUnicodeString(down, count, units);
    CGEventKeyboardSetUnicodeString(up, count, units);
    // Source PID remains our process: keyboard_darwin.m ignores synthetic input.
    if (!fh_input_modifiers_released() || !fh_input_validate(o, o->pid, o->started)) goto cleanup;
    CGEventPostToPid((pid_t)o->pid, down);
    if (!fh_input_modifiers_released() || !fh_input_validate(o, o->pid, o->started)) goto cleanup;
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
