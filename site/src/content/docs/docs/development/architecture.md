---
title: Architecture
description: Runtime ownership, platform boundaries, and application data flow.
---

## Durable settings storage

[ADR 0006](../../decisions/0006-sqlite-storage-contract/) governs the implemented
SQLite store: modernc, embedded goose migrations, and sqlc-generated queries.
`internal/storage` owns the database lifecycle and adapters; `internal/settings`
retains coherent saves and immutable request profiles. See the
[storage maintenance guide](../storage/) for schema changes and recovery.
Transcript history remains optional and memory-only.

Named connections represent reusable servers, with explicit supported uses and
independent active selections for transcription, cleanup, and playback. One ID
can be selected by multiple features; their models and runtime options remain
independent while URL, profile, authentication, and credential reference are shared. A dedicated Connections page owns creation,
editing, duplication, deletion, endpoint/authentication/profile fields, and saved
metadata tests. Creation is inactive; feature pages own active selection, model,
language, presets, voice, and other runtime options. Fresh catalogs are empty.
Selection restores remembered model options for the connection and use; an
unconfigured connection starts with defaults and disables optional features.
The domain contract lives in
`internal/savedconnection`; SQLite adapters remain in storage. The existing
settings owner commits catalog mutations, active configuration, and credential
references together. UI connection actions reuse that transaction and reject
stale selected IDs. Captured requests retain their settings and key strings
across connection switches and deletion. See [saved connections](../../guides/saved-connections/).

## Server compatibility ownership

The [backend maintenance guide](../backend-compatibility/) describes the shared
app/site catalog export, public capability matrix, and evidence requirements.

`internal/compatibility` owns the stable profile IDs, operation-scoped catalog,
implemented capability flags, and route contracts. The Settings DTO exposes a
fresh catalog for presentation; the renderer uses generated types and cannot
change the authoritative registry. Planned entries carry no capabilities and
cannot be selected through settings or connection-test bindings.

`internal/config` persists one compatibility selection for each existing STT,
post-processing, and TTS connection. `internal/settings` captures these with
the endpoint/model and credentials in the same transaction. Inference uses an
immutable client view per captured selection (TTS carries it on its focused
request DTO); the shared HTTP client is never mutated. Request construction
resolves and validates the operation before network or upload work, then uses
the resolved route and capabilities. The stream decoder consumes explicit
typed-event and legacy-segment permissions. File streaming observations include
the effective profile in their cache key.

Profiles identify implemented server contracts. The S1-mini preset continues
to own prompt construction independently. Future model-specific options must
add qualified capabilities and bounded request/response handling here; catalog
availability alone cannot prove model support. Named saved connections retain
these provider contracts with their endpoint and opaque credential reference.

## Authority map

| Concern | Owner |
|---|---|
| Live dictation state machine | `internal/dictation` |
| Cross-feature start admission and recording preemption | `internal/activity` |
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
| Durable non-secret configuration | `%LOCALAPPDATA%\Freehand\settings.db` (SQLite) |
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
internal/activity         cross-feature admission, without duplicated feature state
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

Custom health targets retain the base-relative path-joining contract, including
the leading slash in saved values. Settings help and validation describe that
contract; no configuration migration or fallback model probe is performed.

Model probes require a valid JSON model-list shape; reachability remains distinct from inventory validation and inference compatibility. Health probes accept opaque successful bodies. Invalid model responses cannot satisfy first-run readiness.

After loading the applied profile, the renderer runs one bounded metadata-only STT probe and repeats it only when that profile's connection identity changes. First-run readiness is an exclusive app-shell content state backed by a persisted completion flag; the Go recording command rejects both renderer and global-shortcut starts until it is complete. A later failed probe may take over the content area once, but the established user can continue without correction while the persistent status strip retains the warning. Dismissal is scoped to the exact failed endpoint/model/authentication/microphone condition rather than mutating durable settings.

Ordinary Go collaborators remain ordinary types: `history.Store`, the inference client, post-processor, capture adapter, and insertion policy are injected by `internal/app` and are not registered with Wails. The settings/profile transaction remains one owner even though consumers receive narrow snapshot functions.

The storage package opens one connection with a per-process file lock, validates
Freehand database identity and goose history, and upgrades forward before
settings-dependent services start. Four typed singleton tables hold preferences,
transcription, cleanup, and playback; related tables hold bounded request headers,
credential references, deferred credential deletion, and initialization state.
The read-only legacy JSON importer runs only when no database exists. It rejects
unknown fields or invalid input and leaves the original file untouched.

A load failure or uncertain commit creates an explicit recovery state: ordinary
saves and new request profiles are blocked. Retry validates and reloads committed
settings; explicit Reset archives the database and sidecars before replacing it.
An upgrade requires a successful SQLite backup first. No API key or transcript is
stored in these tables. Credentials are staged under new native accounts; settings
and account references commit together, then obsolete accounts are reclaimed.
Native startup registration and shortcut configuration reconcile from committed
settings on restart. No SQL transaction spans an inference call.

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

## Cross-feature activity admission

`internal/app` constructs one ordinary `activity.Coordinator` and supplies live
activity predicates from dictation and file transcription plus the speech
owner's Stop operation. The coordinator is not a Wails service, scheduler, or
aggregate state machine. Construction installs sources without calling them;
all participating services are composed before admission is used.

- Recording excludes active file transcription and synchronously stops and
  releases speech before microphone capture. A failed playback stop rejects
  recording instead of allowing capture over potentially audible output.
- File transcription excludes active dictation. It deliberately does not stop
  speech that was already running; the policy is not symmetric three-way
  exclusion.
- New speech requests exclude active dictation and file transcription.
- Shortcut capture uses the same activity predicates for its initial check.
  Input and native shortcut owners still own capture serialization and
  suspend/restore. This check is not a reservation for the whole user prompt.

Admission spans the owner's start checks, any playback preemption, and the
publication of its active state. Every start defers release, including failed
profile, file-selection, and device preparation paths. There is no activity
reservation retained for a running operation: subsequent checks read the
feature owner's current state, including cancellation and completion.

The lock order is admission **before** feature control locks. Speech start
must acquire admission before its player-control lock, because recording may
hold admission while calling speech Stop. Sources and synchronous start-status
callbacks must not reenter admission. Stop, cancel, and shutdown do not acquire
admission, so they can terminate work while another start is waiting.

Closing admission is terminal and wakes waiting callers without joining a
start or clearing feature state. Participating service shutdown closes the
shared coordinator before resource teardown, regardless of shutdown order.
Owners still fence late preparation, cancel their contexts, and join their
workers; file worker registration occurs inside its closed-state admission
fence. Recording rechecks closure after potentially blocking speech preemption.

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

Settings clusters use the shared `SettingsCard` component with the same
`layer-fill` background and single `hairline` border as saved-connection cards.
Internal row dividers remain; card outlines do not stack with elevation shadows.
The shared switch uses a pill track and an inset circular thumb, retaining
Bits UI state, keyboard semantics, and visible focus indicators.

Connection editor navigation is owned by `SettingsScreen`: it remembers the originating
feature and returns there after Back or save. The editor compares non-credential fields
with its opening snapshot; credential presence/removal is checked separately without
copying a password into that snapshot. Opening a form alone is not dirty, but still
reserves the draft against external-window updates. Navigation confirms before
discarding changed connection fields and clears the transient key on exit. Settings
groups separate Capture, Features, and Application; visual and keyboard section order
match.

## Model behavior contracts

`internal/modelprofile` owns explicit role-specific model IDs, requirements, and
backend/model capability intersection. Generic preserves the existing baseline;
S1-mini is qualified for cleanup only. `SettingsDTO.ModelProfiles` exposes the
resolved catalog for each active backend. Feature controls read that metadata,
and Go validates selections even when a feature is disabled. Unknown or
wrong-role IDs fail closed before inference; model inventory names never select
behavior. Backend routes, event formats, and server-loaded-model semantics remain
in `internal/compatibility`.

The post-process owner still builds trained prompts. It takes S1-mini's language
and reasoning requirements from the shared model contract, preserves raw fallback,
and enforces reasoning off through qualified adapters. Generic servers retain
the explicit requirement for server-side reasoning configuration. Microphone,
checkpoint, file, and speech request paths carry model-profile selection in their
existing immutable settings snapshot. Model-specific fields never become
connection credentials or connection-owned settings.

SQLite migration 00006 adds transcription and speech `model_profile` columns with
Generic defaults. Cleanup's existing `preset` column and JSON key already hold
its model-profile ID; retaining those names preserves legacy import and saved
choices without a second source of truth. The existing cleanup descriptor service
supplies prompt/control metadata; shared catalog metadata supplies behavior names,
capabilities, and requirements.

`internal/modelsettings` defines the value-only, non-secret subset remembered for
(connection ID, purpose, exact model ID). It excludes transport, credentials,
feature enablement, capture policy, and timeouts. `internal/storage` owns typed
sqlc queries for `remembered_models` and loads them with connection state. The
settings owner's save lock coordinates changes; active settings, remembered
options, connection selection, and credential references commit in one SQLite
transaction. Failed commits do not publish a new in-memory catalog. The renderer
receives only models for the selected connections, plus Go-owned defaults.

Model selection restores a copied option draft; changing an ID never derives a
profile from its spelling. The editor retains a draft per model while switching. Save submits a bounded
batch of option edits with the active selection; the storage owner validates
every edit against its selected connection and commits the whole batch atomically. Quick controls restore and save in their existing queue.
Forget requests carry connection, purpose, and model identity and run through the
same settings transaction. Removing the active model clears selection and disables
features that require configuration. Captured jobs remain immutable.

Connection URL/backend changes clear model preferences; renames and credential
rotation preserve them. Copies start independently, removed uses discard their
preferences, and connection deletion cascades. A bounded 32 models per use keeps
catalogs finite; exceeding the limit fails atomically without eviction. A blank
ID is permitted only for whisper.cpp's explicit server-loaded slot.

## Diagnostics boundary

`internal/app` creates one hierarchy from Wails' default structured logger and assigns bounded component attributes before injecting it into feature services, the post-processor, and native overlay. Runtime code records lifecycle metadata and fixed error categories rather than formatting underlying errors. It never logs transcript/audio content, credential or header material, model IDs, full paths, URL paths/query, or insertion-target identity. High-frequency audio, VAD, progress, delta, and renderer-event traffic remains off the logging path.

Wails stays at `Info` because the pinned bridge's debug tracing serializes binding arguments and results. Root `main.go` has the only direct standard-library log call: a content-free bootstrap category for failures outside the Wails logger lifetime. The complete versioned policy and field vocabulary live in [the logging contract](../../safety/logging/).

## Configuration boundaries

Durable settings contain ordinary STT, VAD, shortcut, window, appearance, history, post-processing, and optional speech-playback configuration. STT, stored-file STT, post-processing, and TTS have independent validated request budgets; STT, post-processing, and TTS retain independent runtime models and selections. Selecting the same reusable connection explicitly shares its endpoint, HTTP policy, backend profile, and credential reference; selecting separate connections keeps those identities independent. Stored credentials remain in Windows Credential Manager; SQLite contains only their opaque references. Payload and retained-memory ceilings are implementation safety invariants rather than user-tunable settings.

`internal/tts` is deliberately on-demand and provider-neutral. History/file renderer calls identify a backend-retained entry/version or completed stored-file result rather than resending transcript text. The first-class Text to speech workspace is the single deliberate exception: it accepts a bounded user-authored input (4,096 Unicode characters) and does not write that output-oriented content into transcript history. Synthesized bytes never become bridge results. The service captures one coherent TTS settings/credential profile, sends a bounded `/v1/audio/speech` WAV request, validates PCM before native playback, and emits only typed scalar status/progress. The ordinary connection service may discover speech model IDs with authenticated `GET /v1/models` metadata, but voice remains an explicit provider ID because the compatible API defines no voice-list operation. One in-memory playback session owns pause/resume/restart/stop/save/clear. Replay reads the retained PCM without another request; Save reconstructs a canonical PCM16 WAV and writes only to a native-dialog destination; Clear zeroes and releases the session. A new request replaces it, recording preempts and releases it before capture, native progress follows audible time rather than output-buffer submission, and shutdown cancels generation and closes native output deterministically.

The main, Settings, and About windows use the opaque product palette by default.
Dark surfaces adapt the website’s navy to a tighter desktop ladder (`#111722`,
`#171f2c`, `#1c2635`) with its cobalt accent (`#4d8dff`). Inputs use `#121925`
for a gentle recess; subdued strokes and supporting text keep dense forms calm. The semantic CSS roles in `frontend/src/app.css`
cover cards, inputs, popovers, dialogs, and navigation. Native dark captions and
startup backgrounds in `internal/app/window.go` match the ground; overlay colour
constants in `internal/platform/overlay.go` match the panel and accent. Status
colours keep their separate meanings. Dark Mica applies one translucent navy
tint at `#app` plus translucent panels, while native captions remain under DWM
control. Light-mode tokens remain independent.
 Windows Mica is an explicit persisted opt-in applied when all three native windows are created, so changing it requires a process restart. The service reports the launch-time material separately from the editable preference; Svelte continues rendering the launch-time material until restart rather than making its surfaces translucent over solid native windows. Shell chrome uses the same material-aware layer roles, including the main header/status strip, Settings navigation/action bar, and About action bar.

`internal/app` owns three named Wails windows: the normal `main` shell plus hidden, reused `settings` and `about` renderers. It creates them from Wails' `ApplicationStarted` lifecycle event, after the framework has populated its screen manager, so the saved main-window placement is supplied directly through `WebviewWindowOptions`. Renderers reach the windows only through the narrow generated `internal/windowing` binding. `internal/windowstate` persists only the main window's normal bounds relative to its display work area and independently from product settings. On launch, the saved display is matched by Wails screen ID and stable device name, then its bounds are clamped to the current work area; a missing display falls back to a centered primary window. Immediately before Settings or About is revealed from a hidden state, its Wails logical bounds are centered over the main window and clamped to the main window's current display work area. No auxiliary placement is persisted. Each WebView has independent Svelte state, while the transactional Go settings service remains authoritative and broadcasts its renderer-safe committed snapshot to every window that needs it. Settings reloads from Go whenever it is revealed, preserves an active draft against external events, and routes native close requests through its existing discard confirmation before asking Go to hide it. About has no editable state and hides immediately from either its native close action or footer. Because Wails parent-blocking modal attachment is not supported on Windows, the main window's rack is inert while the modeless Settings window is visible.

The native status overlay is enabled by default but has an independent persisted opt-out plus curated layout, work-area anchor, phase visibility, motion, surface, visualizer, proportional-size, opacity, edge-distance, and glow settings. `internal/overlay` owns that feature lifecycle: the settings transaction supplies applied configuration, dictation supplies authoritative status, and the package translates both into a narrow `platform.OverlayOptions`/`platform.OverlayStatus` contract. Enabling creates one native surface and bounded level tap; disabling closes its HWND, message-loop thread, timers, fonts, and graphics resources instead of retaining a hidden renderer. Capsule/glass/bars/top-center remains the compatibility default.

The Win32 renderer queues all changes onto its locked message-loop thread, uses the foreground application's monitor work area captured at the start of a recording, and does not chase later focus changes. Windows Animation Effects and the saved Reduced policy control decorative frames, while the coordinator-owned silence deadline remains live. The overlay draws from the window's own palette rather than one of its own: a single ground, one accent hue, and the shared status colours, with each visible state kept distinguishable by glyph and stage rather than by colour alone. Windows contrast themes force a system palette, solid opaque surface, and no glow. Detailed may render only fixed Freehand labels and bounded operational values (normalized shortcut, elapsed time, checkpoint count); transcript text and provider/user metadata never enter the native contract.

Settings can request a presentation-only native preview through a narrow Wails binding. Draft presentation changes update the same renderer, real dictation preempts preview, Settings close stops it, and the applied saved configuration is restored. Preview can temporarily create a surface while the applied feature is disabled, but stopping it destroys that surface. Overlay creation remains a degraded optional capability: native failure is logged without failing dictation or rolling back the saved preference.

The home-screen rack is a narrow immediate-save surface for STT and post-processing models, explicit processing-profile selection, trained S1-mini output controls, and the capture/delivery switches. It is composed of `RackModule` panels grouped as Speech to text, Cleanup, Capture and Delivery, so the main window and the Settings navigation name the same concerns identically. Each update starts from the backend-confirmed settings snapshot and restores remembered options when a model changes, then applies the named quick fields before calling the same transactional settings owner with no credential mutation. It must never save the full editable Settings-window draft, and it is disabled while that modeless window is visible. The rack uses the same active connection selector as feature settings, with separate links for editing settings. Connections exclusively owns endpoint credentials, authentication, HTTP policy, and provider profiles; custom instructions and runtime options remain on feature pages.

The main renderer treats transcript-list disclosure as a presentation-only WebView preference. It is written to versioned local storage and falls back safely to open when storage is missing, malformed, or unavailable. The preference never enters the Go settings transaction and does not affect configuration, history retention, or runtime authority. The rack does not collapse: its modules are compact enough to stay open, and the rack column scrolls on its own at the minimum window height.

Every input mode and every dictation state shares one `TransportShell`: a fixed 116px control cell, an elastic stage, and a 236px readout cell spanning the window under the header. Because that geometry never changes, starting a recording, switching input modes, or failing a request never moves anything else on screen. `TransportBar`, `AudioFileTranscription`, and `TextToSpeech` supply the three cells; the shell owns the progress rail, which is indeterminate for endpoint work that reports no progress and determinate only for the file-upload leg, whose length is known.

The post-processing package owns a small renderer-visible profile catalog so names, descriptions, editability, and fixed protocol instructions stay aligned with request construction. Model IDs are never used to infer behavior. The custom profile persists a bounded user system instruction in the ordinary settings database, while its API key remains in Credential Manager. The S1-mini profile keeps its exact system instruction in code and persists only its trained styling, structure, and context selections. Switching profiles preserves inactive profile values rather than destructively rewriting them.

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

An active operation observes one coherent request profile. The transactional settings owner captures endpoint, model, headers, authentication mode, post-processing configuration, and both credentials under its save lock before microphone capture or stored-file upload begins. Renderer-safe settings reads use that same lock, so no window can combine an old saved configuration with credential or native state already changed by an in-progress save. Every failed native or credential stage attempts its own restoration plus all earlier restorations in reverse order; rollback failures remain inspectable by Go while their renderer-visible messages omit provider and credential-store details. The profile remains private to Go and fixed for the operation; settings edits save normally but affect only later operations. Segmented dictation therefore does not read a credential at its first checkpoint, and stored-file post-processing does not reread one after upload.

`postprocess.Processor` accepts the captured configuration and credential explicitly through `ProcessWithCredential`; it has no credential-store dependency or alternate store-reading entry point. Credential acquisition remains with the transactional settings/profile owner.

## Audio contract

The capture adapter may receive the Windows mix format, commonly 48 kHz float/stereo. Before upload, the client produces a bounded WAV payload with explicit format metadata. The initial target is mono signed 16-bit PCM at 16 kHz.

Each recording also owns a bounded, non-blocking interruption signal from the native stop callback. The callback never tears down audio or enters UI/state-machine code. Dictation fences that signal by recording generation, cancels the recording-only timer, discards partial PCM, and releases the device before publishing failure. System-default capture keeps miniaudio's shared-mode WASAPI rerouting; an explicit device is never silently replaced.

No audio is persisted after request completion, error, or cancellation.

Microphone requests (including local checkpoints) explicitly request completed
JSON with multipart `response_format=json` and `Accept: application/json`.
The generic chat decoder classifies a reported `finish_reason: "length"` as
`incomplete_response` before optional metadata redaction. It returns bounded
safe diagnostics and no partial cleaned text. The existing processing outcome
policy selects raw text for both microphone and file workflows, preserves
cancellation and focus checks, and shows an output-limit-specific notice.
There is no provider-specific branch or automatic cleanup retry.

The file service submits at most one transcription request per explicit start.
An unsupported stream records endpoint/model/compatibility-profile capability evidence and stops;
a new completed-mode attempt requires a user action. A completed JSON response
to a streaming request is consumed in place. The inference parser requires a
final text event for typed SSE while retaining EOF completion for legacy
Speaches segments. Incomplete typed streams and read/server failures return
accepted partial text alongside an error. The file service marks it failed,
skips cleanup, and permits explicit copy and bounded opt-in history retention.

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

### Transcription option snapshots

`internal/compatibility.TranscriptionOptions` defines value-only recognition
controls and shared bounds/capability validation. `config.Settings` persists this
nested value; Settings validates it transactionally with connection changes. The
frontend editor copies it independently for drafts and applied settings.

Dictation captures options at recording start, including every checkpoint;
stored-file jobs capture them with their request profile. The immutable inference
client copy receives that value, validates it before I/O, and writes the same
fields for completed and streaming requests. File multipart sizing uses the same
writer to preserve Content-Length. Optional values are omitted by default;
explicit zero temperature uses a separate override flag. These hints are not
added to diagnostic logs or history details. Cleanup presets remain independent.

### Cleanup generation requirements

`PostProcessingSettings.GenerationOptions` stores a value-only
`compatibility.CleanupOptions`. Config validation checks bounds and profile
support even while processing is disabled. The frontend editor deep-copies the
nested value to keep drafts independent. Jobs capture it with the existing
connection/credential snapshot, before either transcription or later cleanup.

The post-processing preset layer derives S1-mini's required reasoning-off
behavior whenever the selected contract advertises qualified enforcement. The
inference adapter maps the effective options to `max_tokens` and
`reasoning_effort: "none"`, without mutating persisted custom-model preferences.
Generic still requires external reasoning configuration for S1-mini. No model-ID
sniffing, arbitrary JSON extensions, automatic retries, or chunking are added.


## Native transcription and vLLM adapters

The compatibility catalog owns native server-loaded model semantics and the
vLLM stream capability. Config validation permits a missing STT model only for
a qualified server-loaded model profile. Settings and readiness consume the
same exported catalog. Connection owns whisper.cpp's metadata-only default
health probe; explicit custom paths retain base-relative behavior.

Inference owns native multipart model omission and the vLLM decoder. File
transcription applies the profile's completed-only restriction before upload
and reports that restriction separately from observed stream incompatibility.
vLLM per-audio-chunk stop markers cannot complete the whole file; the decoder
requires `[DONE]` after a successful final chunk and fails closed on provider
errors or incomplete streams. Accepted deltas retain manual recovery semantics.
Postprocess derives mandatory S1-mini reasoning-off for either qualified
cleanup adapter without changing the optional custom preference.

## Language selection ownership

`internal/speechlanguage` supplies a fresh language-name/code catalog to the
renderer-safe settings snapshot and validates bounded custom values. The catalog
is compiled reference data, not a list of verified model capabilities or a
persisted settings collection. `compatibility.Contract.TranscriptionLanguage`
owns provider mapping, consumed before both microphone and file multipart bodies
are built. The existing `config.Settings.Language` string preserves selections;
no new database or reusable model/connection records are introduced.

The existing S1-mini profile descriptor declares English. Each workflow owner
uses `postprocess.ValidateLanguage` after transcription and before cleanup, with
the captured language choice and sanitized detected-language metadata (aggregated
for dictation segments). Rejection makes no cleanup request and uses existing
raw fallback, cancellation precedence, history projection, and focus-safe
delivery. Unknown language assumes English by explicit product policy. Provider
capability, model language support, and a selected input language remain distinct.

## Connection assessments

`internal/connection` adds bounded diagnostic checks to its existing metadata
results. Endpoint and credential failures remain distinct from advertised model
presence and local model-option validation. Optional value-only model settings
are validated through `internal/modelsettings` and `internal/modelprofile`;
invalid options do not prevent metadata discovery. Saved-connection tests assess
only transport/access and never claim readiness for all enabled uses.

The renderer captures a non-secret input signature when each feature check starts.
A later draft with different transport, selected model, or model options marks
that assessment stale while retaining useful model-list choices. Signatures are
not persisted, logged, or sent to the server; unrelated capture preferences do
not invalidate them. A response completing after a draft edit retains its original
signature. Existing connection revisions still discard superseded connection
results. Request options and signatures contain no credential values.

Checks make one bounded GET request to the configured metadata route. HTTP success
is scoped to that route; advertised IDs do not imply feature support. There are
no inference probes, automatic inventory iterations, capability guesses, or new
readiness gates for unlisted aliases. Model and option checks do not replace
normal runtime admission and captured request settings.


### Voice discovery ownership

`internal/compatibility` advertises voice discovery only for qualified speech
backends. `internal/connection.ListSpeechVoices` captures saved connection
details and credentials through the existing settings owner. `internal/inference`
performs bounded metadata GETs and normalizes provider shapes into voice IDs,
display names, language labels, and model/server scope. Generic remains manual.
The renderer never supplies a key or endpoint to this operation, and no model
lifecycle or inference routes are used. Logs contain only bounded outcomes and
counts, not URLs, model IDs, voice IDs, or credentials.

The settings editor rejects late results from an obsolete connection revision
or model selection. The voice picker preserves custom IDs and never treats
inventory membership as request admission. Lists are ephemeral; selected voices
use the existing modelsettings/sqlc save transaction, requiring no migration.
Kokoro's `stream: false` is a backend wire adaptation, not a model preference.
