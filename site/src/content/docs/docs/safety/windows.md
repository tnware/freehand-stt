---
title: Windows engineering invariants
description: Contributor requirements for focus, insertion, credentials, audio, and lifecycle safety.
---

Use these invariants when reviewing Windows adapters and workflow changes. Pair
them with the [native acceptance checklist](../native-test-checklist/); passing
deterministic tests alone does not establish desktop behavior.

| Change area | Start with |
| --- | --- |
| Text delivery or Copy | [Focus and insertion](#focus-and-insertion), [Clipboard](#clipboard) |
| Recording or shortcuts | [Audio](#audio), [Keyboard hooks and hotkeys](#keyboard-hooks-and-hotkeys) |
| Settings or new renderer methods | [Credentials](#credentials), [WebView boundary](#webview-boundary) |
| Background work or shutdown | [Runtime configuration](#runtime-configuration-and-shutdown), [Process lifecycle](#process-lifecycle) |
| Retention or server probes | [Transcript history](#transcript-history), [Endpoint safety](#endpoint-safety) |

## Focus and insertion

:::caution[The recording target is the delivery boundary]
Capture the foreground target when recording begins. Never choose a new target
after transcription, reactivate an application to paste, or deliver after
cancellation, timeout, stale generation, or shutdown.
:::

| At this boundary | Required behavior |
| --- | --- |
| Recording starts | Capture the target's HWND/thread/process identity |
| Before delivery and each chunk | Revalidate the captured target and focus |
| Before every Unicode dispatch | Wait within a fixed bound for Ctrl, Alt, Shift, and Windows keys to be released; recheck focus and cancellation while waiting |
| Focus changes or modifier wait times out | Keep the result in backend memory and offer explicit Copy; never write to the clipboard automatically or release keys synthetically |

## Clipboard

| Concern | Requirement |
| --- | --- |
| Content | Use Unicode; never log clipboard content |
| Contention | Retry clipboard-open failures only within a bound |
| Explicit Copy ownership | Own a message-only window and pin the complete transaction to one OS thread |
| Transaction order | Open with that owner before emptying; close before destroying the owner |
| Failure before mutation | Allocation/cancellation failure leaves existing contents untouched |
| History | Every entry needs an explicit Copy action |

:::note[Clipboard-paste insertion is disabled]
Any future implementation must capture every existing format before mutation,
then restore only while the clipboard still holds Freehand's data object. Never
overwrite newer user clipboard content.
:::

## Transcript history

| Rule | Required behavior |
| --- | --- |
| Default | Disabled; retain nothing until a saved setting enables history |
| Allowed content | Finalized raw/processed text and bounded non-secret run details |
| Storage | History-owned **20-entry, 2 MiB** in-memory ring |
| Every mutation | Reapply both limits, including completion and cleanup updates; one entry cannot exceed the total byte budget |
| Processed text overflows | Prefer an explicitly marked raw-only fallback |
| Raw text cannot fit | Remove the entry; never truncate text or exceed the budget |
| Removal | Release an individually removed entry immediately |
| Disable, Clear, shutdown | Clear the ring |

Never retain audio, provisional text, target identity, credentials, headers, or
full paths. Never send transcript content to logs/crash reports, or historical
text to generic status/overlay events.

:::note[Cancellation and captions do not grant delivery permission]
Cancellation discards unfinished text. A finalized raw transcript already stored
before cleanup may remain in enabled history, but does not authorize insertion
or current-result Copy after cancellation. Current-result DTOs and opt-in
realtime captions have separate bounded presentation contracts; captions never
authorize history, Copy, cleanup, or insertion.
:::

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

1. **Snapshot before work starts.** Capture the complete profile and both
   applicable credentials under the settings transaction lock. Later saves affect
   later operations; never combine an old endpoint/model with a new credential.
2. **Own work through the application context.** Microphone, stored-file,
   connection-test, shortcut-capture, and preparation work derive from Wails and
   have explicit cancellation or timeout paths.
3. **Fence and cancel during shutdown.** Stop accepting work, cancel active work,
   suppress late publication, and close stored-file/dictation work before history.
   Native capture checks its closed fence around preparation so late warmup cannot
   recreate resources.

| Owner | Teardown wait budget |
| --- | --- |
| Dictation service | 5 seconds |
| Stored-file service | 5 seconds |
| Speech service | 2 seconds |

These are per-service waits, not a shared process-exit deadline or a guarantee
that every native call is interruptible.

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

| Location or action | Requirement |
| --- | --- |
| Durable API key | Windows Credential Manager only |
| Renderer password draft | Transient and strictly bounded; clear after save, settings exit/hide, and teardown |
| Returned Wails/Svelte data | Availability and credential-reference metadata only; never a stored key |
| Logs | No Authorization, API keys, cookies, secret headers, transcripts, or audio |
| Change/delete profile | Never implicitly copy another profile's credential |

## Process lifecycle

- One process owns recording resources.
- A second launch cannot register duplicate hooks or start capture.
- Quit cancels requests and waits for bounded cleanup.
- Crashes must not leave the microphone active because capture belongs to the process.

## Endpoint safety

Only metadata may be checked automatically:

```http title="Permitted metadata routes"
GET /health
GET /v1/models
GET /v1/audio/voices  # qualified speech-profile metadata only
```

:::caution[Never probe a model inventory with inference]
Do not call chat completions across discovered models, preload/cycle models, or
run parallel inference probes. Live checks use only an explicitly selected
endpoint/model and respect its resource limits.
:::

The settings UI must describe its endpoint test as metadata-only.
Discovered model IDs may populate a selector, but selection itself performs no network request. Connection failures cross the Wails boundary only as stable status metadata; peer-controlled response bodies and credential material do not.
