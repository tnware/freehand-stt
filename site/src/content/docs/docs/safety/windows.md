---
title: Windows engineering invariants
description: Contributor requirements for focus, insertion, credentials, audio, and lifecycle safety.
---

## Focus and insertion

- Capture the foreground target when recording begins.
- Never resolve the paste target only after transcription completes.
- Never restore/focus another application silently merely to paste.
- If focus changed, retain the result in backend memory and show an explicit Copy action. Never write it to the clipboard automatically.
- Do not paste after cancellation, timeout, stale generation, or shutdown.
- Before every Unicode dispatch, wait within a fixed bound for Ctrl, Alt, Shift,
  and Windows keys to be released. Revalidate focus and cancellation while
  waiting; require explicit Copy on timeout without releasing keys synthetically.

## Clipboard

- Use Unicode clipboard content.
- Clipboard-paste insertion remains disabled. If implemented later, capture every existing format before mutation and restore only while the clipboard still contains the data object this app set. Never overwrite newer user clipboard content.
- Treat clipboard-open failures as retryable and bounded.
- Explicit Copy owns a message-only window and pins its complete clipboard
  transaction to one OS thread. Open with that owner before emptying; close the
  clipboard before destroying the owner. Allocation and cancellation failures
  before mutation must leave clipboard contents untouched.
- Do not log clipboard content.
- Transcript history never writes automatically to the clipboard; each historical entry requires an explicit Copy action.

## Transcript history

- History is disabled by default and retains nothing until a saved setting enables it.
- Retain only finalized raw/processed transcript text and bounded non-secret run details in the history-owned 20-entry, 2 MiB in-memory ring.
- Enforce both limits after every mutation, including completion and post-processing updates; a single entry may not exceed the total byte budget.
- Never retain audio, provisional text, target application identity, credentials, headers, or full file paths. Cancellation discards unfinished text; a finalized raw transcript already retained before cleanup may remain in enabled history, without authorizing insertion or current-result copy after cancellation.
- Reapply the 20-entry/2-MiB budget after every history mutation. Prefer an explicitly marked raw-only fallback when processed output causes overflow; remove an entry rather than truncate transcript text or silently exceed the budget when raw cannot fit.
- Release an individually removed entry immediately. Clear the ring when history is disabled, when the user clears it, and during shutdown.
- Never put transcript content in logs or crash reports. Do not put historical text in generic status/overlay events. Current-result DTOs and opt-in realtime captions are separate bounded presentation contracts; captions never authorize history, copy, cleanup, or insertion.

## Keyboard hooks and hotkeys

The complete action matrix and normalization rules are documented in
[Shortcut policy](../../reference/shortcuts/).

- Toggle mode may use `RegisterHotKey` with non-repeat behavior.
- Hold-to-talk requires both press and release events through a low-level hook or another proven key-state mechanism.
- Keep callbacks minimal and move work to the owning Go feature.
- Match the native callback ABI: `nCode` is a signed 32-bit integer. Negative
  codes forward immediately without dereferencing the event pointer.
- Unhook and unregister deterministically during shutdown.
- Hold-hook and temporary-capture Close each wait at most two seconds for their
  native source and tracked callback work together. Fence new input first;
  recorder callbacks must not block the native unhook/message-loop completion
  or prevent Wails from reaching bounded feature shutdown.
- Report shortcut conflicts instead of silently falling back.

## Runtime configuration and shutdown

- Treat endpoint settings and their credential as one coherent operation snapshot. Do not pair an endpoint/model captured before a settings save with a credential loaded after that save.
- Capture the complete request profile, including both applicable credentials, under the settings transaction lock before microphone or stored-audio work starts. Allow later settings changes, but apply them only to later operations.
- Derive microphone, stored-file, connection-test, shortcut-capture, and preparation work from the Wails application context and give each operation an explicit timeout or cancellation path.
- Shutdown stops accepting work before cancelling active work, suppresses late publication, and closes stored-file and dictation work before history. Dictation and stored-file services each have a five-second teardown wait budget; speech has two seconds. These are not a shared process-exit deadline or a guarantee that every native call is interruptible. Native capture checks its closed fence around preparation so no late warmup may recreate a resource after close. See [shutdown ownership](../../development/architecture/#shutdown-and-audio-export-ownership).

## WebView boundary

- The WebView is presentation, not a file or device authority. Go opens the native stored-audio picker and retains its result as a private selection capability; the renderer has no bound path argument.
- Reject direct symbolic-link selections. Before upload, reopen the private path and require the same regular-file identity, size, and modification time; disappearance, replacement, or mutation requires reselection.
- Explicitly deny WebView microphone, camera, geolocation, notification, and clipboard-read permissions because those capabilities are implemented by native Go code.
- Keep Wails simple event emission disabled. Adding remote content, raw HTML rendering, or a new WebView permission requires a security review.

## Audio

- Shared mode only for the initial release.
- Bound recording duration and in-memory buffer size.
- Handle microphone removal and default-device change without wedging the state machine.
- An unexpected native stop signals ordinary Go control flow without blocking the audio callback. Discard and zero the interrupted recording; never transcribe a partial utterance after device loss.
- Follow Windows default-device rerouting only when the user selected System default. Never silently fall back from an explicitly selected microphone.
- Device enumeration refreshes only on user-visible settings actions, not background polling, and never rewrites a missing saved device choice.
- Do not retain audio or write predictable temporary paths.
- If a temporary file becomes necessary, use restrictive unique creation and delete it on every terminal path.

## Credentials

- Store API keys in Windows Credential Manager.
- A user-entered key may exist transiently in the renderer password field, with a strict size bound and cleanup after save, settings exit/hide, and teardown.
- Return only availability and credential-reference metadata to Wails/Svelte; never return a stored key.
- Redact Authorization, API keys, cookies, extra secret headers, transcripts, and audio from logs.
- Changing or deleting a profile must not copy another profile's credential implicitly.

## Process lifecycle

- One process owns recording resources.
- A second launch cannot register duplicate hooks or start capture.
- Quit cancels requests and waits for bounded cleanup.
- Crashes must not leave the microphone active because capture belongs to the process.

## Endpoint safety

Safe automatic checks:

```text
GET /health
GET /v1/models
GET /v1/audio/voices  # qualified speech-profile metadata only
```

Forbidden automatic checks:

```text
POST /v1/chat/completions across discovered models
preloading or cycling models
parallel inference probes
```

The settings UI must describe its endpoint test as metadata-only.
Discovered model IDs may populate a selector, but selection itself performs no network request. Connection failures cross the Wails boundary only as stable status metadata; peer-controlled response bodies and credential material do not.
