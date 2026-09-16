---
title: Architecture
description: Runtime ownership, platform boundaries, and application data flow.
---

## Product boundary

Freehand is a lightweight native desktop speech client for user-chosen,
self-hosted and OpenAI-compatible infrastructure. Dictation is the default;
file transcription is independently usable, and the optional TTS composer is
a third, on-demand workflow:

```text
hotkey -> microphone capture -> optional VAD checkpoints -> STT
       -> optional cleanup -> focus-safe insertion -> optional history
native file selection -> STT -> optional cleanup -> explicit copy -> optional history
bounded user-authored text -> explicit Speak -> TTS -> native playback
       -> optional explicit generated-audio export
```

Separately, speech playback can synthesize backend-retained history text or a
completed voice or file transcript. It does not replay captured/source audio, and the
composer does not require transcript history. Restart reuses generated PCM in
the memory-only playback session without another inference request.

Readiness is task-specific, not an application-wide setup prerequisite:
dictation owns recording setup, file transcription needs STT but no microphone
or completed dictation setup, and the TTS composer needs its own enabled speech
configuration but neither STT nor a microphone. Conversation mode remains out of
scope. Inference runs on the user's chosen server or an explicitly installed
optional Windows or macOS managed runtime; models and runtime binaries are not bundled.

## Managed local speech

`internal/managedruntime.Manager` owns the runtime inventory and independent
per-instance workers. Provider adapters own installation, metadata catalogs,
model acquisition, and process launch for NeMo transcription and speech,
llama.cpp cleanup, and whisper.cpp completed transcription. The manager
exposes a small instance-targeted Wails boundary and
bounded status events, not upstream flags. Official versioned archives are
checksum-verified before extraction is published. Model downloads use NeMo's
own manager with a Freehand-owned cache; the GGML adapters download only their
pinned catalog revisions with size and SHA-256 verification. Listing the catalog
never loads models. Each provider is bounded to one installation and one process
tree, including retired workers. Existing duplicate entries remain explicitly
repairable without ID, Connection, or model rewrites. Different providers can run concurrently.

NeMo's instance has a required transcription `Model` and an optional
`SpeechModel`; these resolve separate task contracts in one process. MagpieTTS
v2602 acquisition verifies the pinned generator GGUF, NanoCodec decoder, and ten
tokenizer members as a complete bundle. A partial or damaged bundle cannot be
marked installed or launched. The fixed launcher supplies both engine configurations,
and readiness checks require the expected transcription and speech identities
before publishing role-specific endpoints. Start, Stop, Restart, cancellation,
and process ownership remain shared. There is no second ASR slot or arbitrary
YAML/flag editor.

The optional speech selection persists with runtime preferences through the
same storage transaction. Built-in Connections advertise speech only while a
qualified speech model is selected. Removing that selection requires first
deselecting its speech Connection; validation prevents a silent fallback.
Removing the role clears its remembered model options through the existing
Connection-use ownership, while downloaded files and transcription remain.

NeMo and GGML archive installers share the bounded download and checksum
verification path. Each installer still owns its staging directory, extraction,
publication, and recovery; sharing transfer code does not merge provider recipes.

Each configured runtime has a built-in row in connection state. Storage derives its
stable identity and qualified uses and materializes it through ordinary settings
transactions, preserving selection and remembered-option foreign keys. There is
no additional runtime-event routing registry. Go rejects edits, rename, duplicate,
and delete for built-in rows; legacy managed aliases and their selections remain
intact. Runtime-owned transport/model fields are read-only in Connections, while
task options retain their existing owners. Stopped rows remain available for
explicit selection and repair, but requests fail closed until ready.

When the runtime inventory changes, the settings owner reprojects selected
Connections and gates managed Voice's realtime flag against the new model's
capabilities before validation. The model and compatible mode commit together;
switching back to a realtime-capable model does not enable realtime implicitly.
Manual Voice connections retain their mode and settings. Runtime admission and
settings-save failures return distinct bounded errors.

Built-in name projection removes only the GGML default `(CPU)` suffix
for its matching provider, leaving durable instance preferences, custom names,
legacy aliases, IDs, and task selections intact. Installed backend presentation
comes from the referenced runtime's `Status.backend`, never from a saved name.
Runtime management, quick controls, and Connection status share that formatter;
the binary label is not an inference-offload measurement.

Acquisition events contain only bounded operation IDs, phases, counters, and
terminal outcomes. GGML reports bytes written; NeMo observes metadata for the
selected model's exact owned partial/final paths while its own manager runs.
Magpie progress aggregates its model, decoder, and tokenizer bundle; extraction
and final checksums must complete before the catalog marks it installed.
Neither raw child diagnostics nor paths enter renderer progress. Full transfer
does not imply successful verification, and request admission does not imply
operation success.
The runtime manager renders this same status at the setup action and the matching
model's catalog row, including cancellation and terminal outcomes. Catalog rows
retain their position instead of regrouping on installation state changes.
The selected provider exposes acquisition, start/stop, cancellation, and bounded
status through the same runtime store. The inventory sidebar selects its detail
pane; entering runtime management from Settings resolves the current draft.
Provider artwork comes from the shared local branding registry. Managed Connection
icons resolve the referenced instance's provider, since durable managed details
intentionally contain no manual API profile. Quick settings pass their runtime
inventory into connection pickers so artwork and status use the same session.

The settings owner persists runtime instance definitions and ordinary Connections
through the existing SQLite transaction. A managed Connection references an
instance, not an ephemeral port. Each task independently selects its Connection;
there is no global managed-mode routing override. Runtime instances own loaded
models; Voice owns realtime, and task/model option ownership remains unchanged.
Request admission resolves the selected instance and qualified contract into an
immutable, credential-free loopback transport. Manual credentials and headers
are never inherited by managed requests. An unavailable selected instance blocks
that task without choosing another server. Cleanup remains independently selected
and retains raw fallback when its captured endpoint is unavailable. Selecting a
manual Connection is an explicit routing change, not a runtime lifecycle action.

Managed Nemotron setup recommends realtime Voice. Files still use completed
transcription. The existing STT and NeMo WebSocket clients remain transport
owners; no runtime install/start logic belongs in them. Readiness must reflect
the selected task's instance status rather than an unrelated runtime or manual
connection. Runtime installation/catalog management remains separate from, and
linked by, Connections and task quick settings.

Task settings and quick settings replace manual model/profile pickers with
instance-targeted controls below the Connection picker when that Connection is
managed. Runtime identity and API-model diagnostics stay in runtime management,
not repeated above the quick-settings model selector. The shared
speech controls preserve voice, speed, and qualified language/style options;
cleanup preserves its instruction and generation controls. These surfaces do not
register providers or imply additional roles: NeMo qualifies transcription and
Nemotron realtime, whisper.cpp qualifies completed transcription, and llama.cpp
qualifies S1-mini cleanup. NeMo also qualifies MagpieTTS v2602 speech with its
independent model selection. Catalog sections group these choices by task
capability, retaining Whisper family and variant grouping within transcription.

The adapter probes `/ready` and `/v1/models` at the server origin, but publishes
`http://127.0.0.1:<port>/v1` as the speech API base. Completed microphone/file
clients append `audio/transcriptions`; the NeMo realtime client appends
`audio/transcriptions/realtime`, and speech appends `audio/speech`.
Keep this distinction at the adapter boundary, not in shared client URL handling.

llama.cpp likewise publishes `/v1` for the cleanup client, with S1-mini reasoning
disabled. whisper.cpp instead publishes the server origin for native `/inference`
requests and `/health` metadata. GGML installations use pinned CPU/CUDA recipes,
host-aware recommendations, selected-model GPU warm-up, and a private output viewer.
The adapter verifies the server and companion libraries as a pinned installation;
its recorded backend determines controlled launch arguments. Backend changes
require stopped, idle worker ownership and preserve model files, instance IDs,
Connections, and startup preferences. Failed or cancelled acquisition retains
the previous installation. NeMo's device policy is unchanged.

`GetBinaryOptions` is an advisory, metadata-only boundary for new GGML
installations. Shared platform recipes separate OS/architecture, pinned archives,
layout, accelerator, and dependencies from provider/model contracts. On Windows
x64, NVIDIA device 0 with driver >=551.78 and compute capability >=5.0 qualifies
the pinned CUDA 12.4 recommendation; unknown or unsupported metadata selects CPU.
Recommendation and installation admission share compatibility rules. The UI
requires explicit acceptance before `InstallBackend`; the legacy `Install`
operation retains its CPU default. No latest-release lookup, installation
mutation, model execution, or free-VRAM-driven switching occurs during selection.
NeMo retains its own binary-selection policy: Metal on Apple Silicon, CPU on
Intel Macs, and its existing Windows driver checks. llama.cpp recommends Metal
on Apple Silicon and CPU on Intel. Its pinned macOS binaries require 13.3;
OS-version admission is separate from Freehand's 13.0 minimum. No macOS
whisper.cpp recipe exists because upstream does not publish a server executable.
Provider `backends` metadata drives the switching controls; unsupported providers
remain visible with installation disabled.

Pinned macOS tar.gz archives retain their upstream directory layout. The installer
validates every entry, bounds expanded size, and resolves same-directory dylib
aliases solely against regular files in the verified archive. It materializes
aliases as copies and verifies their bytes and executable permissions before
publication and launch. Traversal, escaping or cyclic links, hard links, special
files, duplicate names, and unexpected installed files fail closed. Windows ZIP
recipes retain their existing extraction and integrity rules.

### Download source metadata

Download provenance is additive metadata on `GetProviders`: runtime sources
project the platform recipes; model sources project the same
specifications used for acquisition and verification. It is not a second pin
registry or evidence of the currently installed files. The renderer uses the
generated `RuntimeSource`/`ModelSource` DTOs, with provider-catalog metadata as
the source for status rows. No filesystem, network, or model execution is needed
to inspect sources. Runtime artifacts identify platform/backend and include
companion archives. Direct GGML model downloads identify their Hugging Face
repository; NeMo identifies delegated acquisition and its release-index pins
without fabricating an upstream hosting URL. Source links open through Wails'
external browser API, never as remote content inside the settings WebView.

### Startup ownership and progress

Windows owns children through a Job Object, including model-manager subprocesses.
macOS re-execs a private supervisor before application startup. A lifetime pipe
from Freehand and kqueue child-exit observation own a dedicated runtime process
group. Pipe EOF on cancellation, Quit, or parent crash kills the group. Natural
server exit also kills remaining descendants. The supervisor observes exit before
reaping the leader, preventing process-group identity reuse during cleanup.
Only the exact child PID is published internally; libproc verifies its IPv4
loopback listening socket before endpoint admission. The inherited environment
excludes DYLD, user PATH, proxy/auth, and runtime configuration overrides; NeMo
gets only a fixed system PATH for its explicit curl downloads.
The server listens only on `127.0.0.1`; verification, selected-model readiness,
and required GPU warm-up precede endpoint admission. Runtime/model hashing before
launch is cancellable. After process creation, readiness and warm-up share a 120-second timeout;
failure or cancellation has an additional four-second owned-process drain bound.
A deadline is not proof of child exit: retain ownership and reject replacement
until the process has actually stopped. Wails shutdown cancels work and closes
all owned process trees against its separate overall eight-second bound.

GPU llama.cpp and NeMo retain upstream built-in warm-up before readiness; CPU
launches retain `--no-warmup`. CUDA whisper.cpp instead receives one multipart
`/inference` request containing one second of synthetic silence after readiness,
against only its already-loaded selected model. The response is bounded and
discarded, never routed through cleanup, history, or user result publication.
This is startup work, including saved start-at-launch intent, not a health probe
or model-discovery operation. No inventory is invoked and no remote fallback
is admitted.

`Status.startupProgress` reports phase-specific elapsed time for
`verifying_runtime`, `verifying_model`, `launching`, `waiting_ready`,
`loading_warming`, and `warming_up` as applicable. Upstream loading and warm-up
share `loading_warming` when their boundary is not observable. Status derives
from lifecycle transitions, not output parsing or elapsed-time percentages.

Terminal operation state and admission reopening commit together under the worker
mutex. A separate publication mutex orders the captured terminal snapshot before
the next operation's initial notification; callbacks run outside the worker mutex.
The manager consumes that immutable status/active-model snapshot rather than
rereading mutable worker state. Seeing a completed operation must not itself
cause an immediate follow-up action to fail as still busy.

Start, Stop, and Restart bindings acknowledge admission; status events report
progress and completion. Restart owns stop and relaunch as one worker operation,
waits for the old process exit, and never relaunches after failed stopping,
cancellation, or shutdown. Auto-start publishes the same `starting` transition
before adapter launch. The renderer displays pending requests immediately, then
uses backend lifecycle stages and outcomes without optimistic running state.

### Explicit process-output observation

Each worker privately captures a rolling stdout/stderr tail by default, bounded
to 256 KiB, 1,024 chunks, and 4 KiB per chunk. A separate bounded prefix remains
for existing diagnostic parsers; it is not the viewer source. Strict command
metadata capture remains distinct from both, so tail truncation cannot validate
an incomplete catalog. Process-generation fencing rejects stale callbacks.

The workspace's **Runtime output** panel displays the selected runtime's output
immediately when its tab opens, including during startup. Runtime management and
workflow controls reveal that shared panel. The mounted reader enables access
only while the panel, workspace, and document are visible. Hiding the panel,
switching panel tabs or runtime targets, entering global Settings, hiding the
workspace, or unmounting clears renderer output and disables reads. Reopening
the visible viewer enables fresh reads directly.

`internal/windowing` also owns one reusable standalone **Process output** window.
That window requires sensitive-output consent on each opening or runtime switch
before enabling reads. Both viewers observe the runtime without owning it.
Cursor-based, bounded request/response deltas are polled without overlap; no
output is published in events, application logs,
bridge tracing, crash reports, or files. Frontend accumulation is also bounded.
A read-only xterm.js surface displays bounded UTF-8 text, newline, tab, carriage
return, backspace, validated bounded SGR colors/styles, and erase-line progress
controls. Other control families are filtered, including OSC, DCS/APC/PM/SOS,
terminal queries, input modes, and alternate-screen operations. Decoder state is
bounded and independent per stream, including fragmented or malformed controls.
Filtering is not sensitive-content redaction. The viewer has no process-input,
link, title, or escape-triggered clipboard handlers; terminal-generated responses
never reach child stdin. Fit/search addons and search terms are viewer-local.
Terminal scrollback and write queues are bounded; reset, eviction, and reader
access changes discard stale rendered state and pending writes. Follow controls
scrolling; Clear drops the tail.
Explicit Copy selection sends only selected rendered text to the native clipboard,
without reading clipboard contents, automatic copy, Copy-all, export, or shell input.
Copied text can outlive the viewer and be visible to other applications. Teardown
disposes terminal, addon, and resize resources.

Closing a viewer or switching its runtime clears visible renderer data and revokes retrieval. Disabling
access does not erase private memory. The tail may survive process exit for
inspection; Clear, the next start attempt, runtime removal, and shutdown release
it. Viewer actions never start, stop, restart, or orphan a process. llama.cpp
uses normal non-debug `--log-verbosity 3 --log-colors on` output in the existing
private capture. NeMo uses `--access-log --log-format json` to emit structured
HTTP request records alongside its plain startup output. Global `--json` stays
off so normal loading diagnostics remain available. Structured access records
have an `http.request` event, request ID, method, path, status, and remote address;
they are private child output, never application telemetry.

**Highlight logs** derives bounded display-only SGR colors from recognizable
severity, status, and JSON tokens, preserving upstream ANSI and the original
search/copy text. Fragmented records, renderer resets, stream changes, and hidden
viewers retain the existing bounds and generation fences. JSON values are never
decoded into terminal controls or interpreted as actions.
The pinned logger defaults to no disk sink; file/prompt logging flags and
inherited logging/config overrides remain excluded. Normal logs may contain
sensitive content; whisper verbosity is not enabled.
See the [logging contract](../../safety/logging/#managed-process-output) for the
narrow renderer exception and unchanged application-log prohibitions.

Other platforms expose unsupported managed-runtime status without changing
native capture or manual endpoints.

## macOS platform boundary

Windows and macOS use native adapters under shared feature owners.
Shared audio selects CoreAudio or WASAPI. Quartz/Accessibility own Mac keyboard and
safe delivery, Security.framework owns Keychain, and a nonactivating Cocoa panel
owns the passive overlay. Wails retains the interactive shell, tray and single instance.

Mac delivery captures NSWorkspace's frontmost PID/process-start identity and a
retained AX focused window. It verifies that app/window before delivery and each
Unicode chunk, not editor-element identity. Same-window field changes deliver to
the current field. Editor roles, AXEnabled, protected-content metadata and AXValue
settability are not admission requirements. Permission, Secure Input, physical
modifier, cancellation and no-activation guards remain mandatory; Secure Input
does not guarantee detection of every custom secure field.

`internal/insertion` owns allowlisted rejection stages/reasons and generic fallback.
The dictation recorder retains capture rejection within its recording generation;
native cleanup preserves the first failure without retaining diagnostic target
metadata. Copy-required presentation must not equate every rejection with moved
focus or zero typing. Quartz posting can be partial or ambiguous, so delivery is
not retried automatically and clipboard recovery remains explicit.

`shortcut.Controller.Start` tolerates an unavailable saved hold hook without
rolling back independent Carbon Toggle/Show registrations; explicit `Configure`
changes remain transactional. The settings service's `RetryHoldShortcut` binding
serializes with settings publication/saves and invokes the injected controller
retry after activity admission. It returns/publishes refreshed availability without
writing preferences. Native rearm checks released keys before and after opening a
fresh tap and starts a fresh reducer; a failed replacement leaves the previous
working hook intact. The Shortcuts UI retries only on an explicit click and
refreshes availability after a rejected Wails call without overwriting dirty drafts.

`internal/input` exposes non-secret permission status and explicit recovery actions.
`internal/settings` and `internal/buildinfo` publish native platform metadata; renderer
labels do not infer permissions from a user agent. Mica is Windows-only. AppKit
operations belong to the main thread, and `internal/app` releases native target state
and SQLite after feature shutdown through Wails `PostShutdown`.

### NeMo metadata and model preferences

NeMo connection checks retain bounded `/v1/models` IDs, capability labels, and
device strings. Saved-Connection diagnostics retain the combined inventory;
per-task checks and model pickers select their own capability.
Transcription choices exclude explicitly incompatible capabilities;
missing capability metadata stays unknown. An optional, two-second `/health`
probe beside the configured `/v1` prefix supplies the runtime version without
changing origin or overriding a custom health path. Failure of this optional
probe does not invalidate a successful model inventory. Metadata never chooses
model profiles or triggers inference.

NeMo speech voice discovery reads only the selected `speech` row in `/v1/models`,
including bounded `voices` and `languages`. It never falls back to another model
or a general voice endpoint. Language metadata intersects the explicit Magpie
profile; optional Japanese/Chinese frontends can narrow the offered languages.
An unknown or absent speech row is a metadata failure, not permission to probe
models by running synthesis.

`compatibility.NeMoOptions` is a value-only snapshot shared by completed and
realtime transports. `config` owns active controls, `modelsettings` owns the
per-connection/use/model subset, and migration 00004 plus sqlc queries persist
both in the existing transaction. Zero values preserve punctuation-on, verbatim,
filter-off behavior; endpointing zero omits the override. Asset availability is
not inferred from metadata. Runtime inventory changes restore the selected
model's remembered controls, or defaults for a newly selected model, preserving
task languages and manual connection selections. The internal inventory save
accepts bounded retained language preferences; renderer saves and request
admission still enforce the selected model's language contract.

## Optional realtime dictation

Qualified realtime combinations include Nemotron 3.5 on NeMo-Speech.cpp v0.1.0
and Qwen3-ASR on vLLM 0.28.0. The vLLM adapter uses model-only configuration,
JSON/base64 audio, and distinct delta/done events. `internal/realtime` owns the versioned WebSocket
adapters and bounded audio/text transport; `internal/dictation` owns capture,
generation fencing, immutable profiles, finalization, cleanup, and safe delivery.
Voice has one active connection/model/profile and an optional qualified realtime
mode. `VoiceTranscription` owns microphone settings; root STT fields own audio
files. Dictation captures only
the Voice credential for either transport, while files capture only their own key.
The completed Voice snapshot adapts onto the existing STT request fields without
changing persistent file settings. Native captions carry a bounded transient tail
in one fixed row; they never become a delivery source or take focus.

The realtime reader retains the immutable request credential only for response
admission. It rejects literal credential reflections in accumulated deltas,
wire finals, language evidence, and parsed/joined transcript text before publishing
that value or returning deliverable text. Rejection signals capture failure and
drains owned audio through the normal cancellation path; diagnostics expose only
`credential_reflection`. The key is never stored on the session or in results.

NeMo uses `audio/transcriptions/realtime` beneath the HTTP API root; vLLM retains
its separate `realtime` route. Completed NeMo requests use `verbose_json` for
language/duration metadata. Nemotron terminal language tags are stripped only
from text and merged with reported languages. Realtime finals accumulate bounded
language evidence into response details for the existing cleanup gate. Structured
realtime fields are optional; v0.1.0 relies on the terminal tag. Unknown structured
locale evidence and overflow are represented as multilingual without publishing
arbitrary peer text. Provisional text never contributes authoritative language
metadata or delivery content.

The Qwen profile intersects model languages with the vLLM language map and
publishes restricted choices plus mode-specific language-hint metadata. Realtime
omits saved completed context, language, vocabulary, and temperature. The model
layer removes fragmented structured Qwen headers; it never deduplicates speech.
Only an explicit final text field after local stop is deliverable. A mixed
language result is reported as multilingual so English-only cleanup falls back.

The main window owns one Session and one activity rail for Voice, audio files,
speech, Connections, local runtimes, history, and global Settings. Workflow
configuration renders in the contextual right inspector; Connections owns the
shared server editor. `ShellNavigation` retains accepted task origin and return
section; all exits from configuration resolve the active draft before changing
panes. An ignored reveal cannot replace the draft or its completion intent.
`internal/windowing` validates sections, origins, and connection requests;
`ShellReady` gates native navigation until renderer subscriptions and initial
loading complete. Global and contextual Save apply in place; closing or crossing
configuration realms resolves the draft before completing navigation. Connections'
explicit Save and return commits setup and resumes its workflow only after success.
Native close resolves Save,
Discard, or Keep editing before hiding the workspace. Failure retains the draft
and error. Hiding clears transient credentials, shortcut capture, overlay
preview, and sensitive runtime output. Go-owned jobs keep their own lifetimes.

Runtime preference changes refresh through the shared editor's snapshot
reconciliation. A refresh preserves settings and connection drafts, including
edits begun while the runtime save was pending. Read sequence and snapshot
revision fences reject responses superseded by a newer read, event, or save
acknowledgement. Renderer disposal rejects late reads and save acknowledgements
and prevents queued saves from starting.

## Durable settings storage

The SQLite store uses modernc, embedded Goose migrations in `internal/storage/schema/`,
and sqlc-generated queries. The distinct `freehand.db` identity starts from safe
defaults and an empty catalog. It never reads, imports, converts, or deletes
`settings.db`, `settings.json`, or their legacy native credentials.
Goose alone owns schema versions.
`internal/storage` owns the database lifecycle and adapters; `internal/settings`
retains coherent saves and immutable request profiles. See the
[storage maintenance guide](../storage/) for schema changes and recovery.
Transcript history remains optional and memory-only.

Named connections represent reusable servers, with explicit supported uses and
independent active selections for Voice transcription, audio-file transcription, cleanup, and playback. One ID
can be selected by multiple features; their models and runtime options remain
independent while URL, profile, authentication, and credential reference are shared. The shared connection editor owns endpoint/authentication/profile fields.
The Connections rail page and workflow pickers use the same connection editor.
Its searchable list contributes to the shared primary sidebar; at compact widths
the title-bar toggle exposes the list over the editor. It owns library creation, editing,
duplication, deletion, and saved metadata tests. Its renderer guards list, row,
workflow, and close navigation with save/discard/keep-editing handling; credential
drafts remain in the Settings editor. Workflow pickers search the same catalog and retain
Add and Manage actions outside the scrolling results. Library creation offers
Save for later or a workflow setup destination. Its explicit Save and return action sends
`Change.ActivateFor` with Create so catalog, selection, settings, and key references
commit together. Storage rejects invalid or unsupported activation purposes and
activation on other actions. New selections still require model configuration;
no inference or optional feature is enabled by creating a connection. Feature pages own active selection, model,
language, presets, voice, and other runtime options. Fresh catalogs are empty.
Selection restores remembered engine options while preserving task intent; an
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

| Concern                                                                                           | Owner                                                                  |
| ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| Live dictation state machine                                                                      | `internal/dictation`                                                   |
| Cross-feature start admission and recording preemption                                            | `internal/activity`                                                    |
| Global shortcuts                                                                                  | Platform shortcut adapters                                             |
| Audio capture and normalization                                                                   | Go audio service                                                       |
| Endpoint requests and cancellation                                                                | Go provider-neutral inference client                                   |
| Native stored-file grant and transcription job                                                    | `internal/filetranscription` + inference client                        |
| Optional transcript post-processing                                                               | `internal/postprocess` request/outcome policy; feature-owned execution |
| API credentials                                                                                   | Windows Credential Manager / macOS Keychain adapters                   |
| Original target and insertion                                                                     | Platform focus/input adapters                                          |
| Optional transcript history                                                                       | `internal/history` memory store                                        |
| Optional authored-text/transcript synthesis, native playback, and explicit generated-audio export | `internal/tts` + inference speech capability + native playback adapter |
| Optional passive status overlay                                                                   | Go overlay service + native Win32/Cocoa renderers                      |
| Native tray presentation/actions                                                                  | `internal/tray` consuming bounded domain snapshots                     |
| Tray ownership, startup, single instance                                                          | Go/Wails platform lifecycle                                            |
| Task, settings, and status rendering                                                              | Svelte through generated Wails bindings                                |
| Durable non-secret configuration                                                                  | Platform configuration directory + `freehand.db` (SQLite)              |
| Structured runtime diagnostics                                                                    | One Wails default logger hierarchy, injected by `internal/app`         |
| Release identity and version                                                                      | `build/config.yml`, parsed by `internal/releaseinfo`                   |
| Release discovery and staged executable updates                                                   | `internal/updates` + Wails updater GitHub provider                     |

The frontend never receives a stored API key, raw audio, or selected filesystem path. A key being entered by the user exists only as a bounded, transient password-field draft until it is saved to the native credential store or the settings flow is left. Go opens the Wails native file picker and converts its result into a backend-only selection capability; the zero-argument renderer binding cannot nominate another path. Status events contain the opaque operation generation, base file name, byte progress, mode, and transcript text, never the full path or audio bytes.

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
internal/platform/audio    native capture, playback, permissions, and level taps
internal/platform/keyboard native hold-to-talk and temporary shortcut capture
internal/platform/input    native target identity, Unicode insertion, explicit copy
internal/platform/overlay  passive native overlay presentation and rendering
internal/platform/startup  per-user native startup registration
internal/managedruntime    runtime inventory, provider adapters, worker lifecycle
internal/managedruntime/internal/artifact verified files, archives, and installation
internal/managedruntime/internal/process  owned process trees and listener identity
internal/storage           SQLite transactions, migrations, and generated queries
internal/realtime          shared session lifecycle with distinct NeMo/vLLM protocols
internal/postprocess       transcript-cleanup request and outcome policy
internal/tray              native status, last-activity, recovery, and window actions
internal/updates           persisted polling policy and Wails updater lifecycle
frontend/src/lib           testable settings/status state
frontend/src               thin Svelte components
```

### Organizing implementation files

Freehand uses one Go module. Domain packages own their mutable state and expose
narrow capability boundaries. Within each domain, filenames describe the work:
dictation admission, capture, processing, status, and delivery; file selection,
transcription, and streaming; speech sources, generation, playback, and export.
These files share their existing owner, locks, generations, and shutdown budget.
Settings keeps snapshots, save/rollback, recovery, and request-profile capture
inside the same package and transaction boundary. Configuration values and
validation are grouped by Voice, speech, cleanup, overlay, and transport.

Native adapters are grouped by capability under `internal/platform`. Each
capability contains its Windows and macOS implementations, unsupported-platform
stubs, native bridge sources, and tests. `internal/app` composes them directly.
The platform-neutral `audio`, `hotkey`, `insertion`, and `shortcut` packages retain
their contracts and policy. Capture and playback share native audio setup;
hold-to-talk and shortcut capture share one keyboard coordination boundary.
The overlay consumes a small level-source interface satisfied by the audio tap,
without depending on the native audio implementation.

The managed-runtime package owns inventory admission, provider selection,
worker state, ordered publication, and output access. Its private `artifact`
package owns pinned download verification, safe extraction, owned-path checks,
and installation rollback. Its private `process` package owns native process
trees, bounded private output, and listener identity. Neither leaf imports its
parent. Process launch accepts explicit output/start callbacks; the parent
applies worker-generation and shutdown guards before publishing observations.
`Manager` is the renderer boundary, and each runtime instance uses the same
worker engine. Provider recipes and model qualification stay with that owner.

Use provider-specific names for provider-specific files, such as
`nemo_adapter.go` and `nemo_catalog.go`. Shared mechanisms use their actual
responsibility as their name. Keep tests beside the implementation they exercise;
generated SQLite code remains in `storage/dbgen`, with queries and immutable
migrations in their existing directories. A new subpackage must have a small,
one-way dependency boundary and must not require exporting an owner's locks or
mutable state.

Action feedback remains renderer presentation. `SessionMessages` distinguishes a
speech-status failure by operation generation from an unrelated command error.
The workspace suppresses its shared copy only while the matching failed speech
session has visible local feedback; other tasks and Settings retain the shared
fallback. New speech status clears only its own stale failure, and replaying an
unchanged failure event does not resurrect a dismissed notice. This does not
change backend admission, retry capabilities, or request state. Shared notices
float without participating in workspace layout; task details use bounded,
keyboard-accessible popovers.

The renderer sees small Wails services registered from the package that owns each capability. Wails is the bridge boundary, not the application's package hierarchy:

| Bound package       | Renderer responsibility                                                                                                                                                     | Backend authority                                                    |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `settings`          | Renderer-safe snapshot and one atomic settings/credential/startup/shortcut save request                                                                                     | Settings transaction owner                                           |
| `input`             | Microphone inventory, permission status, and native shortcut capture                                                                                                        | Audio and platform keyboard/permission adapters                      |
| `connection`        | Focused STT and post-processing health/model probes                                                                                                                         | Inference metadata capability                                        |
| `dictation`         | Live commands and status snapshot                                                                                                                                           | Package-owned recorder                                               |
| `history`           | Bounded history queries, copy, delete, and clear                                                                                                                            | Package-owned synchronized store                                     |
| `filetranscription` | Native picker grant, upload/transcription state, retry, cancellation, copy                                                                                                  | Package-owned file job using the injected history store              |
| `tts`               | Speak bounded user-authored text; listen to backend-owned history versions or the completed file transcript; preview, pause, resume, restart, stop, save, clear, and status | Package-owned synthesis/playback session; native-dialog audio export |
| `updates`           | Current bounded status and explicit user-initiated update review                                                                                                            | Package-owned polling policy over the configured Wails updater       |

Focused operations use generated request DTOs. A connection probe receives only its endpoint, model-discovery, headers, authentication policy, and bounded transient credential draft, so unrelated shortcut, VAD, window, history, or processing drafts cannot invalidate it. Settings save similarly receives one generated `SaveSettingsRequest` instead of a positional credential argument list. Svelte constructs and consumes those generated shapes directly.

Shortcut settings follow the same boundary. `internal/hotkey` owns one bounded action matrix for toggle recording, Show Freehand, and hold to talk. `internal/input` exposes that matrix as generated policy metadata, accepts one action-specific capture request, and emits bounded normalized chord progress while the native hook is active. The renderer never selects native validation flags or maintains a second key grammar. Expected capture rejections return structured categories; a cross-process global conflict remains knowable only when the transactional settings save asks Windows to register the replacement.

All shortcut assignments are optional, including toggle recording. Empty values
survive settings transactions and SQLite reload without default substitution.
`internal/shortcut` separates saved preferences from successfully bound chords:
startup reports conflicts but preserves independent registrations and a usable
capture guard. Capture suspends/restores only those bound chords, not rejected
startup preferences. Explicit saves retain register-before-unregister rollback.
`ConfigureWithRollback` captures the controller's saved preferences and effective
toggle/show/hold bindings together before applying a change. Its returned rollback
belongs to the single settings transaction, including configuration recovery: a
later startup, credential, or persistence failure attempts to restore that exact
snapshot, not register previously rejected saved chords. Native rollback failures
remain part of the reported save error. The controller keeps no implicit
previous-settings history and substitutes no fallback.
The controller calls the hold adapter's `Start` for initialization and `Configure`
for subsequent changes, including when the initial assignment is empty; the
Windows reducer emits no edges for an empty chord.
Voice readiness does not require a global shortcut. Recording admission and
focus-safe insertion remain unchanged.

Custom health targets retain the base-relative path-joining contract, including
the leading slash in saved values. Settings help and validation describe that
contract; no configuration migration or fallback model probe is performed.

Model probes require a valid JSON model-list shape; reachability remains distinct from inventory validation and inference compatibility. Health probes accept opaque successful bodies. Invalid model responses cannot satisfy first-run dictation readiness.

After loading the applied profile, the renderer runs one bounded metadata-only STT probe and repeats it only when that profile's connection identity changes. The main shell initially selects dictation, but readiness is presented within the selected task rather than replacing the app shell or preventing task selection. The persisted setup-completion flag gates only dictation: the Go recording command rejects both renderer and global-shortcut starts until it is complete. File readiness uses the STT connection/authentication checks without microphone, shortcut, or dictation-completion requirements. The TTS composer does not show STT readiness and validates its own enabled speech configuration. A later failed STT check may show task-scoped recovery; dismissal leaves the persistent status warning and is scoped to the exact failed condition rather than mutating durable settings.

Ordinary Go collaborators remain ordinary types: `history.Store`, the inference client, post-processor, capture adapter, and insertion policy are injected by `internal/app` and are not registered with Wails. The settings/profile transaction remains one owner even though consumers receive narrow snapshot functions.

The storage package opens one connection with a per-process file lock, validates
Freehand database identity and goose history, and upgrades forward before
settings-dependent services start. Typed tables hold preferences, Voice/file
transcription, cleanup, playback, vocabulary, connections, remembered models, and
deferred credential deletion. Saved connections are the durable authority for
transport, headers, and opaque native credential references. Fresh initialization
writes defaults transactionally; there is no JSON importer or import marker.

A load failure or uncertain commit creates an explicit recovery state: ordinary
saves and new request profiles are blocked. Retry validates and reloads committed
settings; explicit Reset archives the current database and sidecars before
replacing it with fresh settings. Users reconfigure connections and keys; native
credential records are not automatically purged.
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
domain reports it available, and authoritative Quit. The native menu has no
recording-start action. Left-click instead opens the app-owned `tray-popover`:
a status-item-anchored nonactivating NSPanel on macOS, or an activating,
keyboard-interactive Wails window on Windows. The Windows panel has no taskbar
entry and uses Wails' native placement with an 8-DIP offset from the work-area
edges. Right-click retains the native menu; Windows explicitly hides the panel
before opening it, while macOS uses AppKit's native pre-click tracking.
Its separate renderer Session presents existing dictation/settings state, with
explicit copy, cancellation, and navigation. It exposes no capture or quick-save
controls. Dismissal does not cancel Go-owned work or restore a destination;
recording starts from a destination-focused shortcut or the main workspace.
Main and Settings navigation hides the panel. The runtime controller does not synthesize or repurpose
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

Dictation owns rejected-start feedback as well as active-run status. Rejection
before microphone capture must reach the status subscribers used by both the
workspace and passive overlay, even when a global shortcut has no renderer call
awaiting its error. Failure presentation must not replace pending copy recovery,
release retained result capabilities, or overwrite active work. `StartRejected`
distinguishes admission feedback from the retained result's delivery outcome, so
the overlay shows a failure without old run timing and the workspace keeps Copy
available alongside recovery. Runtime readiness
changes update the mounted recording transport rather than replacing it with a
different layout; Go remains authoritative for start admission.

The overlay owner dismisses failed and copy-required presentations after five
seconds without changing dictation state or retained result capabilities. Its
version-fenced timer cannot hide newer feedback, active work, or a preview.
Settings updates and closing a preview do not revive an expired outcome; a fresh
status publication starts a new presentation interval. Shutdown stops the timer
and fences callbacks before releasing the native surface.

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

Settings and Connection navigation use the controlled `PendingChangesDialog`.
Parents own drafts, pending destinations, and save/discard actions; the shared
dialog owns presentation and blocks Escape/outside dismissal while saving.
Confirmed settings adoption invalidates metadata once through `SettingsEditor`,
including credential changes that are invisible to the renderer.

`RuntimesPane` owns provider and installation selection. `RuntimeDetail` owns
installation choices, action guards, and removal confirmations;
`RuntimeModelCatalog` renders the qualified model list. These components use
the existing runtime store and action guards; component mounting does not
acquire models or start processes.

Each WebView composes its own `Session` from feature owners under
`frontend/src/lib/stores`. `Session` owns construction, initial loading order,
aggregate busy presentation, and presentation teardown; it is not a second
command facade or a container for feature state.
The default main-window Session belongs to the WebView module, so a development
hot replacement of `App` unsubscribes the old view and clears credential drafts
without disposing that shared Session. Actual page unload or Session module
replacement disposes it; a BFCache page suspension preserves it for restoration.
Disposal is terminal: `Session` stops scheduling subsequent initialization steps
and suppresses late initialization failures. Already-started operations retain
their feature owners. The app mount separately guards metadata replies,
appearance updates, and the `ShellReady` handshake against component teardown.

- `SettingsEditor` owns the applied settings snapshot, independent editable
  settings/credential draft, connection probes, microphone choices, recovery,
  and serialized quick saves. These remain together to preserve one coherent
  editing transaction. Pure snapshot copying and quick-control patch projection
  live in `utils/settingsDraft.ts`; that helper owns no bindings, credentials,
  save queue, or reactive state.
- `DictationState` owns live-dictation projection and commands. It rejects older
  generations before changing state or triggering history and notification
  reactions. Snapshot responses use the same admission rule; a newer generation
  wins even when overlapping reads finish in the opposite order. Same-generation
  outcomes remain valid, while intervening events supersede same-generation reads.
- `FileTranscriptionState` owns stored-file projection, generation/revision
  reconciliation, delta-gap recovery, and explicit file commands. It also owns the
  renderer-session streaming preference independently of backend capability status.
  Effective streaming intersects that preference with current support; component
  remounts and unsupported connections cannot overwrite it. A successful explicit
  capability reset enables the preference for the next manually started request.
  Resetting capability never starts inference and blocks conflicting local starts
  until it completes. This preference is not persisted across renderer reloads.
  Pending start, clear, cancel, and picker commands also have renderer guards for
  immediate feedback and duplicate suppression; Go retains authoritative admission.
  Cancellation remains available once Go reports an active job even before the Start
  binding returns. Picker responses use the same generation/revision reconciliation
  as status events so a delayed reply cannot replace newer capability or progress state.
- `SpeechState` owns playback projection and speech commands, including preview
  admission. It admits one pending Listen binding across voice, file, and history,
  retaining the source/history identity for immediate control feedback. Listen is
  unavailable during that pending call or backend-reported generation, and during
  composer/preview admission. Rejection releases the guard for retry. Playback and
  pause still allow an explicit replacement after generation completes. Go owns
  synthesis cancellation and audio replacement. The generated speech and dictation `CurrentStatus` methods remain
  in separate service namespaces.
  Its save-pending state spans the complete SaveAudio binding, including the native
  dialog and file write, and is shared by compact and embedded playback controls.
  Duplicate renderer requests return before clearing feedback. Success, cancellation,
  and rejection release that guard without blocking playback. Go retains its atomic
  save-dialog guard, generation validation, PCM snapshot, and file-write ownership.
- `HistoryState` owns history refresh/mutation ordering. Successful refresh
  acknowledges the completed file generation through an injected callback.
  `HistoryPane` owns presentation-only search, source filtering, and selection for
  its sidebar, expanded reader, and right-side details contribution. It filters the existing bounded entries by
  final/raw/processed text or file base name and by All, Voice, or Audio files;
  it adds no storage, history acquisition, or inference operation. Sidebar controls
  read applied retention settings and call the existing settings and clear actions.
  `HistoryList` owns local disclosure/comparison state. In compact Home and Settings
  lists, the leading result opens fully, older entries start collapsed, and a
  different leading result resets manual expansion. Updates to the same entry
  preserve those choices. An ephemeral file result takes the compact list's leading
  position without changing retained history or persistence. The full History pane
  presents the selected retained entry expanded with transcript actions and passes
  that same entry snapshot to embedded `HistoryDetails`. It does not invoke the
  native details-selection service or make another details request.

- `SessionMessages` owns shared presentation notices and their timers, not
  workflow state.

Components access these owners directly. Generated services and DTOs remain
wire authority; feature dependencies use narrow types derived from those
services rather than handwritten transport shapes. Feature owners do not import
`Session` or acquire subscriptions during construction.

The main renderer installs `subscribeSessionEvents` before loading snapshots.
The workspace, global Settings, contextual options, and Connections page share one Session
and draft owner. Accepted terminal transitions refresh history. Renderer teardown
disposes subscriptions, timers, and credential drafts without stopping Go-owned
recording, transcription, or playback. Leaving or hiding configuration releases
preview/capture resources; backend snapshots preserve unsaved draft conflicts.

`App` provides one `WorkbenchLayout` context and composes `WorkbenchFrame` around
the active page. Title-bar controls toggle the primary sidebar, bottom panel,
and secondary sidebar. `SidebarContribution` registers the active area's content:
workflow settings, global Settings navigation, Connections, History browsing,
or runtime inventory.
The primary sidebar follows the area while its wide-window visibility remains a
workspace preference. Below 700px it uses a separately controlled compact overlay
and starts hidden; page navigation closes that overlay. Standalone components
without the layout context render their sidebar locally. `WorkspaceSplit` keeps
its local layout only for those standalone consumers; shell Home supplies the
current result without a second bottom panel.

`WorkbenchPanel` supplies Recent, Runtime output, and Diagnostics independently
of page navigation. The selected tab survives page changes and user hiding.
Global Settings suppresses bottom-panel rendering to give configuration the full
vertical area. This page-level suppression does not mutate the selected tab or
saved visibility preference; returning elsewhere restores both. The bottom panel
is also unavailable below 560px viewport height without changing its saved
visibility choice. The secondary sidebar is contextual: Voice exposes
Transcription, Audio, Cleanup, Vocabulary, Overlay, and Delivery; Audio file
exposes Transcription, Cleanup, and Vocabulary;
Text to speech exposes Speech. History offers the selected run's `HistoryDetails`
and History settings. The main editor stays mounted while those options are edited.
Changing pages closes contextual options, and the right toggle opens the current
workflow's options or History details. Each sidebar header exposes an X routed
through `App`'s existing draft-resolution guard. History initializes wide-window
details visibility on each visit, respects explicit closure during that visit,
and offers a Details action in its page header. That action also resolves a
pending History settings draft before replacing it with details.
Voice, Audio file, and Text to speech keep their detailed options explicit.
Their primary sidebar reuses immediate connection/model/language and speech
controls against the applied snapshot. Cleanup quick controls are shared by
Voice and files; managed model selection remains runtime-owned. Quick saves use
the existing serialized settings transaction, and scoped control IDs keep the
primary controls independent from the contextual draft editor.
`ManagedRuntimeControls` follows each local connection in the primary sidebar,
including shared cleanup independently of its enabled switch. The same component
owns runtime status presentation and explicit lifecycle/model commands in detailed
options; it never starts a process on mount. Both placements use the same bounded
runtime surface, icon, visible model label, status, and management/output actions.
The contextual connection picker receives the same runtime inventory from its
owning session for provider icons and status summaries.
Detailed settings add vertical spacing outside that surface so settings-group
row separators cannot strip its inner inset. Sidebar field labels use the shared
value typography, and advanced-option links share `SidebarSettingsLink`. Delivery
rows adapt to their container width, including a compact-window drawer.
Active workflow and draft locks remain
separate from the runtime's own pending state so cancellation stays reachable.
Metadata-only connection probes do not block runtime commands. Runtime management
links queue an ephemeral exact-instance selection for the inventory through the
existing guarded navigation; output links use the shared bottom panel.
Opening clean contextual options does not lock runtime commands or quick controls;
active edits and transactions retain their existing mutation guards.
Below 1100px viewport width, the sidebar
starts hidden and opens as an explicit right-side overlay. Escape/backdrop
dismissal follows the same draft-resolution guard as navigation. It never moves
below the reader. Responsive hiding
preserves the wide-window visibility preference for restoration. Each visible
region owns its scroll area. `LayoutDivider` draws a 1px hairline with a larger
pointer target and supports bounded pointer and keyboard resizing, including
Home/End limits and orientation-aware arrow keys.

`WorkbenchLayout` persists only visibility booleans, bounded bottom/right sizes,
and the selected bottom tab in WebView local storage. Sidebar snippets, History
search and selection, configuration drafts, credentials, output runtime identity,
standalone-viewer consent, transcript text, and runtime output never enter layout
storage. Runtime output uses explicit runtime tabs independent of page navigation.
The initial target is a running runtime, or the first installed runtime if none
is running; later page changes preserve that target. Removing the selected
runtime leaves it unavailable until another target is selected. Its reader is
disposed when the bottom panel hides or changes tabs, including suppression on
the global Settings page. Selecting another runtime releases the previous reader
and displays the new target immediately. Window or document hiding disables reads
and clears buffered viewer output. Showing the region again resumes direct reads
for the retained target; pending reads from a previous viewing cannot repopulate it.
Recent history continues
to use compact expandable rows.
Runtime model browsing uses one catalog component before and after installation.
Whisper families, search, and expanded model details are local presentation state;
they do not change qualification or invoke runtime operations. The runtime detail
owner retains action admission, process controls, and removal confirmations.
Catalog data and model acquisition remain Go-owned and revision/checksum pinned.
The title bar, activity rail, sidebar, and status bar share neutral surface roles;
the status bar exposes capture state from every pane. The command palette routes
through the same navigation and busy-state guards as the rail.
Every main editor page uses `PaneHeader`: a fixed 44px row with a 12px inset,
16px icon, and 16px interface-font title. Workflow, loading, readiness, connection,
runtime recovery, and Settings states retain that first divider and title baseline.
Transport state, selected-connection details, runtime metadata, and recovery notices
belong below it. Settings may keep this header sticky while its content scrolls.
Interface text is 13px and help text is 12px. Reader and toolbar gutters use 12px;
full-page settings forms retain a 20px content inset.
Sidebar headers use the shared 34px `workbench-header` role; docked controls use
the 28px minimum `workbench-toolbar` role. Titles retain the established 16px
semibold type; compact uppercase headings and section icons remain muted.
Header surfaces inherit their parent background and use hairline dividers.
Toolbars and page actions can wrap when
their own pane narrows. `workbench-tab` provides the same selected underline,
hover, and inset keyboard focus for bottom-panel tabs, runtime output targets,
and History sidebar views. Navigation lists use full-width rows and an edge
selection marker. Page and section headings use the interface typeface.
Shared `content-*` roles in `app.css` distinguish page titles, section headings,
field values, technical metadata, and compact uppercase labels. Small semantic
icons identify sections; neutral summary surfaces group key facts. Brand and
status accents retain their action, selection, and state meanings.
Transcript and composer text use a separate 15px/26px reading rhythm. Recording,
file, playback, and output controls sit in docked strips separated by hairlines.
Settings clusters use `SettingsCard` and `SettingsDisclosure` for flat groups
separated by rules. Native expandable sections share `Disclosure`, with a leading
chevron, optional description/icon, expanded-surface contrast, and an indented body.
`DisclosureButton` gives state-owned runtime panels the same treatment; their
controlled targets remain mounted while their contents retain conditional mounting.
The opt-in `disclosure-trigger`, `disclosure-chevron`, and `disclosure-body` styles
also cover catalog rows and workflow disclosures without changing page or workbench
header geometry. Native summaries retain browser keyboard semantics; controlled
buttons expose expanded state and target IDs. `WorkflowSection` supplies the
collapsible quick-control group, with one leading disclosure action. Field groups and catalogs respond to
their container width, including a narrow sidebar inside a wide window. Shared fields
and pickers use 32px controls, actions use 28px controls, and compact toolbars
use the smaller variants. Native control semantics, visible focus, input
borders, and floating menu/dialog surfaces stay explicit. Page styling never
owns workflow state, credentials, or scrolling behavior.
Model and voice metadata refreshes share `PickerRefreshButton`: an icon action
in the sidebar and an outlined action in detailed settings, with a stable label,
busy semantics, and a reduced-motion-aware spinner. Loading and failure text
belongs to the picker and is associated with its input; refresh failures preserve
manual model entry and qualified voice presets.
Transcript and History placeholders share `EmptyState` with explicit pane and
compact variants. It owns the decorative icon tile, text rhythm, and optional
action spacing; callers retain state-specific copy, actions, and scroll ownership.
Auto margins center content only when space permits, keeping short panels scrollable.
Settings rows share `FieldCaption` for label, help, and inline
validation typography while retaining their own layout and transaction behavior.
`SaveIndicator` reserves an icon slot for pending, saved, and failed states;
`QuickSaveStatus` owns the corresponding live text and reserves one line even
when idle. Long recovery messages may expand and wrap. Indicators are decorative,
so each owning settings group announces the result once. Backend validation remains
the source of field errors; shared presentation does not introduce new validation.
`ConnectionDiagnostics` constrains its value column and wraps unbroken server
metadata inside the owning pane. `FeedbackDetails` keeps its title and optional
recovery action outside a keyboard-focusable message scroll region. Opening
focuses the message at its beginning; Escape restores focus to the trigger.
Search pickers share the `ui/combobox` content, option, and trigger components.
The content owns portal placement, viewport limits, one scrollable option region,
and an optional stationary action footer. Feature owners retain filtering,
metadata refresh, selection validation, and draft behavior. `ToggleGroup.Root`
uses its `segments` layout for framed equal-width options; its utility classes
must override the root and item defaults together. Shared component classes
must not silently lose geometry to registry utility classes. Component styles must
not declare a named CSS layer before the global Tailwind layer order; shared
picker presentation uses utility classes so mounting a picker cannot change the
cascade of unrelated headings or workbench layout.
`DownloadDetails` retains a native disclosure with a visible chevron.
History fact grids stack labels and values at narrow container widths; literal
chips wrap long IDs and checksums. `StatusBadge` distinguishes recording from
ordinary processing and errors, and owns its optional status dot. Every command
palette entry supplies a semantic icon, including the title bar's layout icons.

The shared switch uses a pill track and an inset circular thumb, retaining
Bits UI state, keyboard semantics, and visible focus indicators.

The renderer uses cool neutral light and charcoal surfaces with blue brand
accents. Page headings and field labels establish the reading order; descriptions
use readable secondary text. Primary actions use dedicated action tokens for
legible normal and hover states. The shared `Button` soft variant identifies
contextual actions, and `StatusBadge` pairs status text with semantic tones;
color alone never communicates readiness, failure, or selection. Compact
Connection and runtime rows keep identity, current state, and the next action
visible together. Their source provenance and ownership explanations remain in
contextual disclosures. Shortcut rows give the action and chord priority, show
capture feedback only when present, and keep allowed-key rules in a keyboard
accessible disclosure associated with the recording control. Shared visual
primitives do not own credentials, runtime lifecycles, or saved values.

Task-local connection creation opens the Connections page from the activity rail,
preserving the originating task and its configuration context. Home's
first-run Voice panel reuses quick connection/model controls. Connection changes
use their own save action; model/task drafts apply with Save. Switching, adding,
or editing a connection with a dirty runtime draft requires Save and continue,
Discard and continue, or Keep editing. Failed saves retain the draft and do not
continue the action. Guarded confirmation dialogs intercept dismissal before
Bits UI closes them; `onOpenChange` is a notification, not a dismissal veto.
The connection editor retains its draft during confirmation, and failed saves
keep their error/retry prompt available.

Connections page navigation remembers the originating feature and returns there
after task-local setup. The editor compares non-credential fields
with its opening snapshot; credential presence/removal is checked separately without
copying a password into that snapshot. Opening a form alone is not dirty, but still
reserves the draft against external-window updates. The editor retains the latest non-secret external snapshot and adopts it after drafts are discarded, so cancelling task-local creation does not leave an obsolete selection. Navigation confirms before
discarding changed connection fields and clears the transient key on exit.
Global Settings exposes only General, Shortcuts, Overlay, and Vocabulary.
Workflow options reuse `SettingsScreen` in inspector mode with a bounded section
list. Shared sections identify their scope and edit the existing shared preferences:
Vocabulary appears in Voice and files, and Overlay appears in Voice. Voice's local
Delivery view reuses only microphone transcript delivery controls from General,
excluding startup and appearance. Global General, Shortcuts, Overlay, and Vocabulary
remain available independently. These presentations introduce no separate draft,
save transaction, or persistent copy of shared settings.
Closing an inspector or navigating away resolves Save, Discard, or Keep
editing before changing its context; failed saves retain the draft and visible
inspector. Local option tabs do not replace the workflow's main content.

The content scroll offset resets on section/editor transitions and Settings
re-entry without changing draft ownership. Wide navigation retains section focus;
closing compact navigation returns focus to its title-bar toggle. Library connection
actions live in the persistent footer, with the submit button associated with
the connection form; task-originated setup reuses those actions in the same editor.
Connection saves remain independent of runtime settings saves.

Go configuration validation returns a `FieldError` containing bounded guidance
and a Settings JSON property path, never the rejected value or underlying error.
Wails' existing error marshalling delivers it in `RuntimeError.cause`; no extra
validation RPC or renderer copy of validation rules is needed. `SettingsEditor`
owns the presentation issue until editing, discard, successful save, or snapshot
adoption clears it. `SettingsScreen` maps known fields to sections/controls,
reveals the destination, and focuses the control or section heading. Shared
value inputs/rows associate guidance and invalid state through a settings-local
context. Unknown errors remain ordinary plain-message failures.

The home footer selects transcription or speech metadata according to the active
task and compares completed checks with the applied settings, not an unsaved
draft. Local TTS readiness is explicitly separate from metadata reachability;
no new automatic check or inference request is introduced by presentation.
`taskConnectionDetails` supplies the same applied task projection to the footer
and its popover, including the selected saved connection, model, result, pending
check, and stale state. Explicit footer checks use
`SettingsEditor.testAppliedConnection`: Voice delegates to its saved-connection
check, and file/speech checks pass the applied snapshot with an empty credential
draft. Disabled speech, missing selections, configuration recovery, pending
saves, and duplicate checks cannot issue requests through that action. Editing
opens the existing connection manager with the active saved ID and purpose;
it does not select another connection or modify task settings.

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

The schema persists feature-owned model-profile IDs with Generic
defaults. Cleanup's `preset` field holds its model-profile ID, not a second
source of truth. The cleanup descriptor service
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

`internal/diagnostics` owns the bounded error classifier and an immutable
workflow/generation context value. Dictation and file transcription attach that
value only when invoking the shared post-processor; it derives a child from its
injected logger without replacing its component identity or mutating a global
logger. Storage supplies `DiagnosticKind()` for wrapped configuration failures.
Connection metadata calls and asynchronous speech stages record explicit
terminal outcomes, distinguishing normal cancellation from failure. These are
diagnostic observations, not another state machine or proof of native teardown.
The production-tagged Wails default logger still discards output: there is no
new sink, persistence, export binding, or renderer logging facility.

## Configuration boundaries

`internal/settings/request_profiles.go` contains ordinary Go request-profile
capture and preview validation. These methods remain on the settings owner and
hold its save lock while resolving settings, managed endpoints, and credentials.
They do not add a Wails service or a second settings transaction.

Durable settings contain ordinary STT, VAD, shortcut, window, appearance, history, post-processing, and optional speech-playback configuration. STT, stored-file STT, post-processing, and TTS have independent validated request budgets; STT, post-processing, and TTS retain independent runtime models and selections. Selecting the same reusable connection explicitly shares its endpoint, HTTP policy, backend profile, and credential reference; selecting separate connections keeps those identities independent. Stored credentials remain in Windows Credential Manager or macOS Keychain; SQLite contains only their opaque references. Payload and retained-memory ceilings are implementation safety invariants rather than user-tunable settings.

`internal/tts` is deliberately on-demand and provider-neutral. History/file renderer calls identify a backend-retained entry/version or completed stored-file result rather than resending transcript text. Current Voice playback passes only the displayed dictation generation. The dictation owner rejects stale, active, cleared, and closed results, then supplies an immutable text snapshot through the injected `tts.TranscriptSources` collaboration boundary; history retention is not required. The first-class Text to speech workspace is the single deliberate exception: it accepts a bounded user-authored input (4,096 Unicode characters) and does not write that output-oriented content into transcript history. Synthesized bytes never become bridge results. The service captures one coherent TTS settings/credential profile, sends a bounded `/v1/audio/speech` WAV request, validates PCM before native playback, and emits only typed scalar status/progress. The ordinary connection service may discover speech model IDs with authenticated `GET /v1/models` metadata. Generic has no portable voice-list operation; qualified speech profiles add metadata-only voice discovery, and model profiles may restrict selectable voices. One in-memory playback session owns pause/resume/restart/stop/save/clear. Replay reads the retained PCM without another request; Save reconstructs a canonical PCM16 WAV and writes only to a native-dialog destination; Clear zeroes and releases the session. A new request replaces it, recording preempts and releases it before capture, native progress follows audible time rather than output-buffer submission, and shutdown cancels generation immediately and serializes native output teardown within the service wait budget described below.

`WorkbenchFrame` owns the main renderer's region geometry and `WorkbenchLayout`
owns its presentation preferences, independently of Go-owned settings. Shared
sidebar contributions open task-specific options alongside the main content; retained standalone
consumers may still use local splits and quick-setting popovers. Every edit
passes through `SettingsEditor` and the existing coherent settings transaction;
opening a control performs no inference request. Transcript and history
scrolling do not resize the workspace.

The main workspace, About, and Transcription details windows use the opaque product palette by default.
Dark surfaces use a neutral charcoal ladder (`#121212`, `#1b1b1b`,
`#242424`) with Freehand’s brand blue (`#4d8dff`). Light mode pairs neutral
white surfaces (`#f4f4f4`, `#ffffff`) with blue (`#326fe5`). Inputs use `#151515`
in dark mode for a gentle recess. Archivo headings distinguish page and
workspace titles; controls retain the native system typeface. The semantic CSS roles in `frontend/src/app.css`
cover cards, inputs, popovers, dialogs, and navigation. The workspace title bar,
auxiliary native captions, and startup backgrounds in `internal/app/window.go`
match the ground; overlay colour
constants in `internal/platform/overlay/overlay.go` match the panel and accent. Status
colours keep their separate meanings. Dark Mica applies one translucent charcoal
tint at `#app` plus translucent panels, while auxiliary native captions remain
under DWM control. Light-mode tokens remain independent.
Windows Mica is an explicit persisted opt-in applied when native windows are created, so changing it requires a process restart. The service reports the launch-time material separately from the editable preference; Svelte continues rendering the launch-time material until restart rather than making its surfaces translucent over solid native windows. Shell chrome uses the same material-aware layer roles, including the title/status bars, Settings navigation/action bar, and About action bar.

Main has platform-specific title-bar presentation. On Windows,
`mainWindowOptions` enables `Frameless`, `NonClientRegionSupport`, and
`WebView2CompositionHosting`, retaining DWM shadow and corner decorations.
The renderer supplies File/View/Help menus and minimize/maximize/close buttons
beside the layout controls. Separate, non-overlapping
`--wails-non-client-region` rectangles identify caption drag surfaces and each
caption button. Wails maps them to native hit testing, including Windows 11
Snap Layout hover over maximize, and forwards button input to the renderer's
window actions. A caption rectangle must never contain ordinary menu, command,
or layout controls: the runtime does not subtract a child's `none` region from
its parent's rectangle. Selective `--wails-draggable` regions and ordinary button
handlers retain basic interaction if composition hosting falls back.

On macOS, Main keeps `Frameless: false` and uses `MacTitleBarHiddenInset` with
`MacToolbarStyleUnifiedCompact` for full-size content with native traffic lights.
The renderer keeps a 44px header and reserves 96px on the left for those controls
and a gap before the Freehand mark. AppKit owns the button positions; toolbar
style selects native spacing without manually moving the buttons.
macOS retains its existing application menu and native
fullscreen behavior. `InvisibleTitleBarHeight` remains zero so native dragging
does not intercept the whole interactive header. Only intended drag surfaces use
`--wails-draggable: drag`; controls opt out. Wails handles macOS title-bar
double-click according to the user's system preference. About, Transcription
details, and standalone Process output retain their native frames.

The workspace close control calls `Window.Close()`, entering the same native
`WindowClosing` hook as Alt+F4 or the macOS red traffic light. The hook cancels
destruction and requests the existing configuration draft decision; successful
completion hides Main. It does not bypass the guard with `Window.Hide()` or
change authoritative tray Quit and service shutdown.

`internal/app` owns the cross-platform Wails windows: `main`, `about`,
`transcription-details`, and the opaque `tray-popover` panel. Global Settings,
Connections, and contextual configuration share Main's Session. Windows are created from Wails'
`ApplicationStarted` event after screen initialization. `internal/windowstate`
persists only main-window normal bounds relative to its display work area.
Display matching, work-area clamping, and missing-display fallback remain native
responsibilities. About and transcription details retain owner-relative centering
without persisted auxiliary placement. Main installs navigation listeners before
its readiness handshake, protects drafts on repeated reveals, and clears
transient editor state on hide. About uses `internal/windowing`;
`internal/history.Service` validates completed entry IDs and owns the selected ID
while `internal/app` owns the details handle. About and Transcription details
have no editable state and hide immediately from their native close action or
footer. Opening another history entry updates and focuses the same details
window. Details subscribes before fetching its selection, ignores superseded
responses, and refreshes after history actions, settings changes, and workflow
status events. Deleting, clearing, disabling, or evicting history makes details
unavailable; closing clears its selection. Go does not retain a separate details
snapshot or persist it. This native details path remains available from Recent.
The full History pane contributes the same details
component to the shell's right sidebar and omits the redundant native-window
action there.

The native status overlay is enabled by default but has an independent persisted opt-out plus curated layout, work-area anchor, phase visibility, motion, surface, visualizer, proportional-size, opacity, edge-distance, and glow settings. `internal/overlay` owns that feature lifecycle: the settings transaction supplies applied configuration, dictation supplies authoritative status, and the package translates both into a narrow `platform.OverlayOptions`/`platform.OverlayStatus` contract. Enabling creates one native surface and bounded level tap; disabling releases its native surface, timers and graphics resources instead of retaining a hidden renderer. Windows additionally owns its HWND/message-loop thread; macOS dispatches AppKit work to the main thread. New settings default to Capsule/minimal/envelope/bottom-center. Existing saved appearance remains authoritative.

The Win32 renderer queues all changes onto its locked message-loop thread, uses the foreground application's monitor work area captured at the start of a recording, and does not chase later focus changes. Windows Animation Effects and the saved Reduced policy control decorative frames, while the coordinator-owned silence deadline remains live. The overlay draws from the window's own palette rather than one of its own: a single ground, one accent hue, and the shared status colours, with each visible state kept distinguishable by glyph and stage rather than by colour alone. Windows contrast themes force a system palette, solid opaque surface, and no glow. Detailed may render only fixed Freehand labels and bounded operational values (normalized shortcut, elapsed time, checkpoint count); provider/user metadata never enter the status-label contract. Optional realtime captions are a separate bounded, transient text projection and never become a delivery source.

Settings can request a presentation-only native preview through a narrow Wails binding. Draft presentation changes update the same renderer, real dictation preempts preview, Settings close stops it, and the applied saved configuration is restored. Preview can temporarily create a surface while the applied feature is disabled, but stopping it destroys that surface. Overlay creation remains a degraded optional capability: native failure is logged without failing dictation or rolling back the saved preference.

Home presents the selected task and current result first. Its primary-sidebar
contribution exposes the active task's configuration; the shared shell owns
compact visibility and the bottom panel across empty, working, recovery, and
completed states.
`HistoryList` owns the bounded transcript scroll viewport; its outer frame and
drawer pass through the available height instead of nesting another scroll area.
This keeps mouse-wheel input over transcript text and row controls in the visible
scroller. Newest-entry arrival in a compact list resets that viewport to the top.
The full History pane gives its transcript-list sidebar an independent scroll
area beside the selected reader. At viewport widths of at least 1100px, the
optional secondary sidebar shows that run's details; narrower widths hide it
until the user opens the right-side overlay, preserving the wide-window
visibility preference. `HistoryDetails` uses an embedded
44px header and component-scoped heading IDs. Container queries stack field labels
and values and reflow checkpoints within narrow details panes without omitting
metadata. Below 700px, the title-bar primary-sidebar toggle exposes the list over
the main area; selecting an entry returns to the reading surface.
Task-sidebar rows open the relevant right-side options; direct switches use
immediate-save controls. Microphone and delivery controls appear only for
dictation. TTS shows its own connection and model/voice settings links.
Each quick update starts from backend-confirmed settings, restores only engine options
when a model changes, and calls the same transactional owner without credential mutation.
Main quick controls remain disabled while global Settings or a contextual inspector
owns the editable configuration.
The activity rail identifies the active pane with an accent marker and
`aria-current`. Selection and keyboard focus remain separate states.
Open quick-setting triggers use the shared accent wash and text roles; a hairline
separates them from result actions. History dates, counts, and status badges use
the interface typeface, with tabular figures for changing numeric metadata.
`SettingsEditor` owns shared, generation-fenced metadata for Voice, Audio file,
Cleanup, and Text to speech. First entry to model settings or a picker ensures
bounded metadata for the applied connection with empty renderer credential drafts;
Go resolves saved credentials. Concurrent entries within each renderer reuse
pending work and cached results. Committed settings changes synchronize renderers
and invalidate old completions. Ordinary effects do not
loop on failure; deliberate picker re-entry or explicit refresh can retry.
Discovery works before model selection or optional-feature enablement and never
selects a model/profile, enables a feature, or invokes inference. Voice has no
separate component-owned model cache.

For managed connections, the editor includes the selected runtime's lifecycle
and operation identity in each metadata cache key. Probes wait for the runtime
to report `running` and for pending Start/Restart/Stop actions to settle. Runtime
transitions invalidate earlier results, including late completions; readiness
permits one fresh check without creating a retry loop. The footer and connection
panel use that same state to show startup or shutdown before metadata outcomes.

`RuntimeModelPicker` supplies model search, manual IDs, discovery actions, and
saved/server/draft provenance across Voice, files, cleanup, and speech. Its profile
summary is descriptive; model IDs never select behavior. `QuickSaveStatus` reads
field-scoped pending, saved, and failed state from the existing editor owner.
The connection manager keeps metadata results by catalog ID in that same editor;
every confirmed settings adoption or credential-draft teardown invalidates the
cache and in-flight revisions, including credential-only changes invisible to the
renderer. No diagnostic result is persisted.

Global Settings navigation contains General, Shortcuts, Overlay, and Vocabulary
in the same visual and keyboard order. Workflow sections live in the contextual
inspector, and Connections has its own activity-rail page. Navigation
and content scroll independently. The section heading stays visible; the content
scroll padding follows its measured height so validation targets remain exposed.
Current results and playback controls stay accessible independently of recent history.
Optional history is a secondary disclosure. These disclosures do not alter retention.

`HomeScreen` composes task-specific `VoiceBar`, `FileBar`, and `TextToSpeech`
controls beside the shared result reader. Their docked actions remain available
while content scrolls and wrap within narrow editor panes. `WorkbenchFrame`
owns the surrounding sidebars and bottom panel; each feature retains its own
capture, upload, playback, and progress state.

The post-processing package owns a small renderer-visible profile catalog so names, descriptions, editability, and fixed protocol instructions stay aligned with request construction. Model IDs are never used to infer behavior. The custom profile persists a bounded user system instruction in the ordinary settings database, while its API key remains in the native credential store. The S1-mini profile keeps its exact system instruction in code and persists only its trained styling, structure, and context selections. Switching profiles preserves inactive profile values rather than destructively rewriting them.

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

The reusable CI workflow owns source selection and the final validation gate.
Release Please supplies the tag and expected version; CI resolves that tag once,
validates the same commit across all jobs, and packages it on native Windows and macOS runners. The release
publisher consumes only that run's validated artifacts before attestation and
publication. PR validation and Pages deployment share site build checks but not
deployment concurrency or write permissions. See [GitHub Actions](../github-actions/).

An active operation observes one coherent request profile. The transactional settings owner captures endpoint, model, headers, authentication mode, post-processing configuration, and both credentials under its save lock before microphone capture or stored-file upload begins. Renderer-safe settings reads use that same lock, so no window can combine an old saved configuration with credential or native state already changed by an in-progress save. Every failed native or credential stage attempts its own restoration plus all earlier restorations in reverse order; rollback failures remain inspectable by Go while their renderer-visible messages omit provider and credential-store details. The profile remains private to Go and fixed for the operation; settings edits save normally but affect only later operations. Segmented dictation therefore does not read a credential at its first checkpoint, and stored-file post-processing does not reread one after upload.

`postprocess.Processor` accepts the captured configuration and credential explicitly through `ProcessWithCredential`; it has no credential-store dependency or alternate store-reading entry point. Credential acquisition remains with the transactional settings/profile owner.

## Audio contract

`internal/filetranscription/audio_file.go` owns the private selected-file
capability, size/type checks, and identity revalidation before opening. The file
service retains selection and job lifetime. Completed Voice and file responses
share JSON and safe metadata decoding in `internal/inference/transcription_response.go`;
each caller retains its response limit, transport errors, and final-text checks.

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

Stored-audio transcription is a separate service-owned cancellable job so it can continue while the workspace is hidden. It consumes only the Go-owned native selection, revalidates its identity and metadata, and streams multipart bytes from the opened file. Go accumulates progressive transcript text once and emits typed generation/revision deltas across the Wails bridge; it does not republish the complete growing string for every chunk. Renderer stores reject stale or duplicate revisions, request the authoritative snapshot after a gap, and reconcile once with the terminal full result. Upload progress remains throttled and full snapshots are reserved for real phase boundaries or explicit recovery. If the fixed 8 MiB stored-file transcript ceiling is reached, the service stops accepting deltas, publishes an explicit partial-result state, and preserves already accepted text for manual copying rather than silently truncating it. The job is mutually exclusive with microphone dictation. Completed stored-file text is never auto-inserted because no safe target was captured when recording began; it can be explicitly copied and, when enabled, retained in the same bounded history.

Both interactive WebViews explicitly deny microphone, camera, geolocation, notification, and clipboard-read permission requests. Native Go owns microphone capture and clipboard writes, file drop remains disabled, and Wails simple renderer event emission remains disabled.

When a VAD-dependent microphone feature is enabled, the native callback writes into a fixed pool of 20 ms frames and never runs VAD or network work itself. One pinned `libfvad`/WebRTC detector feeds the stabilized live indicator, optional leading/trailing trim boundaries, the speech-armed automatic-stop policy, and optional checkpoint boundaries. Trimming retains bounded configurable speech padding. Automatic stop cannot arm until a configured amount of confirmed speech has accumulated, and resumed speech cancels its silence countdown. Recording control mode is captured explicitly at start: toggle recordings may arm automatic stop, while hold-to-talk recordings continue to use VAD feedback, trimming, and checkpoints but can end only on shortcut release, cancellation, interruption, or the hard duration limit.

When silence-aware splitting is enabled, segments target the configured duration and wait for their own sustained-silence threshold, with a separate 240-second capture ceiling when no pause arrives. Each completed segment is sent sequentially to the ordinary transcription endpoint with a fresh configured microphone-request budget while capture continues; only ordered final text is delivered. The total recording remains bounded, backpressure fails closed, and every PCM/WAV buffer is zeroed after use.

## Insertion contract

At recording start, capture a platform-tagged target: HWND/thread/process identity
on Windows, or frontmost PID/process-start and retained AX focused window on
macOS. The macOS app/window contract above allows field changes within that same
window. At insertion time:

- direct-input mode plus the same valid, focused target: use platform-bounded Unicode dispatches and revalidate the target before each dispatch; Windows uses adaptive dispatch sizes and macOS uses bounded Quartz chunks;
- manual-copy mode: retain the transcript and expose Copy without attempting insertion or touching the clipboard;
- target changed or invalid: retain the transcript and require explicit Copy;
- insertion failure: preserve the transcript for explicit copy without retrying simulated input blindly.

The native input adapter keeps UTF-16 surrogate pairs within one dispatch and checks cancellation before each dispatch. A partial Windows `SendInput` result is an ambiguous partial insertion. macOS Quartz posting has no application-delivery acknowledgement and can also be partial or ambiguous. Neither is retried automatically; only bounded delivery metadata may be logged. Diagnostics include UTF-16 unit count, dispatch count, strategy, duration, and a fixed failure stage; they never include transcript text or target identity.

Windows checks held modifiers before each dispatch and waits only within a
250 ms interval, preserving focus and cancellation checks throughout the wait.
A timeout requires explicit Copy. Explicit clipboard writes use an owned
message-only window and one pinned OS thread for the complete open/empty/set/close
transaction; the owner is destroyed after closing the clipboard. No window is
activated and automatic insertion never uses this clipboard path.
Clipboard admission and open retries share a 500 ms context budget. After the
clipboard is emptied, the synchronous write completes without an intervening
cancellation check, and failures report that clipboard contents may have changed.

Clipboard-paste mode is represented as a deferred policy boundary but cannot be selected or executed. It must remain fail-closed until complete multi-format clipboard capture, bounded paste synchronization, and conditional restoration that never overwrites newer user clipboard content are implemented.

## Optional transcript history

History is an opt-in recovery surface for finalized transcripts, not durable storage or a notes workspace. `history.Store` owns an in-memory oldest-first ring capped at 20 entries and 2 MiB across transcript text and bounded run details. Reaching either limit evicts the oldest entries first. A single entry is never allowed to grow past the total byte budget.

The History sidebar searches and filters only that retained ring. Final, raw,
and processed text and file base names supply its text matches; source filters
distinguish Voice and Audio file. Selection opens the existing read-only transcript
reader with copy, optional Listen, raw/cleaned comparison, and removal alongside
the same entry's full run details.
Search, filters, and selection do not alter retention or create a persistent index.
The sidebar's retention status, History settings shortcut, and Clear history action
use the existing settings and history owners; Clear history applies to the full
ring, including entries hidden by a filter.

Each entry can retain finalized raw and processed text, processing status, completion time, Unicode character count, delivery outcome, selected delivery mode, and bounded request metadata such as source, endpoint host, route, model, response mode, audio duration, segment timing, and file base name/size. When an STT or post-processing response supplies additional metadata, history may also retain a fixed, bounded subset: request/response identity, effective model, provider, finish reason, service tier, system fingerprint, detected language, server-reported audio duration, standard token/duration usage, provider-reported cost values, and llama.cpp-style timing metrics. These fields are optional rather than synthesized. Cost has no assumed currency, and the client never estimates tokens, duration, or price from transcript text. For checkpointed dictation, additive values are aggregated and explicit report counts show whether usage, cost, and performance covered every request; per-request IDs are omitted from the aggregate.

History never contains a URL path supplied by the user, target-window identity, credentials, headers, audio, provisional text, or an unbounded provider response object. The buffer enforces its 20-entry/2-MiB invariant after insertion and every mutation. If a processed copy alone makes an entry too large, it is discarded with visible `history_budget` raw-fallback metadata; if the bounded raw entry still cannot fit, the entry is removed. Removing an entry releases that transcript immediately; turning history off, choosing **Clear history**, or shutting down releases every retained entry. Nothing is written to disk or emitted through status/overlay events. Historical text reaches the clipboard only through an explicit copy action.

## Lifecycle

The update service checks release metadata after 30 seconds, with a two-minute
request timeout, then daily after success or after 15 minutes on failure. Wails
owns the update window, download, verification, staging, and explicit restart.
About derives interactive outcomes from Wails' state: successful staging does
not mean the running version is current. Shutdown rejects new checks, cancels
the service context, and waits at most two seconds across repeated shutdown calls.
This bound lets the native shutdown thread proceed if updater presentation is
waiting for it. A late worker cannot publish or replace service status. The
service cannot guarantee completion of Wails' native window teardown; that
requires packaged Windows and macOS acceptance.

- Wails single-instance ownership uses encrypted second-instance messages and the stable product identifier parsed from `build/config.yml`, shared deliberately with packaging and updates.
- First process owns hotkeys, tray, capture, the main workspace, and reusable About and Transcription details windows.
- Second process asks the first to reveal the main window and exits.
- Closing any native window hides that window; hiding the main workspace first resolves its active configuration draft.
- Appearance changes that affect native window creation are saved transactionally but applied only after tray Quit and relaunch.
- Tray Quit cancels active work, unregisters hooks/hotkeys, stops capture, and exits.
- Tray Quit clears the optional in-memory transcript ring before process exit.
- Automatic startup uses an app-owned HKCU entry on Windows and an ownership-checked per-user LaunchAgent referencing the exact bundle executable on macOS. Neither requires elevation.
- Automatic release checks are opt-out, quiet metadata reads scheduled by `internal/updates`; Wails owns GitHub release comparison, checksum verification, its review window, download, executable staging, and restart. The service owns a fixed initial delay and daily interval, exercised in virtual time without test-only constructor options. It stops polling and rejects new checks during shutdown.
- Services that own asynchronous work retain a child of Wails' application context themselves. Live `StopRecording` owns only the serialized native capture-stop transition; it then submits exactly one generation-scoped completion to the dictation service's single managed worker, which owns transcription, post-processing, history finalization, and insertion. Renderer, toggle, hold-release, duration-limit, and automatic-silence callers therefore share status events as their outcome contract instead of blocking a bridge or native callback on inference. Shutdown atomically stops admission and cancels the service root before waiting for native or workflow locks. Dictation and stored-file transcription each allow five seconds for the complete teardown; speech allows two seconds. These are per-service wait budgets, not a global process-exit guarantee. Wails closes shortcut capture before the dictation/audio owner. Native capture has a closed-state fence before and after device preparation so a late warmup cannot recreate resources.

### Shutdown and audio export ownership

Each workflow starts one tracked cleanup operation and fences new work before
waiting. Repeated shutdown calls join that same operation. Its deadline includes
state/control locks, native teardown, and worker drain. A deadline returns an
explicit error; it does not close a resource concurrently with a native call that
still owns it. Cleanup retains ownership until that call returns, and late work
cannot publish a result, insert a transcript, or start a subsequent playback step.
Restart separates native stop/reset (`Rewind`) from `Play`; the speech owner
rechecks cancellation between them, just as it does between `Load` and `Play`.
A failed rewind retains the previous generation and its monitor. After a
successful rewind, failed playback starts publish paused audio at its actual
position with Resume, Restart, Save, and Clear still available. The replacement
monitor observes resumed playback; neither failure synthesizes new audio or
forwards raw native errors to the renderer.

Windows hold-to-talk separates the native hook/message-loop completion from its
single callback consumer. Closing fences new edges and gives both completions
one two-second wait budget, so a blocked recorder callback cannot prevent Wails
from reaching feature shutdown. Late callbacks remain tracked and cannot replay
queued presses. Temporary shortcut capture also stops its native source on
cancellation and bounds Close to two seconds across source and callback waits.
Native callback parameters preserve Win32's signed 32-bit `nCode` and native
event-pointer types; negative codes forward without inspecting event data.

Speech export takes an independent canonical WAV snapshot under player control,
then releases control before disk I/O. The export worker owns and clears that
snapshot, so Stop, Clear, replacement, and native teardown can proceed. Once the
save location and generation are validated, export stays attached to that audio
snapshot even if playback changes. Shutdown cancels export; the writer checks
cancellation before opening the destination and between chunks. A save that
returns after shutdown cannot report success.

Go cannot forcibly interrupt arbitrary driver calls or every filesystem syscall.
A timed-out operation may retain its resources until it returns or the process
exits. An interrupted write may leave a partial file at the explicitly selected
destination. These service budgets do not bound unrelated Wails shutdown hooks,
window-placement persistence, native dialog dispatch, or all OS cleanup. Preserve
that distinction in acceptance reports; do not replace serialized cleanup with
concurrent frees or an unconditional process kill.

<span id="shelved-conversation-research"></span>

## Conversation scope

Conversation mode is outside the product boundary. On-demand speech generation
does not automatically send transcripts to chat or start a conversational turn.

## Optional post-STT normalization

The client provides an optional transcript-processing capability:

```text
STT -> raw transcript -> selected processing profile -> clean transcript -> insertion
```

Raw STT remains first-class and selectable. The processor is orchestrated by the client through a separately configured OpenAI-compatible `/chat/completions` endpoint; it is never hidden inside Speaches and is never bundled into the Windows executable. The default custom-instruction profile works with an ordinary compatible chat model. S1-mini by Superwhisper is a separate purpose-built profile whose styling, structure, context, and fixed request contract apply only when explicitly selected. Processing failures preserve raw text; owning-operation cancellation still prevents delivery. See the [S1-mini model guide](../../models/s1-mini/).

<span id="shelved-realtime-transcription-research"></span>

<span id="historical-speaches-realtime-research"></span>

## Realtime transport boundaries

The [qualified realtime adapters](#optional-realtime-dictation) use distinct NeMo
and vLLM protocols under unified Voice selection. Audio formats, session setup,
and finalization belong to the selected adapter; OpenAI compatibility alone does
not establish realtime support. Uploaded-file SSE streams response text after
upload, not live microphone audio. Outgoing audio queues are bounded; transport
failure discards provisional text rather than reconnecting and replaying audio.
Only authoritative final text may proceed to cleanup and focus-safe insertion.

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
are built. `config.Settings.VoiceTranscription.Language` and `config.Settings.Language`
preserve independent Voice/file selections. Model profiles further restrict
language support and defaults; model selection preserves task language.

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
use the modelsettings/sqlc save transaction.
Kokoro's `stream: false` is a backend wire adaptation, not a model preference.

### Ordered runtime publication and speech lifecycle

Settings mutations serialize through runtime publication, including recovery. The
transaction lock is released before callbacks so subscribers can capture settings;
a separate publication lock prevents a later commit from overtaking those callbacks.
Overlay snapshots refresh when recording shortcuts change.

Speech player operations and status publication share the service control boundary.
Stop, Clear, replacement, and Restart fence prior work with a new generation. Stale
completion cannot pause or unload the current player. Save Audio releases control
while the native dialog is open and rechecks the generation before writing. Shutdown
cancels work and allows two seconds for inference workers; late workers cannot use
the closed player or publish status.

### Current work and optional retention

The dictation owner exposes one current transcript in its bounded status DTO.
Copy and Clear carry a generation and reject stale commands; failed-insertion
pending-copy remains backend owned. Starting a recording, Clear, cancellation,
and shutdown release the current transcript. File transcription continues to own
its current result and source-file capability. Neither relies on history being on.
The speech renderer store owns an unsent composer draft for the WebView lifetime;
component unmount does not discard it, and session disposal clears it. No draft or
current result is persisted in SQLite or browser storage.

### Task preference ownership

`modelsettings.Select` and the renderer model editor share the same selection
contract. Language, cleanup instructions, trained S1-mini styling/structure/context,
and speaking speed survive model/connection switches; engine options and voice
remain scoped to a model. Remembered-model rows store only model-owned options,
not task settings. Generic cleanup instructions are not translated into trained
S1-mini controls.

### Shared vocabulary

`config.VocabularySettings` owns task-level terminology and Voice/file opt-ins. `modelprofile.VocabularyMode` resolves qualified hint fields; the renderer preview and request projection share Go validation. `settings.captureProfile` projects only into immutable workflow snapshots. Completed NeMo speech contexts use request-only transcription fields, excluded from JSON/model preferences. The vocabulary table and sqlc queries persist the shared settings in the same transaction. Remembered-model rows do not store the shared phrase list.

## File and speech workspace presentation

`HomeScreen` gives the speech composer the same flexible content area as transcript
results. Speech options use the shared `SpeechModelControls` in the contextual
inspector; retained standalone quick controls use the same model/voice pickers
and speed control. Quick saves use the editor’s serialized queue from its confirmed
snapshot; inspector settings keep draft/save semantics.
Voice discovery distinguishes applied quick settings from the editable draft
and rejects stale results. Model changes restore remembered voice/profile options
while preserving task-owned speed. Speed commits on release rather than saving
every intermediate slider value. `TextToSpeech` owns the draft editor. Before
configuration it shows compact setup guidance and a Configure speech action
inside the composer, with an editable textarea and disabled Speak action. It
does not reserve an empty audio strip for setup. Configured or active sessions
retain the playback area; `PlaybackBar` supports embedding in that area.
Its seek slider keeps a local preview during dragging and commits a generated
request only for user input. Native progress is projected onto valid slider steps,
including the exact audio endpoint, before rendering so slider normalization cannot
start a false drag. The full-width track sits below the playback controls. The generated
`tts.SeekRequest` carries the retained session generation and position. Go validates
the generation, capability, and bounds before the Windows adapter stops output
and aligns the PCM cursor to a whole frame through `Player.SeekTo(milliseconds)`.
This audio-position method is distinct from the standard `io.Seeker` file-offset contract. The service preserves playback intent,
rechecks cancellation before resuming, and replaces the progress monitor under
the same control lock used by recording preemption and shutdown. Seeking retains
the audio generation identity and full export snapshot; replacement audio invalidates
an unfinished drag. No synthesized bytes cross the bridge. The composer handles
Ctrl+Enter locally, preserves ordinary Enter, and guards duplicate submissions.
Generation shows an indeterminate message; playback time starts with decoded audio.
The speech draft remains editable throughout generation and playback. Editing and
clearing it never call the playback service. Speak is blocked during generation or
pending submission but can explicitly replace playing/paused audio. The Go service's
captured text argument remains independent of subsequent renderer draft edits.
Audio-file transport keeps its summary, response-mode option, and actions in stable
slots. These components consume existing backend status and capability flags;
window geometry and visual transitions do not alter inference or persistence.

`config.InspectVocabulary` supplies bounded, text-free line diagnostics alongside
existing admission results: source line numbers, duplicate references, and per-use
restriction messages. Counts use the same trimming and exact deduplication as
request projection. The UI selects the corresponding local textarea range; it
neither normalizes stored vocabulary nor invents adapter restrictions. The shared
Vocabulary editor groups task opt-ins with the phrase list, keeps support summaries
visible, and discloses provider explanation, line feedback, and tuning on demand.
Its debounced preview retains the open feedback panel while checking newer input,
disables stale line selection, and ignores completions for superseded drafts.
A local retry repeats only the Go preview; it neither saves settings nor invokes
inference.

`TranscriptText` supplies shared read-only textbox semantics for current results,
expanded history, and raw/cleaned comparisons. Its `transcriptReader` action owns
plain-text DOM nodes, scoped Select All, and keyboard scrolling of the nearest
scrollable ancestor. Native copy operates on the browser selection; it does not
call the whole-transcript copy binding. While a nonempty selection intersects the
text, the action keeps the displayed snapshot and only the latest pending update.
Selection clearing flushes that update; a changed recording/entry key resets the
selection. Stable result markup preserves the snapshot across live finalization.
No HTML is parsed, and teardown removes the document selection listener.

The `followTranscript` DOM action owns only result scrolling. New recording keys
reset following; an active selection suspends following, and scrolling away from the end pauses it until the reader returns
or chooses Jump to latest. Final text replacement preserves paused reading, and
teardown disconnects its resize observer, scroll listener, and scheduled frame.
The conditional Jump to latest row occupies normal layout space outside the
scroll viewport, including in compact windows; transcript padding is not a
substitute for keeping controls out of the reading area.

`CaptureClock` tracks the last valid recording start within a dictation generation.
The transport samples it only during capture and once when capture ends, then
freezes it through transcription, cleanup, or failure. Go's serialized zero time
(`0001-01-01T00:00:00Z`), invalid timestamps, and missing timestamps cannot become
elapsed durations. Idle and new generations reset the display; a pane mounted
after capture without a known start does not invent a duration.

### Speech settings preview

`tts.PreviewVoice` accepts a narrow `settings.TextToSpeechPreview` draft containing
connection ID and non-secret speech options. The settings owner checks the active
saved speech connection, overlays draft options onto a copied runtime profile,
validates them, and reads its credential under the same settings transaction lock.
The renderer cannot supply a preview endpoint, transport profile, or credential.
The captured profile feeds the existing playback admission, cancellation, timeout,
and native audio lifecycle. No preview options enter persistent model preferences
or active settings; ordinary composition/history/file playback uses saved settings.

## Speech family contracts

Parakeet TDT v3 and Cohere profiles use completed adapters, Voxtral uses the vLLM
realtime transport, and vLLM-Omni speech supports Qwen3-TTS CustomVoice options.
`modelprofile` owns languages, preset voices, and option admission; Svelte uses
that resolved metadata. Only Qwen realtime output passes through the Qwen header
parser. Voxtral finals remain ordinary text under the same stop/final authority.

Speech language and delivery instructions are value-only model options. The
baseline stores them with active speech settings and remembered models through
the same transaction. `settings.TextToSpeechPreview` accepts the same options as
the ordinary request and validates before reading credentials. Previews remain
unsaved; running jobs retain their immutable settings and credential snapshot.
The speech adapter explicitly requests buffered WAV from Kokoro-FastAPI and
vLLM-Omni, and NeMo. Qwen CustomVoice never submits reference audio or
uploaded-voice tasks.

MagpieTTS Multilingual 357M is qualified against NeMo v0.1.0 and the v2602
checkpoint. Its profile owns five speaker presets and nine language choices;
the server can further restrict languages to compiled frontends. Empty language
means server default, never automatic detection. Validation rejects unsupported
language, style, and speed before HTTP. Requests retain model/input/voice,
`response_format: "wav"`, and fixed `speed: 1`, adding `language` only when set.
No reference audio, voice cloning, sample-rate override, or streaming synthesis
is exposed. Voice metadata stays transient; active and remembered language/voice
options use the same settings transaction and immutable request snapshot as
other speech profiles.

The settings sidebar filters the global General, Shortcuts, Overlay, and Vocabulary
sections with presentation-only search terms. Workflow options, Connections, and
runtime management remain in their own contexts. Search and native `details`
disclosures do not own settings values:
the existing editor draft and save transaction remain authoritative. Audio and
Overlay keep common controls visible and group fine tuning in `SettingsDisclosure`.
Validation opens ancestor disclosures before focusing a rejected field. Shared
setting rows associate switch labels with their controls; sliders forward their
accessible labels to the focusable thumb.

Workflow pages use the same disclosures for request timeouts and metadata
checks. `RequestSettings` preserves visible warning/stale summaries while
keeping detailed diagnostics below everyday controls. Speech preview actions
are composed into the voice picker; they retain the draft-preview service path.
Voice's workflow-level validation opens its disclosures because the backend does
not currently identify individual voice fields. Disclosures never own option
values or change capability admission, persistence, or request snapshots.

Voice reuses `ModelProfilePicker` and `LanguagePicker` in draft and immediate
settings. Shared pickers present the provided model contracts and delegate edits
to their existing callbacks; they do not infer capabilities or own save policy.
`FieldHelp` presents supporting copy without moving the surrounding controls.
Restrictions and unavailable states stay inline rather than depending on help.

Shared button variants own control shape and medium label weight, including
menu and tooltip triggers. Workflow actions use 28px buttons with 16px icons;
compact transcript actions use 24px buttons with 12px icons. Primary fill marks
Record, initial file selection, Transcribe, Speak, and connection Save. Auxiliary
checks and changes use outline, Copy and setup guidance use soft accent, and Clear
uses ghost. `ButtonIcon` preserves the icon slot during pending states and respects
reduced motion; labels remain visible and callers own busy/disabled behavior.
Settings navigation uses the shared primary-sidebar overlay below 700px;
trailing row controls wrap when the content column needs more room.

Speech uses the same toolbar geometry and picker surfaces as transcription. Its
generate action reserves its width across ready and busy states. General settings
choices wrap to one column in narrow content areas, and selected toggle groups
use the shared blue accent while retaining their keyboard focus indicators.

Quick-setting and feedback popovers use the shared floating-surface slot. History
actions use 32px targets, while comparison labels distinguish plain-language
labels from monospace model IDs. The application status bar is 24px high and retains a visible open state.

The Settings pane fills the workspace viewport. Navigation and content own
independent scrolling; the document body has no viewport-height minimum in this
pane, preventing an outer scrollbar under zoom. Voice's contextual Transcription options group
model, profile, language and recognition controls in the shared settings card;
first-run setup controls continue to use the compact presentation and immediate saves.

### Host resource sampling

`internal/resources.Service` owns the status bar's aggregate CPU and physical
memory and GPU snapshots. `internal/app` registers it as a narrow read-only Wails boundary.
It samples native counters on demand, serializes calls, caches for one second,
and rejects reads outside its application lifecycle. There are no worker
goroutines, subprocesses, model probes, process metadata reads, persisted
metrics, or new permissions. Runtime supervision is independent of this reader.

Windows uses `GetSystemTimes` and `GlobalMemoryStatusEx`; CPU is unavailable on
machines with more than 64 logical processors because that timing API reports
only the caller's processor group. macOS uses Mach host CPU/VM statistics and
`hw.memsize`, releasing the host port on each read. Its available-memory estimate
includes free and inactive pages; speculative pages are already counted as free.
The RAM percentage is total minus estimated available memory, not macOS memory
pressure. CPU uses counter deltas across cores, with a new baseline after failed
reads, counter resets, or gaps longer than five seconds.

Windows GPU sampling uses English PDH GPU Engine and GPU Adapter Memory
counters with bounded buffers. Engine counters are aggregated across processes
and the busiest engine represents each adapter; instance strings containing
process IDs never leave Go. DXGI supplies adapter names and dedicated-memory
capacity, without creating a graphics device; software adapters are excluded
when identified. The PDH query is collected only on demand, rebuilt after a
long pause, and closed at shutdown. macOS reads optional `IOAccelerator`
`PerformanceStatistics` properties through IOKit, checking types and releasing
every object. `Device Utilization %` supplies activity; Apple GPU
`In use system memory` is labeled as unified RAM with no separate capacity.
These driver properties are not a guaranteed macOS schema: missing fields stay
unavailable. At most eight adapters cross the binding; no GPU contexts, private
IOReport APIs, external utilities, or permission prompts are used. Process
attribution and remote-host utilization remain outside this sampler.

The session's `ResourceState` serializes two-second polling while the main
workspace and document are visible and the window is not minimized. Visibility
events, teardown, and session disposal stop polling and clear renderer snapshots;
generation checks discard late responses. A stalled read expires displayed
metrics after six seconds. The status bar reserves metric width, and its popover
uses explicit measuring/unavailable states. Amber at 90% is a latest-sample usage
hint, not a contention diagnosis or runtime admission rule.
