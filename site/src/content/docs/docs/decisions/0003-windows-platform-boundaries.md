---
title: "ADR 0003: Windows platform implementation boundaries"
description: Assign Windows responsibilities between Wails and focused native adapters.
---

- Status: Accepted for the dictation MVP; overlay ownership superseded by ADR 0004
- Date: 2026-08-29
- Wails version: v3.0.0-beta.16
- Go version family: 1.27

## Decision summary

Use framework-owned behavior where Wails already has a tested Windows implementation, and keep custom Win32 code narrow:

| Capability | Owner |
|---|---|
| Tray, settings-window lifecycle, close interception | Wails |
| Single instance and second-launch reveal | Wails |
| Toggle global shortcut | Wails GlobalShortcut |
| Hold-to-talk press/release | Custom `WH_KEYBOARD_LL` adapter |
| Microphone capture and format conversion | malgo/miniaudio using WASAPI |
| Credential persistence | Windows Credential Manager through a narrow Go store |
| Target capture and insertion | Narrow `user32` adapter plus policy layer |
| Autostart | HKCU Run adapter |
| Runtime state and stale-result fencing | Go coordinator |

Do not duplicate framework tray or single-instance infrastructure with another hidden-window/mutex implementation unless a pinned Wails defect becomes a demonstrated blocker.

## Audio choice

Pin:

```text
github.com/gen2brain/malgo v0.11.26
```

malgo wraps bundled miniaudio v0.11.25, uses WASAPI on Windows, requires CGO, and does not require a separate runtime audio DLL. It is preferable to owning the unsafe COM ABI for `IMMDevice`, `IAudioClient`, and `IAudioCaptureClient` in the MVP.

The Windows adapter:

- initializes a WASAPI-only malgo context;
- enumerates capture devices;
- persists an opaque device ID rather than an array index;
- represents system default separately;
- opens shared-mode capture with an explicit requested `FormatSpec`;
- copies callback bytes into a fixed-capacity bounded recorder;
- performs no allocation, UI, HTTP, encoding, or state-machine work in the callback;
- reports an unexpected stop through a one-item per-recording channel so cleanup stays out of the native callback;
- releases device before context shutdown;
- re-enumerates before explicit-device open;
- discards interrupted PCM and fails visibly on explicit-device invalidation instead of silently changing microphones mid-utterance;
- leaves miniaudio's shared-mode WASAPI auto-routing enabled only for the system-default selection.

MVP file STT requests 16 kHz, mono, signed 16-bit PCM. Future realtime Speaches requests 24 kHz mono PCM16, so format is a parameter rather than a global constant.

The WAV writer is pure Go and independently tested. It performs container encoding only—no gain, AGC, clipping, or silence trimming.

## Wails lifecycle

Create platform objects before `App.Run`, but defer exclusive resources until service startup after first-instance election.

Use:

- `application.Options.SingleInstance` with a stable unique ID;
- `OnSecondInstanceLaunch` to invoke one `Show -> Restore -> Focus` closure;
- an atomic window pointer plus pending-reveal flag for early second launches;
- `app.SystemTray.New()` with explicit Settings and Quit menu entries;
- `RegisterHook(events.Common.WindowClosing, ...)` to hide and cancel close;
- `Windows.DisableQuitOnLastWindowClosed: true` as a fail-safe;
- lifecycle services for coordinator/platform startup and reverse-order shutdown;
- Wails cleanup for Wails-owned global shortcuts, windows, tray, and single-instance infrastructure.

`OnWindowEvent` is observational and asynchronous. Only `RegisterHook` provides synchronous close cancellation.

Constructors passed to `application.New` remain side-effect-free because Wails may terminate a second process from inside application creation without running defers.

## Hotkeys

Toggle mode uses Wails `GlobalShortcut` and reports registration conflicts.

True hold-to-talk cannot use `RegisterHotKey` alone because release is required. The Windows adapter uses a dedicated goroutine locked to one OS thread and a minimal `WH_KEYBOARD_LL` callback:

- track physical down/up state;
- ignore repeat downs and injected events;
- emit one pressed and one released edge;
- always call `CallNextHookEx` and never suppress normal keyboard input;
- never perform capture, HTTP, UI, logging, or blocking channel sends in the hook;
- use a bounded nonblocking edge queue;
- force cancellation/release on overflow, teardown, session loss, or message-loop failure;
- unhook deterministically during shutdown.

If the hook cannot start safely, hold mode is explicitly unavailable; the product does not fake it with toggle behavior.

## Credentials

The domain owns a fakeable credential interface. The Windows implementation stores generic credentials in Windows Credential Manager, either directly or through a narrowly pinned wrapper such as `zalando/go-keyring` backed by `danieljoos/wincred`.

Durable configuration contains only an opaque credential reference and presence status. The renderer never receives stored secret bytes or Authorization headers. A key typed by the user may exist only as a bounded, transient password-field draft and is cleared after save, settings exit/hide, and teardown.

Credential Manager protects at-rest storage but is not isolation from another process running as the same Windows user. Documentation must not claim hardware-backed or cross-process secrecy.

## Target and insertion

Capture target identity immediately before recording:

- foreground top-level HWND;
- focused child HWND when available;
- owning thread/process ID;
- process creation identity from `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` and `GetProcessTimes`.

Before insertion, validate the same target still owns focus. Never call `SetForegroundWindow` or restore focus merely to paste.

Primary insertion uses Unicode `SendInput` with `KEYEVENTF_UNICODE`. A partial `SendInput` count is an ambiguous partial insertion: do not retry or switch strategies automatically.

If any required identity cannot be captured, focus changed, or insertion is unsafe, retain the transcript in bounded Go memory and enter explicit copy-required state. Explicit Copy intentionally writes Unicode text to the clipboard. Automatic paths never write to the clipboard.

The alpha status overlay is disabled after native testing proved the Wails/WebView2 first-show path could activate and invalidate the captured target. ADR 0004 assigns any future passive overlay to a narrow Go-owned native Win32 adapter and keeps it disabled until real-Windows no-activation acceptance passes.

UIPI may prevent a normal-integrity process from inserting into an elevated application. Fall back to explicit copy; never auto-elevate the client.

## Autostart

Use only:

```text
HKCU\Software\Microsoft\Windows\CurrentVersion\Run
```

Write a fixed app-owned value containing the quoted absolute executable path and `--startup`. No elevation. Status distinguishes missing, exact, stale/moved executable, access denied, and command-too-long conditions. Disable removes only the app's value.

## Build contract

malgo requires `CGO_ENABLED=1`. The authoritative CI artifact is built natively on the Windows amd64 runner with the repository's Wails task:

```powershell
wails3 task build CGO_ENABLED=1 ARCH=amd64
```

The result is one unsigned `bin/freehand.exe`; miniaudio is compiled into it and WebView2 remains an operating-system runtime dependency. The runner must provide the Go 1.27+, Node 22+, pinned Wails v3 beta.16, and native C toolchain contract. Linux cross-compilation may remain available as an explicit local fallback, but it is not the normal CI path and cannot establish native behavior.

## Test seams

Platform-neutral interfaces and fakes cover:

- bounded capture and WAV output;
- device selection and failure;
- hotkey edge reduction, repeats, modifier order, overflow, and forced release;
- coordinator generations and stale completion;
- target changes at every insertion boundary;
- Unicode input construction and partial counts;
- credential not-found/replacement/deletion and no-secret DTOs;
- startup exact/stale/missing registry state;
- metadata-only endpoint checks.

Native Windows acceptance covers actual microphone conversion, privacy denial, device removal, hook lifetime, shortcut conflicts, Unicode insertion, focus changes, UIPI failure, Credential Manager, startup, second launch, Explorer/tray restart, and shutdown cleanup.

## Historical pinned-framework caveats

The original Wails v3 beta.3 review found that its Windows single-instance code used `CreateMutex` and a message-only HWND, with two limitations requiring native verification:

- the mutex handle is kept alive until process exit rather than explicitly released through the stored lock field;
- second-instance `SendMessage` does not provide a strong acknowledgement that the reveal callback completed.

Neither justified replacing the framework implementation before a real failure was reproduced. The application is now pinned to beta.16 and uses Wails single-instance encryption; inspect the pinned source before treating these beta.3 observations as current defects.
