---
title: Testing contract
description: Deterministic, integration, and native acceptance responsibilities.
---

## CI workflow acceptance

Run the dependency-free selection/gate regressions and workflow wiring checks:

```sh
node --test .github/workflows/scripts/validation.test.mjs
go test ./build/scripts/cicontract
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 -shellcheck= -pyflakes=
```

The Node fixtures use temporary Git repositories to cover prose, site, shared
inputs, unknown paths, renames, unavailable baselines, and full release/manual
validation. Gate cases cover selected successes, intentional skips, failures,
cancellation, and missing outputs. Go fixtures check trigger/gate wiring,
same-run release artifacts, site reuse, and CI's frozen dependency installation.
Actionlint checks workflow syntax and expressions; the flags above disable its
optional external ShellCheck/Pyflakes integrations, not its own analysis.

Verify actual GitHub runs before enabling the required **Validation** check.
Exercise an application-changing PR and a documentation-only PR; confirm the
latter skips application jobs but still reports the gate and relevant site
result. Compare cold and warm cache timings separately. A task dry run confirms
`npm ci` selection but is not a package build. Release gating and same-run artifact
publication also require a real release run; local tests do not establish those
GitHub effects. See [GitHub Actions](../github-actions/) for the rollout contract.

## Product site and onboarding acceptance

Run `npm --prefix site run build` after site or documentation edits. Check the
rendered homepage and getting-started guide at desktop and narrow mobile widths:
task links must reach the intended section, text must remain readable without
horizontal scrolling, and headings/keyboard navigation must retain their order.
Also build with `CI=true` when checking GitHub Pages base-path links.

Keep the README, homepage metadata, and user prerequisites aligned: dictation
leads, file transcription needs no microphone or shortcut, and optional
text-to-speech needs no STT connection. Cleanup is subordinate to transcription.
Existing readiness fixtures cover task gating; a documentation build alone does
not establish native behavior or live-server compatibility. Keep generated
artwork descriptions faithful to the artwork rather than changing alt text to
advertise capabilities it does not depict.

## Compatibility profile acceptance

The published catalog fixture compares the website export with the app-owned
profile registry. Public directory rendering requires an editorial entry for
every profile. Review desktop/mobile navigation, matrix scrolling, and provider
guide links when changing the site.

Automated fixtures cover old-document Generic defaults, independent selection
round-trips, unavailable/wrong-operation rejection even for disabled features,
and preserved configuration recovery. Catalog/resolver agreement prevents a
planned entry from silently becoming a usable contract. Transport fixtures
check unchanged microphone/file multipart fields, the five-field speech body,
shared chat requests and truncated-cleanup rejection, typed final/empty-final
semantics, legacy segment EOF, and rejection of vLLM-shaped SSE. Unavailable
profiles fail before network work. Snapshot tests retain captured compatibility
selections, and file tests isolate streaming observations by profile.

For interactive Windows review, check that settings cards and saved-connection
cards share a single hairline border and subtle fill in light/dark themes with
solid and Mica materials, preserving internal row dividers. Check switch thumb
spacing in both sizes and states, disabled appearance, keyboard focus/Space,
and right-to-left placement. Check each Settings selector, keyboard access,
available-only choices and the explanation for an unavailable saved selection, save/reopen persistence, and
independent STT/processing/TTS choices. Confirm that S1-mini controls remain a
separate preset. Using only a model explicitly chosen by the operator, compare
Generic and the applicable dedicated profile for normal transcription, file
streaming, cleanup, and speech playback. Never invoke model inventories.
Builds and fixture tests do not establish this interactive or live-server acceptance.

## Task-local setup acceptance

Use an isolated configuration for fresh-install review, preserving the operator's live
settings and credential store. Start independently with Voice, Audio file, and Text to
speech. Add a connection from each picker, verify the purpose is preselected and cannot
be removed, and verify Save and use returns to the same task with the new selection.
File and speech setup must work without completing dictation setup. Choose or discover
a model; test metadata access without invoking model inventories.

Repeat from Transcription, Cleanup, and Text to speech Settings. Cancel an unchanged
form; discard a changed form; inject a failed save and retry. Verify selections and
unsaved task/model edits survive Keep editing, Save and continue failures do not advance,
and Discard and continue applies no discarded edits. Test keyboard entry, Escape,
focus return, window-hide credential cleanup, and a narrow viewport. Library creation
must still remain inactive. Windows service fixtures use temporary SQLite files and a
fake vault to check atomic activation, unsupported-purpose rejection, durable selection,
credential rollback, and unchanged in-flight snapshots.

Browser tests are an optional local aid for layout and interaction review, not a
required CI or packaging gate. Run `npm --prefix frontend run test:browser` after
generating bindings and installing frontend dependencies. This uses the standard
`@playwright/test` runner described in
[Svelte's testing guide](https://svelte.dev/docs/svelte/testing#End-to-end-tests-with-Playwright).
`frontend/playwright.config.ts` owns the browser, viewport, test server lifecycle,
and failure reports; `frontend/tests/browser/vite.config.ts` serves the fixture without
the native Wails bridge. Vitest excludes the browser specs.

The browser fixture mounts the actual Settings screen, connection picker, and dialogs
with the installed Bits UI library. Fake services reuse synthetic DTOs at the Wails
boundary. Tests edit fields through the UI; the fixture API only waits for a save to
start and completes that save with success or failure. Assertions inspect visible
forms, selected connections, and error/retry actions, without exposing stores or
inspecting event handlers. Separately named Escape and outside-click cases cover
Keep editing, discard, unchanged-form close, pending connection/settings saves,
failed-save retry, and idle prompt dismissal.

Local Windows runs use Edge; on other platforms run `npx playwright install chromium` from
`frontend`, or set `PLAYWRIGHT_CHANNEL` to an installed browser channel. Pass standard
runner options after `--`, for example `npm --prefix frontend run test:browser -- --ui`
or `--grep "Keep editing"`. Failure traces and screenshots go to `frontend/test-results`;
the HTML report is in `frontend/playwright-report`. No app settings, credentials,
or inference are accessed. Browser coverage does not establish native Wails acceptance.

## Settings navigation and validation acceptance

The settings browser suite covers normal/narrow content scrolling, keyboard
section changes, persistent connection actions, and an invalid Audio save made
from History. Controlled save failures carry the Go validation cause shape;
assertions cover the destination, focus, associated message, section marker,
retained unrelated edits, and retry. Go config/settings fixtures separately
verify both recording-limit modes, nested validation causes, safe error JSON,
and rejection before applied settings, credentials, native adapters, or events
can change. Frontend metadata fixtures distinguish independent STT/TTS outcomes
and prevent unsaved-draft checks from being attributed to applied settings.

Repeat the audit interactively on Windows before native acceptance: scroll long
pages and navigate by mouse/keyboard, use connection-editor Back/Cancel, save an
invalid Audio draft from another section, and correct it without losing other
edits. Check standard/narrow layouts, local errors and focus announcements,
Generic cleanup guidance, details disclosures, stable product/delivery names,
and the file-specific empty state. With independently configured endpoints,
verify capability labels and metadata outcomes without invoking model inventories.
These UI checks do not establish microphone, inference, insertion, or playback
acceptance.

## Diagnostic logging acceptance

Run the affected backend packages on Windows:

```sh
go test ./internal/diagnostics ./internal/storage ./internal/connection ./internal/postprocess ./internal/dictation ./internal/filetranscription ./internal/tts
```

Captured structured-record fixtures cover bounded storage/inference categories,
unknown-category fallback, isolated operation contexts, dictation/file cleanup
correlation, metadata admission/provider/cancellation outcomes, and cleanup
cancellation with raw text preserved. Speech fixtures check one generation
start/terminal pair, separate playback pairs, inference/decode/load/play
failures, explicit cancellation, replacement, Clear, shutdown, and restart
without another inference generation. File shutdown coverage also checks the
late worker's cancellation record without late history or renderer publication.
Fixtures use fake transports/players or local `httptest` servers; no model
inventory, inference server, microphone, or playback device is exercised.
Sentinel credentials, request text, model/voice identifiers, URLs, paths, native
device labels, and provider errors must not appear in captured records.

For interactive review, use `wails3 task dev` and inspect its terminal, not a
production-tagged release executable: the existing Wails production sink
discards output. With only the operator-selected model, observe a successful
operation, cancellation, a recoverable failure, and quit during active work.
Check workflow/generation correlation, timing and terminal levels against the
[logging contract](../../safety/logging/). No per-frame/progress spam or content
should appear. Terminal records do not prove process exit, audible playback, or
focus-safe delivery; retain the separate native acceptance below.

## Shutdown and cancellation acceptance

Run `go test -race ./internal/dictation ./internal/filetranscription ./internal/tts
./internal/platform ./internal/activity` on Windows. Controlled capture, player,
transport, and export boundaries exercise blocked teardown, cancellation before
lock acquisition, late completion, repeated shutdown, and independent export
ownership. Tests hold and release operations explicitly; deadline cases check the
returned timeout and then join cleanup after releasing the blocked call. The
Restart regression runs the actual speech service and Windows playback adapter
with a fake device whose native Stop is held through shutdown; releasing it must
not cause another device Start, and cleanup must close it once. Normal
CI uses fake devices and services and never performs inference.

For opt-in hardware acceptance on a Windows desktop, set
`$env:FREEHAND_NATIVE_AUDIO_ACCEPTANCE = "1"`, then run
`go test ./internal/platform -run '^TestNativeAudioShutdown$' -count=1 -v -timeout 20s`.
Remove the environment variable afterward. This exercises the default microphone
for 100 ms in memory and discards it, then closes real WASAPI output while playing
silence, while paused, and after an explicit rewind/play. It does not save audio or contact a server. A pass proves
those native device paths on that machine, not interactive tray/dialog behavior.

Finish with the [native checklist](../../safety/native-test-checklist/): Quit during
recording, upload, cleanup, generation, playback, and an open Save Audio dialog.
Check process exit, tray/hotkey removal, and absence of late insertion or playback.
For a slow export destination, verify playback controls stay responsive. A blocked
OS call may outlive its service wait deadline; keep that limitation distinct from
successful native cleanup, and inspect an interrupted export before using it.

## SQLite acceptance

Run `go run ./build/scripts/storage -check -base main` and
`go test ./internal/storage ./internal/settings ./build/scripts/storage`.
Storage fixtures use temporary files and real modernc/goose/sqlc paths, including
abrupt subprocess exits during import, writes, and upgrades. Check backups,
constraints, lock waits, read-only/full-disk failures, incompatible history, and
staged credential consistency. Run the storage/settings race tests on Windows;
CI also checks portable storage fixtures on Linux. Native service fixtures use a
fake vault/startup adapter and do not establish interactive Windows acceptance.

## Unit and deterministic integration tests

- Configuration validation, URL joining, header filtering, read-only legacy import with unknown-field rejection, and explicit database recovery without implicit replacement.
- OpenAI multipart transcription request shape against an in-process HTTP server.
- Inference redirect denial through the production constructor: 301/302/303/307/308 across STT, chat, completed/streamed file upload, speech generation, models, and health routes. Local fake transports assert exactly one request with canary credentials/payload, covering same-origin, cross-origin, and HTTPS-to-HTTP targets; failures expose neither Location nor peer body.
- Successful-response credential canaries across completed STT/chat/file, SSE and buffered-SSE terminal metadata, all retained request-ID headers, nested languages/usage type, and discovered model IDs. Check before truncation (including straddling and long credentials), preserve benign text/metrics and unauthenticated controls, and serialize the real history service DTO to prove safe publication. Preserve text-reflection rejection, including split-stream guard behavior. These fixtures never contact inference infrastructure or invoke models.
- Stored-file multipart streaming, exact declared body length, upload progress, current OpenAI SSE, older Speaches SSE, buffered-SSE JSON normalization, ordinary JSON fallback, cancellation, and provider-size rejection.
- Capability-owned request deadlines, cancellation-versus-timeout classification, malformed responses, bounded response handling, and explicit stored-file partial publication when the transcript safety ceiling is reached.
- Shared post-processing outcome tests distinguish success, partial/error output, unavailable processing, timeout, processor-local cancellation, and owning-operation cancellation overriding late success. History projection tests cover Unicode counts, metadata/timing, enabled/disabled/absent retention, deletion or disabling during processing, and history-budget fallback without changing delivery.
- Dictation and file tests exercise the real processor and inference client with a deterministic HTTP transport (no inference server): raw mode, success, unavailable processing, HTTP failure, empty output, and timeout preserve the correct text and history status. Dictation cancellation and replacement-generation tests reject late success; stored-file cancellation retains raw history without making cancelled text copyable. A gated finalization test cancels after the cleanup failure is resolved and proves the workflow rechecks cancellation before raw fallback. Automatic insertion remains dictation-only; file copying requires an explicit action even with history disabled or absent.
- Dictation state transitions and stale-generation rejection.
- Segmented-dictation fixtures serialize fake PCM writes against Stop, Cancel, and Close, matching the native sink contract. Gated overlap tests require sink closure after the admitted write, reject later writes, and close exactly once. Synthetic frame feeding waits for detector acknowledgement rather than bursting or retrying rejected frames; production queue limits and fail-closed backpressure remain unchanged. Run `go test -race -count=1 ./internal/dictation` for segmented, automatic-stop, and hold-release coverage using local fakes only.
- Activity admission tests read the real owners rather than a cached busy flag. The rule matrix covers dictation/file exclusion, speech-start rejection, recording preemption, and initial shortcut-capture checks. Gated tests compose real dictation/file/TTS services with fake native adapters and a cancellation-aware transport: competing starts, speech-control lock order, cancellation followed by a different feature, failed capture/selection/preemption, queued shutdown, late preemption, and file preparation crossing shutdown must not admit conflicting work or strand admission. Run `go test -race ./internal/activity` for this boundary; no model or native device is invoked.
- Direct Unicode insertion uses one dispatch through the ordinary-text threshold, larger zero-pause batches above it, complete target checks before every dispatch, cancellation between dispatches, surrogate-safe boundaries, and fail-closed partial-dispatch handling.
- Focus-change policy: same target inserts; changed target never inserts.
- Clipboard restore only when app-owned clipboard content remains current.
- Credential interface tests using a fake store; no test credential reaches logs or frontend DTOs.
- WAV header, channel conversion, sample conversion, and duration bounds.
- Per-recording device interruption delivery, intentional-stop suppression, stale-generation fencing, partial-audio cleanup, and retry after capture loss.
- Startup configuration semantics.
- Main/Settings window identity, startup visibility, singleton Settings reuse, section validation, close-to-hide behavior, light/dark solid native colors, explicit Mica opt-in, launch-time versus saved-material state, main-window placement restoration, missing-display fallback, and owner-relative auxiliary-window geometry.
- Tray presentation mapping for live dictation, post-processing, VAD/silence, checkpoint, stored-file upload/streaming, completion, failure, cancellation, copy recovery, last activity, and main-window visibility. Tests must prove arbitrary renderer messages, transcript text, and file identity cannot enter labels or tooltips.
- Overlay preference and appearance defaults, sparse-config compatibility, bounds validation, post-persistence runtime application, DPI/opacity/glow composition, idempotent live enable/disable/configure, current-state restoration, and shutdown fencing.
- Frontend feature-owner tests beside `editor`, `files`, `history`, `speech`, and `messages` cover committed-settings synchronization, active-draft preservation, serialized quick saves, transient credential cleanup, file delta/snapshot ordering, history mutation ordering, and playback commands. `session.svelte.test.ts` covers composition, namespaced status bindings, aggregate busy state, and presentation-only teardown. `session-events.test.ts` executes the shared main/Settings subscription wiring with a typed event source: subscribe-before-snapshot, accepted-transition history refresh, stale-event rejection, delta-gap recovery, independent window drafts, unsubscribe/remount, and partial-registration cleanup. These are deterministic renderer tests, not native Windows acceptance.
- The main view's transcript-list disclosure preference persists across launches, uses a safe default for missing or malformed values, and remains usable when WebView storage throws.
- Immediate rack saves start from the applied snapshot, mutate only allowed endpoint/model/S1 fields, send no credential changes, replace both snapshots only after backend confirmation, and preserve the applied state on failure. Settings transaction tests block persistence after native and credential mutation to prove renderer snapshots wait for a coherent commit, exercise reentrant post-commit publication, and inject shortcut, startup, STT credential, post-processing credential, persistence, and rollback failures.
- File-stream reliability fixtures count requests through the real service: parameter rejection, incompatible/malformed SSE, typed premature EOF, empty SSE, and buffered typed SSE never cause a second POST or cleanup of partial text. Explicit completed retry succeeds; original completed JSON uses one request. Partial failures remain copyable and retain a failed history outcome when enabled. Parser fixtures cover required final text, empty final replacement, missing/null fields, read/server errors, and legacy EOF compatibility.
- Generic microphone fixtures require explicit JSON negotiation. Chat fixtures reject reported length limits even when credential redaction removes the diagnostic finish reason, preserve bounded usage metadata, and tolerate missing/other finish reasons without guessing truncation. Real microphone/file workflow fixtures verify raw delivery/copy, no second cleanup request, output-limit notices, discarded partial cleanup, and failed-processing metadata with history enabled, disabled, or absent as applicable.
- Custom health-path fixtures verify origin, versioned, nested, and trailing-slash bases, plus one request only on both successful and failing health checks. Saved path semantics remain base-relative.
- Model metadata fixtures reject HTML, malformed JSON, missing/null/wrong-shaped inventories while preserving reachability; valid empty inventories and opaque health successes remain supported. Renderer fixtures distinguish health reachability from model inventory and block first-run completion on an HTTP 200 invalid model response.
- Metadata-only connection result mapping for credentials, DNS/network, TLS, HTTP status, malformed/oversized responses, bounded model inventory, repeat-action debouncing, and generated frontend enums.
- First-run readiness requires a successful metadata-only check, persists completion without credential drafts, blocks recording at the Go service boundary until complete, and allows an established user to dismiss one exact recovery condition while retaining the endpoint warning.
- Shortcut chords use one Go-owned action matrix for required/optional forms, modifier-only hold, unmodified F13-F24, F12 reservation, aliases, modifier ordering, duplicate detection, and normalized registration. Capture progress and structured rejections feed one tested Windows-facing keycap/spoken renderer without a parallel frontend grammar, and every terminal capture path restores the prior working bindings.
- Transcript-history count/byte eviction after both insertion and every mutation, raw-only fallback for oversized processed output, removal when bounded raw text cannot fit, oldest-entry mutation accounting, bounded segment and optional endpoint-response details, deep-copy isolation, opt-in gating, individual removal, disable/shutdown clearing, finalized-only retention, explicit copy, and renderer refresh ordering.
- Completed and streamed transcription usage parsing, chat usage/cost and llama.cpp timing parsing, malformed optional-metadata tolerance, and partial-report coverage when checkpoint responses are aggregated. These are local protocol-contract tests; they never discover or invoke an installed inference model.
- Stored-file streaming emits thousands of ordered typed generation/revision deltas without publishing a full status per chunk, preserves a recoverable authoritative snapshot and terminal result, rejects stale/duplicate frontend revisions, repairs gaps from Go, and reconciles both main and Settings listeners. The test contract keeps bridge payload growth linear in transcript size.
- Active-operation configuration coherence: changing or clearing STT and post-processing credentials cannot pair a previously captured endpoint/model with a newer credential.
- Wails-lifetime cancellation for connection and transcription work, tracked microphone preparation, closed-state capture fencing, late-publication suppression, and deterministic managed-worker tests proving that live stop returns before remote inference, cancellation reaches an active completion request, shutdown waits for the worker, and closed services reject new completion work.
- OpenAI-compatible speech request shape, bounded/safe response handling, WAV validation and buffer ownership, backend-owned transcript selection, bounded first-class composer input, one-session playback transitions, canonical native WAV save, explicit memory clear, disabled-state dormancy, recording preemption, and shutdown cleanup. Automated tests use fixtures and fake players; they never invoke an installed TTS model.
- Stored-file selection authority: renderer calls cannot bypass the native-selection capability to open an arbitrary supported-looking path.
- Focused generated binding contracts: connection probes carry only compatibility-profile/endpoint/model-discovery/auth/header/transient-credential values, unrelated shortcut/VAD/window/history/processing drafts cannot invalidate them, and settings save uses one request DTO rather than positional credential arguments.
- Captured structured-log records: meaningful operations pair starts with terminal outcomes; error fields use bounded categories; settings, file selection, endpoint, post-processing, segmented-transcription, and direct-input logs exclude credential drafts, headers, transcript text, model IDs, full paths, URL paths/query, and target identity.
- The configured Wails/application log level remains `Info`, so ordinary test and production configurations do not serialize bridge arguments or results.
- Release-source parsing, semantic-to-Windows version derivation, generated-asset drift detection, and immutable renderer-safe About metadata.
- Update-policy scheduling, disabled/development behavior, shutdown fencing, bounded renderer status, and exact platform-binary selection without network access.
- Canonical brand-source parsing, deterministic PNG/ICO generation, complete Windows ICO size directories, and byte-for-byte generated-asset drift detection.

## Build evidence

- Go tests on Linux for platform-neutral packages.
- Windows target compilation for Windows adapters.
- Wails binding generation with the exact `v3.0.0-beta.16` CLI.
- Frontend unit tests, Svelte check, and production frontend build.
- Packaged Windows executable creation.
- Per-user NSIS installer creation with SHA-256 sidecar files.

Cross-compilation proves source/build compatibility only.

## Native Windows acceptance

Tyler's first alpha test should verify:

1. The unsigned executable launches after the expected SmartScreen warning.
2. Exactly one tray instance exists. Its tooltip, status row, and bounded detail row track Recording, Transcribing, Cleaning, stored-file upload/streaming, cancellation, success, attention, failure, and idle last activity. Confirm no transcript, file name/path, model, endpoint, or raw error enters the tray. Show/Hide, Settings, About, active cancellation, available Copy transcript, and Quit must appear only in valid states; tray Quit remains authoritative. A tray action must never start recording or change the insertion target. Confirm the mark-only tray artwork switches between its light/dark treatments with the Windows theme and remains crisp at 100%, 125%, 150%, and 200% display scale. Confirm taskbar, Explorer, installer, uninstaller, Installed apps, and Start Menu surfaces retain the tiled application icon rather than the tray mark.
3. Settings save and reopen.
4. The API key persists through Windows Credential Manager without appearing in the config file.
5. Toggle recording works with the selected microphone.
6. Hold-to-talk starts on press and stops on release.
7. Dictation inserts Unicode text into Notepad.
8. Changing focus while transcription runs does not paste into the new application.
9. Cancellation produces no insertion.
10. Network failure preserves recoverable text state and leaves the microphone stopped.
11. Startup setting survives sign-out/sign-in.
12. Quit removes the tray icon and releases the hotkey.
13. With history off, a completed dictation leaves no history entry.
14. With history enabled, force copy-required, open History without changing focus during dictation, and explicitly copy the retained Unicode transcript.
15. Fill the visible history area and confirm entries scroll independently of the app shell. Expand a processed entry to read only its final transcript, enter and leave raw-versus-cleaned comparison, verify long sparse edits remain highlighted, verify wide rows compare side by side while narrow rows stack, and confirm a raw fallback remains visible in the metadata bar while collapsed. Collapse the entry from its sticky header, then remove it and confirm the other entries remain.
16. Clear history, turn history off, and quit/restart; each action leaves the history view empty.
17. From Server settings, test the displayed endpoint with both a stored credential and an unsaved bounded credential draft. Confirm status/timing stays beside Base URL, model inventory stays beside Model, selection makes no request, and DNS, TLS, HTTP, and offline failures remain actionable without exposing response bodies or key material.
18. With System default selected, change the Windows default input during recording and confirm WASAPI reroutes or the app fails promptly and can restart. With an explicit USB microphone selected, unplug it during recording; confirm partial audio is not transcribed, the state leaves Recording, the saved choice remains, Refresh shows it again after reconnection, and the next recording succeeds.
19. Choose a stored audio file and verify both **Stream transcript** on and off. Confirm the upload rail advances, streamed text appears progressively, completed mode appears only once, cancellation stops the request, copy returns the full text, optional history labels it as an audio file, and microphone recording is disabled while the file job is active. After selection, separately delete, replace, resize, and modify the file before Start; each must require reselection without uploading. A direct symbolic-link selection must be rejected.
20. Try a file above the endpoint or reverse-proxy limit and confirm the HTTP 413 message identifies the server upload limit. No automatic retry or client-side split should occur.
21. Install the per-user NSIS package without elevation. Confirm it appears in Installed apps and creates one Start Menu shortcut but no Desktop shortcut.
22. Start the app, rerun the installer, and confirm it asks for the tray process to be quit before continuing. Repeat from the uninstaller.
23. Upgrade in place and confirm settings, the Credential Manager API key, history opt-in, and startup preference remain intact. Confirm exactly one Installed apps entry and one Start Menu shortcut remain.
24. Uninstall and confirm the executable, install directory, Installed apps entry, and Start Menu shortcut are removed while user settings and Credential Manager entries remain available to a later reinstall.
25. Verify the executable and installer SHA-256 sidecars against their artifacts. For a signed release, verify both Authenticode signatures and timestamp chains on a clean Windows system.
26. Start microphone dictation and a stored-audio run, then change or clear the STT and post-processing endpoints and credentials before their requests complete. Confirm the save affects the next operation while the active non-segmented, pre-checkpoint segmented, and stored-file post-processing requests retain their complete start-of-run profiles.
27. Start and cancel stored-audio work from a slow, disconnected, removable, or network-backed file and then choose tray Quit. Confirm shutdown remains bounded, releases the hotkeys/tray/audio resources, and no late status or warmup work recreates a native resource.
28. Confirm the settings WebView cannot request microphone, camera, geolocation, notification, or clipboard-read permission. Inspect the generated service contract and confirm stored-audio selection is a zero-argument native-picker operation: no renderer call can supply an ungranted path, and status/events never contain the granted full path.
29. In both Windows light and dark modes, confirm the default app shell is opaque and uses the product palette. Enable Mica, save, and confirm the current window remains unchanged with a restart notice; after tray Quit and relaunch, confirm Mica is visible through the intended shell surfaces. Disable it and repeat the restart check back to opaque mode. On a Windows version without Mica support, confirm the opted-in fallback remains legible and the default remains opaque.
30. In the rack, commit STT and post-processing endpoint/model edits by blur or Enter, select discovered models directly, and switch between **Custom instruction** and **S1-mini by Superwhisper**. Confirm each successful change takes effect immediately and reports success in the main-screen message channel; invalid values preserve the prior applied configuration and report an actionable error. In full Processing settings, edit the custom instruction, exercise empty and multibyte-over-limit validation, restore the recommended instruction, save, restart, and confirm it persisted. Switch to S1-mini and confirm the custom editor disappears, the exact built-in instruction is read-only, and the effective control line changes with every styling step plus both structure and context radio choices. Switch back and confirm the custom instruction was preserved. Connection status must reset only when its endpoint/model changes and refresh only after an explicit metadata test.
31. Run the development app from a terminal and exercise recording start/stop/cancel, one VAD checkpoint, post-processing fallback, stored-file selection/transcription/cancel, both connection tests, settings save, and tray Quit. Confirm lifecycle records have stable components/correlation fields, use `duration_ms`, and produce one terminal outcome per start. Search the output for the real test credential, transcript phrases, model IDs, selected file name/path, endpoint path/query, custom headers, and target-window identity; none may appear. Confirm a failed title-bar action is visible in the app rather than only the WebView console.
32. In **Settings → Overlay**, preview all four layouts, six anchors, three surfaces, four recording visualizers, three visibility policies, and both motion policies before saving. Confirm the real native surface cycles through speech, silence, countdown, transcription, post-processing, delivery, copy-required, and failure; draft controls update the same HWND; closing/discarding restores applied settings; and starting a real dictation preempts preview. Turn the saved overlay off and confirm preview can temporarily create it but Stop/Settings close destroys it. For real dictation, confirm the target monitor is captured at recording start, placement stays inside that monitor's work area at 100%/150%/200% DPI and with taskbars on every edge, and focus changes do not move it mid-operation. Exercise minimum/default/maximum size, opacity, edge distance, and glow without entrance replay, extra taps, or GDI/thread leaks. Windows Animation Effects off and Reduced must stop decorative motion while the countdown remains live; a Windows contrast theme must force a readable opaque system palette. Detailed must show only fixed labels, the normalized shortcut, bounded elapsed time, and checkpoint count—never transcript, filename, provider, endpoint/model, prompt, credential, or raw error content. Across all cases verify unchanged focus/caret/target identity, click-through behavior, no taskbar/Alt+Tab entry, bounded shutdown, and ordinary dictation/insertion behavior.
33. Navigate the main, native Settings, native About, and native Transcription details windows plus every remaining in-window dialog using only Tab, Shift+Tab, Enter, Space, Escape, Alt+F4, and the settings sidebar's Arrow/Home/End keys. Confirm every Settings action reuses/focuses one native window at the requested section, section selection and focus move together, the rack is inert while that window is visible, clean close hides immediately, and dirty close routes through the discard confirmation without losing a cancelled draft. Reopen Settings and confirm it reloads the latest Go-owned values and no credential draft survived. Confirm every About action reuses/focuses one window and that native chrome, Alt+F4, and its footer all hide it. Transcription details opens independently without trapping focus in its source window; Tab can focus the details region for keyboard scrolling. With Narrator, confirm recording/file phases, settings saves, errors, and notices are announced once without reading transcript text unexpectedly. Turn Windows **Animation Effects** off while the app is running: WebView decoration and native-overlay entrance/morph/stage motion must stop, while the overlay's automatic-stop countdown continues to advance and all states remain visually distinct. Repeat in a Windows contrast theme and confirm keyboard focus remains visible.
34. Open About in a development build and a packaged build. Confirm its compact metadata matches `build/config.yml`, the executable's Details tab, and Installed apps; only the development build shows **Development**. Run `wails3 task common:check:release-info`, deliberately make one generated Windows version field stale, and confirm the check and package build fail until `wails3 task common:update:build-assets` repairs it.
35. Move and resize the main window on a non-primary display, open Settings, About, and Transcription details, and confirm each hidden auxiliary window opens centered over the main window without leaving that display's usable work area. Move an already-open auxiliary window and invoke it again; confirm it is focused without jumping. Hide and reopen it; confirm it returns relative to the main window rather than retaining independent placement. Choose tray Quit, relaunch, and confirm only the main window restores its normal size and screen-relative position. Then disconnect the saved display and relaunch; confirm the main window is fully visible and centered on the primary work area.
36. In direct-input mode, compare short, long, multiline, and non-ASCII transcripts in Notepad, a Chromium text field, VS Code or another editor, a terminal, and an Office-style rich-text target. Ordinary text should appear in one immediate update; long text should complete without visible fixed-delay stepping, truncation, or broken surrogate pairs. Change focus during a long insertion and confirm delivery stops before the next dispatch rather than redirecting its remainder. Confirm the terminal records only UTF-16 unit count, batch count, duration, strategy, and bounded failure stage—never text or target identity.
37. Resize the result/history divider by dragging and with the keyboard, quit through the tray, and relaunch. Confirm the split restores and both panes scroll independently. At the 560x560 minimum window size, verify Result/History view switching and that audio, transcription, cleanup, and delivery popovers stay within the window without moving the transcript. Exercise nested selectors, Escape focus return, pending saves, and failed-save recovery. Check both light/dark palettes and opaque/Mica modes. Clearing WebView site data may reset pane widths but must not alter Go-owned settings or transcript history.
38. In an isolated user-data directory, import a valid old JSON file, save settings, and reopen. Confirm only SQLite changes and the legacy file stays intact. Unknown legacy fields must block initial import. Exercise corrupt/newer SQLite, locked files, and uncertain save recovery: both windows must pause new work and ordinary saves; Retry reloads committed state, and only explicit Reset archives and replaces the database. Restore an upgrade backup with Freehand closed. Verify credential references, setup review, native shortcuts/startup, and no transcript/audio persistence; never reset personal data as a fixture. See [the storage guide](../storage/).
39. Configure a dedicated local or remote `/v1/audio/speech` endpoint under **Speech playback**. Press **Test**, confirm the authenticated `GET /v1/models` result populates the model picker, enter the provider's voice ID, and save. Then press **Preview**. Verify the fixed preview phrase plays, pause/resume preserve progress, restart begins at zero, and stop releases the session.
40. Enable History, create raw-only and successfully cleaned entries, and verify Listen reads the selected final version. Complete a stored-audio transcript with History off and verify its result can still be listened to. Start a toggle or hold recording during playback and confirm playback stops before capture begins without transcript/history mutation.
41. Under **Server**, **Processing**, and **Speech playback**, confirm the saved microphone/checkpoint, stored-audio, cleanup, and speech-generation budgets reload exactly and the fixed safety ceilings remain visible but not editable. Against a deliberately slow endpoint, set each budget low and confirm the affected phase reports a timeout, logs bounded `error_kind=timeout`, and records the budget in opt-in History details. A cleanup timeout must still insert or expose the raw transcript and mark the fallback. A stored-file request configured above 90 seconds must remain active beyond 90 seconds; Cancel must still end immediately as cancellation rather than timeout. Exercise the streamed transcript safety ceiling with a deterministic fixture and confirm accepted text remains copyable under an explicit partial-result message rather than stopping silently.
42. In a packaged build, verify About reports automatic updates enabled, **Check now** opens the Wails update window, and declining leaves the running executable unchanged. Disable automatic checks, restart, and confirm no background request occurs. Using a controlled newer prerelease, verify the exact `freehand-windows-amd64.exe` asset passes `SHA256SUMS`, stages successfully, and restarts into the version shown by About. A checksum mismatch must fail closed. Confirm this direct-binary path does not claim to run or update through NSIS.

## Model safety

CI and native acceptance must not enumerate models and then call them. Model-list requests are metadata only. Automated tests use fakes or in-process protocol fixtures; real inference is limited to one route and model deliberately selected by the user for the workflow under test. Post-processing acceptance may call its separately selected model only after the user deliberately enables and runs that stage.

### Optional transcription controls

Request fixtures cover microphone, completed-file, and typed-streamed-file
requests with omitted controls, explicit zero temperature, Unicode context, and
Speaches hotwords. They verify exact fields, unchanged audio, Content-Length,
and one request per operation. Invalid controls must fail before transport or
file reads. Persistence fixtures cover migration from absent options and an
inactive retained temperature. Workflow fixtures verify forwarding through the
recorder and file service, and checkpoint snapshots after settings changes.
Frontend coverage protects separation of unsaved and applied nested options.

For native acceptance, use only an explicitly chosen model. Compare baseline
and short context/hotword samples, confirm zero versus default temperature, save
and reopen settings, and test one microphone/checkpoint and stored-file workflow.
Confirm a profile switch with hotwords cannot save until cleared, and clearing
the numeric input cannot silently save zero. Save new hints during a recording
and confirm only the next recording uses them. Record runtime/model versions and
observed behavior; fixtures and upstream source inspection are not live inference
acceptance. No model invocation belongs in automated CI.

### Cleanup generation controls

Fixtures verify default omission, retained disabled limits, exact token-limit
and reasoning fields, rejection before HTTP, no retries on provider rejection,
and preservation of prompts and temperature zero. The S1-mini preset must derive
reasoning off through llama.cpp even with its saved custom override unset.
Migration and settings validation cover missing fields, disabled processing,
invalid/zero enabled limits, and Generic reasoning rejection. Workflow fixtures
exercise generation options with raw fallback, truncation, and cancellation;
frontend tests protect nested draft isolation.

For native acceptance with one explicitly chosen model: save/reopen an output
limit, clear or enter a fractional value and verify validation, disable the limit
and confirm it retains a valid value, and compare Generic versus llama.cpp
reasoning controls. S1-mini with llama.cpp must show a checked, disabled
**Required** reasoning control; S1-mini with Generic must show **Server required**.
Verify preset changes preserve the custom-model preference. A deliberately low
limit must retain raw text when the server reports truncation. Record runtime
build and template/model details for reasoning behavior; neither a model list nor
a successful client fixture proves a runtime honored the override. CI performs
no provider inference.

## whisper.cpp and vLLM acceptance

Use one explicitly chosen server/model at a time. Record the server revision,
model, Freehand revision, operation, and outcome; do not probe model inventories
with inference or publish private inputs/configuration.

- whisper.cpp: save and complete setup with no client model ID; verify a health
  test makes only one GET to `/health` (or the configured prefixed override).
  Dictate and transcribe a WAV file. Check optional context/language/temperature,
  completed-only file controls, cancellation, and switching back to Generic.
- vLLM transcription: test a specific speech model with completed microphone
  and file requests, then streamed files. Exercise Unicode and a multi-chunk
  file; a chunk stop must not cut off later text. Disconnect, malformed-event,
  length, error, and credential-reflection cases use deterministic fixtures.
- vLLM cleanup: test a specific compatible text model, an optional token limit,
  and reasoning-off behavior. For S1-mini, reasoning must remain off even with
  the optional custom switch disabled. Preserve trained prompts and raw fallback.
- Verify Settings, first-run readiness, home quick settings, profile switching,
  and the public backend matrix. Include vLLM-Omni speech using the Qwen3-TTS
  acceptance steps below.

A Windows compile and fixture tests are separate from interactive acceptance
against a live deployment. Metadata success does not prove inference behavior.

## Language selection acceptance

Fixture tests cover default/automatic/named/custom multipart fields across all
four available transcription profiles for both microphone and files, including
validation before upload and exact file framing. S1-mini policy tests cover
English codes/names/regions, unknown-language assumptions, non-English and mixed
reports, and independence of custom cleanup. Recorder and file workflow tests
assert no cleanup request on mismatches, raw delivery/copy, bounded notices,
and enabled/disabled/absent history behavior.

For native Windows acceptance, search names and codes, choose Server default,
Automatic detection, and a supported named language, then save/reopen settings.
Verify an existing custom value remains unchanged. Exercise keyboard selection,
Escape, narrow layouts, light/dark themes, and the English-only S1-mini notice.
With a chosen model and fixed sample, verify raw fallback for non-English input
and S1-mini cleanup under the displayed English assumption when metadata is
absent. Live model runs remain manual and scoped; do not probe inventories.

## Reusable server acceptance

Create one Speaches connection with transcription and speech uses. Confirm it is
listed once in Connections and appears in both feature selectors, but not cleanup.
Select it independently and choose different models. Edit its endpoint/key once;
new requests from both features must use the new coherent connection snapshot,
while running requests keep the old one. Unselecting one feature must leave the
other active. Removing an active use or deleting an active shared connection must
fail without changes. Duplicate and independently replace its key; deleting one
entry must retain a key referenced by the other. Check failed SQL writes, restart,
and v3/v4 upgrades. A vLLM deployment may implement one or both offered operations;
no automatic inference checks may be used to discover that.

## Saved connection acceptance

Fixtures cover forward migration, empty initialization, inactive create/duplicate,
independent selections, model reset on switching, runtime/connection save boundaries,
inactive-key isolation, stale editors, SQL rollback, durable reopen, and credential
retention until the last referencing entry is deleted. Metadata tests must use
only the requested saved connection's key and must never invoke inference.
Frontend tests cover connection draft lifetime and stale metadata results.

On Windows, begin with an empty catalog and create a connection. Confirm it stays
inactive until selected on its feature page. Choose a model, save feature options,
then edit the connection in Connections. Duplicate it, independently replace its
key, switch to it, select None, and delete the inactive entries. Repeat for cleanup
and playback. Check keyboard navigation, dirty-draft guards, restart persistence,
and active-request isolation. Verify clean defaults: capsule/envelope/minimal,
bottom-center, 85% opacity, 70% glow, and an unassigned Show Freehand shortcut.
Assign then clear that shortcut and confirm tray access remains available.
Repeat selection from the home rack, including failed saves and role filtering;
confirm its selectors match Settings and its cards use the same single border
and fill. Browser fixtures and native builds do not replace interactive Windows acceptance.
Use only operator-selected models for deliberate live inference acceptance.

For connection navigation, open Settings → Connections and Manage connections from
each workflow picker: both open the list. Edit connection opens the selected row.
At wider widths, keep the searchable list visible beside the editor; below 760px,
All connections must return to the list. Verify all seven representative entries
fit comfortably in a normal window, list and form scroll independently, and Save
and Cancel remain visible at compact heights. Check both action menus for readable
single-line labels, workflow icons/checkmarks, and compact disabled Delete state.

Change a field, then switch rows, use All connections, or close: Keep editing and
Escape retain the draft, Discard clears it without saving, and Save and continue
completes the pending navigation only after a successful save. A failed save keeps
the draft and shows the error. Repeat with a transient replacement key. Add from a
workflow, save and set up, and confirm the matching model settings open. Creating
with Save for later must leave active selections unchanged. Use for selects only
its chosen workflow, while existing model settings and unsaved Settings drafts
retain their protections. Check keyboard navigation, searchable pickers, and list
empty/no-match states. Metadata discovery must target only the opened workflow,
reuse results, require explicit retries after failure, and reject old completions
when the selected connection changes.

For dark-palette review, check the main window, Settings, About, connection/model
menus, dialogs, inputs, focused controls, disabled controls, and recording overlay.
Confirm navy surfaces and cobalt accents in solid dark mode, readable muted text,
and unchanged status colours. Compare light mode and dark Mica over both light
and dark desktop backgrounds; the Mica result depends on Windows and wallpaper.
Check native active/inactive title bars and startup background after relaunch.
Browser material fixtures establish CSS composition, not native DWM acceptance.

For model-profile acceptance, verify Generic is shown beneath the model in
Transcription and Speech playback, while Post-processing offers Generic and
S1-mini. Selecting S1-mini exposes its trained controls, English-only admission,
and reasoning-off requirement; Generic restores the custom instruction editor.
Check supported reasoning overrides on llama.cpp/vLLM and the explicit server
configuration requirement on Generic. Save and reopen; verify connections,
models, and credentials remain independent. Changing an inventory model ID must
not infer or select a model profile. Check home labels and model profile selection.

Automated tests cover backend/model capability intersection, wrong-role and
unknown IDs while features are disabled, rejection before HTTP, immutable request
selection, S1-mini prompt/reasoning/language regressions, and migration from v5
with existing selections and credential references. New specialized model
profiles need per-role request fixtures and explicit runtime qualification; a
catalog entry alone does not establish support. Do not probe model inventories.

## Remembered model acceptance

- Save distinct model choices for transcription, cleanup, and speech on one
  shared connection. Switch away and back; verify profile, language, instruction,
  trained controls, voice, and speed restore only for the matching use and model.
- A new model ID starts from Generic defaults. Verify changing a name does not
  infer S1-mini, and model edits leave capture settings and timeouts unchanged.
- Switch freely between modified model drafts, save all edited options together,
  and test invalid batch rollback. Discard must restore applied values and clear
  every model draft; saved catalog objects must not mutate through a draft.
- Verify quick model controls restore options before saving, without discovery
  or inference. Settings pickers also list remembered IDs without a server probe.
- Switch connections, restart, rename, rotate a key, change the URL/backend,
  duplicate, remove an inactive use, and delete: check the documented retention
  rules. Forgetting must clear active selection and survive restart.
- Exercise version-six upgrades, failed SQL saves, stale forget requests, role
  validation, and the model count bound with real SQLite. Previously captured
  request settings and credentials must remain unchanged.

Browser fixtures cover presentation and editor behavior only. Native Windows
execution separately covers SQLite, migration, settings transactions, and builds;
interactive Windows acceptance remains a user review step. No inference inventory
probes are part of these checks.

## Connection diagnostic acceptance

- Exercise a valid model list with listed, absent, and unselected model IDs, and a
  health-only whisper.cpp endpoint. Health success must not verify a model ID.
- Distinguish 401/403, bad metadata routes, malformed/oversized responses, network
  failures, and successful metadata access. Success must not claim inference
  permissions or feature support.
- Check invalid provider options, a missing speech voice, and S1-mini on both
  Generic and qualified reasoning adapters. Metadata refresh must remain possible
  while local model options need fixing.
- Edit model options during a pending check and after a completed check. Results
  must be stale for the new draft; unrelated capture edits must not stale them.
  Cleanup, speech, and transcription keep separate assessment inputs.
- Saved-connection checks assess that saved entry only, without selecting it.
  Fixtures must assert GET-only metadata routes and avoid inference calls. Check
  that returned diagnostic text does not reflect user instructions or credentials.
- Review the results panel in light/dark themes and a narrow settings window.
  Pair browser presentation checks with native Windows tests and builds.

### Voice discovery and Kokoro playback

Run inference and connection tests for current/legacy Kokoro voice shapes,
Speaches model-scoped and server-wide fallback lists, explicit empty lists,
credentials, HTTP failures, redirects, malformed/oversized responses, bounded
IDs/counts, and service shutdown. Check that Generic makes no voice-discovery
request and its speech request remains unchanged while Kokoro sends `stream: false`.

In settings, refresh voices, search by ID/name/language, choose and save a voice,
and switch models to verify remembered options. Manual entry must remain usable
without metadata, after a failed refresh, and for an unlisted alias. Change the
connection/model while discovery is pending: old results must not populate the
new selection. Inspect keyboard selection and light/dark/narrow layouts.

Manual Kokoro acceptance used one fixed `af_heart` sample with `kokoro` at speed
1.0 against API 0.6.0. The Windows adapter and WAV decoder accepted 24 kHz mono
PCM16 audio. Audio was not retained and the manual harness is not committed or
run in CI. Native speaker-device preview is a separate interactive check.
Verify the site matrix exposes accessible support labels with visual checkmarks
and links the Kokoro guide from the directory and documentation navigation.

### Consolidation regression coverage

Settings tests block a runtime callback while a second save attempts to commit,
then assert ordered retention updates and callback-safe settings reads. TTS tests
cover stale completion, Stop fencing, and native save-dialog interleavings with Stop
and shutdown. Windows CI runs the complete Go suite, including platform-specific
input, playback, storage, and settings tests; these do not invoke inference servers.

Current-result tests cover copy/clear generation admission without history.
Browser workspace fixtures cover desktop pointer/keyboard resizing, restored
pane widths, narrow view switching, and quick-settings popovers with nested
device/model selectors and asynchronous save outcomes. They use the actual home
components with mocked Wails services and no inference traffic.
The split-restoration test waits for the persisted percentage to match the
separator's final value before reloading, then checks the restored pixel width.
A storage change alone is insufficient because a debounced earlier drag write
can precede the keyboard adjustment.
Renderer coverage checks that speech commands preserve the composer draft and
that file readiness excludes microphone and shortcut prerequisites while retaining
endpoint and authentication checks. Interactive acceptance should switch between
all three tasks and Settings, confirm unsent text survives, then verify current
results can be copied and cleared with history disabled.

### Transcription details window

Open details from both recent history and Settings > History. Verify one independently resizable native window opens without blocking either source window. Open a different completed entry while details is visible or minimized: the same window must update and focus. Check long model names and response metadata at the minimum 480 x 400 size, light/dark appearance, and Mica after restart. Native close, Alt+F4, Escape, and the Close button must hide details without quitting Freehand; reopening and renderer reload must recover the selected run correctly. Delete the selected entry, clear history, disable retention, and evict the entry with later runs: stale details must disappear. Automated history service tests cover invalid/pending IDs, selection switching, deep-copy isolation, removal, and close/shutdown behavior. Native runtime acceptance remains separate from the Windows compile.

## Realtime and Connection Manager acceptance

Run `go test ./internal/realtime ./internal/dictation ./internal/settings ./internal/storage ./internal/overlay ./internal/platform ./internal/windowing`
and the frontend checks/tests. Fixtures cover exact session configuration,
binary PCM frames, authoritative final replacement, missing final failure,
cancellation, stale preview fencing, independent credentials, transactional
realtime preferences after restart, and reusable window navigation. Caption tests
cover bounded Unicode text, one-row whitespace normalization, and fit-cache invalidation.
These fixtures use fake transports and do not invoke inference.

Qualify a chosen Nemotron model manually against NeMo-Speech.cpp v0.1.0. Record
the server/runtime/model revision and distinguish transport inference evidence
from interactive Windows acceptance. Exercise the native checklist below; a
successful compilation or synthetic socket probe does not establish focus safety
or window behavior. Never load inventories or add automatic inference to CI.

### Unified Voice transcription

Migration fixtures exercise old realtime-enabled and completed configurations, preserving the file selection and inactive remembered realtime models. Completed Voice profile tests isolate endpoint, model, options, headers, and credential snapshots from Audio file. Backend eligibility tests must reject realtime on Generic or a mismatched server/profile.

For native acceptance, open Voice → Transcription: there must be no separate Live button. Choose NeMo-Speech.cpp, its loaded model, and the explicit Nemotron profile; enable Realtime inside that panel. Verify live results and one-row captions. Turn realtime off and record using the same connection/model. Switch to an ineligible model/connection and verify mode is disabled. Configure Audio file separately, switch between tasks, and verify independent connection/model/language settings and truthful footer status. Restart and repeat. Test Voice-only first-run setup with Audio file unconfigured. These native checks are separate from successful builds and deterministic tests.

Language dropdown acceptance: check the searchable language picker and the qualified Nemotron select inside quick-settings popovers at short and normal window heights. Menus must remain within the window, scroll internally with the wheel, and expose the final option through keyboard navigation.

### Shared vocabulary acceptance

Check Vocabulary at regular and small Settings window sizes in both themes. Edit names, independently toggle Voice/files, save, and reopen. Follow Vocabulary links from a dirty Voice/file settings draft and verify that edits survive navigation. Confirm the NVIDIA mark appears for NeMo in quick controls, connections, and the Vocabulary page. Switch supported/unsupported models and preserve the list and use preferences. For NeMo, verify limits and strength, then test completed Voice, audio files, and realtime against only the explicitly selected model when native inference acceptance is authorized. Backend fixtures cover migration, immutable workflow projections, Unicode limits, omission, and multipart speech contexts without automatic inference.

## Qwen3-ASR and vLLM realtime

Run `go test ./internal/modelprofile ./internal/realtime ./internal/inference`.
Fixtures verify the explicit Qwen/vLLM intersection, supported language hints,
model-only realtime setup, JSON/base64 PCM16, fragmented language headers,
authoritative final replacement, cancellation, disconnect, premature or missing
finals, peer-error redaction, and transcript bounds. No inference runs in CI.

For manual Windows acceptance, select vLLM and the Qwen3-ASR profile under Voice,
then enable realtime and live captions. Verify provisional results, single-row
captions, stop/finalization, cancellation, and focus-safe delivery. Toggle back to
completed transcription and verify saved language/context controls return.
Select Audio file independently. Confirm shared vocabulary applies to completed
audio only and remains saved when realtime is active. Check the Qwen model mark
and restricted language menus in quick settings and Settings. Successful builds
and synthetic transport checks do not substitute for native microphone acceptance.

## Audio file and speech workspace layout

Check Audio file and Text to speech at normal and compact desktop sizes, including
1156×760 and 650×550. Use synthetic renderer fixtures for selected, busy, completed,
and error states without invoking inference. File summary, response-mode switch,
actions, and result pane keep their positions across these transitions. Long file
names truncate without displacing actions; errors remain readable in the result.
The speech editor and its bottom playback area retain their heights across idle,
generating, playing, paused, completed, and failed states. Check the Speech settings
popover, nested connection menu, Escape focus return, draft retention across tabs,
and visible keyboard focus in both themes. The application Settings button must
remain on screen when the shortcut hint is hidden at compact widths.

## Speech controls, vocabulary feedback, and transcript reading

Connection-manager tests retain separate results, reject missing IDs, preserve active
selections, and invalidate cached/in-flight checks on confirmed settings snapshots,
even when public endpoint fields are unchanged. Model-source tests cover combined
saved/server/draft labels and manually entered IDs. Navigation tests keep keyboard
order aligned with the displayed groups.

Review Connections at normal and compact window sizes: list-first navigation,
collapsed diagnostics for the selected entry, active-use summaries, and independent
list/form scrolling. Review the shared model picker in every workflow and quick panel,
including a failed save, retry, manual ID, and long list. Sticky Settings headings
must leave focused validation controls visible. These checks use metadata or
synthetic fixtures and require no inference inventory probes.

Run the config/modelprofile/settings tests and frontend suite. Vocabulary cases
cover UTF-8 phrase limits, exact duplicates with source line numbers, the first
excess phrase and byte budget, context consumption, unsupported workflows, and
bounded oversized-draft feedback. Speech quick-save tests cover remembered voice
restoration, task-speed preservation, credential/draft exclusion, and failed saves.

The optional `polish.spec.ts` browser checks exercise speech voice save/failure/retry
at a short desktop size and live text following, reader scrollback, finalization,
Jump to latest, and the next recording. Fixtures contain synthetic text and invoke
no inference. Recheck the shared model/voice menus and speed slider in both the
popover and full Settings page, including keyboard entry and nested-menu scrolling.

In native Windows, review Vocabulary with duplicate and over-limit draft lines,
activate a line link and verify selection/scrolling, then discard the draft. Check
normal and compact windows, light/dark themes, long transcripts, and popup Escape
focus return. Builds and synthetic streaming do not establish microphone or model
inference acceptance.

### Unsaved speech preview

Settings tests cover draft model/profile/voice/speed/timeout capture, draft enable
while saved speech is disabled, immutable endpoint/credential snapshots, no saves,
and rejection of invalid options or stale connection IDs before credential access.
The speech service test checks that synthesis receives captured draft options;
renderer tests verify that preview forwards only the bounded draft DTO.

For native acceptance, change voice and speed without saving, preview the fixed
phrase, stop, change them again, and preview again. Discard edits and confirm normal
Text to speech still uses the saved options. Test draft enable with saved speech
disabled, invalid settings, a connection changed in another window, cancellation,
and recording admission. Use only the explicitly chosen model; no inventory probes.

## Speech family expansion acceptance

Fixtures in `internal/modelprofile`, `internal/inference`, and `internal/realtime`
cover the Parakeet, Cohere, Voxtral, and Qwen3-TTS contracts. HTTP fixtures assert
request fields and voice metadata; both Qwen and Voxtral run the vLLM final,
disconnect, error, cancellation, oversized-text, and premature-final scenarios.
SQLite fixtures reconstruct alpha.4's schema and test upgrade, Unicode speech
options, restart, and transaction rollback. Preview tests prove unsaved language
and instructions are captured without saving or forwarding transport fields.

Native acceptance, with one explicitly selected model at a time:

1. Select Parakeet on NeMo-Speech.cpp. Confirm automatic detection, completed
   recording/checkpoints, and independent audio-file transcription.
2. Select Cohere on vLLM. Confirm its 14-language selector labels the default as
   English, and that context/vocabulary controls are absent. Check a completed
   microphone recording and a file response.
3. Select Voxtral Mini Realtime on vLLM. Enable realtime and captions, record,
   and stop. Confirm provisional text is replaced by the final, captions remain
   one row, and changing focus prevents insertion. Disconnect mid-recording and
   confirm provisional text is never inserted.
4. Select Qwen3-TTS 1.7B CustomVoice on vLLM-Omni. Check the nine preset voices,
   ten languages, style field, and keyboard navigation in full and quick settings.
   Edit language/style and Preview before Save. Verify ordinary playback retains
   applied settings until Save; Discard restores them. Switch models and back,
   restart, and verify saved options remain independent per connection/model.
5. Metadata refreshes must never generate speech or transcribe a sample. Record
   actual inference acceptance separately from fixture and Windows build results.
