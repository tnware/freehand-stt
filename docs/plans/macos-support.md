# macOS Native Support Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Deliver a locally runnable Freehand macOS build covering dictation, file transcription, speech playback, shortcuts, native delivery, overlay, credentials and lifecycle, without publishing before operator acceptance.

**Architecture:** Keep the existing Go feature owners and Wails/Svelte shell. Share miniaudio resource/PCM logic while selecting CoreAudio on Darwin; use narrow Cocoa/ApplicationServices/Security adapters for platform behavior. Windows identity and release behavior remain supported; native evidence is recorded separately from automated tests.

**Tech Stack:** Go 1.27, Wails v3.0.0-beta.16, malgo v0.11.26, Svelte 5, Objective-C via cgo, SQLite/sqlc/goose.

## Constraints and baseline

- Local branch `feat/macos-support`, baseline `60ee2a9`; no pushes, PRs, releases or live-server model inventory invocations.
- Initial `go test ./...` failed because frontend assets were not built. Generating bindings and building the real frontend resolved it: full Go baseline passes on Darwin arm64.
- Existing Darwin adapters are unavailable stubs. Windows-only assumptions occur in target identity, shortcut names, credential descriptions, startup, tray icons, release matching and bundle metadata.
- Pinned Wails CLI must itself be built with Go 1.27; installing with toolchain auto selected Go 1.26 and produced binding-parser warnings. Set `GOTOOLCHAIN=go1.27.0` explicitly when installing.
- macOS deployment target must be consistent with Go's supported minimum (13.0) across cgo and bundle metadata, not the template's 11/12 mix.
- Preserve no automatic focus restoration, bounded resources, no secrets in argv/logs/storage, raw-text fallback and explicit copy recovery.
- Native permission prompts require operator consent. No automatic TCC manipulation. Tests requiring input focus, audio or keychain writes are opt-in and use disposable fixtures.

## Task 1: Shared native audio and microphone permissions

**Files:** rename `internal/platform/audio_windows.go` and `playback_windows.go` to shared native files with Windows/Darwin tags; split backend selection into OS files; port corresponding deterministic audio/playback tests. Create Darwin microphone permission adapter and opt-in acceptance tests.

1. Add failing Darwin tests for cancellation/closed capture and CoreAudio selection; run focused tests, preserving actual RED evidence.
2. Move existing bounded capture/playback implementation, do not duplicate it. Select WASAPI on Windows and CoreAudio on macOS.
3. Preserve explicit device IDs, default-device policy, stop/cancel failure cleanup, streaming sinks, level taps and seek/playback semantics.
4. Background Prepare must not prompt or capture when microphone authorization is absent. Explicit recording can request permission with a cancellable bounded wait; denied/restricted access yields actionable errors.
5. Run deterministic tests and race tests. Native microphone/playback acceptance remains opt-in and records actual execution separately.

## Task 2: Native target identity, Unicode delivery and copy

**Files:** `internal/insertion/policy.go`, `policy_test.go`; create `internal/platform/input_darwin.go`, associated `.m/.h` and tests.

1. Test that a Darwin target is an explicitly represented opaque identity, not fabricated HWND/thread values. Existing Windows equality/validity behavior remains unchanged.
2. Capture PID/process-start identity plus retained AX focused window and element under a bounded native owner. Compare CFEqual identities and current frontmost application before delivery.
3. Never activate another application. Reject own-process, missing, stale, secure or changed targets; preserve copy-required recovery.
4. Prefer Unicode Quartz delivery without clipboard mutation; validate before each bounded chunk, handle surrogate pairs and modifier release without automatic retries after ambiguous delivery.
5. Explicit Copy uses NSPasteboard and bounded UTF-8 input, no shell argv. Release retained target state on replacement and shutdown.
6. Run pure identity tests and opt-in native tests with a disposable text target; no arbitrary user-app typing.

## Task 3: Hold-to-talk and native shortcut capture

**Files:** new Darwin keyboard/hold/capture `.go/.m/.h` files and tests; `internal/hotkey` policy as required; neutral shortcut error wording in `internal/shortcut/controller.go`.

1. Test Darwin keycode/modifier mapping, left/right modifier state, repeat suppression, modifier-only edges, injected-event exclusion, overflow and forced cancellation.
2. Use a narrowly owned event tap/CFRunLoop with bounded queues and deterministic close. Do not run recorder/network/UI from event callback.
3. Hook availability must reflect actual Accessibility/Input Monitoring authorization. Start with empty hold configuration succeeds without prompting; disabled hold does not block toggle settings.
4. Shortcut capture is cancellable, temporary, suppresses only its scoped input and returns existing reducer DTOs. Global toggle/show registration remains Wails-owned unless a demonstrated pinned defect requires replacement.
5. Align command/control aliases and supported keys with actual macOS registration, preserving Windows policy and saved settings.
6. Run focused/race tests and separately record actual shortcut behavior.

## Task 4: Passive Cocoa overlay

**Files:** create `internal/platform/overlay_darwin.go`, `.m/.h`, tests. Reuse existing overlay presentation helpers rather than owning workflow state.

1. Test native presentation projection for all four layouts, anchors, surfaces, visualizers, bounded captions and countdown; establish RED before implementation.
2. Create nonactivating, click-through NSPanel, excluded from app switcher; all Cocoa operations on main thread, never key/main, no focus calls.
3. Coalesce updates with a single bounded owner/timer, freeze operation monitor placement, stop updates when hidden and release resources on close.
4. Honor scale/opacity/edge offset, system reduced motion/contrast and transient caption policy. No raw content logging.
5. Verify first show/update/hide does not change target using native harness after integration.

## Task 5: Keychain, autostart and package metadata

**Files:** credential platform split and tests; Darwin Startup adapter/tests; `build/darwin`, `build/scripts/releaseinfo` and tests, relevant Taskfiles.

1. Test bounded keyring operations and status mapping. Use Security.framework directly, never `security` command (secrets in argv).
2. Preserve credential Store API, service/account names and ErrNotFound contract; no cross-platform storage migration or plaintext fallback.
3. Implement per-user login startup referencing the exact `.app` executable and `--startup`, securely written/validated with app-owned identity; do not overwrite unrelated registration.
4. Generate Darwin version/identity from existing release source; valid numeric CFBundleVersion, microphone usage and minimum macOS version. Dev and production bundle identities must be deliberate.
5. Package/sign local `.app`, reproducible archive and optional DMG. Never claim notarization or publish unsigned release artifacts as accepted.
6. Test generation, startup plans against temporary dirs, Keychain using opt-in disposable account only.

## Task 6: Shell and renderer integration (parent owner)

**Files:** `internal/app`, `internal/settings`, `internal/buildinfo`, `internal/config`, root embedded tray assets; frontend platform presentation helper and affected settings/about/readiness components.

1. Test platform capability metadata and Mac shell options; use generated binding types, not renderer UA as native capability authority.
2. Preserve native titlebar, add platform-appropriate app/edit menu, close-to-hide and Dock reopen behavior, Wails encrypted single-instance lifecycle. Mac tray uses template PNG rather than Windows ICO.
3. Gate Windows-only Mica while keeping appearance setting coherent on macOS; show macOS Keychain/login terminology and Command/Option shortcut labels.
4. Provide explicit native permission status/recovery in settings; denial must not make file transcription/TTS unusable.
5. Select Mac update archives correctly or explicitly disable unsupported update application in local builds; never look for `.exe` on Mac.
6. Run Svelte autofixer on touched files, formatting, check, Vitest, browser fixtures and production build.

## Task 7: Cutover, review and native acceptance

- Exclude all unavailable stubs from Darwin only when corresponding real implementations exist. No silent no-op macOS features.
- Build with consistent Go/cgo deployment target and regenerate bindings using pinned CLI after APIs settle.
- Run `go test ./...`, focused race suites, frontend check/tests/build, packaging generation tests, `git diff --check`.
- Review spec completeness first, then code quality/security/resource bounds; fix important findings before handoff.
- Build and launch local `.app`. Verify real process/window/tray startup, close/reopen, second launch, settings persistence and shutdown.
- Native tests: microphone authorization/denial, recording/stop/cancel, capture PCM format, output playback/seek, toggles/hold/capture, Unicode insertion, focus-change fallback, clipboard preservation, passive overlay show/update/hide.
- Backend E2E uses only operator-selected endpoint/model. If endpoint or permissions are unavailable, report the exact unverified acceptance rather than substitute synthetic inference.
- Update a new superseding macOS ADR, contributor architecture/testing and user setup/troubleshooting docs. Preserve historical Windows ADRs.
- Keep incremental conventional commits local. Operator acceptance precedes any push or PR.

## Parallel ownership

Audio, native insertion, keyboard, overlay, and credential/package agents have disjoint file sets. Parent owns composition, UI, shared stub cutover, docs, final checks and all integration. No worker edits another worker's files, commits shared working tree state, invokes inference or changes user permissions. Coordinate through explicit interface reports.
