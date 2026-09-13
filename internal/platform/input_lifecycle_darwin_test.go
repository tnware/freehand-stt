//go:build darwin && cgo

package platform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDarwinNativeCopyLifecycle(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	source := `
#import <AppKit/AppKit.h>
#include <assert.h>
#include <stdatomic.h>
static atomic_int calls;
static dispatch_semaphore_t running, finish;
static int fake_copy(NSString *text) {
 assert([NSThread isMainThread]); assert([text isEqualToString:@"private"]);
 atomic_fetch_add(&calls, 1);
 dispatch_semaphore_signal(running);
 dispatch_semaphore_wait(finish, DISPATCH_TIME_FOREVER);
 return 1;
}
#define FH_INPUT_COPY_NOW fake_copy
#include "` + filepath.Join(wd, "input_darwin.m") + `"
int main(void) { @autoreleasepool {
 // Retained native ownership without AX queries or permissions.
 fh_input_owner *o = fh_input_create();
 CFMutableStringRef w = CFStringCreateMutable(NULL, 0);
 o->window = (AXUIElementRef)CFRetain(w);
 o->pid = 42; o->started = 99;
 CFIndex wr = CFGetRetainCount(w);
 fh_input_reset_target(o);
 assert(!o->window && !o->pid && !o->started);
 assert(CFGetRetainCount(w) == wr-1);
 fh_input_reset_target(o); fh_input_destroy(o); CFRelease(w);
 // Main-thread submission is asynchronous, even when AppKit is slow.
 void *q = fh_input_copy_begin("private", 7); assert(q);
 assert(fh_input_copy_status(q) == FH_COPY_QUEUED);
 assert(fh_input_copy_cancel(q) == FH_COPY_CANCELLED);
 assert(((FHInputCopyRequest *)q)->text == nil);
 fh_input_copy_release(q);
 for (int i=0; i<10000; i++) assert(!fh_input_copy_begin("private", 7));
 assert(atomic_load(&calls) == 0);
 running = dispatch_semaphore_create(0); finish = dispatch_semaphore_create(0);
 dispatch_async(dispatch_get_main_queue(), ^{
   assert(atomic_load(&calls) == 0);
   void *r = fh_input_copy_begin("private", 7); assert(r);
   dispatch_async(dispatch_get_global_queue(QOS_CLASS_DEFAULT, 0), ^{
     dispatch_semaphore_wait(running, DISPATCH_TIME_FOREVER);
     assert(fh_input_copy_cancel(r) == FH_COPY_RUNNING);
     for (int i=0; i<10000; i++) assert(!fh_input_copy_begin("private", 7));
     dispatch_semaphore_signal(finish);
     dispatch_async(dispatch_get_main_queue(), ^{
       assert(fh_input_copy_status(r) == FH_COPY_SUCCEEDED);
       assert(atomic_load(&calls) == 1);
       fh_input_copy_release(r);
       CFRunLoopStop(CFRunLoopGetMain());
     });
   });
 });
 CFRunLoopRun();
 dispatch_release(running); dispatch_release(finish);
 puts("native target references released; queued copy revoked; running copy ambiguous; admission bounded");
} return 0; }
`
	dir := t.TempDir()
	path, exe := filepath.Join(dir, "fixture.m"), filepath.Join(dir, "fixture")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "clang", "-fblocks", "-mmacosx-version-min=13.0", "-framework", "AppKit", "-framework", "ApplicationServices", "-framework", "Carbon", path, "-o", exe)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, exe).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}
