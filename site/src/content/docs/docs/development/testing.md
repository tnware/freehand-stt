---
title: Testing contract
description: Deterministic, integration, and native acceptance responsibilities.
---

## Choosing useful coverage

Test observable outcomes at the owning boundary: admitted requests, immutable
settings, persisted data, emitted status, rendered controls, and actions a user
can perform. Cover failure, cancellation, and out-of-order completion where they
can change those outcomes. Small mechanical refactors do not need new tests that
repeat their implementation.

Avoid assertions about source spelling, prop forwarding, private helper names,
or incidental markup counts. Exercise state owners directly, render static
presentation, and use browser tests for component wiring, interaction, and the
accessibility tree. When removing a brittle test, identify the behavior coverage
that replaces it or explain why the asserted detail is not a contract. Preserve
string assertions for actual protocol values, safety-relevant copy, generated
artifacts, and release configuration; reading a file is not itself a test smell.

Use controlled promises or channels for asynchronous results and Go's
[`testing/synctest`](https://pkg.go.dev/testing/synctest) for timer-driven owners.
Updater tests advance the production schedule in virtual time and exercise
disabling updates during a pending result, metadata timeouts, retry after failure,
and returning to daily checks after recovery. Manual-check fixtures distinguish
a staged update from an up-to-date result. A blocked presentation fixture verifies
the shared two-second shutdown deadline, rejection of new work, and suppression
of late status changes. These checker fixtures do not qualify
the native Wails updater window or its shutdown behavior. Window option tests
cover renderer permissions across surfaces; retained-instance tests check reuse
without claiming native close, focus, or startup acceptance.

## Renderer state and navigation

Frontend CI runs unit tests, Svelte checks, and the production build. Keep the
frontend Playwright workflow suite out of CI because of its runtime cost; run
relevant browser tests locally for concrete behavior changes. Browser fixtures
use synthetic services and do not qualify native desktop behavior.

Exercise dictation events and overlapping status reads in both completion orders.
Older generations must not replace newer state, refresh history, or emit result
notifications. Same-generation completion, rejection, and clear events remain
valid, and a newer generation must not be lost behind a slower snapshot read.
Settings metadata tests cover the single confirmed-snapshot invalidation path,
including changes to credentials that do not alter renderer-visible settings.

Suspend session initialization at each stage, dispose the session, then resolve
or reject the outstanding request. No subsequent initialization or session-level
failure reporting may start. Real App mount/unmount browser fixtures also reject
late metadata and readiness replies, verifying that teardown prevents a new
`ShellReady` call and late error notices.

The shared pending-changes dialog must preserve each parent's labels, errors,
and navigation actions. Browser checks keep the dialog open during a pending
save despite Escape or outside interaction, then allow recovery after failure.
Compare completed transcription metadata fixtures through both microphone and
file clients while retaining their separate response limits and text admission.

## Managed desktop runtimes

Keep managed-runtime tests isolated from the user's application-data directory.
Use small synthetic archives, fake HTTP listeners, and disposable child
executables for deterministic checks; these fixtures are not official NeMo
runtime or model qualification.

Exercise the shared archive downloader through both the single-archive and
bundle installers. Failed HTTP status, mismatched declared length, short or
oversized unknown-length bodies, checksum mismatch, and cancellation must leave
no published runtime or staging directory.

Coverage must include archive checksum failure, path traversal, cancellation,
interrupted install/retry, bounded child output, metadata-only catalog filtering,
model download failure/removal, unsupported platforms, premature child exit,
readiness timeout, port conflicts, and shutdown during install/pull/start/run.
Windows process tests must prove descendants cannot outlive the owner's Job
Object, not just that the immediate child receives a kill request. macOS native
tests exercise cancellation, natural exit (including commands that exit before
exit-watch registration), parent SIGKILL, descendant cleanup, and libproc
rejection of foreign and dead listeners. Temporary fixture roots use
the physical macOS temporary directory so production path guards remain strict.
Owned-process output lifecycle tests run only on Windows and macOS; portable
output-buffer tests still run on Linux. Synthetic tar extraction tests run on
all hosts, checking regular-file content and host file-mode semantics: exact
POSIX permissions on Unix and writable-file attributes on Windows.

`TestDarwinOfficialRuntimeArchives` is opt-in via
`FREEHAND_MACOS_RUNTIME_ARCHIVES`, an isolated directory containing the official
filenames pinned in `platform_recipe.go`. It verifies and installs the Mac
archives, checks Mach-O metadata, and runs only `--help` on the native
architecture. It neither downloads, enumerates, nor loads models. Archive
verification on another architecture is not native execution acceptance.
Keep this opt-in qualification out of automatic CI inference checks.

Real SQLite tests exercise fresh empty inventory, forward schema upgrades,
enabled/disabled runtime preferences, save/reopen, and failed persistence.
Settings tests prove independent task selection, credential-free managed request
snapshots, rejected unsupported roles, and unavailable-instance admission without
remote fallback. Runtime tests exercise independent worker lifetime, retired
directory ownership, stale endpoint generations, and the shared shutdown bound.
Provider-admission checks reject new duplicate installations and keep a provider's
process slot occupied through actual child exit. Distinct-provider concurrency
must remain independent. Verify existing duplicate inventories can still be
loaded and explicitly repaired without rewriting Connections.
Assert external behavior at these boundaries rather than source strings, markup
snapshots, or private call ordering. Existing realtime fixtures continue
to check loaded-model identity, final-only delivery, and no automatic replay.

Rejected dictation starts must publish user feedback before capture, including
unavailable managed profiles reached through toggle and hold commands. Exercise
repeated rejection, a subsequent successful start, and rejection while a prior
result remains available to copy. Keep cancellation, shutdown, and active-work
fences intact. Native acceptance separately checks that an offline hotkey attempt
shows the configured error overlay without activation, and that stopping or
starting a runtime leaves the recording controls mounted at the same height.

Overlay timeout tests use Go's virtual-time `synctest` to verify the five-second
failure/copy notification lifetime, retained result state, renewed attempts,
active-work protection, settings replay, preview isolation, and shutdown. Native
acceptance checks the fade (or immediate hide with reduced motion) without focus
changes; the in-app transcript and Copy action must remain available afterward.

`TestManagedEndpointReachesSpeechClients` passes the real adapter's published
endpoint to the production microphone, file, and realtime clients against a
versioned NeMo fixture. It covers both qualified completed profiles and the
Nemotron handshake, catching a missing `/v1` prefix even when readiness succeeds.
The fixture does not load a model or establish native inference acceptance.

Built-in Connections tests exercise automatic catalog membership, selecting and
reopening task settings, immutable runtime-owned fields, legacy alias preservation,
manual quotas, and stopped runtime rejection through the real storage/settings
owners. UI acceptance selects a running local Cleanup Connection without creating
an alias or entering a URL. Acquisition tests must observe changing counters and
distinct verification and terminal outcomes for GGML and NeMo. NeMo fixtures use
the pinned model manager's documented partial-file layout, not fabricated output
lines. A 100% transfer must not render as success before verification completes.

`TestManagedGGMLProductionClientRoutes` exercises the actual managed adapter
boundary with production cleanup and completed-transcription clients. It checks
llama.cpp `/v1/chat/completions` with S1-mini reasoning disabled and whisper.cpp
`/inference` without introducing model discovery or inference inventory scans.
`TestGGMLPinnedRuntimeZIPs` is opt-in: set `FREEHAND_TEST_GGML_ZIPS` to an isolated
directory containing the pinned `llama.zip` and `whisper.zip`. It verifies the
official archives through the installer, then launches owned processes with the
managed arguments and `--help`; it neither downloads nor loads models.

`TestGGMLPinnedCUDABundles` uses `FREEHAND_TEST_GGML_GPU_ZIPS` with the official
CUDA ZIP filenames from the pinned recipes. It verifies and installs both
providers through the real backend-selection path, reopens the installation,
executes owned `--help` commands, and enumerates llama.cpp's CUDA device. It
requires compatible NVIDIA hardware and a driver; it never downloads or loads
a model. NVIDIA's NVML probe requires the OS `ProgramFiles` variable even with
an absolute executable path. Keep this explicit environment allowance rather
than inheriting user PATH or CUDA configuration.

The Windows runtime pins are llama.cpp `b10809` and whisper.cpp `v1.8.3`; release and
runtime URL/size/SHA-256 metadata live in `internal/managedruntime/platform_recipe.go`;
model pins live in `provider_ggml.go`.
Qualify pin changes using the exact official binary, not only fixtures matching
the intended arguments. Native acceptance separately covers explicitly selected
S1-mini cleanup alongside NeMo, whisper completed requests, cancellation, and
Quit with both providers running.

For CPU/CUDA backend changes, exercise the real installer against bounded archive
fixtures: preserve models and Connections, verify every companion archive, and
retain the old installation on cancellation or corrupt downloads. Reject changes
while running or active speech work holds admission. Exercise the UI action in
both directions without implicitly starting a model. Qualify official CUDA
archives with the production sanitized child environment, not a developer PATH
that happens to provide missing libraries. CLI help and device metadata are not
GPU inference acceptance; record selected-model execution separately, including
GPU memory contention when NeMo and cleanup run together.

Browser fixtures should cover a fresh installation with no manual connections,
recommended Nemotron realtime setup, supported catalog browsing without pulls,
explicit download/cancel/retry, switching models, unsupported realtime, status
refresh across windows, removal confirmation, and dirty-draft protection.
At compact desktop sizes, assert that provider-row install, selected-model download,
progress/cancel, and start/stop remain in view without scrolling or Playwright's
automatic click scrolling. Download completion must not implicitly start inference.
Repeat from an alternative model's catalog row after scrolling to it: progress,
Cancel, and terminal feedback must stay in the viewport without an additional
scroll, including after the model becomes installed. Check the Voice and audio-file
sidebars place Connection above Model and open the corresponding contextual options.
Verify provider images load from bundled assets and agree across runtime headings,
built-in Connection details/pickers, and managed task shortcuts. Check both themes;
status text and action labels must stay readable independently of the artwork.
Exercise CPU/CUDA status updates in workflow sidebars and Connection details without
changing the Connection selection. Reopen real SQLite settings containing the
`llama.cpp (CPU)` and `whisper.cpp (CPU)` default names: built-in names
must be backend-neutral while instance preferences, selected IDs, and custom
names remain intact. These checks establish display consistency, not GPU offload.
Exercise download/cancel/retry and start/stop directly from a collapsed runtime
row, checking that details stay collapsed and dirty drafts still guard mutations.
Exercise keyboard navigation, narrow layouts, light/dark appearance, and reduced
motion. Generated Wails DTOs remain the fixture contract.

The `managed-runtime` and `runtime-sources` browser suites cover the composed
runtime section, including its setup/preferences and model-catalog components.
Keep assertions at the visible controls and runtime commands so component
extraction preserves confirmation, cancellation, and metadata-only browsing.

Native acceptance uses only the explicitly selected model. Check installation
from an official verified archive, NeMo's model pull, a real ready listener,
realtime microphone finals, completed files, Stop/Start, Quit during a model
download, restart, and complete removal. Verify no listener binds to the LAN,
no child remains after exit, and manual connections still work when explicitly
selected again. Browser fixtures are not native inference evidence. No model
inventory inference belongs in CI. See the
[Windows checklist](../../safety/native-test-checklist/#managed-local-runtime).

### Startup, recommendation, and output-viewer validation

Test host recommendations, GPU warm-up, startup reporting, and the output viewer
at their ownership boundaries. Separate deterministic results from native
startup, rendering, clipboard, and selected-model inference observations:

- Source projection tests compare metadata with actual acquisition specifications,
  including companion archives and NeMo release-index pins. Intercept the real
  GGML HTTP request before network access to verify the advertised source.
  Browser tests inspect sources before installation without mutation calls,
  distinguish NeMo's model manager from direct downloads, and follow explicit
  CPU/CUDA preview choices without fetching files.
- llama.cpp normal-output tests retain bounded private memory-only capture, reject
  environment/config logging overrides, and qualify the exact pinned CPU binary
  with a model-free upstream warning through the owned launcher. This does not
  establish native viewer behavior or GPU inference acceptance.
- Recipe/host selection checks Windows x64 CPU/CUDA choices, NVIDIA device 0,
  driver 551.78 and compute capability 5.0 thresholds, unknown or malformed
  metadata, unsupported platforms, and explicit CPU override. The metadata
  recommendation cannot download, start, upgrade, or replace an installation;
  installation admission uses the same compatibility rule, not free VRAM.
- Launch-argument checks preserve GPU llama.cpp/NeMo built-in warm-up and CPU
  `--no-warmup`. Synthetic HTTP fixtures validate only CUDA whisper.cpp's
  selected-model startup multipart request, one-second silence, bounded/discarded
  response, failure, and cancellation. Assert that ready metadata alone cannot
  admit a warm-up-pending endpoint. Pinned-source evidence belongs with the
  fixtures; fixture success is not first-request latency evidence.
- Startup lifecycle checks cover cancellable prelaunch hashing, phase transitions
  and phase-specific elapsed time, the shared 120-second post-creation readiness/warm-up
  deadline, and the additional four-second drain. A cancelled or timed-out start
  cannot mark Running or release ownership while the child is still alive.
- Operation publication checks immediately admit a follow-up action from a
  terminal notification while preserving terminal-before-next-start event order.
- Output checks cover 256 KiB/1,024-chunk/4 KiB-per-chunk limits, cursor deltas and
  truncation, interleaved streams, split UTF-8 and terminal controls, generation
  fencing, reader admission, revocation without private-tail erasure, explicit Clear,
  next-start/remove/shutdown cleanup, and the separate bounded parser prefix.
- Use split and malformed controls to verify the display allowlist: bounded SGR
  colors/styles and erase-line progress may pass; OSC clipboard/link/title,
  DCS/APC/PM/SOS, queries, input modes, and alternate-screen controls may not.
  Check per-stream decoder isolation, bounded terminal scrollback/write queues,
  eviction/reset propagation, and stale writes after close or revoked access.
  Explicit Copy selection must copy only selected rendered text through the native
  boundary, never read the clipboard or react to upstream escape sequences.
- Windowing and frontend state tests must cover immediate reads for visible embedded
  runtime tabs, standalone consent per opening or runtime switch, viewer reuse,
  bounded non-overlapping polling, late-read rejection, visible
  state clearing, and no process lifecycle calls from viewer actions. Browser
  fixtures must cover recommendation before installation, explicit acceptance,
  phase/elapsed display without fake percentages, **View output** during startup
  from management and quick controls, direct embedded display, standalone consent,
  read-only terminal rendering,
  colors and carriage-return progress, local search, resize, selection/copy admission,
  follow/pause scrolling, Clear, close/reopen, and sparse upstream output. Use bounded
  synthetic output; viewer interactions must never send input or change process lifetime.

`process-output.spec.ts` checks the standalone viewer in both themes at wide and
narrow widths, stable consent geometry, terminal content fitting its viewport, and actual clicks on
search and Follow controls. Searching forward and backward to the same match
must leave Copy enabled when the terminal still has a selection.

For native Windows acceptance, use only a user-selected model to measure startup
and first/subsequent request behavior, confirm actual GPU preparation and CPU
behavior, inspect host recommendations,
and exercise real window close/reuse, cancellation, and process-tree shutdown.
Neither browser mocks, `--help`, a docs build, nor successful deterministic tests
establish those results. Use the
[native runtime acceptance section](../../safety/native-test-checklist/#managed-local-runtime).
macOS currently has shared recipe-extension
contracts only: no qualified managed packages or native acceptance.

## Shortcut recovery regression checks

The controller and input-service tests cover unavailable startup toggle/show
bindings, preservation of independent registrations, replacement capture before
and after clearing, and explicit hold-hook initialization despite a global
conflict. An unavailable startup hold preference must not block capture of a
replacement toggle. Empty chords never reach global registration or emit hold
edges. Settings and real SQLite reopen tests cover saving an empty toggle without
restoring its default; frontend readiness treats unassignment as nonblocking.
Real controller/settings-service integration tests inject persistence failure
after replacement or Clear has already changed native bindings. They assert the
complete prior settings snapshot, persisted settings, startup preference, effective
globals and hold assignment survive, no rejected startup chord is retried, and no
failed draft is published. Degraded toggle, degraded hold, and fully bound Clear
cases also exercise capture after rollback and a subsequent successful save.
Hold lifecycle coverage starts with an empty assignment, then assigns a chord and
captures again, asserting exactly one `Start` and subsequent `Configure` use.

On Windows, additionally exercise a real startup conflict, **Clear → Save**,
restart with the toggle unassigned, and **Record → Save** with an unused chord.
Check hold press/release separately, including that clearing hold stops modifier
keys from starting recording. On macOS repeat optional-toggle persistence and
capture recovery in a packaged app with the required permissions. Deterministic
fixtures do not establish OS registration ownership or cross-platform acceptance.

Run `go test -race ./internal/shortcut ./internal/settings ./internal/input ./internal/platform ./internal/app`
and `npx playwright test --config shortcut-recovery.config.ts` from `frontend`
(for the browser command only). The browser fixture uses the real generated retry
binding and Wails request envelope with a mocked transport, not native keyboard or
permission access. It covers failure/success refresh, no mount retry, pending/busy
disabling, clearing unavailable hold, and preserving a dirty draft. Native-hook
unit tests cover unchanged-chord rearm, loss cancellation, fresh reducer edges,
released-key checks around tap creation, and failed replacement preservation.
These checks do not substitute for packaged-app keyboard/permission acceptance.

## Native macOS acceptance

See the [macOS platform boundary](../architecture/#macos-platform-boundary).
Run the deterministic Go and frontend suites on macOS, then exercise a packaged
app with a stable bundle identity. Tests guarded by `FREEHAND_NATIVE_*` are opt-in:
a skipped acceptance test proves nothing about native behavior. Do not prompt, type
into arbitrary user apps, record audio, or access a live credential account in CI.

Native verification must separately cover:

- First launch, menu-bar icon, native titlebars/Edit menu, close-to-hide, Dock reopen
  (main window only), encrypted second launch and bounded Quit cleanup.
- Denied/authorized microphone behavior, explicit-device/default routing, unplug,
  repeated start/stop/cancel, live PCM format, output pause/seek/restart and close.
- Toggle/show and hold press/release, repeat suppression, modifier-only chords,
  both physical modifier sides, Secure Input, tap disablement and shortcut capture
  cancellation. Re-test after granting/revoking Input Monitoring/Accessibility.
- Disposable editable targets: Unicode/emoji/newlines and unchanged clipboard.
  Changing fields within the same window delivers to the current field; changing
  app/window, closing/reusing the process or losing verifiable window identity
  requires explicit copy. Do not restore target focus automatically.
- Delivery guards: unavailable Accessibility, active Secure Input, held physical
  modifiers and cancellation before/during chunks. Editor roles, AXEnabled,
  protected-content metadata, AXValue settability and element identity are not
  required. Do not claim detection of every custom secure field.
- Rejection diagnostics: capture/validate/send reasons remain bounded and scoped
  to their recording generation; unknown errors use generic copy guidance. Check
  partial-delivery warnings and explicit Copy without automatic clipboard fallback
  or retries. A rejected send is not evidence that nothing was typed.
- Overlay first show/update/hide/close without key-window or target changes; every
  layout, anchor and visualizer, multi-monitor placement, reduced motion/contrast,
  bounded captions, hidden timers and shutdown.
- Disposable Keychain account set/get/update/delete, denial mapping and no plaintext
  fallback. Start-at-login exact bundle path, disable/removal and actual next login.
- Real bundle/signature/entitlement inspection; Intel execution and signed update
  application are distinct from an Apple Silicon build or a local ad-hoc signature.

Record commands, actual results, and unverified cases. Live inference
uses only an explicitly selected endpoint/model; do not qualify a model inventory.
Microphone or keyboard denial must leave file transcription and TTS independently usable.

Insertion fixtures in `internal/platform`, `internal/insertion` and
`internal/dictation` cover native predicate categories, app/window validation,
first-failure preservation, raw-error redaction and capture-rejection lifetime.
`frontend/tests/browser/insertion-diagnostics.spec.ts` uses synthetic service
responses to check copy-required explanations and recovery presentation. These
fixtures do not establish delivery into another application. Rerun the integrated
suites after policy changes; focused race passes do not qualify the full build.

## Public download selection

Run the pure release/OS-selection tests with `npm --prefix site test`. Browser
checks reuse the frontend's pinned Playwright installation and never download or
execute release binaries:

```sh
npm ci --prefix frontend
npm exec --prefix frontend -- playwright install chromium
npm --prefix site run build
npm --prefix site run test:browser
CI=true npm --prefix site run build
CI=true SITE_TEST_BASE=/freehand-stt/ npm --prefix site run test:browser
```

The second build/run pair exercises the production GitHub Pages base path.
Keep the preview environment consistent with the build. Browser fixtures cover
Windows, both known Mac architectures, ambiguous Mac architecture, unsupported
platforms, partial/missing/malformed releases, API failure/timeout and no JavaScript.
Checks also cover platform grouping, decorative icons surviving release-label
updates, narrow download layouts, and accessible capability indicators in the
backend tables. Visually review both Mac and Windows recommendation rows and
confirm the neutral glass surfaces remain faint with readable text.
All supported alternatives remain visible. Synthetic release assets establish
selection behavior, not public availability or native application acceptance.
The test preview uses its own port and bypasses Astro's development preview lock.

## Website app preview

After building the site, run `cd site && node tests/app-preview.browser.mjs`.
It uses the frontend's pinned Playwright installation and a separate production
preview port. It checks the default dictation scene, short recording beats,
three cumulative insertions and automatic replay, the second-tab realtime scene
with latest-word captions and final-only insertion, pause, screenshot switching,
enlargement and Escape, shared capability-column row alignment, narrow layouts,
reduced motion, and the no-JavaScript
fallback. Use `SITE_TEST_BASE=/freehand-stt/` after a Pages-base build.

The macOS-style scene is an explicitly labelled illustration with sample text,
not a native recording. It never requests microphone access or runs inference.
Realtime uses an illustrative shorter finalization beat, not measured model
latency; the support note links to qualified model/backend combinations.
Reduced motion initially shows the completed document without autoplay;
offscreen and hidden-page playback pauses. The other preview tabs use labelled
UI-review captures. None of these assets establish native acceptance.

The Features page adds a static overlay-appearance explorer. Check each layout
at every position on desktop and narrow screens, selected controls, no-JavaScript
fallback, and the linked guide anchors. It illustrates layout choices only;
it does not change application settings or establish native rendering acceptance.

The Features hero uses decorative neutral icons alongside its overlay-setting
descriptions. Keep page titles consistent and supporting feature sections
text-first. The app-preview suite checks hero icon visibility on narrow screens
and without JavaScript, while preserving the explanatory text and guide links.

For local setup discovery, check the homepage's local and manual setup paths,
the runtime/model choices on Features, and installation labels in the Backends
and Models directories. Only NeMo, Windows whisper.cpp, and llama.cpp with
S1-mini should offer installation in Freehand; compatible remote models and
speech generation must not inherit that label. Follow the local setup links
through to their guide sections with the production base path. Check narrow
layouts and no-JavaScript access as well as desktop presentation.

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
GitHub effects. See [GitHub Actions](../github-actions/) for workflow configuration.

## Product site and onboarding acceptance

Run `npm --prefix site run build` after site or documentation edits. Check the
rendered homepage and getting-started guide at desktop and narrow mobile widths:
task links must reach the intended section, text must remain readable without
horizontal scrolling, and headings/keyboard navigation must retain their order.
Also build with `CI=true` when checking GitHub Pages base-path links.

The documentation sidebar follows the reader's path: Get started, Use Freehand,
Backend profiles, Model profiles, Reference, then Contribute. Installation,
connections, and choosing a model profile precede specialized guides. Keep
provider/model directories with their corresponding guide groups, and sort peer
backend/model guides by name after their overview material. Verify that sidebar
links and previous/next links retain this order on desktop and mobile.

Keep the README, homepage metadata, and user prerequisites aligned: dictation
leads, file transcription needs no microphone or shortcut, and optional
text-to-speech needs no STT connection. Cleanup is subordinate to transcription.
Existing readiness fixtures cover task gating; a documentation build alone does
not establish native behavior or live-server compatibility. Keep generated
artwork descriptions faithful to the artwork rather than changing alt text to
advertise capabilities it does not depict.

### Desktop setup and recovery

`frontend/tests/browser/readiness.spec.ts` exercises the real workspace with
synthetic settings and metadata services. It checks prerequisite gating, model
discovery before selection, failed-check retry, failed completion save and
retry, completed-check disclosure, optional controls, and focused microphone and
connection recovery. First-run completion has no dismiss action; established
recovery retains its existing dismissal behavior. File readiness remains
independent of microphone readiness. The presentation-priority unit tests
supplement the existing readiness policy tests.

Review setup at normal and narrow window sizes, including a short window: the
next action must remain reachable, expandable sections must work by keyboard,
and controls must not introduce horizontal scrolling. On Windows, separately
verify opening Audio/Shortcuts/connection settings, returning to setup after
saving, unplugging the selected microphone, retrying a failed connection check,
and persisting setup completion across restart. Browser fixtures and executable
compilation do not establish those native interactions or invoke inference.

### Action feedback

`frontend/tests/browser/feedback.spec.ts` uses synthetic speech and file services
to verify one visible speech error, full keyboard-accessible details, preserved
composer text across retry, settings shortcuts, and shared fallback when leaving
the failed composer. It compares editor/control bounds before and after failure
and success notices at desktop and compact widths. Playback controls must remain
clickable while a confirmation is visible. Unit tests cover operation-generation
ownership, stale failure clearing, and preservation of unrelated command errors.

Native review should include a speech-generation failure, a cancelled or failed
audio save, failed Settings save before switching connections, and a dictation
failure with a long explanation. Check that notices leave task controls reachable,
Escape returns focus from Details, and opening the relevant settings pane
preserves the current work. These fixtures do not invoke inference or establish
native file-dialog or window-focus acceptance.

### Footer connection panel

`frontend/tests/browser/connection-status.spec.ts` covers task-specific names,
models and check results, retry after failure, stale draft checks, missing
connections, disabled speech, keyboard activation/Escape, and expandable
technical details at compact and desktop widths. The fixtures use synthetic
metadata services and the production footer/panel components. Editor tests
verify that explicit workspace checks use applied endpoints and empty credential
drafts, route to the correct service, and respect disabled/pending states.

On Windows, separately verify that **Edit connection** opens the correct saved
connection, **Choose connection** opens the manager for the current task, and
**Speech settings** focuses that settings section. Switch tasks and connections
while checks complete; an earlier result must never appear to belong to a newer
selection. Opening the footer panel must not record, run inference, or trigger
another automatic metadata probe. Native focus/window behavior is separate from
these renderer fixtures and compilation.

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
separate preset. Using only an explicitly selected model, compare
Generic and the applicable dedicated profile for normal transcription, file
streaming, cleanup, and speech playback. Never invoke model inventories.
Builds and fixture tests do not establish this interactive or live-server acceptance.

## Task-local setup acceptance

These are checks to perform, not a record of native acceptance. Run the affected
flows separately on Windows and packaged macOS. Assert one main workspace and
Session for global Settings, the Connections rail page, and contextual options.
Voice, files, and speech keep their main result or composer visible while options
are edited on the right. Test startup requests, repeated opens, page navigation,
right-sidebar dismissal, and native close-to-hide with a dirty draft. Save must
commit before continuing, Discard must restore applied values, and Keep editing
must retain the current inspector and draft. Global Settings exposes only General,
Shortcuts, Overlay, and Vocabulary; About and native history details retain their
existing lifecycle. No additional configuration window opens.

Check first-entry metadata discovery for Voice, Audio file, Cleanup, and Text to
speech before model selection or optional-feature enablement. Controls within each renderer must reuse pending and completed results; committed
settings events invalidate metadata coherently across all configuration surfaces. Delay a response, change the
connection, and verify the old result is rejected. Exercise empty lists, failures,
explicit retry and deliberate picker re-entry without reactive retry loops.
Automatic requests must use applied connections and empty renderer credential
drafts; metadata discovery must never invoke inference or select/enable models.

Use an isolated configuration for fresh-install review, preserving existing live
settings and credential store. Start independently with Voice, Audio file, and Text to
speech. Add a connection from each picker, verify the purpose is preselected and cannot
be removed, and verify Save and return resumes the task's configuration with the new selection.
File and speech setup must work without completing dictation setup. Choose or discover
a model; test metadata access without invoking model inventories.

Repeat from contextual Transcription, Cleanup, and Speech options. Cancel an unchanged
form; discard a changed form; inject a failed save and retry. Verify selections and
unsaved task/model edits survive Keep editing, Save failures do not advance,
and Discard applies no discarded edits. Test keyboard entry, Escape,
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
For HTML injected by a browser test, import shared Svelte and singleton state
through a fixture module processed by Vite, such as `presentation-runtime.ts`.
Raw inline imports can otherwise create a second singleton when Vite adds a
module-version URL to the production component's import.

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

The configuration browser suite covers normal/narrow content scrolling, keyboard
section changes, persistent connection actions, and validation that reveals the
owning contextual section. Controlled save failures carry the Go validation cause shape;
assertions cover the destination, focus, associated message, section marker,
retained unrelated edits, and retry. Go config/settings fixtures separately
verify both recording-limit modes, nested validation causes, safe error JSON,
and rejection before applied settings, credentials, native adapters, or events
can change. Frontend metadata fixtures distinguish independent STT/TTS outcomes
and prevent unsaved-draft checks from being attributed to applied settings.

`visual-hierarchy.spec.ts` mounts the actual Settings screen in light and dark
modes at desktop and narrow widths. It checks enabled primary-action contrast in
normal and hover states, reachable save actions, horizontal overflow, and
keyboard access to shortcut rules. `builtin-connections.spec.ts` covers compact
resource rows, selected-connection semantics, visible runtime controls, and
expansion into the model catalog; opening these views must issue no runtime
commands. Their screenshots support visual review rather than replacing
behavioral assertions. Inspect heading and label hierarchy, readable secondary
text, status labels, and disclosure focus alongside the existing navigation,
shortcut-recovery, source-provenance, and managed-runtime interaction suites.
Apply the same visual review to Voice, audio files, Text to speech, History and
its detail pane, Connections, runtimes, and global/contextual Settings. Check
that key values stand out from helper text, neutral grouping surfaces remain
subtle in both themes, and section icons supplement visible labels. Exercise
narrow widths and zoom so added hierarchy does not hide actions or reduce
transcript readability.
Keep checks tied to user outcomes rather than exact class names or palette values.

Runtime catalog coverage checks the same browsing controls before and after
installation, family expansion and search, keyboard access to model sources,
and disabled model actions while uninstalled or running. Browsing must make no
runtime calls. Offline Go tests cover the published Whisper variant inventory,
pin structure, completed-transcription qualification, and removal that preserves
sibling models; they do not download or invoke catalog models.

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
discards output. With only an explicitly selected model, observe a successful
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
abrupt subprocess exits during initialization, writes, and forward upgrades.
Check file isolation on both platforms: `settings.db` and `settings.json`
remain untouched, legacy credentials are never read or deleted, and `freehand.db`
starts with defaults and no connections. Renamed foreign databases must fail identity
validation. Temporary Git fixtures reject edits, removal, or renaming of published
`schema/` migrations, new versions below the published maximum, and nontransactional
annotations in every accepted letter case. Backup tests block source access to
verify that incomplete copies are never published, cancel that work, and restore
the completed snapshot after retention ignores temporary and unrelated files.
Catalog tests fill all four manual allowances alongside the maximum managed
inventory and reopen it through the real storage owner. Inject native-vault write
and deletion failures to exercise rollback, deferred cleanup, and cleanup-cap recovery.
Check backups,
constraints, lock waits, read-only/full-disk failures, incompatible history, and
staged credential consistency. Run the storage/settings race tests on Windows;
CI also checks portable storage fixtures on Linux. Native service fixtures use a
fake vault/startup adapter and do not establish interactive Windows acceptance.

## Unit and deterministic integration tests

- Configuration validation, URL joining, header filtering, fresh initialization with legacy-file isolation, and explicit database recovery without implicit replacement.
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
- Main window identity, startup visibility, single-shell Settings navigation, retained draft preservation, section/origin validation, guarded shell close-to-hide behavior, light/dark solid native colors, explicit Mica opt-in, launch-time versus saved-material state, main-window placement restoration, missing-display fallback, and owner-relative About/history-details geometry.
- Tray presentation mapping for live dictation, post-processing, VAD/silence, checkpoint, stored-file upload/streaming, completion, failure, cancellation, copy recovery, last activity, and main-window visibility. Tests must prove arbitrary renderer messages, transcript text, and file identity cannot enter labels or tooltips.
- Overlay preference and appearance defaults, sparse-config compatibility, bounds validation, post-persistence runtime application, DPI/opacity/glow composition, idempotent live enable/disable/configure, current-state restoration, and shutdown fencing.
- Frontend feature-owner tests beside `editor`, `files`, `history`, `speech`, and `messages` cover committed-settings synchronization, active-draft preservation, serialized quick saves, transient credential cleanup, file delta/snapshot ordering, history mutation ordering, and playback commands. `session.svelte.test.ts` covers composition, namespaced status bindings, aggregate busy state, and presentation-only teardown. `session-events.test.ts` executes the shared main/Settings subscription wiring with a typed event source: subscribe-before-snapshot, accepted-transition history refresh, stale-event rejection, delta-gap recovery, independent window drafts, unsubscribe/remount, and partial-registration cleanup. These are deterministic renderer tests, not native Windows acceptance.
- The main view's transcript-list disclosure preference persists across launches, uses a safe default for missing or malformed values, and remains usable when WebView storage throws.
- Immediate controls in first-run readiness and explicit workflow mode switches start from the applied snapshot, mutate only allowed task/model fields, send no credential changes, replace both snapshots only after backend confirmation, and preserve the applied state on failure. Workflow sidebar rows open contextual options; their edits remain drafts until Save succeeds. Endpoint and credential edits use the Connections rail page. Settings transaction tests block persistence after native and credential mutation to prove renderer snapshots wait for a coherent commit, exercise reentrant post-commit publication, and inject shortcut, startup, STT credential, post-processing credential, persistence, and rollback failures.
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

Run these checks on a non-elevated Windows desktop with disposable settings and
explicitly selected inference endpoints. Record actual results separately; this
list is an acceptance procedure, not a claim that the checks have passed:

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
17. From the Connections rail page, test the displayed endpoint with both a stored credential and an unsaved bounded credential draft. Confirm status/timing is scoped to that connection. In workflow model settings, verify metadata discovery, selection without inference, and actionable DNS, TLS, HTTP, and offline failures without response bodies or key material.
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
30. From each workflow sidebar, open the corresponding right-side options to select saved connections and discovered models, and switch between **Custom instruction** and **S1-mini by Superwhisper** under Cleanup. Edit endpoint/authentication details only on the Connections page. Confirm contextual edits remain drafts until **Save** succeeds and the main workflow remains visible. A failed save must retain the editable draft, preserve the prior applied configuration, and report an actionable error. Discard must restore the applied values. In Cleanup settings, edit the custom instruction, exercise empty and multibyte-over-limit validation, restore the recommended instruction, save, restart, and confirm it persisted. Switch to S1-mini and confirm the custom editor disappears, the exact built-in instruction is read-only, and the effective control line changes with every styling step plus both structure and context radio choices. Switch back and confirm the custom instruction was preserved. Committed connection changes must invalidate stale metadata. Model pickers may perform bounded first-entry discovery; deliberate re-entry or explicit refresh may retry, but ordinary effects must not loop after failure.
31. Run the development app from a terminal and exercise recording start/stop/cancel, one VAD checkpoint, post-processing fallback, stored-file selection/transcription/cancel, both connection tests, settings save, and tray Quit. Confirm lifecycle records have stable components/correlation fields, use `duration_ms`, and produce one terminal outcome per start. Search the output for the real test credential, transcript phrases, model IDs, selected file name/path, endpoint path/query, custom headers, and target-window identity; none may appear. Confirm a failed title-bar action is visible in the app rather than only the WebView console.
32. In **Settings → Overlay**, preview all four layouts, six anchors, three surfaces, four recording visualizers, three visibility policies, and both motion policies before saving. Confirm the real native surface cycles through speech, silence, countdown, transcription, post-processing, delivery, copy-required, and failure; draft controls update the same HWND; closing/discarding restores applied settings; and starting a real dictation preempts preview. Turn the saved overlay off and confirm preview can temporarily create it but Stop/Settings close destroys it. For real dictation, confirm the target monitor is captured at recording start, placement stays inside that monitor's work area at 100%/150%/200% DPI and with taskbars on every edge, and focus changes do not move it mid-operation. Exercise minimum/default/maximum size, opacity, edge distance, and glow without entrance replay, extra taps, or GDI/thread leaks. Windows Animation Effects off and Reduced must stop decorative motion while the countdown remains live; a Windows contrast theme must force a readable opaque system palette. Detailed must show only fixed labels, the normalized shortcut, bounded elapsed time, and checkpoint count—never transcript, filename, provider, endpoint/model, prompt, credential, or raw error content. Across all cases verify unchanged focus/caret/target identity, click-through behavior, no taskbar/Alt+Tab entry, bounded shutdown, and ordinary dictation/insertion behavior.
33. Navigate the activity rail, command palette, Settings pane and Connections editor, native About and Transcription details windows, and remaining dialogs using only the keyboard. Verify Tab/Shift+Tab, Enter, Space, Escape, Alt+F4, rail focus, panel Arrow/Home/End navigation, and Ctrl+K (⌘K on macOS). Every Settings entry point must resolve to the requested section in Main. Dirty exits through the rail, palette, runtime links, native task requests, Done, and close offer Save, Discard, or Keep editing; failed saves retain the draft. Hiding clears credential drafts and displayed runtime output, disables output reads, and stops shortcut capture/overlay preview. Reopening a visible Runtime output tab begins fresh reads for its retained target. Connection creation's Save and return resumes the originating workflow. Contextual Save applies in place; Done closes the options. About and Transcription details reuse their own windows and hide from native close or footer. With Narrator, state changes are announced without repeated timer or transcript readings. Repeat with Windows Animation Effects disabled and in a contrast theme; countdowns continue while decorative motion stops and keyboard focus remains visible.
34. Open About in a development build and a packaged build. Confirm its compact metadata matches `build/config.yml`, the executable's Details tab, and Installed apps; only the development build shows **Development**. Run `wails3 task common:check:release-info`, deliberately make one generated Windows version field stale, and confirm the check and package build fail until `wails3 task common:update:build-assets` repairs it.
35. Move and resize the main window on a non-primary display. Open Settings and confirm it stays inside the main workspace. Open About and Transcription details, and confirm each hidden auxiliary window opens centered over the main window without leaving that display's usable work area. Move an already-open auxiliary window and invoke it again; confirm it is focused without jumping. Hide and reopen it; confirm it returns relative to the main window rather than retaining independent placement. Choose tray Quit, relaunch, and confirm only the main window restores its normal size and screen-relative position. Then disconnect the saved display and relaunch; confirm the main window is fully visible and centered on the primary work area.
36. In direct-input mode, compare short, long, multiline, and non-ASCII transcripts in Notepad, a Chromium text field, VS Code or another editor, a terminal, and an Office-style rich-text target. Ordinary text should appear in one immediate update; long text should complete without visible fixed-delay stepping, truncation, or broken surrogate pairs. Change focus during a long insertion and confirm delivery stops before the next dispatch rather than redirecting its remainder. Confirm the terminal records only UTF-16 unit count, batch count, duration, strategy, and bounded failure stage—never text or target identity.
37. Toggle the primary sidebar, bottom panel, and contextual secondary sidebar from the title bar; drag both dividers and resize them with the keyboard. Quit through the tray and relaunch to check visibility, size, and selected bottom-tab restoration. Cross the 700px width, 1100px width, and 560px height thresholds: primary navigation becomes an explicit overlay, History details auto-hide but remain reachable through the right-side overlay, and the bottom panel hides without losing its tab or visibility preference. Details must never stack below the reader. Check independent scrolling, overlay Escape/backdrop dismissal, focus return, and restoration when room returns. Open task options from the sidebar and confirm the transcript or composer stays visible. Exercise nested selectors, Save/Discard/Keep editing, pending saves, and failed-save recovery in light/dark and opaque/Mica modes. Clearing WebView site data may reset presentation preferences but must not alter Go-owned settings or retain history/output.
38. In an isolated user-data directory, place `settings.db` and `settings.json` fixtures before launch. Confirm first-run defaults and an empty connection catalog, with those files and legacy credentials untouched. Save and reopen `freehand.db`. Exercise corrupt/newer SQLite, foreign database identity, locked files, and uncertain save recovery: the workspace must pause new work and ordinary saves; Retry reloads committed state, and only explicit Reset archives and replaces the current database. Restore a `freehand.db` upgrade backup with Freehand closed. Verify credential references remain coherent without exposing keys.
39. Configure a dedicated local or remote `/v1/audio/speech` connection on the Connections page and select it in **Text to speech → Speech** options. Press **Test**, confirm the authenticated `GET /v1/models` result populates the model picker, enter the provider's voice ID, and save. Then press **Preview**. Verify the fixed preview phrase plays, pause/resume preserve progress, restart begins at zero, and stop releases the session.
40. Enable History, create raw-only and successfully cleaned entries, and verify Listen reads the selected final version. Complete a stored-audio transcript with History off and verify its result can still be listened to. Start a toggle or hold recording during playback and confirm playback stops before capture begins without transcript/history mutation.
41. Under **Voice**, **Audio file**, **Cleanup**, and **Text to speech**, confirm the saved microphone/checkpoint, stored-audio, cleanup, and speech-generation budgets reload exactly and the fixed safety ceilings remain visible but not editable. Against a deliberately slow endpoint, set each budget low and confirm the affected phase reports a timeout, logs bounded `error_kind=timeout`, and records the budget in opt-in History details. A cleanup timeout must still insert or expose the raw transcript and mark the fallback. A stored-file request configured above 90 seconds must remain active beyond 90 seconds; Cancel must still end immediately as cancellation rather than timeout. Exercise the streamed transcript safety ceiling with a deterministic fixture and confirm accepted text remains copyable under an explicit partial-result message rather than stopping silently.
42. In a packaged build, verify About reports automatic updates enabled, **Check now** opens the Wails update window, and declining leaves the running executable unchanged. Disable automatic checks, restart, and confirm no background request occurs. Using a controlled newer release, verify the exact `freehand-windows-amd64.exe` asset passes `SHA256SUMS`, stages successfully, and restarts into the version shown by About. Exercise both an alpha-to-non-prerelease upgrade and a later non-prerelease upgrade. A checksum mismatch must fail closed. Confirm this direct-binary path does not claim to run or update through NSIS.

## Model safety

CI and native acceptance must not enumerate models and then call them. Model-list requests are metadata only. Automated tests use fakes or in-process protocol fixtures; real inference is limited to one route and model deliberately selected by the user for the workflow under test. Post-processing acceptance may call its separately selected model only after the user deliberately enables and runs that stage.

### Optional transcription controls

Request fixtures cover microphone, completed-file, and typed-streamed-file
requests with omitted controls, explicit zero temperature, Unicode context, and
Speaches hotwords. They verify exact fields, unchanged audio, Content-Length,
and one request per operation. Invalid controls must fail before transport or
file reads. Persistence fixtures cover clean-baseline defaults and an
inactive retained temperature. Workflow fixtures verify forwarding through the
recorder and file service, and checkpoint snapshots after settings changes.
Frontend coverage protects separation of unsaved and applied nested options.

For native acceptance, use only an explicitly chosen model. Compare baseline
and short context/hotword samples, confirm zero versus default temperature, save
and reopen settings, and test one microphone/checkpoint and stored-file workflow.
Confirm unsupported combinations omit shared vocabulary without erasing the list
or use preference, and clearing the numeric input cannot silently save zero. Save new hints during a recording
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
- Verify Settings, first-run readiness, workflow sidebar navigation, profile switching,
  and the public backend matrix. Include vLLM-Omni speech using the Qwen3-TTS
  acceptance steps below.

A Windows compile and fixture tests are separate from interactive acceptance
against a live deployment. Metadata success does not prove inference behavior.

## Language selection acceptance

Fixture tests cover default/automatic/named/custom multipart fields across the
Generic, Speaches, whisper.cpp, and vLLM completed adapters for microphone and files, including
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
and clean-baseline round trips. A vLLM deployment may implement one or both offered operations;
no automatic inference checks may be used to discover that.

## Saved connection acceptance

Fixtures cover the clean schema, empty initialization, inactive create/duplicate,
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
Repeat selection in each workflow's contextual options, including role
filtering and a failed Save. Confirm failed saves retain the draft,
successful saves update the current workflow, and sidebar summaries reflect
the applied values. Browser fixtures and native builds do not replace interactive Windows acceptance.
Use only explicitly selected models for deliberate live inference acceptance.

For connection navigation, open Connections from the rail and Manage connections from
each workflow picker: both open the list. Edit connection opens the selected row.
Keep the searchable list in the shared primary sidebar; below 700px,
its title-bar toggle must expose the compact list overlay. Verify all seven representative entries
fit comfortably in a normal window, list and form scroll independently, and Save
and Cancel remain visible at compact heights. Check both action menus for readable
single-line labels, workflow icons/checkmarks, and compact disabled Delete state.

Change a field, then switch rows, use All connections, or close: Keep editing and
Escape retain the draft, Discard clears it without saving, and Save
completes the pending navigation only after a successful save. A failed save keeps
the draft and shows the error. Repeat with a transient replacement key. Add from a
workflow, Save and return, and confirm the originating model settings resume. Creating
with Save for later must leave active selections unchanged. Use for selects only
its chosen workflow, while existing model settings and unsaved Settings drafts
retain their protections. Check keyboard navigation, searchable pickers, and list
empty/no-match states. Metadata discovery must target only the opened workflow,
reuse results, permit explicit refresh or deliberate picker re-entry after failure, and reject old completions
when the selected connection changes.

For dark-palette review, check the main window, Settings, About, connection/model
menus, dialogs, inputs, focused controls, disabled controls, and recording overlay.
Confirm neutral charcoal surfaces and brand-blue accents in solid dark mode, readable muted text,
and unchanged status colours. Compare light mode and dark Mica over both light
and dark desktop backgrounds; the Mica result depends on Windows and wallpaper.
Check the workspace title bar, auxiliary native active/inactive captions, and
startup background after relaunch.
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
selection, S1-mini prompt/reasoning/language regressions, and baseline round trips
with independent selections and credential references. New specialized model
profiles need per-role request fixtures and explicit runtime qualification; a
catalog entry alone does not establish support. Do not probe model inventories.

## Remembered model acceptance

- Save distinct model choices for Voice, files, cleanup, and speech on one
  shared connection. Switch away and back; verify engine options, model profile,
  and voice restore only for the matching use and model. Language, cleanup
  instructions, trained S1-mini controls, speaking speed, and shared vocabulary
  must preserve current task settings, which are absent from remembered-model
  rows.
- A new model ID starts from Generic defaults. Verify changing a name does not
  infer S1-mini, and model edits leave capture settings and timeouts unchanged.
- Switch freely between modified model drafts, save all edited options together,
  and test invalid batch rollback. Discard must restore applied values and clear
  every model draft; saved catalog objects must not mutate through a draft.
- Verify first-run readiness model controls restore options before saving, without
  discovery or inference. Workflow sidebar model rows open Settings, whose pickers
  also list remembered IDs without a server probe.
- Switch connections, restart, rename, rotate a key, change the URL/backend,
  duplicate, remove an inactive use, and delete: check the documented retention
  rules. Forgetting must clear active selection and survive restart.
- Exercise baseline round trips, failed SQL saves, stale forget requests, role
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
- Review the results panel in light/dark themes and a narrow Settings pane.
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

For native Kokoro playback, use one explicitly selected model/voice and verify
WAV decoding and speaker-device preview separately from metadata discovery.
Keep generated test audio memory-only; do not run live synthesis in CI.
Verify the site matrix exposes accessible support labels with visual checkmarks
and links the Kokoro guide from the directory and documentation navigation.

### Consolidation regression coverage

Settings tests block a runtime callback while a second save attempts to commit,
then assert ordered retention updates and callback-safe settings reads. TTS tests
cover stale completion, Stop fencing, and native save-dialog interleavings with Stop
and shutdown. Windows CI runs the complete Go suite, including platform-specific
input, playback, storage, and settings tests; these do not invoke inference servers.

Current-result tests cover copy/clear and Voice playback generation admission without
history, including stale, active, cleared, and closed results. Fake speech clients
verify backend-owned Voice text selection without retaining history and the 4,096
Unicode code-point boundary, including supplementary characters.
Browser workspace fixtures cover desktop pointer/keyboard resizing, restored
pane sizes and visibility, responsive region hiding, and task-sidebar links with
asynchronous save outcomes in contextual configuration. They use the actual home
components with mocked Wails services and no inference traffic. Workspace checks
also cover Voice Listen with history off, hover/keyboard action tooltips, expiring
copy confirmation, and preservation of supplementary Unicode in the composer.
Copy-feedback unit tests cover repeated clicks, out-of-order completion, failure,
and teardown.
Layout restoration checks wait for the saved size to match the separator's final
value before reloading, then check the restored geometry and visibility. Inspect
the stored payload: only layout booleans, bounded sizes, and the bottom tab belong
there, never configuration drafts, credentials, transcript text, History search/selection,
output runtime identity, consent, or output bytes.
Renderer coverage checks that speech commands preserve the composer draft and
that file readiness excludes microphone and shortcut prerequisites while retaining
endpoint and authentication checks. Interactive acceptance should switch between
all three tasks and Settings, confirm unsent text survives, then verify current
results can be copied and cleared with history disabled.

### Transcription details window

Open Transcription details from Recent. Verify one independently resizable native window opens without blocking the workspace. Open a different completed entry while details is visible or minimized: the same window must update and focus. Check long model names and response metadata at the minimum 480 x 400 size, light/dark appearance, and Mica after restart. Native close, Alt+F4, Escape, and the Close button must hide details without quitting Freehand; reopening and renderer reload must recover the selected run correctly. Delete the selected entry, clear history, disable retention, and evict the entry with later runs: stale details must disappear. Automated history service tests cover invalid/pending IDs, selection switching, deep-copy isolation, removal, and close/shutdown behavior. Native runtime acceptance remains separate from the Windows compile.

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

Fresh-store fixtures exercise completed and realtime Voice configurations, independent file selection, and remembered model options without importing legacy data. Completed Voice profile tests isolate endpoint, model, options, headers, and credential snapshots from Audio file. Backend eligibility tests must reject realtime on Generic or a mismatched server/profile.

For native acceptance, open Voice → Transcription: there must be no separate Live button. Choose NeMo-Speech.cpp, its loaded model, and the explicit Nemotron profile; enable Realtime inside that panel. Verify live results and one-row captions. Turn realtime off and record using the same connection/model. Switch to an ineligible model/connection and verify mode is disabled. Configure Audio file separately, switch between tasks, and verify independent connection/model/language settings and truthful footer status. Restart and repeat. Test Voice-only first-run setup with Audio file unconfigured. These native checks are separate from successful builds and deterministic tests.

Language dropdown acceptance: open contextual Transcription options from the workflow sidebar or title-bar right toggle and check the searchable language picker and qualified Nemotron select at short and normal window heights. Repeat in first-run readiness, where available controls save immediately. Menus must remain within the window, scroll internally with the wheel, and expose the final option through keyboard navigation.

### Shared vocabulary acceptance

Check Vocabulary at regular and small workspace sizes in both themes. Edit names, independently toggle Voice/files, save, and reopen. Follow Vocabulary links from a dirty Voice/file settings draft and verify that edits survive navigation. Confirm the NVIDIA mark appears for NeMo in quick controls, connections, and the Vocabulary page. Switch supported/unsupported models and preserve the list and use preferences. For NeMo, verify limits and strength, then test completed Voice, audio files, and realtime against only the explicitly selected model when native inference acceptance is authorized. Backend fixtures cover fresh-schema round trips, immutable workflow projections, Unicode limits, omission, and multipart speech contexts without automatic inference.

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
and restricted language menus in first-run readiness and Settings. Successful builds
and synthetic transport checks do not substitute for native microphone acceptance.

## Audio file and speech workspace layout

File streaming state tests cover enabled and disabled preferences across file and
capability changes, explicit request flags, failed resets, and duplicate/pending
reset admission. `file-streaming.spec.ts` checks tab remounts, simulated connection
capabilities, completed-only profiles, and **Try streaming** enabling the next
explicit request without uploading automatically. Native endpoint support remains
owned by the existing Go capability checks; these renderer fixtures invoke no models.
Deferred-command tests cover duplicate/conflicting file actions, failure recovery,
cancellation while the Start reply is pending, and older/same/newer-generation picker
replies. The file browser fixture also delays Start to check immediate **Starting…**
feedback, disabled conflicting controls, restored controls after rejection, and
transition to the backend-owned Cancel action after admission.

Capture-clock tests cover missing/invalid/Go-zero start times, generation changes,
and freezing the last take across transcription, cleanup, and failure.
`capture-reading.spec.ts` reproduces the recording-to-transcribing zero-time
transition at compact width and verifies no growth during processing. Its short
viewport cases assert that Jump to latest sits outside the transcript scroll area,
preserves reading position during updates/finalization, and still scrolls to the
end when activated.

Check Audio file and Text to speech at normal and compact desktop sizes, including
1156×760 and 650×550. Use synthetic renderer fixtures for selected, busy, completed,
and error states without invoking inference. File summary, response-mode switch,
actions, and result pane keep their positions across these transitions. Long file
names truncate without displacing actions; errors remain readable in the result.
The speech editor and its bottom playback area retain their heights across idle,
generating, playing, paused, completed, and failed states. Open Speech options from the
Text to speech Settings cog or sidebar, check its connection/model selectors and Escape focus return,
and verify failed saves retain drafts while Save applies in place and Done closes options without clearing the composer.
Check visible keyboard focus in both themes. The application Settings button must
remain on screen when the shortcut hint is hidden at compact widths.

### Transcript selection and keyboard reading

`transcript-reader.spec.ts` exercises scoped Ctrl+A, a native Ctrl+C event for a
partial selection, Page Up/Down and Home/End scrolling, and selection preservation
through streaming and finalization at 560px and 1156px. Copy events are intercepted
only in the fixture to avoid replacing the tester's system clipboard. History
checks cover deferred cleanup updates and independent raw/cleaned selection.
New recordings clear old selections. Native review should confirm copying a
selected excerpt into another application, including Unicode text.

### Listen request feedback

`listen-pending.spec.ts` delays voice, file, and history Listen bindings at 560px
and 1156px. It checks immediate feedback without button-width changes, duplicate
click suppression, continued text/copy access, rejection/retry, and re-enabling
after generation. Speech store tests exercise cross-source admission, release
after rejection, and continued replacement from playing/paused/completed states.
All requests are synthetic; these tests do not invoke inference.

### Saving generated audio

`save-audio-pending.spec.ts` holds a synthetic SaveAudio response open in compact
and embedded playback at 560px and 1000px. It verifies visible pending feedback,
duplicate-click suppression, stable player height, usable pause/resume/seek,
and recovery after cancellation, failure, and success. Store tests additionally
keep the guard across playback-generation changes and verify duplicate requests
do not clear other feedback. These fixtures do not open a native save dialog or
write audio. Native review should confirm cancellation and retry with the Windows
dialog and a chosen WAV destination.

### Speech seeking and composer shortcut

Speech composer fixtures verify editing during generation, playback, and pause,
clearing the draft without releasing audio, and explicit replacement with the next
submitted text. Ctrl+Enter cannot start a second generation while one is running.
History expansion fixtures exercise real mouse-wheel events over the list at 560px
and 1156px, scrolling in both directions, same-entry updates preserving the reading
position, and newest-entry arrival returning the compact list viewport to the top.
In the full History pane, check sidebar-list scrolling independently of the selected
reader and run details, and verify the compact sidebar overlay leaves the main
area usable after selection.

Transcript playback fixtures at 560px and 1000px verify a single row no taller than
48px, keyboard seeking, pause/resume and Stop visibility, keyboard access to
Restart and Save through Playback actions, and clearing completed audio.

`speech-playback.spec.ts` uses synthetic renderer fixtures to check stable generation
and playback geometry at 560px and 1000px, keyboard seeking and retained focus,
drag previews, one commit on release, and rejection of a drag after audio replacement.
Irregular native progress fixtures exercise updates between slider steps before and
after seeking. Geometry assertions check visible fill height, full track width, and
alignment between the filled range and thumb. This catches a frozen display even
when the underlying seek request succeeds. Light/dark fixtures compare standalone
playback surfaces and rounded borders with the adjacent result card; embedded
playback uses its parent's frame.
It also checks Enter/newline and Ctrl+Enter submission, including empty, oversized,
composing, and repeated input. Frontend state tests cover delayed responses and
duplicate submissions. These checks do not invoke inference.

Go seek tests cover playing, paused, and completed intent, stale generations,
out-of-range values, native errors, resume failure, and shutdown during a blocked
seek. Windows adapter tests check whole-frame alignment and unchanged full WAV export.
For an opt-in real Windows output check, set `FREEHAND_NATIVE_PLAYBACK_ACCEPTANCE=1`
and run `go test ./internal/platform -run '^TestNativePlaybackSeek$' -count=1 -v`.
This plays synthetic silence only and checks seek, paused position, resumed clock,
buffer drain, and resource closure. It does not exercise a microphone or inference
server and does not establish audible speech quality. For interactive acceptance,
use explicitly generated speech to review seeking while playing and paused, replay
after completion, full-audio export, and recording preemption.

## Speech controls, vocabulary feedback, and transcript reading

Connection-manager tests retain separate results, reject missing IDs, preserve active
selections, and invalidate cached/in-flight checks on confirmed settings snapshots,
even when public endpoint fields are unchanged. Model-source tests cover combined
saved/server/draft labels and manually entered IDs. Navigation tests keep keyboard
order aligned with the displayed groups.

Review Connections at normal and compact window sizes: list-first navigation,
collapsed diagnostics for the selected entry, active-use summaries, and independent
list/form scrolling. Open the shared model picker from every workflow's Settings
section and first-run readiness,
including a failed save, retry, manual ID, and long list. Sticky Settings headings
must leave focused validation controls visible. These checks use metadata or
synthetic fixtures and require no inference inventory probes.

Run the config/modelprofile/settings tests and frontend suite. Vocabulary cases
cover UTF-8 phrase limits, exact duplicates with source line numbers, the first
excess phrase and byte budget, context consumption, unsupported workflows, and
bounded oversized-draft feedback. Speech quick-save tests cover remembered voice
restoration, task-speed preservation, credential/draft exclusion, and failed saves.

The optional `polish.spec.ts` browser checks exercise speech sidebar navigation to
Settings, voice/model/speed save, failure, retry, discard, and composer preservation
at a short desktop size, plus live text following, reader scrollback, finalization,
Jump to latest, and the next recording. Fixtures contain synthetic text and invoke
no inference. Recheck the shared model/voice menus and speed slider in Settings,
including keyboard entry and nested-menu scrolling.

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
SQLite fixtures exercise the clean baseline, Unicode speech options, restart,
and transaction rollback. Preview tests prove unsaved language
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
   ten languages, style field, and keyboard navigation in Settings, including entry
   from the Text to speech sidebar.
   Edit language/style and Preview before Save. Verify ordinary playback retains
   applied settings until Save; Discard restores them. Switch models and back,
   restart, and verify saved options remain independent per connection/model.
5. Metadata refreshes must never generate speech or transcribe a sample. Record
   actual inference acceptance separately from fixture and Windows build results.

### Settings presentation and discovery

The settings streamlining browser fixture exercises keyword navigation with an
unsaved draft, clickable switch labels, collapsed Audio/Overlay tuning, keyboard
slider changes, and draft preservation after navigation. A synthetic validation
failure on speech padding must reopen its disclosure and focus the slider at
both desktop and narrow widths. Minimal overlay surfaces must disable glow
adjustment without resetting its value. Local runtime must appear neither in the
Settings section list nor its search results; the activity-rail action must still
open runtime management through the normal draft guard. Fixtures use no capture or inference.

The workflow streamlining fixture checks all four workflow pages at 860px and
520px: request controls start collapsed, edits survive collapsing and navigation,
and save validation reveals hidden request limits. Voice workflow validation
also exposes its temperature and timeout controls. Speech preview stays beside
voice selection. All workflow fixtures use synthetic profiles and service
responses; they do not contact inference servers.

### Main workspace title bar

Browser fixtures verify the Windows File/View/Help menu actions, caption-button
labels and binding calls, maximize/restore state updates, keyboard operation,
and close-time draft decisions. macOS presentation fixtures verify reserved
traffic-light space and the absence of duplicate renderer caption controls.
Inspect drag-region CSS so command, menu, and layout controls remain outside
Windows caption rectangles and opt out of inherited dragging on macOS.

Run the [native title-bar checks](../../safety/native-test-checklist/#main-workspace-title-bar)
on each platform. A passing browser suite or Windows executable compile cannot
establish native hit testing, Snap Layouts, mixed-DPI dragging, macOS traffic-light
positioning, fullscreen transitions, or close-to-hide event delivery. Record
Windows interactive evidence separately from packaged macOS acceptance; macOS
remains unverified until those checks run there. Recheck the framed About,
Transcription details, and standalone Process output windows independently.

### Shared workspace layout

Exercise the title-bar primary, bottom, and secondary visibility controls across
Voice, Audio file, Text to speech, Connections, Settings, Runtimes, and History. Primary content
must follow the area without resetting the wide-window visibility preference.
Below 700px wide, it starts hidden and opens as an overlay through the same
title-bar control. Check Escape, backdrop dismissal, page navigation, focus on
opening, and focus return when a region hides or crosses a breakpoint.

The bottom panel retains Recent, Runtime output, or Diagnostics across page
changes and user hiding. Global Settings must use the full vertical workspace
without rendering the bottom panel. Returning to another page restores the selected
tab and the user's visibility preference, including an intentionally hidden panel.
This suppression must not rewrite stored layout choices. The panel automatically
hides below 560px viewport height and restores the user's visibility choice and
tab when room returns. Contextual options
and History details occupy only the secondary sidebar: auto-hide them below 1100px and open them on
demand as a right-side overlay through the title-bar toggle. Check Escape and
backdrop dismissal; never stack details below the transcript, and restore the
wide-window preference on widening. History opens details on each visit at
1100px or wider, even if they were closed on an earlier visit. Closing must remain
respected through selection and history updates within the visit. Its page-header
Details button opens the selected run's details at all widths; narrower windows
must start with details hidden. Check both thin
dividers' larger pointer targets, drag limits, keyboard arrows, Home/End limits,
separator names, orientation, and focus indicators.

Check the contextual section sets: Voice has Transcription, Audio, Cleanup,
Vocabulary, Overlay, and Delivery; Audio file has Transcription, Cleanup, and
Vocabulary; Text to speech has Speech; History has
Details and History settings. Editing keeps the main content visible. Switching
pages closes contextual options, and the right toggle opens the current workflow's
options or History details. Test Save, Discard, and Keep editing on navigation and
close, including a failed or pending save. Every contextual sidebar must expose
one reachable X in its existing header, including when section tabs scroll.
Check keyboard activation and focus return to the title-bar toggle, and verify
the X and History's page-header Details action use the same draft guard.
`right-sidebar-controls.spec.ts` covers these controls and History's responsive
opening policy alongside the existing layout and configuration fixtures.
Global Settings must not duplicate
workflow or connection sections; Connections opens from its own rail action.
Shared Vocabulary and Overlay options must identify their scope, stay beside the
workflow when opened locally, and reflect the same saved values in global Settings.
Voice's Delivery view must expose only microphone transcript delivery, excluding
startup and appearance. Global General, Shortcuts, Overlay, and Vocabulary remain
reachable. Verify shared-control drafts use the same save/discard guards without
creating a second settings copy in layout storage.
Unconfigured speech must keep its draft editable beside compact Configure speech
guidance, with Speak disabled until configured.

Open Runtime output and verify the initial target is a running runtime, or the
first installed runtime if none is running. Output must appear directly without
a Show output step. Select another runtime's tab by pointer and keyboard, then
switch pages: navigation must not choose another runtime. Entering global Settings
must release the reader because the panel is hidden there; returning retains the
runtime target and resumes direct reads. Hiding the bottom panel, changing panel
tabs or runtime targets, hiding the workspace or document, and teardown must
disable retrieval, clear viewer output, and fence late reads. No reader may enable
while hidden. Reopening starts fresh reads for the retained target, whose private
tail may still be available. Removing the selected runtime must not silently pick
another. The standalone Process output window still requires Show output on each
opening or runtime switch. Diagnostics checks use the
explicitly selected workflow's saved connection and remain metadata-only.
Browser fixtures establish renderer behavior; native window-hide events,
clipboard behavior, and runtime process lifecycles require separate OS acceptance.

### Workspace history and error presentation

The full History pane must combine retention status, History settings, Clear history,
search, source filters, and retained transcript previews in one sidebar. Exercise
matches in final, raw, and processed text and file base names, including text absent
from the displayed preview. Combine search with All, Voice, and Audio files filters;
check no-match feedback and returning to the complete list. These controls browse
only the existing in-memory entries and must not persist transcript text or trigger
metadata discovery, inference, or file reads.

Select an entry and verify its expanded reader, Copy, optional Listen, raw/cleaned
comparison, and removal remain available beside the same run's full details.
Switch between Details and History settings without replacing the selected reader;
retention edits use the same guarded save and discard lifecycle.
Neither selecting an entry nor browsing its details should call the native
details-window bindings. Check selection after cleanup
updates, new arrivals, deletion, eviction, disabling retention, and clearing the
full history while a filter hides some entries. At widths below 700px, exercise
the sidebar toggle, keyboard access, overlay dismissal, and selecting an entry
without trapping or obscuring the reader. The right details sidebar must be
available at 1100px and wider and automatically hide below that width without
stacking or resetting its wide-window visibility preference. Its title-bar
toggle must still expose an inspectable right-side overlay at narrow widths.
Verify pointer and keyboard
resizing. The list, reader, and details must scroll independently
without an outer page scrollbar. At a 360px details-pane width, retain readable
long identifiers, model names, timestamps, usage/cost/performance values, and
every checkpoint field. Check component-scoped heading associations and the
embedded 44px header, then verify the standalone details window from Recent
still exposes the same fields and lifecycle.

Compact Recent history lists retain their existing expansion checks:
a fully readable newest result, single-line older previews, keyboard disclosure,
new arrivals resetting manual expansion/comparison, cleanup updates preserving it,
deleting the newest entry, clearing/repopulating, and a full unretained file result.
Native acceptance should confirm the same transitions after real recordings and
cleanup completion.

Synthetic workspace checks exercise history action menus with keyboard opening,
Escape focus restoration, deletion of the selected entry, and raw/cleaned copy.
At 560px and 1156px they verify compact history footers and visible cleanup
fallback explanations. File error details must remain inside the viewport,
preserve transport geometry, and disappear when retry starts. Retained history
and file status are fixture data; these checks invoke no native inference.

### Shared settings pickers

Synthetic picker tests cover keyboard access to help, nested-popover Escape
focus restoration, profile descriptions at desktop and narrow widths, and
restricted-language search by code with navigation to the end of a scrollable
list. They verify that draft selections survive page navigation, custom language
values remain reachable after an unmatched search, and failed Voice Settings saves
retain the draft profile/language while applied values remain unchanged. A successful
retry returns to Voice; reopening Settings must show the saved values. Fixture profiles are renderer data only;
no model inventory or inference call is involved.

### Vocabulary presentation

Synthetic component checks at 520px and 1000px cover visible editor/workflow
controls, collapsed tuning and feedback, keyboard help and focus return, edits
surviving disclosure, independent opt-ins, source-line selection, unsupported
selection explanations, retry without draft loss, stale preview rejection, and
UTF-8 byte overflow. Preview responses are fixtures; Go vocabulary tests remain
the authority for actual model/backend restrictions. Native review should confirm
these controls against the selected profiles and saved settings.

`settings-layout.spec.ts` checks all four workflow pages at 100%, 125%, and 150%
CSS scale: content scrolls, Save remains fixed and visible, and the document has
no extra vertical overflow. It also checks Voice uses the shared settings group.
Native WebView/DPI acceptance remains separate from these browser fixtures.

## Workbench shell regression coverage

Run `npm run test:browser` from `frontend`. The fixtures render the real shell,
settings editor, workflow controls, and runtime inventory against synthetic
services. Settings requests enter through the same readiness/navigation event
contract; fixtures do not create a second Settings iframe. Tests cover guarded
rail and palette navigation, connection drafts, hidden credential cleanup,
record/stop/cancel admission, runtime lifecycle recovery, capability-gated
streaming, and panel collapse/keyboard behavior.

For an isolated local test server, set `PLAYWRIGHT_PORT` to an unused loopback
port. `PLAYWRIGHT_REUSE_SERVER=1` explicitly reuses a running fixture server; CI
always starts its own server. These are browser checks, not acceptance of native
recording, insertion, permissions, or runtime inference. Run the native checklist
on each supported OS and architecture before release.

### Page-content consistency

Review Voice, files, speech, readiness, History, Settings, Connections, and runtime
details in both themes at normal and compact widths. Headers and the first content
column should share the page gutter; related settings should read as divided rows.
Compare fields, pickers, buttons, disclosures, and keyboard focus across pages.
Check that transcript text retains its reading size, docked playback/output
controls remain reachable, and notices do not obscure compact settings. Existing
workspace, readiness, settings-layout, visual-hierarchy, speech-playback, runtime,
and process-output browser suites exercise these surfaces with synthetic data.
Repeat the native WebView review at supported DPI scales before release.
