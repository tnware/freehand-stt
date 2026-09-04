# Agent instructions

## Mission

Build Freehand, a lightweight native desktop client for self-hosted and OpenAI-compatible speech infrastructure. The current product is Windows-first: it records speech, sends it to infrastructure the user chose, and safely inserts the result into the application that originally had focus.

The product is intentionally not a meeting recorder, notes workspace, bundled model runner, or account platform.

Freehand owns the desktop boundary: capture, local speech policy, native shortcuts and overlays, endpoint configuration, observable request state, recoverable processing, and safe delivery. Inference servers own models, accelerators, batching, and model lifecycle. Treat localhost, a private LAN server, a user-managed VPS, and a hosted compatible provider as equal deployment choices; never assume inference runs on the client machine.

## Current delivery boundary

The current alpha supports two first-class input workflows:

```text
hotkey -> microphone capture -> optional VAD checkpoints -> /v1/audio/transcriptions
        -> optional /v1/chat/completions cleanup -> focus-safe insertion -> optional history

native file selection -> /v1/audio/transcriptions -> optional streamed response
        -> optional /v1/chat/completions cleanup -> explicit copy -> optional history
```

Optional S1-mini by Superwhisper processing is implemented as a separate stage after raw STT. It is never part of Speaches and is never bundled into the executable. Follow `site/src/content/docs/docs/decisions/0001-s1-mini-post-processing.md` exactly; preserve raw mode, fall back durably to raw text, and do not invent untrained control values.

Realtime microphone transcription and conversation mode (STT -> LLM -> TTS) are shelved research, not active roadmap commitments. The existing pause-aware checkpoint flow is the default latency strategy. If product evidence revives realtime work, keep completed/file STT and realtime STT independently configurable, implement semantic delta/final events behind a versioned transport adapter, and follow `site/src/content/docs/docs/decisions/0002-realtime-transcription.md` before adding code.

## Non-negotiable safety rules

1. Model discovery and endpoint health checks are metadata-only. Use `/health` or `/v1/models`.
2. Never invoke, preload, iterate through, or smoke-test model inventories. The user's desktop GPU can hold only one Ollama model at a time; a prior multi-model probe crashed Windows.
3. Never paste a completed transcription into a different focused window. Capture the target HWND at recording start and fail closed if focus changed.
4. Store credentials in Windows Credential Manager. User-entered API keys may exist only as a bounded, transient renderer draft; never persist them in JSON, TOML, SQLite, logs, argv, events, or crash reports, and never return a stored credential to the renderer.
5. History is disabled by default. Do not retain audio. Delete temporary audio after each request, including failures and cancellation.
6. Do not report native Windows behavior from cross-compilation. Windows acceptance requires execution on Windows.

## Architecture

- Go owns runtime state, hotkeys, audio, network requests, credential access, focus-safe insertion, startup registration, native platform adapters, and service shutdown.
- Wails v3.0.0-beta.16 owns the interactive settings shell, renderer bindings, tray, and single-instance application lifecycle. Svelte 5 owns settings/status presentation. The passive focus-sensitive overlay remains native Win32.
- Keep domain code under `internal/<domain>` and keep root `main.go` as composition/lifecycle wiring.
- Keep shared workflow code free of accidental Windows dependencies. Windows remains the only supported runtime today, but future platforms should add native adapters for capture, hotkeys, credentials, windows, overlays, insertion, packaging, and updates rather than weakening the desktop contract to a lowest-common-denominator implementation.
- `internal/dictation` owns the live recording state machine. `internal/history` owns transcript retention, and `internal/settings` owns coherent settings/credential snapshots. Platform callbacks and HTTP completions report into their owning feature; they do not mutate UI or insertion state independently.
- OpenAI compatibility is represented as separate STT, post-processing/chat, realtime, and on-demand TTS capabilities. Do not overload one endpoint or credential setting to mean all of them. TTS remains explicit and dormant when disabled. History/file playback selects transcript text through backend-owned capabilities; the first-class TTS composer accepts only bounded user-authored text. Synthesized audio never crosses Wails.

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

## Windows interaction requirements

- Toggle shortcuts may use `RegisterHotKey`.
- True hold-to-talk requires key-down and key-up semantics; do not claim it from `RegisterHotKey` alone.
- Use shared-mode audio capture, handle default-device changes and removal, and normalize capture to mono signed 16-bit PCM WAV.
- Preserve Unicode text.
- Clipboard insertion must not destroy unrelated clipboard state or paste into the wrong HWND.
- Use Wails single-instance ownership with encrypted second-instance messages. A second launch should reveal settings rather than starting another recorder.
- Tray Quit is the authoritative shutdown path; closing settings hides the window.

## Frontend rules

- Svelte 5 runes only: `$state`, `$derived`, `$effect`, `$props`, `$bindable`.
- Keep components thin and put testable frontend state under `frontend/src/lib`.
- Run the Svelte autofixer on every edited `.svelte` file, then scoped formatting and `npm run check`.
- A user-entered API key may exist only in the password input draft. Clear it after a successful save, when settings are left or hidden, and on teardown. Never expose stored credential values to the renderer.

## Work-item and documentation discipline

- One GitHub issue should describe one independently reviewable outcome. Record the observed problem, user impact, relevant invariants, acceptance criteria, validation required, and explicit non-goals.
- Use `priority::P1` only for correctness, data/credential exposure, unsafe insertion, shutdown reliability, or release blockers. Use P2 for meaningful reliability, security hardening, and maintainability; use P3 for deferred capability or polish.
- Keep issue status honest: `ready` means unblocked and sufficiently specified, `validation` means implemented but awaiting native or release evidence, `blocked` must name the actual dependency, and `deferred` means intentionally postponed rather than blocked.
- Keep documentation audiences separate. `README.md` is the concise product and repository front door; task-oriented install, setup, workflow, privacy, and troubleshooting material belongs in the public user guide; architecture, tests, CI, releases, logging contracts, ADRs, and native acceptance belong in contributor or maintainer sections. Do not turn the README or a user guide into a maintainer notebook.
- Each feature or fix pull request updates the affected durable documentation in the same branch: the user guide for user-visible behavior, `site/src/content/docs/docs/development/architecture.md` for ownership/contracts, and `site/src/content/docs/docs/development/testing.md` or the native checklist for acceptance. Track delivery state in the relevant issue or pull request rather than maintaining a second checklist in the documentation site. Update `README.md` only when the product summary, requirements, principal capabilities, safety promises, or public project status changes.
- ADRs preserve decisions and history. Do not silently rewrite an accepted decision; add a superseding note or a new ADR when ownership or protocol direction changes. Correct plainly stale toolchain facts and mark historical caveats as historical.
- Before opening a pull request, reconcile its linked issue and documentation against what actually shipped. Do not leave completed work checked as future work or describe implemented features as post-MVP.
- Keep release identity, human-readable version, Windows resource version, installer version, and About metadata derived from one documented source rather than copied independently.
- Judge proposed capabilities against the accepted remote-first direction in `site/src/content/docs/docs/decisions/0005-remote-first-product-direction.md`. Prefer deeper interoperability, setup clarity, diagnostics, reliability, and native delivery over bundled inference, provider-specific feature accumulation, or workspace expansion.

## Validation

Before publishing:

- `gofmt` all edited Go files.
- Run focused Go tests and `go test ./...` when practical.
- Generate Wails bindings with the pinned CLI.
- Run frontend build and `npm run check`.
- Compile the Windows executable.
- Run `git diff --check`.
- Record the native Windows build separately from real interactive Windows runtime acceptance.

Do not add automatic inference tests to CI.
