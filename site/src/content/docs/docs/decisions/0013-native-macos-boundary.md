---
title: "ADR 0013: Native macOS desktop boundary"
description: Extend Freehand with macOS adapters without weakening native safety and ownership.
---

- Date: 2026-09-12
- Supersedes the Windows-only runtime assumption in ADR 0003; preserves its Windows implementation contract and ADR 0004's native-overlay boundary.
- Toolchain: Go 1.27, Wails v3.0.0-beta.16, macOS 13 or newer, cgo enabled.

## Decision

Keep one application and one set of feature state machines. Add macOS at the
native adapter boundary, not as a renderer-hosted recorder or a parallel product.
Inference remains user-chosen and remote-first. No models or inference runtimes
are added to the app.

| Capability | macOS owner |
| --- | --- |
| Interactive native windows, menu bar, tray and encrypted single instance | Wails and `internal/app` |
| Capture, format conversion and playback | Shared malgo/miniaudio implementation, CoreAudio backend |
| Microphone permission | AVFoundation authorization, checked without prompting during preparation |
| Toggle/show shortcut registration | Pinned Wails Carbon GlobalShortcut implementation |
| Hold edges and temporary shortcut capture | Bounded Quartz event tap and native run-loop owner |
| Target capture and Unicode delivery | NSWorkspace process identity, retained AX focused window and Quartz Unicode events |
| Explicit copy | NSPasteboard, never a shell command |
| Passive status overlay | Nonactivating click-through Cocoa NSPanel |
| Credential persistence | Security.framework generic-password Keychain items |
| Start at login | App-owned per-user LaunchAgent referencing the exact bundle executable |

## Permissions are capabilities, not global readiness

Expose a bounded, non-secret authorization snapshot through `internal/input`.
Only explicit user actions may request authorization or open the corresponding
Privacy & Security pane. Microphone preparation never prompts. Denied microphone,
Accessibility, or Input Monitoring access must not disable file transcription or
text-to-speech. Permission checks are not proof that a target or event tap can be
used: native operations still validate and fail closed.

Native permissions and credential values never travel through JavaScript APIs
other than bounded permission status and transient user-entered credential drafts.
No TCC database edits, permission-dialog automation or plaintext credential fallback
are part of the application.

## Target identity and delivery

A target is a comparable platform-tagged identity, not a fake HWND. Windows
retains HWND/thread/process-start checks. On macOS, NSWorkspace identifies the
frontmost application; retain its PID and process start time plus its AX focused
window. Expose an opaque generation token to the shared insertion policy and
release the retained window on target replacement or shutdown.

Before delivery and each bounded Unicode chunk, verify the same live frontmost
application and window. Reject Freehand itself, missing or stale targets, changed
app/window identity, unavailable Accessibility permission and active Secure Input.
Do not query editor roles, AXEnabled, protected-content metadata or AXValue
settability, and do not require editor-element identity. Moving between fields
within the same window is allowed: delivery goes to the field currently focused
there. Secure Input is a guard, not a promise to recognize every custom secure field.

Never activate or restore another application or synthesize modifier-up events to
overcome physically held keys. Preserve cancellation and surrogate pairs. Quartz
event posting has no application-delivery acknowledgement; do not retry partial or
ambiguous dispatch automatically. Preserve the transcript for explicit copy, the
only path permitted to replace the clipboard.

Rejections expose only allowlisted capture/validate/send stages and static reasons.
Keep the first failure through cleanup and scope capture diagnostics to the recording
generation. Unknown errors reduce to generic copy-required guidance. Neither UI nor
logs may infer that focus moved or that nothing was typed from a generic rejection;
no target identity, editor metadata or transcript belongs in diagnostic reasons.

## Resource and lifecycle ownership

Reuse shared audio session, PCM sink, level tap, cancellation and output bounds.
Select CoreAudio on Darwin and WASAPI on Windows rather than maintaining two
recording engines. Resolve explicit device identity anew before opening it.

Native keyboard callbacks only publish bounded input state. Recording, inference,
UI and feature transitions run outside the callback. Lost input, tap disablement,
secure input and queue overflow must cancel or reject rather than leave recording
latched. Temporary shortcut capture owns its suppression scope and cancellation.

The overlay projects the existing Go presentation state. It cannot become key or
main, receive clicks, activate another app or create a WebView. AppKit operations
belong to the main thread; updates are coalesced and timers stop when hidden.
Operation placement does not follow unrelated later focus changes. Captions remain
bounded, transient presentation-only content. System reduced motion, contrast and
transparency preferences constrain presentation.

Cocoa may terminate inside its event loop, so cleanup cannot rely on code after
`App.Run`. Stop external sources in Wails shutdown hooks; close feature services
before releasing retained target state and SQLite in `PostShutdown`. Retain an
idempotent fallback for startup failures and platforms where Run returns.

## Shell, settings and packaging

Use native titlebars, standard editing/menu shortcuts, close-to-hide and a Dock
reopen hook that reveals only the main shell. Wails' default all-window reopen is
not suitable for hidden connection/credential editors. Use a monochrome template
PNG for the macOS status item, not Windows ICO resources.

Backend platform metadata is the renderer's authority for native capability labels.
Command/Option labels replace Windows-specific terminology; Mica remains Windows-only.
The legacy `startWithWindows` settings key remains the serialized login-startup
preference so existing settings and migrations do not split unnecessarily.

Derive bundle identity/version from the existing release configuration. Bundle
microphone usage text and a consistent minimum deployment target are mandatory.
Development and release identity must be deliberate. A local ad-hoc signature is
not Developer ID signing, notarization or Gatekeeper acceptance. macOS update assets
are matching-architecture app archives, not Windows executables; released update
application requires native signed-bundle verification.

## Verification boundary

Deterministic Go/browser fixtures establish policy, DTO and lifecycle behavior.
They do not establish permissions, actual microphone/output routing, keyboard taps,
Unicode delivery, focus preservation, Keychain prompts or login startup. Those
require a real packaged Mac application and disposable native fixtures. Native
Windows acceptance remains separate. Do not qualify inference by iterating models;
use only an explicitly selected endpoint and model.
