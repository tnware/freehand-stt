# Experimental X11 status overlay

The Linux overlay is presentation-only. It uses one dedicated XCB connection and
native Cairo/Pango rendering, with no WebView. A local X11 session, Shape 1.1
and RandR are required. Wayland and XWayland are unavailable; the desktop must
not be relabeled to enable this path. There is no automatic insertion transport.

The override-redirect window has an empty input shape before its first map,
ICCCM input=False, notification/skip-taskbar hints and no input event selection.
It never activates a window, grabs input, synthesizes events, or changes any
selection. Only its own drawable is painted. Status/caption text is bounded,
transient, passed to Pango as plain text, and erased on hide/close. No text goes
into window titles, properties, logs or files.

The focused window's monitor is captured when a dictation/preview generation
first becomes visible. Pointer location is a geometry-only fallback. The overlay
stays on that monitor while the operation runs, respects the current desktop's
work area, and reclamps after RandR/work-area changes. Six anchors and four
layouts are supported, plus live amplitude visualizers, countdown, and bounded
captions. Microphone amplitude is consumed only as a scalar, not audio.

Transparency requires a compositor and a compatible 32-bit visual. Otherwise the
window is opaque. Glass is translucent without blur. GNOME motion/high-contrast
preferences are read at native initialization when their schema exists; unknown
motion preferences use reduced decorative motion. The explicit Reduced option
always wins. Native multi-monitor/compositor/fractional-scaling acceptance still
needs the intended desktop; an Xvfb test is not that evidence.

One Go owner thread serializes native calls. Updates use a latest-value mailbox;
hidden windows have no timer. Close revokes work/clears the mailbox immediately,
then waits at most one second. A stalled server produces a shutdown error rather
than a false successful-cleanup claim. One pending connection limit prevents
repeated failed handshakes from creating unbounded owners.

## Isolated native qualification

Install the normal native build dependencies plus Xvfb, xauth and dbus-daemon.
The following opt-in tests create their own synthetic focus window and selection
owner and change their private X server's work-area property. **Run only on a
fresh isolated Xvfb**, never an existing user's desktop:

```sh
XDG_SESSION_TYPE=x11 GDK_BACKEND=x11 dbus-run-session -- \
  xvfb-run -a -s '-screen 0 1280x900x24 -nolisten tcp -noreset' \
  env FREEHAND_X11_OVERLAY_ACCEPTANCE=1 \
  go test -count=1 -race -v ./internal/platform/overlay
```

Native assertions check unchanged focus and selection ownership, empty input
shape, nonactivating window hints, all layouts/anchors, work-area changes,
Unicode rendering, hide/destroy and concurrent Go ownership/shutdown. They never
start WebKit or inference and never generate keyboard input. To export only the
synthetic fixture, explicitly set `FREEHAND_X11_OVERLAY_SNAPSHOT` to a PNG path.
The normal PR distro lanes run these probes separately from the frontend smoke.
