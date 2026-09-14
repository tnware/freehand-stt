# Agent instructions

## Mission

Build Freehand, a lightweight native desktop client for self-hosted and OpenAI-compatible speech infrastructure. The application has native Windows and macOS adapters: it records speech, sends it to infrastructure the user chose, and delivers the result only while the captured application/window remains valid and focused.

The product is intentionally not a meeting recorder, notes workspace, general-purpose model runner, or account platform. Models and runtime binaries are downloaded only through explicit setup, not bundled in the executable.

Freehand owns the desktop boundary: capture, local speech policy, native shortcuts and overlays, endpoint configuration, observable request state, recoverable processing, and safe delivery. Inference servers own execution, accelerators, and batching. Optional Windows and macOS managed runtimes have a bounded installation, model, and process lifecycle. Treat localhost, a private LAN server, a user-managed VPS, and a hosted compatible provider as equal deployment choices; never assume inference runs on the client machine.

## Current delivery boundary

The current alpha supports two first-class input workflows:

```text
hotkey -> microphone capture -> optional VAD checkpoints -> /v1/audio/transcriptions
        -> optional /v1/chat/completions cleanup -> focus-safe insertion -> optional history

native file selection -> /v1/audio/transcriptions -> optional streamed response
        -> optional /v1/chat/completions cleanup -> explicit copy -> optional history
```

Optional S1-mini by Superwhisper processing is implemented as a separate stage after raw STT. It is never part of Speaches and is never bundled into the executable. Preserve raw mode, fall back durably to raw text, and do not invent untrained control values.

Optional realtime microphone dictation is qualified for NeMo-Speech.cpp v0.1.0 with Nemotron 3.5 streaming, and vLLM v0.28.0 with the explicit Qwen3-ASR or Voxtral Mini Realtime profiles. Keep their distinct wire protocols separate. Completed and realtime microphone transcription use one Voice connection/model/profile with a capability-gated mode switch. Audio-file transcription remains independently configurable in manual mode. Managed mode supplies its selected local model to Voice and files without overwriting their saved manual selections. Partial text is presentation-only; authoritative finals use the existing cleanup and focus-safe delivery path. The pause-aware completed flow remains the manual default; the recommended managed Nemotron setup enables realtime explicitly. Conversation mode remains out of scope.

## Non-negotiable safety rules

1. Model discovery and endpoint health checks are metadata-only. Use `/health` or `/v1/models`; qualified speech profiles may also read `/v1/audio/voices` for voice discovery. The managed adapter may read NeMo's CLI model index and poll `/ready`; catalog browsing never downloads or loads models.
2. Never invoke, preload, iterate through, or smoke-test model inventories. Live inference checks use only an explicitly selected endpoint/model and must respect its resource limits.
3. Never paste a completed transcription into a different focused window. Capture the platform target at recording start: HWND/thread/process identity on Windows; NSWorkspace frontmost PID/process-start plus retained AX focused window on macOS. Revalidate before delivery and each chunk. On macOS, field changes within the same window deliver to the current field; editor metadata or element identity is not required. Preserve permission, Secure Input, modifier and cancellation guards, no activation, and explicit-only copy recovery. Secure Input does not guarantee detection of every custom secure field.
4. Store credentials in Windows Credential Manager or macOS Keychain through native adapters. User-entered API keys may exist only as a bounded, transient renderer draft; never persist them in JSON, TOML, SQLite, logs, argv, events, or crash reports, and never return a stored credential to the renderer.
5. History is disabled by default. Do not retain audio. Delete temporary audio after each request, including failures and cancellation.
6. Do not report native behavior from cross-compilation or browser fixtures. Windows and macOS acceptance each require execution on the relevant OS and architecture; macOS permission acceptance uses a packaged app.

## Architecture

- Keep the output viewer read-only, with limited color/progress controls and explicit Copy selection. Require sensitive-output consent and bounded memory-only reads; never forward terminal input or enable output-triggered clipboard, link, title, or file actions. Viewer controls do not own the runtime process.
- Use host-aware pinned binary recommendations and platform-specific recipes. Recommendations never change existing installations automatically. Selected-model GPU warm-up belongs to startup; health/catalog operations stay metadata-only. macOS recipes support NeMo on Apple Silicon (Metal) and Intel (CPU), and llama.cpp on macOS 13.3+ (Apple Silicon CPU/Metal, Intel CPU). Managed whisper.cpp remains Windows-only until upstream publishes a macOS server binary.
- List implemented providers directly: one configured installation and one running process tree per provider, with distinct providers allowed concurrently. Preserve existing IDs and expose explicit duplicate recovery rather than merging or deleting user installations. Managed llama.cpp qualifies S1-mini cleanup; managed whisper.cpp qualifies completed transcription. Windows uses pinned x64 CPU/CUDA binaries and revision/checksum-pinned catalog downloads. NeMo retains its hardware checks and model-manager acquisition.
- Runtime instances own installation, model loading, and process lifetime; saved Connections own independent task selection. Keep runtime inventory management separate but linked from Connections and quick settings. Realtime belongs to Voice; S1-mini belongs to cleanup model behavior, not a separate runtime adapter.
- Go owns runtime state, hotkeys, audio, network requests, credential access, focus-safe insertion, startup registration, native platform adapters, and service shutdown.
- Wails v3.0.0-beta.16 owns the interactive settings shell, renderer bindings, tray, and single-instance application lifecycle. Svelte 5 owns settings/status presentation. The passive focus-sensitive overlay is native Win32 on Windows and a nonactivating, click-through Cocoa NSPanel on macOS.
- Keep domain code under `internal/<domain>` and keep root `main.go` as composition/lifecycle wiring.
- `internal/managedruntime` owns optional Windows/macOS runtime installation, catalog, and process supervision. Pin official binaries and checksums; never build from source, alter PATH, or expose arbitrary flags. Keep files and model cache under Freehand app data, bind to `127.0.0.1`, and own subprocess trees through Windows Job Objects or the macOS lifetime-pipe supervisor and process groups. macOS listener admission uses libproc to verify the actual server PID; archive dylib aliases become verified regular copies, never filesystem links. Settings owns durable preferences and coherent endpoint projection. Never leak BYO keys/headers to the managed endpoint or silently fall back to a remote server after failure. Capture child output only in bounded private memory, not renderer events or persistent logs.
- Keep shared workflow code platform-neutral. Windows and macOS provide native adapters for capture, hotkeys, credentials, windows, overlays, insertion, packaging, and updates. Share bounded audio/session logic with WASAPI/CoreAudio backend selection; preserve platform-specific safety contracts rather than weakening them to a lowest-common-denominator implementation.
- `internal/dictation` owns the live recording state machine. `internal/history` owns transcript retention, and `internal/settings` owns coherent settings/credential snapshots. Platform callbacks and HTTP completions report into their owning feature; they do not mutate UI or insertion state independently.
- OpenAI compatibility is represented as separate STT, post-processing/chat, realtime, and on-demand TTS capabilities. Keep capability contracts distinct. Voice selects one completed/realtime transcription combination; audio files, cleanup, and speech retain independent selections and coherent credential snapshots. TTS remains explicit and dormant when disabled. History/file playback selects transcript text through backend-owned capabilities; the first-class TTS composer accepts only bounded user-authored text. Synthesized audio never crosses Wails.

## Model behavior contracts

- `internal/compatibility` owns server APIs; `internal/modelprofile` owns explicit
  model behavior and intersects its capabilities with the selected backend.
  Generic is a baseline, not proof of support by every deployed model. Qwen speech language and style are remembered model options and preview snapshots.
- `internal/modelsettings` owns the non-secret per-connection/use/model option
  subset. Task selection preserves language,
  cleanup intent, and speaking speed; remembered-model rows contain only their
  current model-owned subset, not historical task fields. Persist it with active settings through the existing settings transaction
  and sqlc queries; never store transport or credentials in model preferences.
- Shared vocabulary uses task-owned phrases and Voice/file opt-ins, with qualified adapter projection into immutable requests. Cleanup instructions and prose context remain separate.
- Model profiles belong to feature settings, independently of reusable server
  connections. Do not infer them from model IDs, URLs, or model inventories.
- Add specialized profiles only with a justified, qualified per-role contract,
  backend validation, request fixtures, language/option restrictions, and user
  documentation. UI controls and request admission must use the same resolved
  contract. Retain immutable in-flight settings and raw cleanup fallback.

## Wails binding and lifecycle rules

- Use generated Wails models, enums, and service functions directly in Svelte. Do not duplicate generated wire shapes or hide them behind assertion-heavy adapters.
- Keep bound methods narrow and task-specific. Prefer request DTOs over long positional parameter lists or passing all application settings to a focused operation.
- Validate every renderer-controlled value in Go. A native dialog opened by the renderer does not make a later path argument trusted; privileged file selection and file handles should ultimately be owned by Go.
- Use bindings for request/response operations and events for status, progress, and streaming updates. Keep event payloads bounded, named consistently, and unsubscribed on component teardown.
- `ServiceStartup` owns the Wails application context. Derive long-lived work from that context, add operation-specific cancellation/timeouts, and never start an untracked `context.Background()` goroutine that can outlive shutdown.
- `ServiceShutdown` must be deterministic and bounded. Stop accepting work, cancel children, wait only within a documented bound, then close native resources in ownership order.
- Settings, endpoint configuration, and credentials are one coherent runtime profile. Never allow an active job to combine a previously captured endpoint/model with a newly replaced credential.
- Organize Go by cohesive product capability, not by a generic service layer. A Wails service is only a renderer boundary inside its owning package; ordinary stores, recorders, clients, and platform adapters should not be registered merely to make them "services." Keep each mutable state machine under one feature owner, preserve the single transactional settings/profile owner, and inject narrow dependencies from `internal/app`.
- Leave `AllowSimpleEventEmit` disabled. Explicitly deny WebView permissions the renderer does not need, and treat any future remote content or HTML rendering as a security-boundary change.
- Derive runtime diagnostics from the one Wails default logger created by `internal/app`; inject component children rather than creating package loggers or printing directly. Keep Wails at `Info`: debug bridge tracing can serialize credential drafts, transcripts, and history binding payloads.
- Follow `site/src/content/docs/docs/safety/logging.md`: pair meaningful asynchronous starts with terminal outcomes, use `duration_ms` and bounded `error_kind` values, and never log raw errors, credentials, headers, transcript/audio content, model IDs, full paths, URL paths/query, or target-window identity.

## SQLite persistence contract

Use `freehand.db` with its distinct application identity. Start from defaults and an empty connection catalog; never read, import, convert, or delete alpha `settings.db`, `settings.json`, or legacy native credentials.

- Use `modernc.org/sqlite` with `database/sql`, embedded goose SQL migrations, and sqlc-generated application queries. Do not introduce an ORM, a second migration runner, or handwritten application query/scanning paths. Keep infrastructure SQL confined to the documented storage boundary.
- Pin driver and tool versions. Goose's version table is the sole migration authority. Goose and sqlc both read `internal/storage/schema/`, beginning with `00001_initial.sql`. Published migrations in this lineage are immutable and startup upgrades forward only; the alpha reset is not a general immutability bypass.
- Keep generated rows and database handles inside storage. Existing domain owners retain validation, save transaction coordination, immutable request snapshots, and native/credential recovery.
- Commit generated queries. The implementation must ship reproducible generation and CI checks for stale/untracked generated output, released migration immutability, and query/import boundaries. Test real SQLite migrations, transactions, recovery, and platform-specific behavior.
- Credentials remain in Windows Credential Manager or macOS Keychain; SQLite may store opaque references only. Adding SQLite does not authorize persistent transcript history or audio retention.


## Windows interaction requirements

- Toggle shortcuts may use `RegisterHotKey`.
- True hold-to-talk requires key-down and key-up semantics; do not claim it from `RegisterHotKey` alone.
- Use shared-mode audio capture, handle default-device changes and removal, and normalize capture to mono signed 16-bit PCM WAV.
- Preserve Unicode text.
- Clipboard insertion must not destroy unrelated clipboard state or paste into the wrong HWND.
- Use Wails single-instance ownership with encrypted second-instance messages. A second launch should reveal the main window rather than starting another recorder.
- Tray Quit is the authoritative shutdown path; main-window close hides the task workspace. One reusable Settings window owns configuration and its inline Connections editor; its close action resolves drafts before hiding. Never open a separate native Connection Manager.

## macOS interaction requirements

- Keep Toggle/Show on Wails Carbon shortcuts. Hold-to-talk and temporary shortcut capture use bounded Quartz event-tap/run-loop owners with real press/release semantics.
- Permission checks never prompt. Request access only through explicit actions; do not manipulate TCC. Microphone or keyboard denial must not disable file transcription or TTS.
- Keep AppKit work on the main thread and keyboard callbacks free of recorder, network and UI work. Lost input, Secure Input and overflow must cancel/reject rather than latch recording.
- Do not query editor roles, AXEnabled, protected-content metadata or AXValue settability for insertion admission. Use the app/window boundary above and preserve Unicode surrogate pairs.
- Use Security.framework for credentials and NSPasteboard only for explicit Copy; never pass credentials or transcript text through shell commands.
- Preserve close-to-hide, main-window Dock reopen and encrypted single-instance ownership. Quit must stop sources and feature services before releasing retained targets and storage in Wails shutdown hooks.
- Start at login owns only its per-user LaunchAgent and exact bundle executable. Keep release identity and macOS 13 deployment metadata coherent; ad-hoc signing is not notarization or native release acceptance.

## Frontend rules

- Svelte 5 runes only: `$state`, `$derived`, `$effect`, `$props`, `$bindable`.
- Keep components thin and put testable frontend state under `frontend/src/lib`.
- Run the Svelte autofixer on every edited `.svelte` file, then scoped formatting and `npm run check`.
- A user-entered API key may exist only in the password input draft. Clear it after a successful save, when settings are left or hidden, and on teardown. Never expose stored credential values to the renderer.

## Work-item and documentation discipline

- One GitHub issue should describe one independently reviewable outcome. Record the observed problem, user impact, relevant invariants, acceptance criteria, validation required, and explicit non-goals.
- Use `priority::P1` only for correctness, data/credential exposure, unsafe insertion, shutdown reliability, or release blockers. Use P2 for meaningful reliability, security hardening, and maintainability; use P3 for deferred capability or polish.
- Keep issue status honest: `ready` means unblocked and sufficiently specified, `validation` means implemented but awaiting native or release evidence, `blocked` must name the actual dependency, and `deferred` means intentionally postponed rather than blocked.
- Keep documentation audiences separate. `README.md` is the concise product and repository front door; task-oriented install, setup, workflow, privacy, and troubleshooting material belongs in the public user guide; architecture, tests, CI, releases, logging contracts, and native acceptance belong in contributor or maintainer sections. Do not turn the README or a user guide into a maintainer notebook.
- Each feature or fix pull request updates the affected durable documentation in the same branch: the user guide for user-visible behavior, `site/src/content/docs/docs/development/architecture.md` for ownership/contracts, and `site/src/content/docs/docs/development/testing.md` or the native checklist for acceptance. Track delivery state in the relevant issue or pull request rather than maintaining a second checklist in the documentation site. Update `README.md` only when the product summary, requirements, principal capabilities, safety promises, or public project status changes.

- Before opening a pull request, reconcile its linked issue and documentation against what actually shipped. Do not leave completed work checked as future work or describe implemented features as post-MVP.
- Keep release identity, human-readable version, Windows resource version, installer version, and About metadata derived from one documented source rather than copied independently.
- Prefer interoperability, setup clarity, diagnostics, reliability, and native delivery over general-purpose inference management, provider-specific flag accumulation, or workspace expansion.

## Validation

Before publishing:

- `gofmt` all edited Go files.
- Run focused Go tests and `go test ./...` when practical.
- Generate Wails bindings with the pinned CLI.
- Run frontend build and `npm run check`.
- Compile the Windows executable and package affected macOS architectures.
- Run `git diff --check`.
- Record builds and deterministic tests separately from interactive Windows/macOS runtime acceptance and signing/notarization checks.

Do not add automatic inference tests to CI.
