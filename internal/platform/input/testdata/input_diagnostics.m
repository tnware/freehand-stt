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
static CFStringRef field_name=NULL, scope_window=NULL;
static AXError field_error=kAXErrorSuccess, scope_error=kAXErrorSuccess;
static pid_t field_pid=42;
static int stale_baseline=0, unstable_field=0, field_reads=0, scope_reads=0;
static int field_timeout=0, change_window_on_field=0, malformed_field=0, change_after_post=0;
static int window_reads=0;
// Equal AX app proxies have separate object-local timeout settings.
static struct { AXUIElementRef ref; float timeout; } app_timeouts[512];
static size_t app_timeout_count=0;
static size_t app_slot(AXUIElementRef e) {
 for(size_t n=0;n<app_timeout_count;n++) if(app_timeouts[n].ref==e) return n;
 assert(app_timeout_count < 512);
 size_t n=app_timeout_count++; app_timeouts[n].ref=e; return n;
}
static AXError set_timeout(AXUIElementRef e, float timeout) {
 assert(timeout > 0 && timeout <= 0.15f);
 if(CFEqual(e,CFSTR("app"))) app_timeouts[app_slot(e)].timeout=timeout;
 if(CFStringHasPrefix((CFStringRef)e,CFSTR("field-"))) assert(timeout==0.02f);
 return field_timeout && CFEqual(e,CFSTR("field-a")) ? kAXErrorFailure : kAXErrorSuccess;
}
static CFTypeRef retained(CFTypeRef x) { return CFRetain(x); }
static pid_t frontmost(void) { front_calls++; return change_during && front_calls>1 ? 43 : front_pid; }
static AXUIElementRef app_element(pid_t pid) { assert(pid==front_pid); return (AXUIElementRef)CFStringCreateMutableCopy(NULL,0,CFSTR("app")); }
static AXError read_attribute(AXUIElementRef e, CFStringRef name, CFTypeRef *out) {
 *out=NULL;
 // The two observed failures: no system focused app, no editor metadata.
 if(CFEqual(name,kAXFocusedApplicationAttribute)) return kAXErrorNoValue;
 if(CFEqual(name,kAXFocusedWindowAttribute)) {
  window_reads++; assert(app_timeouts[app_slot(e)].timeout==0.15f);
  if(window_error!=kAXErrorSuccess) return window_error;
  *out=retained(window_name); return kAXErrorSuccess;
 }
 if(CFEqual(name,kAXFocusedUIElementAttribute)) {
  field_reads++; assert(app_timeouts[app_slot(e)].timeout==0.02f);
  if(change_window_on_field) window_name=CFSTR("changed-during-field");
  if(field_error!=kAXErrorSuccess) return field_error;
  if(!field_name) return kAXErrorAttributeUnsupported;
  if(malformed_field) { *out=retained(kCFBooleanTrue); return kAXErrorSuccess; }
  *out=retained(unstable_field && field_reads%2==0 ? CFSTR("field-c") : field_name);
  return kAXErrorSuccess;
 }
 if(CFEqual(name,kAXWindowAttribute)) {
  scope_reads++;
  if(stale_baseline && CFEqual(e,CFSTR("field-a"))) return kAXErrorInvalidUIElement;
  if(scope_error!=kAXErrorSuccess) return scope_error;
  *out=retained(scope_window ? scope_window : window_name); return kAXErrorSuccess;
 }
 // No roles, values, editability, labels or editor content may be queried.
 assert(!"editor metadata query"); return kAXErrorAttributeUnsupported;
}
static AXError element_pid(AXUIElementRef e,pid_t *out) {
 *out=CFEqual(e,CFSTR("app")) ? front_pid :
  CFStringHasPrefix((CFStringRef)e,CFSTR("field-")) ? field_pid : window_pid;
 return kAXErrorSuccess;
}
static int process_info(int pid,int flavor,uint64_t arg,void *buf,int size) {
 (void)pid;(void)flavor;(void)arg; assert(size==sizeof(struct proc_bsdinfo));
 ((struct proc_bsdinfo *)buf)->pbi_start_tvsec=start_time; return size;
}
static void post_event(void) { posts++; if(change_after_post) field_name=CFSTR("field-b"); }
static int no_copy(NSString *text) { (void)text; assert(!"clipboard access"); return 0; }
#define FH_INPUT_FRONTMOST_PID() frontmost()
#define AXIsProcessTrusted() (trusted)
#define AXIsProcessTrustedWithOptions(options) (assert(!"permission prompt"),false)
#define IsSecureEventInputEnabled() (secure)
#define AXUIElementCreateSystemWide() (assert(!"system-wide AX query"), (AXUIElementRef)NULL)
#define AXUIElementCreateApplication app_element
#define AXUIElementGetTypeID CFStringGetTypeID
#define AXUIElementSetMessagingTimeout set_timeout
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
#define CGEventPostToPid(p,e) post_event()
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
 // Unknown capture is compatible, and never upgraded into a new baseline.
 int reads=field_reads;
 field_name=CFSTR("field-a"); assert(fh_input_validate(o,pid,started));
 assert(field_reads==reads && !o->element && !*fh_input_rejection(o));
 // Known same field succeeds. Known different live field rejects before posting.
 capture(o,&pid,&started); assert(o->element);
 field_reads=scope_reads=window_reads=0; assert(fh_input_validate(o,pid,started));
 assert(field_reads==1 && scope_reads==1 && window_reads==1);
 field_name=CFSTR("field-b"); field_reads=scope_reads=window_reads=0;
 assert(!fh_input_send(o,text,1)); assert(posts==2);
 assert(!strcmp(fh_input_rejection(o),"focused_element_changed"));
 assert(field_reads==2 && scope_reads==3 && window_reads==2); assert(!o->element && !o->window);
 // A -> unavailable -> B keeps A, and rejects only the later proven change.
 field_name=CFSTR("field-a"); capture(o,&pid,&started);
 field_error=kAXErrorCannotComplete; assert(fh_input_validate(o,pid,started));
 assert(o->element && !*fh_input_rejection(o));
 field_error=kAXErrorSuccess; field_name=CFSTR("field-b");
 assert(!fh_input_validate(o,pid,started));
 assert(!strcmp(fh_input_rejection(o),"focused_element_changed"));
 // A destroyed/virtualized reference is unknown, not evidence of a new field.
 field_name=CFSTR("field-a"); capture(o,&pid,&started);
 stale_baseline=1; field_name=CFSTR("field-b");
 assert(fh_input_validate(o,pid,started)); assert(!*fh_input_rejection(o)); stale_baseline=0;
 // Inconsistent observations do not claim a confirmed change.
 field_reads=0; unstable_field=1;
 assert(fh_input_validate(o,pid,started)); assert(!*fh_input_rejection(o)); unstable_field=0;
 // Unsupported identity/scope, wrong PID and optional timeout failure stay compatible.
 AXError unknowns[]={kAXErrorAttributeUnsupported,kAXErrorNoValue,kAXErrorCannotComplete,kAXErrorInvalidUIElement};
 for(size_t n=0;n<sizeof(unknowns)/sizeof(unknowns[0]);n++) {
  field_error=unknowns[n]; capture(o,&pid,&started);
  assert(!o->element && !*fh_input_rejection(o)); assert(fh_input_validate(o,pid,started));
 }
 field_error=kAXErrorSuccess; field_name=CFSTR("field-a");
 scope_error=kAXErrorAttributeUnsupported; capture(o,&pid,&started);
 assert(!o->element && !*fh_input_rejection(o)); scope_error=kAXErrorSuccess;
 scope_window=CFSTR("other-window"); capture(o,&pid,&started); assert(!o->element && !*fh_input_rejection(o)); scope_window=NULL;
 field_pid=43; capture(o,&pid,&started); assert(!o->element && !*fh_input_rejection(o)); field_pid=42;
 field_timeout=1; capture(o,&pid,&started); assert(!o->element && !*fh_input_rejection(o)); field_timeout=0;
 malformed_field=1; capture(o,&pid,&started); assert(!o->element && !*fh_input_rejection(o)); malformed_field=0;
 // Current scope loss remains unknown without destroying the original baseline.
 capture(o,&pid,&started); scope_error=kAXErrorCannotComplete;
 assert(fh_input_validate(o,pid,started)); assert(o->element && !*fh_input_rejection(o)); scope_error=kAXErrorSuccess;
 field_pid=43; assert(fh_input_validate(o,pid,started)); assert(o->element && !*fh_input_rejection(o)); field_pid=42;
 // Required window identity still rejects a change during optional sampling.
 capture(o,&pid,&started);
 change_window_on_field=1; assert(!fh_input_validate(o,pid,started)); change_window_on_field=0;
 assert(!strcmp(fh_input_rejection(o),"focused_window_changed"));
 window_name=CFSTR("window"); field_name=NULL; capture(o,&pid,&started);
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
 // Native event boundaries also check fields: no later event after a change.
 field_name=CFSTR("field-a"); capture(o,&pid,&started);
 int before=posts; change_after_post=1;
 assert(!fh_input_send(o,text,1)); assert(posts==before+1); change_after_post=0;
 assert(!strcmp(fh_input_rejection(o),"focused_element_changed"));
 assert(!o->element && !o->window);
 // A new capture can send, but an old target cannot replay following rejection.
 assert(!fh_input_validate(o,pid,started));
 field_name=CFSTR("field-a"); capture(o,&pid,&started);
 assert(fh_input_send(o,text,1)); before=posts;
 field_name=CFSTR("field-b"); assert(!fh_input_send(o,text,1)); assert(posts==before);
 fh_input_destroy(o);
 puts("app/window and optional field identity passed; unknown compatible, confirmed changes rejected; no OS effects");
 } return 0;
}
