// Exact production implementation with all native observations/effects stubbed.
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#include <libproc.h>
#include <assert.h>
#include <stdio.h>
#include <unistd.h>
static int trusted=1, secure=0, posts=0, front_calls=0, change_during=0;
static pid_t front_pid=42, window_pid=42;
static uint64_t start_time=99;
static CFStringRef window_name=CFSTR("window");
static AXError window_error=kAXErrorSuccess;
static CFTypeRef retained(CFTypeRef x) { return CFRetain(x); }
static pid_t frontmost(void) { front_calls++; return change_during && front_calls>1 ? 43 : front_pid; }
static AXUIElementRef app_element(pid_t pid) { assert(pid==front_pid); return (AXUIElementRef)retained(CFSTR("app")); }
static AXError read_attribute(AXUIElementRef e, CFStringRef name, CFTypeRef *out) {
 (void)e; *out=NULL;
 // The two observed failures: no system focused app, no editor metadata.
 if(CFEqual(name,kAXFocusedApplicationAttribute)) return kAXErrorNoValue;
 if(CFEqual(name,kAXFocusedWindowAttribute)) {
  if(window_error!=kAXErrorSuccess) return window_error;
  *out=retained(window_name); return kAXErrorSuccess;
 }
 return kAXErrorAttributeUnsupported;
}
static AXError element_pid(AXUIElementRef e,pid_t *out) { *out=CFEqual(e,CFSTR("app")) ? front_pid : window_pid; return kAXErrorSuccess; }
static int process_info(int pid,int flavor,uint64_t arg,void *buf,int size) {
 (void)pid;(void)flavor;(void)arg; assert(size==sizeof(struct proc_bsdinfo));
 ((struct proc_bsdinfo *)buf)->pbi_start_tvsec=start_time; return size;
}
static int no_copy(NSString *text) { (void)text; assert(!"clipboard access"); return 0; }
#define FH_INPUT_FRONTMOST_PID() frontmost()
#define AXIsProcessTrusted() (trusted)
#define AXIsProcessTrustedWithOptions(options) (assert(!"permission prompt"),false)
#define IsSecureEventInputEnabled() (secure)
#define AXUIElementCreateSystemWide() ((AXUIElementRef)retained(CFSTR("system")))
#define AXUIElementCreateApplication app_element
#define AXUIElementGetTypeID CFStringGetTypeID
#define AXUIElementSetMessagingTimeout(e,t) (kAXErrorSuccess)
#define AXUIElementCopyAttributeValue read_attribute
#define AXUIElementIsAttributeSettable(e,a,b) (assert(!"editor metadata query"),kAXErrorAttributeUnsupported)
#define AXUIElementGetPid element_pid
#define proc_pidinfo process_info
#define CGEventSourceFlagsState(s) (0)
#define CGEventSourceCreate(s) ((CGEventSourceRef)retained(CFSTR("source")))
#define CGEventCreateKeyboardEvent(s,k,d) ((CGEventRef)retained(CFSTR("event")))
#define CGEventSourceSetUserData(s,d) ((void)0)
#define CGEventSetFlags(e,f) ((void)0)
#define CGEventKeyboardSetUnicodeString(e,c,u) ((void)0)
#define CGEventPostToPid(p,e) ((void)posts++)
#define FH_INPUT_COPY_NOW no_copy
#include "input_darwin.m"
static void capture(fh_input_owner *o,uint32_t *pid,uint64_t *started) {
 front_calls=0;
 if(!fh_input_capture(o,pid,started)) { fprintf(stderr,"capture rejected: %s\n",fh_input_rejection(o)); abort(); }
}
int main(void) { @autoreleasepool {
 fh_input_owner *o=fh_input_create(); uint32_t pid; uint64_t started; uint16_t text[]={'x'};
 capture(o,&pid,&started); // Works despite unsupported editor metadata/system AX focus.
 assert(fh_input_validate(o,pid,started));
 assert(fh_input_send(o,text,1)); assert(posts==2);
 window_name=CFSTR("other-window"); assert(!fh_input_validate(o,pid,started));
 assert(!strcmp(fh_input_rejection(o),"focused_window_changed"));
 window_name=CFSTR("window"); capture(o,&pid,&started);
 front_pid=43; window_pid=43; assert(!fh_input_validate(o,pid,started));
 front_pid=42; window_pid=42; capture(o,&pid,&started);
 start_time++; assert(!fh_input_validate(o,pid,started));
 capture(o,&pid,&started); secure=1; assert(!fh_input_send(o,text,1)); assert(posts==2); secure=0;
 trusted=0; assert(!fh_input_capture(o,&pid,&started)); trusted=1;
 window_error=kAXErrorNoValue; assert(!fh_input_capture(o,&pid,&started)); window_error=kAXErrorSuccess;
 window_pid=43; assert(!fh_input_capture(o,&pid,&started)); window_pid=42;
 front_calls=0; change_during=1; assert(!fh_input_capture(o,&pid,&started)); change_during=0;
 front_pid=getpid(); assert(!fh_input_capture(o,&pid,&started)); front_pid=42;
 capture(o,&pid,&started); fh_input_reset_target(o); assert(!fh_input_validate(o,pid,started));
 fh_input_destroy(o);
 puts("app/window policy passed; editor metadata ignored, changed targets/security rejected; no OS effects");
 } return 0;
}
