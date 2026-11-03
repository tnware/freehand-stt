# Linux keyboard adapter validation

The native adapter uses a dedicated XCB connection for each registered shortcut
and for explicit shortcut capture. It does not install Xlib error handlers,
change the desktop keyboard map, synthesize events, or convert input into text.
Build dependencies are `xcb` pkg-config development metadata and the official
X.Org XKB protocol headers (`libxcb1-dev` and `x11proto-dev` on Debian).

## Capability boundary

- Native keyboard support requires `XDG_SESSION_TYPE=x11`, no
  `WAYLAND_DISPLAY`, and a local `DISPLAY` such as `:0`. It does not claim
  Wayland or XWayland global hooks. Wayland Toggle/Show belongs to the portal.
- XKB detectable autorepeat must be supported and enabled for this connection.
  This is a per-client flag, not a desktop-wide repeat setting. See the
  [X.Org protocol specification](https://www.x.org/docs/XKB/XKBproto.pdf).
- Modifier and lock masks come from the server map. Ambiguous modifier maps and
  non-unique primary key mappings fail admission. Key names follow the first
  keysym group; unsupported primary symbols are rejected.
- Modifier-only holds are unavailable. A primary-key hold ends when its primary
  key or either physical side of a required modifier is released. Releasing one
  modifier side ends the hold conservatively even if its other side is down.
- Mapping changes and source loss stop the source. Missing release edges cancel
  rather than submit. A hold lasting two minutes is also canceled as a stale
  hold; release the keys and explicitly retry before recording again.
- Explicit shortcut capture owns an exclusive keyboard grab for at most thirty
  seconds, accepts Escape/cancel, and uses the shared chord reducer. Callbacks
  never own the native resource lifetime. Callback shutdown has a one-second
  bound; native grabs are released before waiting for arbitrary callbacks.
- Local X connection admission has a 1.5-second bound. At most one stalled
  handshake remains pending, and it cannot acquire any keyboard grab.

## Deterministic tests

Run `go test -race ./internal/platform/keyboard ./internal/hotkey` with CGo and
the development packages available. The regular suite never grabs a keyboard.
It covers repeat suppression, modifier sides/masks, loss, cancellation, failed
replacement, disabled holds, and a blocked callback whose pending disable must
be upgraded to cancellation by Close.

## Explicit native acceptance

Do not equate a build or fake-event test with native execution. On the selected
X11 desktop, build the keyboard package's test executable and run individual
probes with `FREEHAND_X11_ACCEPTANCE=1`. Keep `XDG_SESSION_TYPE` truthful; do not
force an XWayland session through the guard. The probes wait one second for the
terminal Enter key to be released and never synthesize input or alter key maps.

```
go test -c -o /tmp/freehand-keyboard-acceptance ./internal/platform/keyboard
FREEHAND_X11_ACCEPTANCE=1 /tmp/freehand-keyboard-acceptance -test.v -test.run '^TestNativeX11GrabLifecycle$'
```

Run the following separately with the same environment and `-test.run`:

- `TestNativeX11GrabLifecycle`: automatic passive-grab conflict, teardown/regrab,
  capture timeout, and capture relinquishing its exclusive grab.
- `TestNativeX11HoldEdges`: after READY, hold Ctrl+Shift+F8 for two seconds, then
  release. Exactly one press and one release are required.
- `TestNativeX11GlobalRepeat`: after READY, hold Ctrl+Shift+F8 for two seconds,
  then release. Exactly one toggle callback is required during the four-second
  observation following its first press.
- `TestNativeX11CaptureChord`: after READY, press and release Ctrl+Shift+F9.
- `TestNativeX11CaptureEscape`: after READY, press Escape.

Also manually qualify modifiers released before the primary, CapsLock/NumLock,
existing desktop shortcut conflicts, closing during capture/hold, and changing
the keyboard layout. Record the actual desktop/session and outcomes separately
from deterministic tests. A conflict or unsupported map is a refusal, not a
successful native acceptance. No test here authorizes focus-safe insertion.
