//go:build darwin && cgo

package overlay

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func runDarwinOverlayHarness(t *testing.T, body string) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	source := fmt.Sprintf("#include %q\n#include <assert.h>\n#include <stdio.h>\n%s", filepath.Join(filepath.Dir(file), "overlay_darwin.m"), body)
	dir := t.TempDir()
	path, exe := filepath.Join(dir, "overlay.m"), filepath.Join(dir, "overlay")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "clang", "-fblocks", "-mmacosx-version-min=13.0", "-framework", "Cocoa", "-framework", "ApplicationServices", path, "-o", exe)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native harness compile: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, exe).CombinedOutput(); err != nil {
		t.Fatalf("native harness: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}

func TestDarwinOverlayCaptionRecentSuffix(t *testing.T) {
	runDarwinOverlayHarness(t, `
void fh_overlay_snapshot(uint64_t id,int reduced,int contrast,fh_overlay_frame *out) { memset(out,0,sizeof(*out)); }
int main(void) { @autoreleasepool {
 NSFont *font = [NSFont systemFontOfSize:14];
 NSString *text = @"Older caption words 界界界 👩🏽‍💻 é older words and the final words";
 double width = [@"and the final words" sizeWithAttributes:@{NSFontAttributeName:font}].width + 2;
 NSString *fit = FHFitCaption(text,width,font);
 assert([fit hasSuffix:@"the final words"]);
 assert([text hasSuffix:fit] && fit.length < text.length);
 assert([fit sizeWithAttributes:@{NSFontAttributeName:font}].width <= width);
 NSString *emoji = @"👩🏽‍💻";
 assert([FHFitCaption(emoji,1,font) isEqualToString:@""]);
 assert([FHFitCaption(text,0,font) isEqualToString:@""]);
 assert([FHFitCaption(text,10000,font) isEqualToString:text]);
 FHOverlayView *view = [[[FHOverlayView alloc] initWithFrame:NSMakeRect(0,0,width+72,52)] autorelease];
 view->frame = (fh_overlay_frame){.visible=1,.captions=1,.scale=1,.surface=1};
 snprintf(view->frame.caption,sizeof(view->frame.caption),"%s",text.UTF8String);
 NSBitmapImageRep *bitmap = [[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:640 pixelsHigh:104 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0] autorelease];
 [NSGraphicsContext saveGraphicsState];
 [NSGraphicsContext setCurrentContext:[NSGraphicsContext graphicsContextWithBitmapImageRep:bitmap]];
 [view drawRect:view.bounds]; assert([view->captionFit isEqualToString:fit]);
 NSString *cached = view->captionFit;
 [view drawRect:view.bounds]; assert(view->captionFit==cached);
 view->frame.scale=2; [view drawRect:view.bounds];
 assert(view->captionScale==2 && view->captionWidth==view.bounds.size.width/2-72);
 assert([view->captionFit sizeWithAttributes:@{NSFontAttributeName:font}].width <= view->captionWidth);
 [view clearCaption]; assert(!view->captionText && !view->captionFit);
 [NSGraphicsContext restoreGraphicsState];
 puts("PASS: native measured recent suffix and composed Unicode boundaries");
} return 0; }
`)
}

func TestDarwinOverlayMinimalAmplitudeDraw(t *testing.T) {
	runDarwinOverlayHarness(t, `
void fh_overlay_snapshot(uint64_t id,int reduced,int contrast,fh_overlay_frame *out) { memset(out,0,sizeof(*out)); }
static NSData *render(FHOverlayView *view) {
 NSBitmapImageRep *bitmap = [[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:56 pixelsHigh:56 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0] autorelease];
 [NSGraphicsContext saveGraphicsState];
 [NSGraphicsContext setCurrentContext:[NSGraphicsContext graphicsContextWithBitmapImageRep:bitmap]];
 [view drawRect:view.bounds];
 [NSGraphicsContext restoreGraphicsState];
 return [NSData dataWithBytes:bitmap.bitmapData length:bitmap.bytesPerRow*bitmap.pixelsHigh];
}
int main(void) { @autoreleasepool {
 FHOverlayView *view = [[[FHOverlayView alloc] initWithFrame:NSMakeRect(0,0,56,56)] autorelease];
 view->frame = (fh_overlay_frame){.visible=1,.layout=0,.stage=0,.scale=1,.opacity=1,.surface=1,.accent=0x00ff88,.background=0x111111,.animated=0};
 NSData *quiet = render(view);
 view->frame.levels[16] = .9;
 NSData *loud = render(view);
 assert(![quiet isEqualToData:loud]);
 view->frame.stage = 1;
 NSData *processing = render(view);
 view->frame.levels[16] = 0;
 assert([processing isEqualToData:render(view)]);
 puts("PASS: real minimal drawing responds to amplitude with decorative motion off");
} return 0; }
`)
}

func TestDarwinOverlayMeterDotsDraw(t *testing.T) {
	runDarwinOverlayHarness(t, `
void fh_overlay_snapshot(uint64_t id,int reduced,int contrast,fh_overlay_frame *out) { memset(out,0,sizeof(*out)); }
static NSData *render(FHOverlayView *view) {
 NSBitmapImageRep *bitmap = [[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:304 pixelsHigh:62 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0] autorelease];
 [NSGraphicsContext saveGraphicsState];
 [NSGraphicsContext setCurrentContext:[NSGraphicsContext graphicsContextWithBitmapImageRep:bitmap]];
 [view drawRect:view.bounds]; [NSGraphicsContext restoreGraphicsState];
 return [NSData dataWithBytes:bitmap.bitmapData length:bitmap.bytesPerRow*bitmap.pixelsHigh];
}
int main(void) { @autoreleasepool {
 FHOverlayView *view = [[[FHOverlayView alloc] initWithFrame:NSMakeRect(0,0,304,62)] autorelease];
 view->frame = (fh_overlay_frame){.visible=1,.layout=2,.scale=1,.surface=1,.accent=0x00ff88,.background=0x111111};
 NSData *zero = render(view);
 view->frame.checkpoint_count = 1; NSData *one = render(view);
 assert(![zero isEqualToData:one]);
 view->frame.checkpoint_count = 5; NSData *five = render(view);
 assert(![one isEqualToData:five]);
 view->frame.checkpoint_count = 100; assert([five isEqualToData:render(view)]);
 view->frame.checkpoint_count = -1; assert([zero isEqualToData:render(view)]);
 puts("PASS: real meter drawing has zero, one, and capped five checkpoint dots");
} return 0; }
`)
}

func TestDarwinOverlayNativeAccessibilityRefreshAndVisibleTimer(t *testing.T) {
	runDarwinOverlayHarness(t, `
#include <objc/runtime.h>
static int snapshots, interval;
static BOOL reduced, contrast, transparency, visible=YES;
static BOOL getReduced(id self,SEL cmd) { return reduced; }
static BOOL getContrast(id self,SEL cmd) { return contrast; }
static BOOL getTransparency(id self,SEL cmd) { return transparency; }
static void *notify(void *unused) { @autoreleasepool {
 for(int i=0;i<100;i++) [[[NSWorkspace sharedWorkspace] notificationCenter] postNotificationName:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil];
} return NULL; }
void fh_overlay_snapshot(uint64_t id,int motion,int opaque,fh_overlay_frame *out) {
 assert([NSThread isMainThread]); snapshots++;
 *out = (fh_overlay_frame){.visible=visible,.width=208,.height=52,.scale=1,.opacity=opaque ? 1 : .5,.surface=opaque ? 1 : 0,.animated=!motion,.timer_ms=interval};
}
int main(void) { @autoreleasepool {
 [NSApplication sharedApplication];
 Class cls = [[NSWorkspace sharedWorkspace] class];
 method_setImplementation(class_getInstanceMethod(cls,@selector(accessibilityDisplayShouldReduceMotion)),(IMP)getReduced);
 method_setImplementation(class_getInstanceMethod(cls,@selector(accessibilityDisplayShouldIncreaseContrast)),(IMP)getContrast);
 method_setImplementation(class_getInstanceMethod(cls,@selector(accessibilityDisplayShouldReduceTransparency)),(IMP)getTransparency);
 FHOverlay *a=fh_overlay_create(1); FHRetain(a); FHRefresh(a);
 assert(a->panel.visible && !a->timer && snapshots==1);
 reduced=contrast=transparency=YES;
 NSNotificationCenter *center = [[NSWorkspace sharedWorkspace] notificationCenter];
 pthread_t notifier; pthread_create(&notifier,NULL,notify,NULL); pthread_join(notifier,NULL);
 assert(a->pending);
 dispatch_async(dispatch_get_main_queue(), ^{
   assert(snapshots==2 && !a->pending);
   assert(!a->view->frame.animated && a->view->highContrast && a->view->reducedTransparency && a->panel.alphaValue==1);
   assert(!a->timer);
   interval=33; FHRefresh(a); assert(a->timer && a->timer.valid);
   dispatch_after(dispatch_time(DISPATCH_TIME_NOW,200*NSEC_PER_MSEC),dispatch_get_main_queue(), ^{
     assert(snapshots>3 && a->panel.visible && a->timer.valid);
     visible=NO; FHRefresh(a); assert(!a->timer && !a->panel.visible && !a->view->frame.visible);
     assert(fh_overlay_close(a)==1); assert(!a->panel && !a->timer && !a->accessibilityObserver);
     int before=snapshots;
     [center postNotificationName:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil];
     dispatch_async(dispatch_get_main_queue(), ^{
       assert(snapshots==before && !a->pending && atomic_load(&a->references)==1);
       FHRelease(a);
       puts("PASS: stationary accessibility refresh, coalescing, visible timer ticks, hidden timer stop, observer teardown"); exit(0);
     });
   });
 });
 CFRunLoopRun();
} return 1; }
`)
}

func TestDarwinOverlayNativePassiveContract(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "overlay_darwin.m"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, required := range []string{"NSWindowStyleMaskNonactivatingPanel", "canBecomeKeyWindow", "canBecomeMainWindow", "ignoresMouseEvents = YES", "NSWindowCollectionBehaviorCanJoinAllSpaces", "NSWindowCollectionBehaviorFullScreenAuxiliary", "orderFrontRegardless", "accessibilityDisplayShouldReduceMotion", "accessibilityDisplayShouldIncreaseContrast", "accessibilityDisplayShouldReduceTransparency"} {
		if !strings.Contains(source, required) {
			t.Errorf("native focus/accessibility contract missing %s", required)
		}
	}
	for _, prohibited := range []string{"makeKeyAndOrderFront:", "activateIgnoringOtherApps:", "AXIsProcessTrustedWithOptions", "WKWebView"} {
		if strings.Contains(source, prohibited) {
			t.Errorf("passive overlay contains %s", prohibited)
		}
	}
}

// This executable tests native ownership without creating a window or requesting
// permissions. It is deliberately not visual/focus acceptance.
func TestDarwinOverlayNativeLifecycle(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	source := fmt.Sprintf(`#include %q
#include <assert.h>
#include <stdio.h>
void fh_overlay_snapshot(uint64_t id,int reduced,int contrast,fh_overlay_frame *out) { memset(out,0,sizeof(*out)); }
static void *storm(void *p) { for(int i=0;i<10000;i++) fh_overlay_wake(p); return NULL; }
static void *backgroundClose(void *p) { assert(fh_overlay_close(p)==0); return NULL; }
int main(void) { @autoreleasepool {
 FHOverlay *a=fh_overlay_create(1); assert(a); FHRetain(a);
 pthread_t threads[4];
 for(int i=0;i<4;i++) pthread_create(&threads[i],NULL,storm,a);
 for(int i=0;i<4;i++) pthread_join(threads[i],NULL);
 assert(a->pending && atomic_load(&a->references)==3);
 assert(fh_overlay_close(a)==1);
 FHOverlay *b=fh_overlay_create(2); assert(b); FHRetain(b); fh_overlay_wake(b);
 pthread_t closer; pthread_create(&closer,NULL,backgroundClose,b); pthread_join(closer,NULL);
 assert(b->closed && atomic_load(&b->references)==3);
 dispatch_async(dispatch_get_main_queue(), ^{
   assert(atomic_load(&a->references)==1 && atomic_load(&b->references)==1);
   assert(!a->panel && !a->timer && !b->panel && !b->timer);
   FHRelease(a); FHRelease(b);
   puts("PASS: coalesced wakes, main close, bounded background close, queued reference drain");
   exit(0);
 });
 CFRunLoopRun();
} return 1; }
`, filepath.Join(filepath.Dir(file), "overlay_darwin.m"))
	dir := t.TempDir()
	path, exe := filepath.Join(dir, "lifecycle.m"), filepath.Join(dir, "lifecycle")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "clang", "-fblocks", "-mmacosx-version-min=13.0", "-framework", "Cocoa", "-framework", "ApplicationServices", path, "-o", exe)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native harness compile: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, exe).CombinedOutput(); err != nil {
		t.Fatalf("native lifecycle: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}

type darwinTestLevels struct{ level float64 }

func (s *darwinTestLevels) TakeLevel() float64 {
	level := s.level
	s.level = 0
	return level
}

func TestDarwinOverlaySnapshotLevelsAndFiniteOptions(t *testing.T) {
	now := time.Unix(10000, 0)
	o, _ := NewStatusOverlay()
	defer o.Close()
	tap := &darwinTestLevels{}
	o.SetLevelSource(tap)
	// Exercise the same snapshot used by the native timer, without opening a panel.
	o.status = OverlayStatus{Kind: OverlayRecording, Generation: 1}
	tap.level = .5
	frame := o.snapshot(now, true, false)
	if frame.Levels[overlayLevelBars-1] <= 0 || frame.TimerMS != 33 || frame.View.Animated {
		t.Fatalf("live meter missing: %+v", frame)
	}
	o.status = OverlayStatus{Kind: OverlayHidden, CaptionEnabled: true, Caption: "discard me"}
	frame = o.snapshot(now, false, false)
	if frame.TimerMS != 0 || frame.Caption != "" || frame.Levels != [overlayLevelBars]float64{} {
		t.Fatal("hidden overlay retains work/content")
	}
	options := DefaultOverlayOptions()
	options.Scale = math.NaN()
	options.Opacity = math.Inf(1)
	options.Glow = math.NaN()
	frame = projectDarwinOverlay(OverlayStatus{Kind: OverlayRecording}, options, now, false, false)
	if math.IsNaN(frame.Options.Scale) || math.IsInf(frame.Options.Opacity, 0) || math.IsNaN(frame.Options.Glow) {
		t.Fatal("nonfinite options crossed native boundary")
	}
}

// Native interactive acceptance is MANUAL, not an always-skipping Go test.
// Run the macOS application with its Cocoa main run loop in an interactive login
// session. Keep a text editor focused and record its foreground PID and insertion
// point before/during/after overlay previews. Exercise all four layouts, all
// surfaces/visualizers, every status, captions on/off, hold/toggle shortcuts,
// elapsed/checkpoints/countdown, and hide/close while updates arrive. Verify
// click-through and unchanged editor focus; repeat across Spaces/full-screen,
// negative-origin mixed-DPI displays, and a mid-operation display switch. Toggle
// Reduce Motion, Increase Contrast and Reduce Transparency and refresh preview.
// Confirm hidden/closed panels have no timer and no retained caption using a
// native debugger. This matrix has NOT been verified by the source-contract or
// projection tests above; setting an environment variable cannot certify it.

func TestDarwinOverlayOperationAnchorAndClose(t *testing.T) {
	o, err := NewStatusOverlay()
	if err != nil {
		t.Fatal(err)
	}
	if o.native != nil {
		t.Fatal("constructor allocated a native window/queue")
	}
	workA := darwinOverlayRect{X: -1400, Y: 40, W: 1400, H: 900}
	workB := darwinOverlayRect{X: 0, Y: 0, W: 1920, H: 1080}
	var state darwinOverlayPlacement
	first := OverlayStatus{Kind: OverlayRecording, Generation: 1}
	if got := state.resolve(first, workA); got != workA {
		t.Fatalf("first operation: %+v", got)
	}
	first.Kind = OverlayTranscribing
	if got := state.resolve(first, workB); got != workA {
		t.Fatalf("monitor moved mid-operation: %+v", got)
	}
	first.Generation++
	if got := state.resolve(first, workB); got != workB {
		t.Fatalf("new operation did not select monitor: %+v", got)
	}
	state.resolve(OverlayStatus{}, workA)
	if got := state.resolve(first, workA); got != workA {
		t.Fatal("hidden must reset anchor")
	}
	for anchor := OverlayAnchorTopLeft; anchor <= OverlayAnchorBottomRight; anchor++ {
		options := DefaultOverlayOptions()
		options.Anchor = anchor
		r := darwinOverlayDestination(workA, 208, 52, options)
		if r.X < workA.X || r.Y < workA.Y || r.X+r.W > workA.X+workA.W || r.Y+r.H > workA.Y+workA.H {
			t.Fatalf("out of work area: %+v", r)
		}
		if anchor <= OverlayAnchorTopRight && r.Y != workA.Y+workA.H-52-18 {
			t.Fatalf("not top: %+v", r)
		}
		if anchor >= OverlayAnchorBottomLeft && r.Y != workA.Y+18 {
			t.Fatalf("not bottom: %+v", r)
		}
	}
	if err := o.Close(); err != nil {
		t.Fatal(err)
	}
	if err := o.Close(); err != nil {
		t.Fatal(err)
	}
	if err := o.Update(first); err == nil {
		t.Fatal("closed overlay accepts updates")
	}
	if err := o.Configure(DefaultOverlayOptions()); err == nil {
		t.Fatal("closed overlay accepts options")
	}
}

func TestDarwinOverlayMeterCheckpointCount(t *testing.T) {
	for _, count := range []int{-1, 0, 1, 5, 9999} {
		frame := projectDarwinOverlay(OverlayStatus{Kind: OverlayRecording, Checkpoints: count}, DefaultOverlayOptions(), time.Now(), false, false)
		if frame.CheckpointCount != min(max(count, 0), 5) {
			t.Fatalf("count %d projected as %d", count, frame.CheckpointCount)
		}
	}
}

func TestDarwinOverlayCaptionForcesLegibleSurface(t *testing.T) {
	options := DefaultOverlayOptions()
	options.Layout, options.Surface, options.Opacity = OverlayLayoutMinimal, OverlaySurfaceMinimal, .25
	for _, caption := range []string{"", "newest words"} {
		frame := projectDarwinOverlay(OverlayStatus{Kind: OverlayRecording, CaptionEnabled: true, Caption: caption}, options, time.Now(), true, false)
		if frame.Options.Surface != OverlaySurfaceSolid || frame.Options.Opacity < .9 {
			t.Fatalf("illegible caption surface: %+v", frame.Options)
		}
	}
}

func TestDarwinOverlayAccessibilityAndConfiguration(t *testing.T) {
	now := time.Unix(10000, 0)
	for layout := OverlayLayoutMinimal; layout <= OverlayLayoutDetailed; layout++ {
		for surface := OverlaySurfaceGlass; surface <= OverlaySurfaceMinimal; surface++ {
			for visualizer := OverlayVisualizerBars; visualizer <= OverlayVisualizerMeter; visualizer++ {
				options := DefaultOverlayOptions()
				options.Layout = layout
				options.Surface = surface
				options.Visualizer = visualizer
				options.Scale = 1.5
				options.Opacity = .5
				options.Glow = .8
				frame := projectDarwinOverlay(OverlayStatus{Kind: OverlayTranscribing}, options, now, true, false)
				if frame.View.Animated || frame.TimerMS != 0 || frame.Options != options || frame.Phase != "Transcribing" {
					t.Fatalf("configuration lost: %+v", frame)
				}
				frame = projectDarwinOverlay(OverlayStatus{Kind: OverlayRecordingCountdown, CountdownDuration: time.Second, CountdownDeadline: now.Add(time.Second / 2)}, options, now, true, true)
				if frame.Progress != .5 || frame.TimerMS == 0 || frame.Options.Opacity != 1 || frame.Options.Glow != 0 || frame.Options.Surface != OverlaySurfaceSolid {
					t.Fatalf("accessibility override/countdown: %+v", frame)
				}
			}
		}
	}
	options := DefaultOverlayOptions()
	options.Motion = OverlayMotionReduced
	frame := projectDarwinOverlay(OverlayStatus{Kind: OverlayRecording}, options, now, false, false)
	if frame.View.Animated {
		t.Fatal("explicit reduced motion ignored")
	}
}

func TestDarwinOverlayProjectionBoundsOperationalContent(t *testing.T) {
	now := time.Unix(10000, 0)
	status := OverlayStatus{Kind: OverlayRecordingCountdown, CaptionEnabled: true, Caption: strings.Repeat("界", 240), Shortcut: strings.Repeat("A", 300), Checkpoints: 2000, StartedAt: now.Add(-2 * time.Hour), CountdownDeadline: now.Add(time.Second), CountdownDuration: 4 * time.Second}
	frame := projectDarwinOverlay(status, DefaultOverlayOptions(), now, false, false)
	if len([]rune(frame.Caption)) != 180 || len([]rune(frame.Shortcut)) > 48 {
		t.Fatalf("unbounded native strings: caption=%d shortcut=%d", len([]rune(frame.Caption)), len([]rune(frame.Shortcut)))
	}
	if frame.Elapsed != "99:59 elapsed" || frame.Checkpoints != "999 checkpoints" || frame.Progress != .25 {
		t.Fatalf("unexpected operational projection: %+v", frame)
	}
	status.CaptionEnabled = false
	status.Kind = OverlayHidden
	frame = projectDarwinOverlay(status, DefaultOverlayOptions(), now, false, false)
	if frame.Caption != "" || frame.Visible || frame.TimerMS != 0 {
		t.Fatalf("hidden frame retains caption/work: %+v", frame)
	}
}
