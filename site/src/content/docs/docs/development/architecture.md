---
title: Architecture
description: Runtime ownership, platform boundaries, and application data flow.
---

## Authority map

| Concern | Owner |
|---|---|
| Live dictation state machine | `internal/dictation` |
| Global shortcuts | Windows platform adapter |
| Audio capture and normalization | Go audio service |
| Endpoint requests and cancellation | Go provider-neutral inference client |
| Native stored-file grant and transcription job | `internal/filetranscription` + inference client |
| Optional transcript post-processing | `internal/postprocess` request/outcome policy; feature-owned execution |
| API credentials | Windows Credential Manager adapter |
| Original target and insertion | Windows focus/input adapter |
| Optional transcript history | `internal/history` memory store |
| Optional transcript synthesis and native playback | `internal/tts` + inference speech capability + Windows playback adapter |
| Optional passive status overlay | Cohesive Go overlay service + narrow native Win32 renderer |
| Native tray presentation/actions | `internal/tray` consuming bounded domain snapshots |
| Tray ownership, startup, single instance | Go/Wails Windows lifecycle |
| Settings and status rendering | Svelte through generated Wails bindings |
| Durable non-secret configuration | `%APPDATA%` JSON |
| Structured runtime diagnostics | One Wails default logger hierarchy, injected by `internal/app` |
| Release identity and version | `build/config.yml`, parsed by `internal/releaseinfo` |
| Release discovery and staged executable updates | `internal/updates` + Wails updater GitHub provider |

The frontend never receives a stored API key, raw audio, or selected filesystem path. A key being entered by the user exists only as a bounded, transient password-field draft until it is saved to Windows Credential Manager or the settings flow is left. Go opens the Wails native file picker and converts its result into a backend-only selection capability; the zero-argument renderer binding cannot nominate another path. Status events contain the opaque operation generation, base file name, byte progress, mode, and transcript text, never the full path or audio bytes.

At selection, Go opens and validates the file as a supported, bounded regular file. A direct symbolic-link selection is rejected. At upload start, Go reopens the private path and requires the same operating-system file identity, size, and modification time before streaming from that opened handle. Disappearance, replacement, or pre-start mutation fails closed and requires reselection. Regular files reached through platform-managed reparse-backed storage remain usable only when they open as regular files and still satisfy the same identity/metadata fence.

## Runtime state

```text
Idle
  -> Recording
  -> Transcribing
  -> PostProcessing (when enabled)
  -> ReadyToInsert
  -> Idle

Any active state
  -> Cancelling
  -> Idle

Recording, Transcribing, or PostProcessing
  -> Failed
  -> Idle or explicit Retry
```

Only the dictation package performs live-recording state transitions. Platform and network adapters return events/results tagged with an operation generation so a late completion cannot affect a newer recording.

## Packages

```text
cmd/ or root main.go       composition and Wails lifecycle
internal/dictation        live recording state machine and renderer commands
internal/history          bounded transcript store and renderer queries/actions
internal/filetranscription native file grant and stored-audio job
internal/settings         transactional settings/profile owner
internal/connection       bounded endpoint metadata probes
internal/input            microphone inventory and shortcut capture
internal/audio             interfaces, WAV normalization, capture ownership
internal/inference         focused STT, chat, streaming, and metadata protocol adapter
internal/tts               on-demand synthesis and single-session playback state machine
internal/config            non-secret profiles and validation
internal/credential        credential interface
internal/insertion         focus-safe insertion policy
internal/platform          Windows implementations plus non-Windows stubs
internal/postprocess       transcript-cleanup request and outcome policy
internal/tray              native status, last-activity, recovery, and window actions
internal/updates           persisted polling policy and Wails updater lifecycle
frontend/src/lib           testable settings/status state
frontend/src               thin Svelte components
```

The renderer sees small Wails services registered from the package that owns each capability. Wails is the bridge boundary, not the application's package hierarchy:

| Bound package | Renderer responsibility | Backend authority |
|---|---|---|
| `settings` | Renderer-safe snapshot and one atomic settings/credential/startup/shortcut save request | Settings transaction owner |
| `input` | Microphone inventory and native shortcut capture | Audio and Windows keyboard adapters |
| `connection` | Focused STT and post-processing health/model probes | Inference metadata capability |
| `dictation` | Live commands and status snapshot | Package-owned recorder |
| `history` | Bounded history queries, copy, delete, and clear | Package-owned synchronized store |
| `filetranscription` | Native picker grant, upload/transcription state, retry, cancellation, copy | Package-owned file job using the injected history store |
| `tts` | Listen to backend-owned transcript versions; preview, pause, resume, restart, stop, and status | Package-owned synthesis/playback session |
| `updates` | Current bounded status and explicit user-initiated update review | Package-owned polling policy over the configured Wails updater |

Focused operations use generated request DTOs. A connection probe receives only its endpoint, model-discovery, headers, authentication policy, and bounded transient credential draft, so unrelated shortcut, VAD, window, history, or processing drafts cannot invalidate it. Settings save similarly receives one generated `SaveSettingsRequest` instead of a positional credential argument list. Svelte constructs and consumes those generated shapes directly.

Shortcut settings follow the same boundary. `internal/hotkey` owns one bounded action matrix for toggle recording, Show Freehand, and hold to talk. `internal/input` exposes that matrix as generated policy metadata, accepts one action-specific capture request, and emits bounded normalized chord progress while the native hook is active. The renderer never selects native validation flags or maintains a second key grammar. Expected capture rejections return structured categories; a cross-process global conflict remains knowable only when the transactional settings save asks Windows to register the replacement.

After loading the applied profile, the renderer runs one bounded metadata-only STT probe and repeats it only when that profile's connection identity changes. First-run readiness is an exclusive app-shell content state backed by a persisted completion flag; the Go recording command rejects both renderer and global-shortcut starts until it is complete. A later failed probe may take over the content area once, but the established user can continue without correction while the persistent status strip retains the warning. Dismissal is scoped to the exact failed endpoint/model/authentication/microphone condition rather than mutating durable settings.

Ordinary Go collaborators remain ordinary types: `history.Store`, the inference client, post-processor, capture adapter, and insertion policy are injected by `internal/app` and are not registered with Wails. The settings/profile transaction remains one owner even though consumers receive narrow snapshot functions.

The configuration store decodes known fields over safe defaults while retaining
the original JSON object. Unknown top-level and nested fields are reported as
newer-version compatibility metadata and merged back unchanged when known
settings are saved. A malformed document or invalid known value creates an
explicit recovery state instead of making defaults look applied: the settings
service rejects ordinary saves and request-profile capture, while its narrow
Retry and Reset bindings either re-read the untouched file or deliberately
replace it. Credentials remain outside this document in Windows Credential
Manager.

The native tray controller is also an ordinary Go collaborator rather than a
renderer service. It consumes the same bounded dictation and stored-file
snapshots already published by their domain owners, reduces them to fixed
status and last-activity labels, and never accepts transcript text, file names,
paths, model identities, or endpoint details into its presentation model. Its
menu exposes safe recovery and lifecycle actions: Show/Hide Freehand, Settings,
About, cancellation for active work, explicit transcript copy when the owning
domain reports it available, and authoritative Quit. It intentionally cannot
start recording because opening a tray menu cannot preserve an honest text
insertion target. The runtime controller does not synthesize or repurpose
brand imagery. It receives generated, purpose-drawn light and dark Windows ICO
families at composition time and lets Wails select the closest frame and react
to system-theme changes. The canonical vector sources and consumer matrix are
documented in [Brand asset pipeline](../brand-assets/).

`internal/inference` is an application-owned, provider-neutral adapter for the
small OpenAI-compatible capability profile this client actually uses. Its
single composed HTTP client is divided by responsibility: transport and safe
errors, completed microphone transcription, stored-file multipart upload,
transcription SSE normalization, post-processing chat completion, and bounded
metadata probes. Consumers use that package name directly rather than aliasing
a provider name.

The shared inference client constructor owns redirect denial for every route;
no redirect may replay credentials, audio, or text, even to the same origin.
Tests that replace network I/O must retain this production policy by using
`inference.New()` and replacing only its `HTTP.Transport`. Response metadata
uses one shared literal-credential sanitizer before publication to feature
owners, history, or renderer DTOs. Bounded parsers apply the same string rule
before truncation, including nested usage/language fields and request-ID
headers. Model discovery filters reflected IDs; speech synthesis exports no
response metadata. Unsafe optional metadata never discards otherwise valid
transcript text; text reflection rejection and privacy-safe errors remain
separate checks.

The adapter remains focused instead of vendoring a general OpenAI SDK. A broad
SDK would not replace the application's exact multipart-length and progress
contract, older Speaches and buffered-SSE normalization, partial accepted-text
handling, metadata-only discovery limits, credential-reflection rejection,
privacy-safe errors, or no-automatic-retry policy. Introducing an SDK without
removing those responsibilities would add another abstraction and could cause
duplicate large-audio uploads. Reconsider that choice only if a measured spike
replaces most of the transport while preserving every compatibility, privacy,
timeout, cancellation, and retry invariant.

The shared transport sets connection-pool and TLS-handshake bounds but no
response-header deadline. Each operation owner derives a capability-specific
deadline from its captured settings profile immediately around the request:
each microphone checkpoint, each stored-file attempt, post-processing, and
speech generation therefore receive independent budgets. The connection
service keeps its fixed 15-second metadata-only probe deadline. Deadline and
user cancellation are separate bounded error kinds; no timeout causes an
automatic inference retry.

## Shared post-processing outcome policy

Dictation and stored-file transcription remain separate state machines. Each
owner decides whether to run cleanup, retains raw text before the attempt when
history permits, and invokes `ProcessWithCredential` using its captured profile.
`Processor` owns request validation, prompts, the cleanup deadline, and empty
response rejection.

After the attempt, `postprocess.Resolve` makes one operation-local decision:
use processed text on success, preserve raw text on failure, and let an already
cancelled owning context override even a successful late response. A cleanup
timeout permits raw fallback; user cancellation does not authorize delivery.
Owners recheck cancellation after finalizing a failed attempt before admitting
raw fallback, because finalization may block after the outcome was resolved.
The resolver owns neither credentials nor history, locks, events, or retries.

`history.Store.FinalizeProcessing` projects that outcome into processing status,
timing, character counts, response metadata, and retained raw/processed versions.
It returns updated run details even with history disabled or absent, or after an
entry was deleted. History budget fallback affects only retained text, never the
workflow's delivery decision. Dictation still owns generation checks, focus-safe
insertion and copy recovery; stored-file transcription still owns terminal file
state and explicit copy. Each owner records its own final run outcome and
publishes its own status and diagnostics.

## Renderer state ownership

Each WebView composes its own `Session` from feature owners under
`frontend/src/lib/stores`. `Session` owns construction, initial loading order,
aggregate busy presentation, and presentation teardown; it is not a second
command facade or a container for feature state.

- `SettingsEditor` owns the applied settings snapshot, independent editable
  settings/credential draft, connection probes, microphone choices, recovery,
  and serialized quick saves. These remain together to preserve one coherent
  editing transaction.
- `DictationState` owns live-dictation projection and commands.
- `FileTranscriptionState` owns stored-file projection, generation/revision
  reconciliation, delta-gap recovery, and explicit file commands.
- `SpeechState` owns playback projection and speech commands, including preview
  admission. The generated speech and dictation `CurrentStatus` methods remain
  in separate service namespaces.
- `HistoryState` owns history refresh/mutation ordering. Successful refresh
  acknowledges the completed file generation through an injected callback.
- `SessionMessages` owns shared presentation notices and their timers, not
  workflow state.

Components access these owners directly. Generated services and DTOs remain
wire authority; feature dependencies use narrow types derived from those
services rather than handwritten transport shapes. Feature owners do not import
`Session` or acquire subscriptions during construction.

Both main and Settings windows install `subscribeSessionEvents` before loading
snapshots. This shared composition routes events to the owners and reacts to
accepted terminal transitions by refreshing history. Window-specific level and
overlay reactions stay in the window callbacks. The disposer releases all
session subscriptions; window teardown clears presentation timers and credential
drafts without stopping Go-owned recording, transcription, or playback. Hiding
the reusable Settings window continues to discard its draft through the existing
settings lifecycle.

## Diagnostics boundary

`internal/app` creates one hierarchy from Wails' default structured logger and assigns bounded component attributes before injecting it into feature services, the post-processor, and native overlay. Runtime code records lifecycle metadata and fixed error categories rather than formatting underlying errors. It never logs transcript/audio content, credential or header material, model IDs, full paths, URL paths/query, or insertion-target identity. High-frequency audio, VAD, progress, delta, and renderer-event traffic remains off the logging path.

Wails stays at `Info` because the pinned bridge's debug tracing serializes binding arguments and results. Root `main.go` has the only direct standard-library log call: a content-free bootstrap category for failures outside the Wails logger lifetime. The complete versioned policy and field vocabulary live in [the logging contract](../../safety/logging/).

## Configuration boundaries

Durable settings contain ordinary STT, VAD, shortcut, window, appearance, history, post-processing, and optional speech-playback configuration. STT, stored-file STT, post-processing, and TTS have independent validated request budgets; STT, post-processing, and TTS also have independent endpoint, model, HTTP-policy, and credential identities even when the user points them at the same server. Stored credentials remain outside the JSON settings file. Payload and retained-memory ceilings are implementation safety invariants rather than user-tunable settings.

`internal/tts` is deliberately on-demand and provider-neutral. History/file renderer calls identify a backend-retained entry/version or completed stored-file result rather than resending transcript text. The first-class Text to speech workspace is the single deliberate exception: it accepts a bounded user-authored input (4,096 Unicode characters) and does not write that output-oriented content into transcript history. Synthesized bytes never become bridge results. The service captures one coherent TTS settings/credential profile, sends a bounded `/v1/audio/speech` WAV request, validates PCM before native playback, and emits only typed scalar status/progress. The ordinary connection service may discover speech model IDs with authenticated `GET /v1/models` metadata, but voice remains an explicit provider ID because the compatible API defines no voice-list operation. One in-memory playback session owns pause/resume/restart/stop/save/clear. Replay reads the retained PCM without another request; Save reconstructs a canonical PCM16 WAV and writes only to a native-dialog destination; Clear zeroes and releases the session. A new request replaces it, recording preempts and releases it before capture, native progress follows audible time rather than output-buffer submission, and shutdown cancels generation and closes native output deterministically.

The main, Settings, and About windows use the opaque product palette by default. Windows Mica is an explicit persisted opt-in applied when all three native windows are created, so changing it requires a process restart. The service reports the launch-time material separately from the editable preference; Svelte continues rendering the launch-time material until restart rather than making its surfaces translucent over solid native windows. Shell chrome uses the same material-aware layer roles, including the main header/status strip, Settings navigation/action bar, and About action bar.

`internal/app` owns three named Wails windows: the normal `main` shell plus hidden, reused `settings` and `about` renderers. It creates them from Wails' `ApplicationStarted` lifecycle event, after the framework has populated its screen manager, so the saved main-window placement is supplied directly through `WebviewWindowOptions`. Renderers reach the windows only through the narrow generated `internal/windowing` binding. `internal/windowstate` persists only the main window's normal bounds relative to its display work area and independently from product settings. On launch, the saved display is matched by Wails screen ID and stable device name, then its bounds are clamped to the current work area; a missing display falls back to a centered primary window. Immediately before Settings or About is revealed from a hidden state, its Wails logical bounds are centered over the main window and clamped to the main window's current display work area. No auxiliary placement is persisted. Each WebView has independent Svelte state, while the transactional Go settings service remains authoritative and broadcasts its renderer-safe committed snapshot to every window that needs it. Settings reloads from Go whenever it is revealed, preserves an active draft against external events, and routes native close requests through its existing discard confirmation before asking Go to hide it. About has no editable state and hides immediately from either its native close action or footer. Because Wails parent-blocking modal attachment is not supported on Windows, the main window's rack is inert while the modeless Settings window is visible.

The native status overlay is enabled by default but has an independent persisted opt-out plus curated layout, work-area anchor, phase visibility, motion, surface, visualizer, proportional-size, opacity, edge-distance, and glow settings. `internal/overlay` owns that feature lifecycle: the settings transaction supplies applied configuration, dictation supplies authoritative status, and the package translates both into a narrow `platform.OverlayOptions`/`platform.OverlayStatus` contract. Enabling creates one native surface and bounded level tap; disabling closes its HWND, message-loop thread, timers, fonts, and graphics resources instead of retaining a hidden renderer. Capsule/glass/bars/top-center remains the compatibility default.

The Win32 renderer queues all changes onto its locked message-loop thread, uses the foreground application's monitor work area captured at the start of a recording, and does not chase later focus changes. Windows Animation Effects and the saved Reduced policy control decorative frames, while the coordinator-owned silence deadline remains live. The overlay draws from the window's own palette rather than one of its own: a single ground, one accent hue, and the shared status colours, with each visible state kept distinguishable by glyph and stage rather than by colour alone. Windows contrast themes force a system palette, solid opaque surface, and no glow. Detailed may render only fixed Freehand labels and bounded operational values (normalized shortcut, elapsed time, checkpoint count); transcript text and provider/user metadata never enter the native contract.

Settings can request a presentation-only native preview through a narrow Wails binding. Draft presentation changes update the same renderer, real dictation preempts preview, Settings close stops it, and the applied saved configuration is restored. Preview can temporarily create a surface while the applied feature is disabled, but stopping it destroys that surface. Overlay creation remains a degraded optional capability: native failure is logged without failing dictation or rolling back the saved preference.

The home-screen rack is a narrow immediate-save surface for STT and post-processing endpoints/models, explicit processing-profile selection, trained S1-mini output controls, and the capture/delivery switches. It is composed of `RackModule` panels grouped as Speech to text, Cleanup, Capture and Delivery, so the main window and the Settings navigation name the same concerns identically. Each update starts from the backend-confirmed settings snapshot and changes only the named quick fields before calling the same transactional settings owner with no credential mutation. It must never save the full editable Settings-window draft, and it is disabled while that modeless window is visible. Endpoint credentials, authentication mode, insecure-HTTP policy, custom instructions, and other advanced settings remain in the full Settings window.

The main renderer treats transcript-list disclosure as a presentation-only WebView preference. It is written to versioned local storage and falls back safely to open when storage is missing, malformed, or unavailable. The preference never enters the Go settings transaction and does not affect configuration, history retention, or runtime authority. The rack does not collapse: its modules are compact enough to stay open, and the rack column scrolls on its own at the minimum window height.

Every input mode and every dictation state shares one `TransportShell`: a fixed 116px control cell, an elastic stage, and a 236px readout cell spanning the window under the header. Because that geometry never changes, starting a recording, switching input modes, or failing a request never moves anything else on screen. `TransportBar`, `AudioFileTranscription`, and `TextToSpeech` supply the three cells; the shell owns the progress rail, which is indeterminate for endpoint work that reports no progress and determinate only for the file-upload leg, whose length is known.

The post-processing package owns a small renderer-visible profile catalog so names, descriptions, editability, and fixed protocol instructions stay aligned with request construction. Model IDs are never used to infer behavior. The custom profile persists a bounded user system instruction in the ordinary JSON settings, while its API key remains in Credential Manager. The S1-mini profile keeps its exact system instruction in code and persists only its trained styling, structure, and context selections. Switching profiles preserves inactive profile values rather than destructively rewriting them.

## Release identity

`build/config.yml` is the only human-edited product identity and semantic
version source. The main package embeds and validates it before constructing
the application, then uses it for the Wails application name and encrypted
single-instance identity. `internal/buildinfo` exposes only immutable,
renderer-safe build metadata to About: product and semantic versions, the
derived four-part Windows version, development/production mode, and toolchain
versions already present in the executable.

`build/scripts/releaseinfo` derives Windows PE, assembly-manifest, and NSIS
version fields from the same source after Wails regenerates build assets. The
Windows resource task runs its read-only check before compilation, preventing a
package whose embedded About version disagrees with its executable or installer
metadata.

An active operation observes one coherent request profile. The transactional settings owner captures endpoint, model, headers, authentication mode, post-processing configuration, and both credentials under its save lock before microphone capture or stored-file upload begins. Renderer-safe settings reads use that same lock, so no window can combine an old JSON configuration with credential or native state already changed by an in-progress save. Every failed native or credential stage attempts its own restoration plus all earlier restorations in reverse order; rollback failures remain inspectable by Go while their renderer-visible messages omit provider and credential-store details. The profile remains private to Go and fixed for the operation; settings edits save normally but affect only later operations. Segmented dictation therefore does not read a credential at its first checkpoint, and stored-file post-processing does not reread one after upload.

## Audio contract

The capture adapter may receive the Windows mix format, commonly 48 kHz float/stereo. Before upload, the client produces a bounded WAV payload with explicit format metadata. The initial target is mono signed 16-bit PCM at 16 kHz.

Each recording also owns a bounded, non-blocking interruption signal from the native stop callback. The callback never tears down audio or enters UI/state-machine code. Dictation fences that signal by recording generation, cancels the recording-only timer, discards partial PCM, and releases the device before publishing failure. System-default capture keeps miniaudio's shared-mode WASAPI rerouting; an explicit device is never silently replaced.

No audio is persisted after request completion, error, or cancellation.

Stored-audio transcription is a separate service-owned cancellable job so it can continue while the settings window is hidden. It consumes only the Go-owned native selection, revalidates its identity and metadata, and streams multipart bytes from the opened file. Go accumulates progressive transcript text once and emits typed generation/revision deltas across the Wails bridge; it does not republish the complete growing string for every chunk. Main and Settings renderers reject stale or duplicate revisions, request the authoritative snapshot after a gap, and reconcile once with the terminal full result. Upload progress remains throttled and full snapshots are reserved for real phase boundaries or explicit recovery. If the fixed 8 MiB stored-file transcript ceiling is reached, the service stops accepting deltas, publishes an explicit partial-result state, and preserves already accepted text for manual copying rather than silently truncating it. The job is mutually exclusive with microphone dictation. Completed stored-file text is never auto-inserted because no safe target was captured when recording began; it can be explicitly copied and, when enabled, retained in the same bounded history.

Both interactive WebViews explicitly deny microphone, camera, geolocation, notification, and clipboard-read permission requests. Native Go owns microphone capture and clipboard writes, file drop remains disabled, and Wails simple renderer event emission remains disabled.

When a VAD-dependent microphone feature is enabled, the native callback writes into a fixed pool of 20 ms frames and never runs VAD or network work itself. One pinned `libfvad`/WebRTC detector feeds the stabilized live indicator, optional leading/trailing trim boundaries, the speech-armed automatic-stop policy, and optional checkpoint boundaries. Trimming retains bounded configurable speech padding. Automatic stop cannot arm until a configured amount of confirmed speech has accumulated, and resumed speech cancels its silence countdown. Recording control mode is captured explicitly at start: toggle recordings may arm automatic stop, while hold-to-talk recordings continue to use VAD feedback, trimming, and checkpoints but can end only on shortcut release, cancellation, interruption, or the hard duration limit.

When silence-aware splitting is enabled, segments target the configured duration and wait for their own sustained-silence threshold, with a separate 240-second capture ceiling when no pause arrives. Each completed segment is sent sequentially to the ordinary transcription endpoint with a fresh configured microphone-request budget while capture continues; only ordered final text is delivered. The total recording remains bounded, backpressure fails closed, and every PCM/WAV buffer is zeroed after use.

## Insertion contract

At recording start, capture a target identity containing at least the foreground HWND and process identity. At insertion time:

- direct-input mode plus the same valid, focused target: insert ordinary transcripts in one bounded Unicode dispatch and use larger adaptive dispatches for long text, revalidating the complete target before each dispatch without a fixed pacing delay;
- manual-copy mode: retain the transcript and expose Copy without attempting insertion or touching the clipboard;
- target changed or invalid: retain the transcript and require explicit Copy;
- insertion failure: preserve the transcript for explicit copy without retrying simulated input blindly.

The native input adapter keeps UTF-16 surrogate pairs within one dispatch and checks cancellation before each dispatch. A partial `SendInput` result is an ambiguous partial insertion: it is logged only as bounded delivery metadata and is never retried. Diagnostics include UTF-16 unit count, dispatch count, strategy, duration, and a fixed failure stage; they never include transcript text or target identity.

Clipboard-paste mode is represented as a deferred policy boundary but cannot be selected or executed. It must remain fail-closed until complete multi-format clipboard capture, bounded paste synchronization, and conditional restoration that never overwrites newer user clipboard content are implemented.

## Optional transcript history

History is an opt-in recovery surface for finalized transcripts, not durable storage or a notes workspace. `history.Store` owns an in-memory oldest-first ring capped at 20 entries and 2 MiB across transcript text and bounded run details. Reaching either limit evicts the oldest entries first. A single entry is never allowed to grow past the total byte budget.

Each entry can retain finalized raw and processed text, processing status, completion time, Unicode character count, delivery outcome, selected delivery mode, and bounded request metadata such as source, endpoint host, route, model, response mode, audio duration, segment timing, and file base name/size. When an STT or post-processing response supplies additional metadata, history may also retain a fixed, bounded subset: request/response identity, effective model, provider, finish reason, service tier, system fingerprint, detected language, server-reported audio duration, standard token/duration usage, provider-reported cost values, and llama.cpp-style timing metrics. These fields are optional rather than synthesized. Cost has no assumed currency, and the client never estimates tokens, duration, or price from transcript text. For checkpointed dictation, additive values are aggregated and explicit report counts show whether usage, cost, and performance covered every request; per-request IDs are omitted from the aggregate.

History never contains a URL path supplied by the user, target-window identity, credentials, headers, audio, provisional text, or an unbounded provider response object. The buffer enforces its 20-entry/2-MiB invariant after insertion and every mutation. If a processed copy alone makes an entry too large, it is discarded with visible `history_budget` raw-fallback metadata; if the bounded raw entry still cannot fit, the entry is removed. Removing an entry releases that transcript immediately; turning history off, choosing **Clear history**, or shutting down releases every retained entry. Nothing is written to disk or emitted through status/overlay events. Historical text reaches the clipboard only through an explicit copy action.

## Lifecycle

- Wails single-instance ownership uses encrypted second-instance messages and the stable product identifier parsed from `build/config.yml`, shared deliberately with packaging and updates.
- First process owns hotkeys, tray, capture, the main window, and reusable Settings and About windows.
- Second process asks the first to reveal the main window and exits.
- Closing any native window hides that window; closing Settings first resolves or discards its renderer-owned draft.
- Appearance changes that affect native window creation are saved transactionally but applied only after tray Quit and relaunch.
- Tray Quit cancels active work, unregisters hooks/hotkeys, stops capture, and exits.
- Tray Quit clears the optional in-memory transcript ring before process exit.
- Automatic startup currently uses an app-owned HKCU entry and never requires elevation; replacement with Wails Autostart remains separate backlog work.
- Automatic release checks are opt-out, quiet metadata reads scheduled by `internal/updates`; Wails owns GitHub release comparison, checksum verification, its review window, download, executable staging, and restart. The service stops polling and rejects new checks during shutdown.
- Services that own asynchronous work retain a child of Wails' application context themselves. Live `StopRecording` owns only the serialized native capture-stop transition; it then submits exactly one generation-scoped completion to the dictation service's single managed worker, which owns transcription, post-processing, history finalization, and insertion. Renderer, toggle, hold-release, duration-limit, and automatic-silence callers therefore share status events as their outcome contract instead of blocking a bridge or native callback on inference. Shutdown atomically stops completion admission, cancels stored-file and dictation work, waits for the managed completion worker within five seconds, suppresses late publications, closes shortcut capture before the dictation/audio owner, and returns within the shared deadline even if an operating-system file read does not respond. Native capture has a closed-state fence before and after device preparation so a late warmup cannot recreate resources.

## Shelved conversation research

Conversation mode is not part of the active product direction. If future
evidence revives it, it must reuse the same coherent-profile and
feature-ownership principles but requires a separate turn state machine,
streamed chat, ordered sentence segmentation, sequential TTS playback,
cancellation, and a single selected LLM. It must not create parallel requests
to different Ollama models.

## Optional post-STT normalization

The client provides an optional transcript-processing capability:

```text
STT -> raw transcript -> selected processing profile -> clean transcript -> insertion
```

Raw STT remains first-class and selectable. The processor is orchestrated by the client through a separately configured OpenAI-compatible `/chat/completions` endpoint; it is never hidden inside Speaches and is never bundled into the Windows executable. The default custom-instruction profile works with an ordinary compatible chat model. S1-mini by Superwhisper is a separate purpose-built profile whose styling, structure, context, and fixed request contract apply only when explicitly selected. All failures fall back to raw text. See [ADR 0001](../../decisions/0001-s1-mini-post-processing/).

## Shelved realtime transcription research

Realtime transcription is not part of the active product direction. ADR 0002
preserves the explored boundary in case live captions or provisional editing
later provide value that pause-aware checkpoints do not. It remains a separate
capability with its own transport, URL, credential reference, audio format,
model, and event codec; it must never be folded implicitly into completed STT.

The application normalizes provider messages into correlated speech, provisional-delta, and finalized-transcript events. Current Speaches v0.8.2 provides VAD plus finalized transcription events but not the newer OpenAI input-transcription delta event, so the UI must work well both with completed utterance chunks and true incremental text.

Audio capture accepts a requested format specification: file STT currently uses 16 kHz PCM16 WAV, while Speaches realtime uses 24 kHz mono PCM16 chunks. Only finalized raw text may proceed to optional S1-mini and insertion. See [ADR 0002](../../decisions/0002-realtime-transcription/).
