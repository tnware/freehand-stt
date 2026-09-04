---
title: "ADR 0004: Keep WebView2 out of the focus-sensitive overlay"
description: Keep the passive focus-sensitive status surface native and non-activating.
---

- Status: Implemented as a configurable native surface; expanded-layout real-Windows acceptance pending
- Date: 2026-08-29
- Supersedes: the overlay ownership in ADR 0003
- Related issues: #4, #6

## Context

The client is one Windows GUI executable and one Go process. Wails is embedded into that process; it is not a second application runtime or a daemon. It currently supplies:

- the Windows UI/message-loop integration;
- the interactive settings window hosted in WebView2;
- Go-to-Svelte events and generated Svelte-to-Go calls;
- the tray menu;
- single-instance routing and settings-window reveal;
- toggle/show global-shortcut registration.

Go owns the product runtime regardless of whether the settings window is open:

- coordinator state and generation fencing;
- microphone capture and WAV encoding;
- STT HTTP requests;
- Credential Manager access;
- low-level hold-to-talk hook;
- foreground identity capture and validation;
- Unicode insertion and copy-required policy;
- startup registration and shutdown sequencing.

The first alpha also created the transient status overlay as a second Wails/WebView2 window. Native testing showed that its first-show path could activate the window even after `WS_EX_NOACTIVATE` and `SWP_NOACTIVATE` were applied. That changed the foreground target and correctly forced focus-safe insertion into copy-required mode. A passive status surface belongs inside the focus-safety boundary; a framework show path that can activate is not acceptable there.

## Decision

1. The native-overlay test build creates no second Wails/WebView2 window. It creates one hidden Go-owned Win32 tool window after the primary Wails application starts; coordinator state updates show, repaint, or hide that native window.
2. The compact overlay is a native Win32 window owned by Go, not a Wails/WebView2 window.
3. The adapter owns a dedicated locked OS thread and message loop. It creates the window hidden with `WS_EX_NOACTIVATE | WS_EX_TOOLWINDOW | WS_EX_TOPMOST | WS_EX_LAYERED | WS_EX_TRANSPARENT`, composites one premultiplied-ARGB surface with GDI+ and presents it through `UpdateLayeredWindow`, returns `MA_NOACTIVATE` and `HTTRANSPARENT`, and shows through `SW_SHOWNOACTIVATE` plus `SWP_NOACTIVATE`. `UpdateLayeredWindow` neither activates a window nor takes focus, so per-pixel alpha buys antialiased edges, a soft shadow and a real accent bloom without weakening any focus-safety property of the original GDI path.
4. The overlay contains no controls, receives no transcript or credential content, never owns coordinator/insertion state, and never calls `SetForegroundWindow` or `AttachThreadInput`. Minimal, Capsule, and Meter remain glyph-led. Detailed may render only a fixed product name, fixed phase/instruction strings, a backend-normalized shortcut, bounded elapsed time, and a bounded checkpoint count. User text, filenames, endpoint/model identifiers, credentials, prompts, provider errors, and transcript content are prohibited.
5. Visibility is a saved policy: recording only; recording plus transcription/post-processing; or all phases and bounded outcomes. Every state retains a distinct glyph, stage, and colour so text is never the only way to understand it. Capsule remains the compatibility default.
6. The implementation is an unsigned native-test candidate. Cross-compilation establishes source/ABI compatibility only. If real-Windows acceptance shows any activation or focus change, the overlay must be disabled again rather than weakening target validation.
7. Wails remains the settings shell. Replacing its tray, single-instance, shortcuts, and settings UI is a separate measured architecture decision, not part of the overlay implementation.
8. The settings renderer may request a native preview using only a validated presentation DTO and normalized shortcuts. Preview is presentation-only, never changes saved configuration, never runs while dictation is active, is preempted by real dictation, and is stopped when Settings closes. The native surface is destroyed afterward when the applied overlay setting is off.
9. Placement follows the foreground application's monitor at the start of each dictation and then remains stable for that operation. The six supported anchors are computed against the monitor work area, not raw screen bounds. Focus changes during an operation do not move the overlay.
10. Decorative motion follows Windows Animation Effects unless the user chooses Reduced. The functional silence countdown continues to update. Windows contrast themes override opacity, surface, glow, and palette so platform accessibility remains authoritative.

## Why not remove Wails immediately?

Removing Wails would remove WebView2 and Svelte, but it would also make the project directly responsible for:

- the main Win32 message loop and thread affinity;
- tray icon recreation after Explorer restarts;
- single-instance activation and second-launch routing;
- global shortcut registration and conflict handling;
- a complete settings UI, including DPI, accessibility, keyboard navigation, dark/high-contrast behavior, validation, and scrolling;
- packaging and lifecycle details currently delegated to Wails.

For a passive overlay, direct Win32 is small and justified. Replacing the full settings shell is a larger product tradeoff and must be measured rather than assumed.

## Target ownership map

| Capability | Owner after this decision |
|---|---|
| Settings window and renderer bridge | Wails + WebView2 + Svelte |
| Tray, single instance, settings reveal | Wails |
| Toggle/show shortcuts | Wails for now |
| Hold-to-talk | Go Win32 adapter |
| Audio, STT, credentials, coordinator | Go |
| Target capture and insertion | Go Win32 adapter |
| Native passive overlay | `internal/overlay` lifecycle/policy service + narrow Go Win32 renderer; real-Windows acceptance required |

## Native overlay acceptance

Real-Windows acceptance must prove:

- foreground top-level HWND, focused child HWND, thread ID, process ID, and process creation identity are unchanged across first show, updates, and hide;
- Notepad and at least one Chromium-based application retain caret and keyboard focus;
- the overlay never appears in Alt+Tab or the taskbar;
- mouse interaction passes through or is ignored according to the final policy;
- DPI and multi-monitor placement remain bounded;
- all four layouts, six anchors, three surfaces, four visualizers, three visibility policies, and both motion policies remain readable and non-activating;
- the native preview uses the same renderer, stops on Settings close, restores applied settings after discard, and is immediately preempted by real dictation;
- Detailed renders only the fixed operational allowlist and never receives content/provider metadata;
- Windows Animation Effects and contrast themes override presentation as documented without disabling the functional countdown;
- repeated show/update/hide does not leak windows, GDI or GDI+ objects, threads, or timers;
- direct insertion succeeds only when the original target is still authoritative;
- a genuine focus change still produces copy-required behavior.

Cross-compilation cannot satisfy these criteria.
