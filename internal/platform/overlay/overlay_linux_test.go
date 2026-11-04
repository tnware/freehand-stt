//go:build linux && cgo

package overlay

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func TestLinuxOverlayProjectionBoundsAndPrivacy(t *testing.T) {
	now := time.Now()
	status := OverlayStatus{Kind: OverlayRecordingCountdown, CaptionEnabled: true, Caption: strings.Repeat("😀", 300), Shortcut: strings.Repeat("⌘", 100) + "\x00\u202e", StartedAt: now.Add(-8 * time.Second), CountdownDeadline: now.Add(time.Second), CountdownDuration: 2 * time.Second}
	f := projectLinuxOverlay(status, OverlayOptions{Scale: math.NaN(), Opacity: math.Inf(1), Glow: math.Inf(-1)}, now)
	if f.visible != 1 || f.scale != 1 || f.opacity < .9 || f.progress != .5 {
		t.Fatalf("invalid projected bounds: scale=%v opacity=%v progress=%v", f.scale, f.opacity, f.progress)
	}
	caption := []byte{}
	for _, b := range f.caption {
		if b == 0 {
			break
		}
		caption = append(caption, byte(b))
	}
	if !utf8.Valid(caption) || utf8.RuneCount(caption) != 180 {
		t.Fatal("caption must remain bounded valid UTF-8")
	}
	status.CaptionEnabled = false
	f = projectLinuxOverlay(status, DefaultOverlayOptions(), now)
	if f.caption[0] != 0 {
		t.Fatal("disabled caption retained")
	}
	status.Kind = OverlayHidden
	f = projectLinuxOverlay(status, DefaultOverlayOptions(), now)
	if f.visible != 0 || f.caption[0] != 0 || f.shortcut[0] != 0 {
		t.Fatal("hidden surface retained text")
	}
}
func TestLinuxOverlayPhasesAndShortcut(t *testing.T) {
	for kind := OverlayRecording; kind <= OverlayRecordingCountdown; kind++ {
		label, instruction := linuxOverlayPhase(OverlayStatus{Kind: kind})
		if label == "" || instruction == "" {
			t.Fatalf("missing phase %v", kind)
		}
	}
	_, instruction := linuxOverlayPhase(OverlayStatus{Kind: OverlayRecording, RecordingMode: OverlayRecordingHold})
	if !strings.HasPrefix(instruction, "Release") {
		t.Fatal("hold instruction must describe release")
	}
	if got := linuxOverlayShortcut("Ctrl\x00\u202e+F8"); got != "Ctrl+F8" {
		t.Fatal("unsafe shortcut controls")
	}
}
func TestLinuxOverlayRejectsWayland(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", ":99")
	if overlay, err := NewStatusOverlay(); err == nil || overlay != nil {
		t.Fatal("Wayland/XWayland must not create an X11 overlay")
	}
}
func TestLinuxOverlayNativeSafety(t *testing.T) {
	if os.Getenv("FREEHAND_X11_OVERLAY_ACCEPTANCE") != "1" {
		t.Skip("explicit isolated Xvfb-only native overlay qualification")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	flags, err := exec.CommandContext(ctx, "pkg-config", "--cflags", "--libs", "xcb", "xcb-shape", "xcb-randr", "cairo-xcb", "pangocairo", "gio-2.0").Output()
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "overlay-probe")
	args := []string{"-std=c11", "-O2", "-D_FORTIFY_SOURCE=2", "-D_DEFAULT_SOURCE", "-Wall", "-Wextra", "-Werror", "-o", binary, "testdata/overlay_probe_linux.c", "overlay_linux.c"}
	args = append(args, strings.Fields(string(flags))...)
	args = append(args, "-lm")
	if out, err := exec.CommandContext(ctx, "cc", args...).CombinedOutput(); err != nil {
		t.Fatalf("native fixture compilation: %v\n%s", err, out)
	}
	probeArgs := []string{}
	if path := os.Getenv("FREEHAND_X11_OVERLAY_SNAPSHOT"); path != "" {
		probeArgs = append(probeArgs, path)
	}
	if out, err := exec.CommandContext(ctx, binary, probeArgs...).CombinedOutput(); err != nil {
		t.Fatalf("native overlay safety: %v\n%s", err, out)
	} else {
		t.Log(strings.TrimSpace(string(out)))
	}
}
func TestLinuxOverlayNativeLifecycle(t *testing.T) {
	if os.Getenv("FREEHAND_X11_OVERLAY_ACCEPTANCE") != "1" {
		t.Skip("explicit isolated Xvfb-only native overlay qualification")
	}
	overlay, err := NewStatusOverlay()
	if err != nil {
		t.Fatal(err)
	}
	defer overlay.Close()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				if err := overlay.Configure(DefaultOverlayOptions()); err != nil {
					t.Error(err)
					return
				}
				if err := overlay.Update(OverlayStatus{Kind: OverlayRecording, CaptionEnabled: true, Caption: "synthetic transient text", Generation: 1}); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()
	// Ensure the owner processes its latest frame before testing hide/Close.
	time.Sleep(100 * time.Millisecond)
	if err := overlay.Update(OverlayStatus{}); err != nil {
		t.Fatal(err)
	}
	if err := overlay.Close(); err != nil {
		t.Fatal(err)
	}
	if err := overlay.Update(OverlayStatus{Kind: OverlayRecording}); err == nil {
		t.Fatal("closed overlay admitted update")
	}
	if err := overlay.Configure(DefaultOverlayOptions()); err == nil {
		t.Fatal("closed overlay admitted configuration")
	}
	overlay.mu.Lock()
	defer overlay.mu.Unlock()
	if overlay.status.Caption != "" || overlay.source != nil {
		t.Fatal("closed owner retained transient data")
	}
	if overlay.failure != nil {
		t.Fatal(overlay.failure)
	}
}
