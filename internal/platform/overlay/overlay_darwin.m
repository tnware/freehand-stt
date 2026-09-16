//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#include <pthread.h>
#include <stdatomic.h>
#include <math.h>
#include "overlay_darwin.h"

// MRC deliberately: queued blocks own explicit C references, not Go pointers.
@interface FHOverlayPanel : NSPanel
@end
@implementation FHOverlayPanel
- (BOOL)canBecomeKeyWindow { return NO; }
- (BOOL)canBecomeMainWindow { return NO; }
@end

static NSColor *FHColor(uint32_t rgb, CGFloat alpha) {
    return [NSColor colorWithSRGBRed:((rgb >> 16) & 255)/255.0
                             green:((rgb >> 8) & 255)/255.0
                              blue:(rgb & 255)/255.0 alpha:alpha];
}
static NSString *FHString(const char *s) {
    return [NSString stringWithUTF8String:s] ?: @"";
}
static double FHClamp(double x) { return isfinite(x) ? fmin(1, fmax(0, x)) : 0; }
static void FHRound(NSRect r, double radius, NSColor *color) {
    [color setFill];
    [[NSBezierPath bezierPathWithRoundedRect:r xRadius:radius yRadius:radius] fill];
}
// Caption input is already bounded by Go. Remove only whole grapheme clusters,
// measuring with the same native font used to draw the newest suffix.
static NSString *FHFitCaption(NSString *text, double width, NSFont *font) {
    if (width <= 0) return @"";
    NSDictionary *attributes = @{NSFontAttributeName:font};
    NSUInteger start = 0;
    while (start < text.length) {
        NSString *suffix = [text substringFromIndex:start];
        if ([suffix sizeWithAttributes:attributes].width <= width)
            return [suffix stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
        start = NSMaxRange([text rangeOfComposedCharacterSequenceAtIndex:start]);
    }
    return @"";
}
static void FHText(NSString *s, NSRect r, double size, NSColor *color, BOOL bold) {
    NSMutableParagraphStyle *p = [[[NSMutableParagraphStyle alloc] init] autorelease];
    p.lineBreakMode = NSLineBreakByTruncatingTail;
    [s drawInRect:r withAttributes:@{NSFontAttributeName:[NSFont systemFontOfSize:size weight:bold ? NSFontWeightSemibold : NSFontWeightRegular], NSForegroundColorAttributeName:color, NSParagraphStyleAttributeName:p}];
}

@interface FHOverlayView : NSView {
@public
    fh_overlay_frame frame;
    BOOL highContrast;
    BOOL reducedTransparency;
    NSString *captionText, *captionFit;
    double captionWidth, captionScale;
}
- (void)clearCaption;
@end
@implementation FHOverlayView
- (void)clearCaption {
    [captionText release]; captionText = nil;
    [captionFit release]; captionFit = nil;
}
- (void)dealloc { [self clearCaption]; [super dealloc]; }
- (BOOL)isFlipped { return YES; }
- (BOOL)acceptsFirstResponder { return NO; }
- (BOOL)isAccessibilityElement { return YES; }
- (NSString *)accessibilityRole { return NSAccessibilityStaticTextRole; }
- (NSString *)accessibilityLabel {
    if (!frame.visible) return @"";
    NSMutableArray *parts = [NSMutableArray arrayWithObjects:FHString(frame.phase), FHString(frame.instruction), nil];
    if (frame.shortcut[0]) [parts addObject:FHString(frame.shortcut)];
    if (frame.layout == 3) {
        [parts addObject:FHString(frame.elapsed)]; [parts addObject:FHString(frame.checkpoints)];
    }
    if (frame.captions && frame.caption[0]) [parts addObject:FHString(frame.caption)];
    if (frame.stage == 6) [parts addObject:[NSString stringWithFormat:@"Countdown %.0f percent remaining", FHClamp(frame.progress)*100]];
    return [parts componentsJoinedByString:@", "];
}
- (void)drawRect:(NSRect)dirty {
    (void)dirty;
    if (!frame.visible) return;
    [NSGraphicsContext saveGraphicsState];
    double scale = frame.scale > 0 ? frame.scale : 1;
    NSAffineTransform *transform = [NSAffineTransform transform];
    [transform scaleBy:scale]; [transform concat];
    double w = self.bounds.size.width/scale, h = self.bounds.size.height/scale;
    NSRect bounds = NSMakeRect(1,1,w-2,h-2);
    NSColor *accent = FHColor(frame.accent,1);
    NSColor *ink = [NSColor colorWithWhite:1 alpha:1];
    NSColor *secondary = [NSColor colorWithWhite:1 alpha:highContrast ? 1 : .76];
    double radius = frame.layout == 3 && !frame.captions ? 18 : fmin(h/2,26);
    double alpha = frame.surface == 1 || reducedTransparency || highContrast ? 1 : (frame.surface == 2 ? .58 : .86);
    FHRound(bounds,radius,FHColor(frame.background,alpha));
    if (frame.surface == 0 && !reducedTransparency && !highContrast) {
        NSGradient *gradient = [[[NSGradient alloc] initWithStartingColor:[NSColor colorWithWhite:1 alpha:.12] endingColor:[NSColor colorWithWhite:1 alpha:0]] autorelease];
        [gradient drawInBezierPath:[NSBezierPath bezierPathWithRoundedRect:bounds xRadius:radius yRadius:radius] angle:90];
    }
    if (frame.glow > 0 && !highContrast) {
        [NSGraphicsContext saveGraphicsState];
        NSShadow *shadow = [[[NSShadow alloc] init] autorelease];
        shadow.shadowColor = FHColor(frame.accent,.32*FHClamp(frame.glow));
        shadow.shadowBlurRadius = 12; shadow.shadowOffset = NSZeroSize; [shadow set];
        FHRound(NSMakeRect(10,h-5,w-20,2),1,FHColor(frame.accent,.5*FHClamp(frame.glow)));
        [NSGraphicsContext restoreGraphicsState];
    }
    NSBezierPath *border = [NSBezierPath bezierPathWithRoundedRect:bounds xRadius:radius yRadius:radius];
    [FHColor(frame.accent, highContrast ? 1 : .3) setStroke];
    border.lineWidth = highContrast ? 2 : 1; [border stroke];

    // Functional amplitude feedback must survive Reduce Motion.
    if (frame.layout == 0 && !frame.captions && frame.stage == 0) {
        double peak = 0;
        for (int i=0;i<17;i++) peak = fmax(peak,FHClamp(frame.levels[i]));
        double diameter = 32+12*peak;
        NSBezierPath *ring = [NSBezierPath bezierPathWithOvalInRect:NSMakeRect((w-diameter)/2,(h-diameter)/2,diameter,diameter)];
        ring.lineWidth = 1.5+2*peak;
        [FHColor(frame.accent,.3+.7*peak) setStroke]; [ring stroke];
    }
    static NSString *const symbols[] = {@"mic.fill", @"arrow.triangle.2.circlepath", @"checkmark", @"doc.on.clipboard", @"exclamationmark.triangle", @"xmark", @"timer"};
    int icon = frame.icon >= 0 && frame.icon < 7 ? frame.icon : 0;
    double ix = frame.layout == 0 && !frame.captions ? (w-22)/2 : 18;
    double iy = frame.layout == 3 && !frame.captions ? 18 : (h-22)/2;
    NSImage *image = [NSImage imageWithSystemSymbolName:symbols[icon] accessibilityDescription:nil];
    image = [image imageWithSymbolConfiguration:[NSImageSymbolConfiguration configurationWithPaletteColors:@[accent]]];
    [NSGraphicsContext saveGraphicsState];
    if (icon == 1 && frame.animated) {
        NSAffineTransform *spin = [NSAffineTransform transform];
        [spin translateXBy:ix+11 yBy:iy+11];
        [spin rotateByDegrees:(frame.animation_ms % 1200)*.3];
        [spin translateXBy:-ix-11 yBy:-iy-11]; [spin concat];
    }
    [image drawInRect:NSMakeRect(ix,iy,22,22) fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
    [NSGraphicsContext restoreGraphicsState];

    if (frame.captions) {
        NSString *text = frame.caption[0] ? FHString(frame.caption) : FHString(frame.phase);
        if (![captionText isEqualToString:text] || captionWidth != w-72 || captionScale != scale) {
            [self clearCaption];
            captionText = [text copy]; captionWidth = w-72; captionScale = scale;
            captionFit = [FHFitCaption(text,captionWidth,[NSFont systemFontOfSize:14]) copy];
        }
        FHText(captionFit,NSMakeRect(54,16,w-72,24),14,ink,NO);
    } else if (frame.layout != 0) {
        NSRect stage;
        if (frame.layout == 3) {
            FHText(@"FREEHAND",NSMakeRect(52,13,w-70,15),10,secondary,YES);
            FHText(FHString(frame.phase),NSMakeRect(52,29,w-70,24),17,ink,YES);
            FHText(FHString(frame.instruction),NSMakeRect(18,57,w-36,20),12,secondary,NO);
            FHText(FHString(frame.shortcut),NSMakeRect(18,80,w-36,18),11,ink,YES);
            FHText(FHString(frame.elapsed),NSMakeRect(18,h-27,(w-36)/2,18),11,secondary,NO);
            FHText(FHString(frame.checkpoints),NSMakeRect(w/2,h-27,w/2-18,18),11,secondary,NO);
            stage = NSMakeRect(18,104,w-36,16);
        } else if (frame.layout == 2) {
            int count = MIN(5,MAX(0,frame.checkpoint_count));
            for (int i=0;i<count;i++)
                FHRound(NSMakeRect(w-22-i*8,16,4,4),2,accent);
            FHText(FHString(frame.phase),NSMakeRect(52,9,w-68-(count ? 44 : 0),19),12,ink,YES);
            stage = NSMakeRect(54,33,w-74,18);
        } else {
            stage = NSMakeRect(56,17,w-75,19);
        }
        double t = frame.animated ? frame.animation_ms/1000.0 : 0;
        if (frame.stage == 0) {
            double peak = 0;
            for (int i=0;i<17;i++) peak = fmax(peak,FHClamp(frame.levels[i]));
            if (frame.visualizer == 1) {
                double d = 5+peak*(stage.size.height-5);
                FHRound(NSMakeRect(NSMidX(stage)-d/2,NSMidY(stage)-d/2,d,d),d/2,accent);
            } else if (frame.visualizer == 3) {
                FHRound(NSMakeRect(stage.origin.x,NSMidY(stage)-3,stage.size.width,6),3,FHColor(frame.accent,.18));
                FHRound(NSMakeRect(stage.origin.x,NSMidY(stage)-3,stage.size.width*peak,6),3,accent);
            } else if (frame.visualizer == 2) {
                NSBezierPath *path = [NSBezierPath bezierPath];
                for (int i=0;i<17;i++) {
                    NSPoint p = NSMakePoint(stage.origin.x+i*stage.size.width/16,NSMidY(stage)-FHClamp(frame.levels[i])*stage.size.height/2);
                    if (!i) [path moveToPoint:p]; else [path lineToPoint:p];
                }
                for (int i=16;i>=0;i--) [path lineToPoint:NSMakePoint(stage.origin.x+i*stage.size.width/16,NSMidY(stage)+FHClamp(frame.levels[i])*stage.size.height/2)];
                [path closePath]; [accent setFill]; [path fill];
            } else {
                double step = stage.size.width/17;
                for (int i=0;i<17;i++) {
                    double bh = fmax(2,FHClamp(frame.levels[i])*stage.size.height);
                    FHRound(NSMakeRect(stage.origin.x+i*step,NSMidY(stage)-bh/2,fmax(1,step-3),bh),2,accent);
                }
            }
        } else {
            NSRect track = NSMakeRect(stage.origin.x,NSMidY(stage)-2,stage.size.width,4);
            FHRound(track,2,FHColor(frame.accent,.2));
            double fraction = FHClamp(frame.progress);
            if (frame.stage == 6) { track.size.width *= fraction; FHRound(track,2,accent); }
            else if (frame.stage == 5) { track.size.height = 2; FHRound(track,1,accent); }
            else if (frame.stage == 4) { FHRound(track,2,FHColor(frame.accent,frame.animated ? .55+.35*sin(t*3) : .8)); }
            else {
                double position = frame.animated ? (sin(t*3)+1)/2 : .5;
                if (frame.stage == 2) position = 1-position;
                double length = frame.stage == 3 ? 24 : 34;
                track.origin.x += position*fmax(0,track.size.width-length); track.size.width = fmin(length,track.size.width);
                FHRound(track,2,accent);
            }
        }
    }
    // Minimal and caption layouts still expose the functional countdown.
    if (frame.stage == 6 && (frame.layout == 0 || frame.captions))
        FHRound(NSMakeRect(12,h-7,(w-24)*FHClamp(frame.progress),3),1.5,accent);
    [NSGraphicsContext restoreGraphicsState];
}
@end

typedef struct {
    pthread_mutex_t lock;
    atomic_uint references;
    uint64_t identifier;
    bool closed, pending;
    FHOverlayPanel *panel; // Main queue only, including teardown.
    FHOverlayView *view;
    NSTimer *timer;
    id accessibilityObserver;
    int timerMS;
    dispatch_semaphore_t destroyed;
} FHOverlay;
static void FHRetain(FHOverlay *o) { atomic_fetch_add(&o->references,1); }
static void FHRelease(FHOverlay *o) {
    if (atomic_fetch_sub(&o->references,1) == 1) {
        dispatch_release(o->destroyed); pthread_mutex_destroy(&o->lock); free(o);
    }
}
static bool FHClosed(FHOverlay *o) {
    pthread_mutex_lock(&o->lock); bool closed = o->closed; pthread_mutex_unlock(&o->lock); return closed;
}
// Notification blocks can outlive observer removal when delivered concurrently.
// Their Objective-C lease keeps the C bridge alive until the last delivery ends.
@interface FHOverlayLease : NSObject {
@public
    FHOverlay *overlay;
}
- (instancetype)initWithOverlay:(FHOverlay *)owner;
@end
@implementation FHOverlayLease
- (instancetype)initWithOverlay:(FHOverlay *)owner {
    if ((self = [super init])) { overlay = owner; FHRetain(owner); }
    return self;
}
- (void)dealloc { FHRelease(overlay); [super dealloc]; }
@end
static void FHStopTimer(FHOverlay *o) {
    [o->timer invalidate]; [o->timer release]; o->timer = nil; o->timerMS = 0;
}
static void FHRefresh(FHOverlay *o) {
    NSCAssert([NSThread isMainThread], @"Overlay drawing requires Cocoa main thread");
    if (FHClosed(o)) return;
    NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
    BOOL reduced = workspace.accessibilityDisplayShouldReduceMotion;
    BOOL contrast = workspace.accessibilityDisplayShouldIncreaseContrast;
    BOOL transparency = workspace.accessibilityDisplayShouldReduceTransparency;
    fh_overlay_frame next = {0};
    // The fixed ABI has two flags: transparency uses the opaque surface override.
    fh_overlay_snapshot(o->identifier,reduced,contrast || transparency,&next);
    if (FHClosed(o)) return;
    if (!next.visible) {
        FHStopTimer(o); [o->panel orderOut:nil];
        if (o->view) { memset(&o->view->frame,0,sizeof(o->view->frame)); [o->view clearCaption]; [o->view setNeedsDisplay:YES]; }
        return;
    }
    if (!o->panel) {
        o->panel = [[FHOverlayPanel alloc] initWithContentRect:NSMakeRect(0,0,1,1)
            styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
            backing:NSBackingStoreBuffered defer:NO];
        o->panel.releasedWhenClosed = NO;
        o->panel.ignoresMouseEvents = YES;
        o->panel.hidesOnDeactivate = NO;
        o->panel.becomesKeyOnlyIfNeeded = YES;
        o->panel.level = NSStatusWindowLevel;
        o->panel.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorFullScreenAuxiliary | NSWindowCollectionBehaviorIgnoresCycle;
        o->panel.opaque = NO; o->panel.backgroundColor = [NSColor clearColor];
        o->panel.hasShadow = NO; o->panel.animationBehavior = NSWindowAnimationBehaviorNone;
        o->view = [[FHOverlayView alloc] initWithFrame:NSMakeRect(0,0,1,1)];
        o->panel.contentView = o->view;
        FHOverlayLease *lease = [[FHOverlayLease alloc] initWithOverlay:o];
        o->accessibilityObserver = [[[workspace notificationCenter]
            addObserverForName:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification
            object:nil queue:nil usingBlock:^(NSNotification *notification) {
                (void)notification;
                // All AppKit reads/painting remain in the coalesced main wake.
                fh_overlay_wake(lease->overlay);
            }] retain];
        [lease release];
    }
    o->view->frame = next; o->view->highContrast = contrast; o->view->reducedTransparency = transparency;
    if (!next.captions) [o->view clearCaption];
    o->panel.alphaValue = FHClamp(next.opacity);
    [o->panel setFrame:NSMakeRect(next.x,next.y,next.width,next.height) display:NO];
    [o->view setNeedsDisplay:YES];
    if (!o->panel.visible) [o->panel orderFrontRegardless];
    int interval = next.timer_ms > 0 ? MAX(16,next.timer_ms) : 0;
    if (interval != o->timerMS) {
        FHStopTimer(o);
        if (interval) {
            o->timerMS = interval;
            // Raw C owner is alive until main-thread cleanup invalidates timer.
            o->timer = [[NSTimer timerWithTimeInterval:interval/1000.0 repeats:YES block:^(NSTimer *timer) {
                (void)timer; @autoreleasepool { FHRefresh(o); }
            }] retain];
            [[NSRunLoop mainRunLoop] addTimer:o->timer forMode:NSRunLoopCommonModes];
        }
    }
}
void *fh_overlay_create(uint64_t identifier) {
    FHOverlay *o = calloc(1,sizeof(*o));
    if (!o) return NULL;
    if (pthread_mutex_init(&o->lock,NULL) != 0) { free(o); return NULL; }
    atomic_init(&o->references,1); o->identifier = identifier;
    o->destroyed = dispatch_semaphore_create(0);
    return o;
}
void fh_overlay_wake(void *bridge) {
    FHOverlay *o = bridge; if (!o) return;
    pthread_mutex_lock(&o->lock);
    if (o->closed || o->pending) { pthread_mutex_unlock(&o->lock); return; }
    o->pending = true; FHRetain(o); pthread_mutex_unlock(&o->lock);
    // Never synchronous: Go calls wake while holding its snapshot mutex.
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            pthread_mutex_lock(&o->lock); o->pending = false; pthread_mutex_unlock(&o->lock);
            FHRefresh(o);
        }
        FHRelease(o);
    });
}
static void FHDestroy(FHOverlay *o) {
    NSCAssert([NSThread isMainThread], @"Overlay teardown requires Cocoa main thread");
    FHStopTimer(o);
    if (o->accessibilityObserver) {
        [[[NSWorkspace sharedWorkspace] notificationCenter] removeObserver:o->accessibilityObserver];
        [o->accessibilityObserver release]; o->accessibilityObserver = nil;
    }
    if (o->view) memset(&o->view->frame,0,sizeof(o->view->frame));
    [o->panel orderOut:nil]; [o->panel close];
    [o->panel release]; o->panel = nil;
    [o->view release]; o->view = nil;
    dispatch_semaphore_signal(o->destroyed);
}
int fh_overlay_close(void *bridge) {
    FHOverlay *o = bridge; if (!o) return 1;
    pthread_mutex_lock(&o->lock); o->closed = true; pthread_mutex_unlock(&o->lock);
    int done = 1;
    if ([NSThread isMainThread]) { @autoreleasepool { FHDestroy(o); } }
    else {
        FHRetain(o);
        dispatch_async(dispatch_get_main_queue(), ^{ @autoreleasepool { FHDestroy(o); } FHRelease(o); });
        done = dispatch_semaphore_wait(o->destroyed,dispatch_time(DISPATCH_TIME_NOW,500*NSEC_PER_MSEC)) == 0;
    }
    FHRelease(o); // Relinquish the Go owner's reference even on bounded timeout.
    return done;
}
uint32_t fh_overlay_capture_display(void) {
    // Window metadata only, never pixels or AX. CG coordinates are top-left.
    CGDirectDisplayID chosen = CGMainDisplayID();
    CFArrayRef windows = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,kCGNullWindowID);
    if (!windows) return chosen;
    for (CFIndex i=0;i<CFArrayGetCount(windows);i++) {
        CFDictionaryRef info = CFArrayGetValueAtIndex(windows,i);
        CFNumberRef layer = CFDictionaryGetValue(info,kCGWindowLayer);
        int value = -1; if (layer) CFNumberGetValue(layer,kCFNumberIntType,&value);
        if (value != 0) continue;
        CFDictionaryRef bounds = CFDictionaryGetValue(info,kCGWindowBounds);
        CGRect rect;
        if (!bounds || !CGRectMakeWithDictionaryRepresentation(bounds,&rect) || CGRectIsEmpty(rect)) continue;
        CGDirectDisplayID displays[32]; uint32_t count = 0;
        if (CGGetDisplaysWithRect(rect,32,displays,&count) == kCGErrorSuccess) {
            double best = 0;
            for (uint32_t j=0;j<count;j++) {
                CGRect intersection = CGRectIntersection(rect,CGDisplayBounds(displays[j]));
                double area = intersection.size.width*intersection.size.height;
                if (area > best) { best = area; chosen = displays[j]; }
            }
        }
        break;
    }
    CFRelease(windows); return chosen;
}
void fh_overlay_work_area(uint32_t display,fh_overlay_rect *rect) {
    if (!rect) return;
    *rect = (fh_overlay_rect){0};
    NSCAssert([NSThread isMainThread], @"NSScreen metadata requires Cocoa main thread");
    NSScreen *chosen = nil;
    for (NSScreen *screen in [NSScreen screens]) {
        if ([screen.deviceDescription[@"NSScreenNumber"] unsignedIntValue] == display) { chosen = screen; break; }
    }
    if (!chosen) chosen = [NSScreen mainScreen] ?: [[NSScreen screens] firstObject];
    NSRect work = chosen.visibleFrame;
    *rect = (fh_overlay_rect){work.origin.x,work.origin.y,work.size.width,work.size.height};
}
