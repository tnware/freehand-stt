# Linux qualification record

Branch implementation is experimental, not a supported release. Test date:
2026-11-03. Original cloud host: Debian 13, x86_64, XFCE/X11. Native dependencies were
extracted from signed official Debian packages into a private workspace prefix.
No system services, security settings, keyboard maps, or user machine were changed.
The managed-cloud follow-up below records the later environment separately.

Latest result (2026-11-04): native Debian 13 and Ubuntu 24.04 packages build,
install, reinstall, replace a versioned fixture and remove successfully. A passive
X11 status overlay is now implemented and passes isolated native tests on both
distros. Automatic insertion remains unavailable. The cloud's installed-app
WebKit namespace denial has not been retried. Later records below supersede the
earlier overlay implementation status and retain the separate desktop gates.

## Established

- Native Go/CGO desktop executable builds against GTK 4.18.6, WebKitGTK 2.54.0,
  and ALSA 1.2.14. Distributable-shape build has no workspace RPATH.
- The production GTK clipboard adapter, exercised without a WebView in the actual
  desktop session, round-trips `Freehand Linux copy: ✓ 漢字 😀`.
- XCB native passive-grab conflict detection, release/regrab, and exclusive
  capture timeout cleanup pass in the actual X11 session.
- Official pinned Linux amd64 CPU NeMo-Speech.cpp v0.1.0 and llama.cpp b10809
  archives pass hash/size verification, private installation, and owned execution.
- Explicitly selected S1-mini model loads and completes a bounded 16-token request
  (HTTP 200); explicitly selected Nemotron model loads and transcribes one second
  of synthetic silence (HTTP 200). Each owned runtime stops afterward. These are
  not speech-quality, microphone, performance, GPU, or model-inventory benchmarks.
- ALSA null-PCM capture/playback lifecycle tests exercise native buffers,
  cancellation, pause/seek, and shutdown. This is not real-device acceptance.
- XDG autostart ownership, quoting, atomic writes, removal, and failure handling
  pass unit/race tests and desktop-file validation. A real login was not tested.

## Original-host qualification gates

These describe the original desktop/prefix environment. Later managed-cloud
records below supersede its package/build results and identify the current
sandbox blocker; hardware and desktop acceptance remain separate.

- Full app shell cannot launch with this rootless dependency prefix: the Debian
  WebKit release binary uses absolute system helper paths. Its developer-only
  helper relocation override is not present. No sandbox disabling, binary
  patching, or loader interception was used. Test on a normal system-installed
  distro desktop before advertising a supported package.
- Xvfb creation is denied by this environment's socket policy, including one
  reviewed escalation. The independently provided XFCE desktop was used for the
  native tests above. There is no physical audio device in this environment.
- Held-key acceptance did not receive a physical press from the available desktop
  automation. Native hold, repeat suppression, capture chord/Escape, layout
  changes, and modifier-first release remain unqualified. Do not substitute
  synthetic unit-event tests for those desktop checks.
- Direct production download DNS is restricted here. Official archives/models
  were obtained via the provided authorized network mechanism, then tested through
  pinned local fixtures and production install/cache/load paths. End-to-end direct
  downloads, interruptions, and network recovery remain to be exercised.
- Install/update/remove of a real `.deb` needs the target distro's installed dpkg
  database and normal system dependencies. Rootless-prefix packaging correctly
  rejects unresolved dependencies/private RPATH. CI defines separate Debian 13
  and Ubuntu 24.04 package lanes, not an assertion that they have run.
- Secret Service, real audio input/output, desktop login, multi-monitor/scaling,
  GPU variants, ARM64, and Wayland portals have not been accepted here.

Automatic text insertion and native overlay are not implemented on Linux.
Explicit Copy is the supported output path in this experimental implementation.
Managed whisper.cpp remains unavailable because the pinned upstream release has
no qualified native Linux server archive. No supported Linux release link should
be published until the documented per-distro acceptance procedure is complete.

## Automated verification

The baseline implementation at `0f4e837` passes `go test ./...`; focused Go race tests include
app/input, keyboard, autostart, hotkey policy, audio, managed runtime, and owned
process lifecycle. Frontend checks report zero Svelte errors/warnings and
64 test files / 532 tests pass. Production frontend build succeeds. Packaging,
CI-contract, and site tests are separate from the unrun native package acceptance.
Independent review verified monotonic shutdown cancellation and the Linux
window-reachability paths. None of these replace the unresolved gates above.

## Packaged frontend bridge smoke contract

The Linux CI lanes now define an unprivileged, isolated Xvfb/session-D-Bus launch
of the installed binary using normal distro WebKit helpers and sandboxing. The
explicit `FREEHAND_LINUX_SHELL_SMOKE=1` switch emits one fixed
`FREEHAND_FRONTEND_BRIDGE_READY_V1` line after the embedded frontend calls its
existing ShellReady bridge. No transcript, credentials, paths, or settings are
reported. The switch is ignored outside Linux and changes no application state.

This marker proves frontend/bridge execution, not healthy session initialization:
the existing initialization recovery paths can still lead to ShellReady. The
harness bounds output and launch time, checks short post-marker liveness, and
terminates its owned process group. This is not graceful-shutdown or arbitrary
escaped-descendant acceptance. Empty private state has no configured endpoint
or model, and the test never requests audio capture or inference.

Harness unit tests, the marker/native callback race tests, and the full Go suite
pass locally after restoring official native development dependencies. At that
point the packaged launch was still pending a suitable runner; a sandbox failure
must remain a failure, never be worked around with sandbox-disabling options.

## Managed-cloud follow-up (2026-11-03)

Restored the published `codex/linux-support` head `ae134b2` into a clean checkout
in the newly attached managed cloud environment. The observed runtime reached
`running`, connected, with configuration revision `1` and an enforced restricted
network policy. The outer shell still had no GTK/WebKit development metadata or
Go language toolchain. No later completed user setup configuration was observed.

The environment's managed rootless Docker daemon is available. Installed normal
system packages inside separate native x86_64 distro containers, using the
provided proxy and CA trust, without changing the user's machine or weakening
WebKit sandboxing:

| Image | Image digest | GTK | WebKitGTK | Go / Node |
| --- | --- | --- | --- | --- |
| `debian:13` | `sha256:2dc419c6d7fe3dad0a2f9fd469c34cc90b0f9ba642d8fde837fe3d7d34a0e7f1` | 4.18.6 | 2.54.0 | 1.27.0 / 24.22.1 |
| `ubuntu:24.04` | `sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852` | 4.14.5 | 2.52.6 | 1.27.0 / 24.22.1 |

New fixes and verification:

- Packaging previously inherited `umask 0077`, which made `DEBIAN` mode 0700
  and caused `dpkg-deb` to reject the archive. Normalize staged directories to
  0755 and control metadata to 0644. The native fixture now explicitly exercises
  private umask and verifies traversable, root-owned payload directories.
- The startup fixture now creates an explicitly private home. Go's numbered
  temporary directory inherited mode 0777 with the container's `umask 0000`;
  production correctly rejected it. Only the fixture changed; ownership and
  writable-directory refusal remain in force.
- Package and shell-harness tests pass on both distros: 14/14 each. These package
  a small C fixture, not the Freehand executable. CI selection/contract tests
  also pass: 54/54.
- Focused Go race suites pass on both distros for keyboard, hotkey, explicit
  input/insertion, startup, and managed-runtime process ownership/lifecycle.
- Native X11 grab/conflict/regrab and capture-cancellation lifecycle probes pass
  five times per distro under unprivileged Xvfb/session D-Bus. `-noreset` keeps
  the virtual server alive between connections, as a desktop client normally
  would; without it, last-client reset interrupted a re-open. No synthetic key
  events, physical press/release, or desktop keyboard qualification are claimed.

Remaining blockers and exact outcomes:

- **Build blocked by network policy:** the pinned Wails CLI was built after
  fetching its module directly from allowed GitHub sources. Production
  `wails3 task package ARCH=amd64 CI=true` then failed when
  `modernc.org/sqlite@v1.58.0` redirected from `proxy.golang.org` to
  `storage.googleapis.com`, which returned `Forbidden`. The supported network
  configuration needs that destination allowed before retrying. The alternate
  allowed `goproxy.io` service returned an upstream Cloudflare 403. No proxy or
  TLS bypass was used. The Ubuntu production build remains unrun pending the
  same dependency access.
- **Normal WebKit sandbox blocked:** Xvfb and session D-Bus start on both
  distros, and the system WebKit helpers are installed at their normal absolute
  paths. The stock distro `MiniBrowser about:blank`, run as `freehand-qa` with
  an isolated home and clean environment, exits with code 133 on both distros:
  Bubblewrap cannot create a namespace, then WebKit reports that `dbus-proxy`
  failed to launch. `unshare --user` independently returns `Operation not
  permitted`. This is a runtime restriction beyond missing packages. No sandbox
  disabling, privileged container, security-profile relaxation, binary patch,
  or helper interception was attempted.
- **Freehand package/install/smoke unqualified:** no new production executable or
  `.deb` was produced, installed, updated, or removed in this environment. The
  installed-Freehand bridge smoke therefore remains unrun. The stock browser
  probe establishes an environment blocker, not Freehand startup acceptance.
- **Dependent validation blocked:** initial frontend checking without generated
  bindings reported missing binding imports. Binding generation, full frontend
  validation, and the full Go suite need the blocked SQLite dependency. Do not
  count the incomplete build or those checks as passes.

Resume with an applied environment configuration that permits the dependency
download and normal WebKit sandbox namespace operations. Rebuild separately on
each distro, verify/install its checksum-matched package, execute the installed
frontend bridge smoke as an unprivileged user, and validate update/removal.
Real microphone/playback, physical shortcuts, Secret Service, Wayland, login,
and the remaining native acceptance procedure still require separate evidence.
Automatic insertion and native overlay remain unimplemented. No PR, merge,
release, model download, or inference was performed in this follow-up.

## Republished-configuration follow-up (2026-11-03)

Restored branch head `eb55c81` in `/workspace/freehand-stt`. The selected managed
instance became `running` and connected, with current observations and observed
spec revision `1`. Its published source configuration version was
`1bc3dcf4-e801-4f48-b17d-7bba75692fb1~cecfgver_6aea6c993bac81a0acf0e7db1312121b`.
The runtime reported an **enforced unrestricted HTTP policy**; the executor's
version-1 policy snapshot agreed and reported no VPN or additional TCP grants.
These observations identify this task's selected runtime, not a later editable
configuration draft.

The completed setup supplies Go 1.27.3, Wails v3.0.0-beta.16 and dependency caches
under `/workspace/.freehand-cloud`; Node is 24.22.1. The outer shell still lacks
system GTK/WebKit development packages and root package-install privileges.
Used the setup's managed Docker daemon and Debian development image, extending
it with normal distro packaging/Xvfb dependencies. Prepared a separate Ubuntu
24.04 image from the pinned official image recorded above. Kept the inherited
proxy, mounted CA trust, TLS verification, package signatures and Go checksum
verification. No private library prefix or sandbox relaxation was used.

The former dependency blocker is cleared: the normal
`proxy.golang.org/modernc.org/sqlite/@v/v1.58.0.zip` request, including its
redirect, returns HTTP 200 (23,114,130 bytes). The production pipeline completes
dependency resolution, and `go mod verify` reports `all modules verified`.

Verification completed independently of GUI acceptance:

- Debian's normal `wails3 task package ARCH=amd64 CI=true` generates bindings,
  builds the production frontend and native executable, and produces a real
  distro-resolved `.deb`. Package/harness tests pass 14/14. GTK is 4.18.6,
  WebKitGTK 2.54.0 and XCB 1.17.0.
- Ubuntu's same production pipeline and 14/14 package/harness tests also pass,
  using a separate Go build cache to avoid reusing Debian native objects.
  GTK is 4.14.5, WebKitGTK 2.52.6 and XCB 1.15.
- Each checksum-matched distro package installs using apt, its desktop entry
  validates, and `ldd` reports no missing libraries. Same-version reinstalls
  succeed. Disposable `0.1.0-1+qa1` packages made from the respective payloads
  upgrade `0.1.0-1`, and each installed executable still matches its original
  binary. Removal succeeds and `/usr/bin/freehand` is absent afterward. This
  tests dpkg replacement, not migration from a previous released executable.
  Ubuntu's fresh image needed a normal apt index refresh before apt could
  acquire the declared ALSA runtime dependency; installation then succeeded.
- Debian `go test ./...`, `go vet ./...`, and focused race tests for native
  keyboard/input/startup, hotkey policy, managed process ownership and app
  lifecycle pass. Svelte checking reports zero errors/warnings; frontend tests
  pass 64 files / 532 tests; CI selection/contract tests pass 56/56. The build
  leaves `go.mod` and `go.sum` unchanged.

Both experimental artifacts and their `.sha256` files are retained in `bin/`:

```text
354db02ba1f3705f28df418da99a2bb10ecf0f371a7eb85a3095d2f75048bf9d  freehand_0.1.0-1_debian13_amd64.deb
561bd9db0f35ea832c38bfbb9a949087818f222964d93610ec3014bc0cfd700b  freehand_0.1.0-1_ubuntu24.04_amd64.deb
```

**Installed-app smoke remains blocked by the normal WebKit sandbox.** Made one
fresh attempt on Debian as UID 1000 with an empty private home, session D-Bus,
Xvfb and normal system helper paths. The existing smoke harness was used with
bounded diagnostic stream observation. It exits 1 before the frontend bridge
marker. The installed `/usr/bin/freehand` reports:

```text
bwrap: No permissions to create a new namespace, likely because the kernel does not allow non-privileged user namespaces.
Failed to fully launch dbus-proxy: Child process exited with code 1
SIGTRAP: trace trap
FAIL: Application exited before readiness stability check
```

ALSA also reports that no default card exists; the environment has no physical
audio device. Do not infer successful renderer initialization or hardware
behavior from the build, package lifecycle or Go tests. After the namespace
denial, no second launch, sandbox-disabling switch, privileged container,
security-profile change or alternate namespace path was attempted. Ubuntu's
installed-app launch remains unrun rather than repeating the same host boundary.
The unresolved action needs a Linux runner that supports WebKit's normal
sandbox. Real audio, physical shortcuts, Secret Service, Wayland, login and the
other desktop gates remain open. No PR, merge, release, model download,
inference or access to the user's Mac was performed.

Local build and lifecycle evidence is retained under
`/workspace/freehand-qualification/`, including `build-debian13.log`,
`build-ubuntu24.04.log`, `lifecycle-debian13.log`, `lifecycle-ubuntu24.04.log`,
`upgrade-debian13.log`, `smoke-debian13.log`,
`go-debian13.log`, `frontend-check.log`, `frontend-test.log`, `ci-tests.log`
and `modules-verify.log`. These logs are workspace evidence, not release assets.

## PR-readiness follow-up (2026-11-03)

Assessed `295b909` against main `5464de0`. The actual workload selector returns
`app=true` and `site=true`: a PR selects storage, Go, frontend, both native
Windows architectures, macOS tests/packages, both Linux distro package/smoke
lanes, site validation, and the final fail-closed Validation gate. This describes
configured checks; no PR or remote CI run was created by this assessment.

Additional cloud-supported evidence:

- Storage generation/ownership checks and storage/guard race suites pass.
- Pinned actionlint passes. The exact selection/PE/release-assembly test command
  from CI passes 95/95. Site tests pass 8/8, the 57-page site builds, and shared
  branding/GitHub Pages paths validate.
- Native GTK Unicode Copy round-trip and XCB conflict/release/capture-cancellation
  probes pass under private Xvfb/session D-Bus without creating a WebView. These
  do not establish physical shortcut delivery or another application's paste.
- Retrieved only the two pinned Linux CPU runtime archives through the inherited
  cloud proxy; both hashes match the recipes. Supplied-archive verification,
  installation and owned `--help` execution pass on Debian and Ubuntu. No models
  were downloaded or loaded. The acquisition/artifact race run records 303
  passing tests/subtests and five skips, including selected-model acceptance.
- Extra Chromium browser fixtures finish 30/31. The failing active-dictation
  assertion expects zero calls but observes `GetBinaryOptions:llama-cpp`;
  mutation controls are disabled. The identical assertion fails on a disposable
  main-source snapshot too, using the same installed test dependencies/bindings.
  This is inherited, not evidence of an unsafe mutation. Playwright browser
  fixtures are not part of the current automatic PR workflow.

Archive retrieval is not live in-app acquisition acceptance. `releaseClient`
and `modelClient` explicitly use `Proxy: nil`; NeMo's child environment omits
proxy/auth configuration. This managed environment requires its inherited proxy.
No direct-network attempt or proxy-policy workaround was made. Full in-app
runtime/model download, interruption/recovery and selected-model behavior still
need separate qualified network/desktop evidence and explicit model selection.

The implementation remains experimental Copy-only Linux support: target capture
and Unicode insertion return `ErrCopyRequired`; native overlay is unavailable.
Supported Linux release assets are intentionally absent from release assembly.
Linux package/smoke CI is nevertheless part of release-tag validation, so a
failed Linux lane also blocks publication of the existing Windows/macOS assets.
PR packaging calls neither Windows `package:signed` nor macOS Developer ID /
notarization tasks; the default macOS package uses ad-hoc signing.
Successful normal-sandbox CI execution and the documented desktop gates remain
necessary; this assessment neither retries the denied WebKit launch nor changes
sandboxing. Additional evidence logs use `pr-*` names under
`/workspace/freehand-qualification/`. No production code or workflow was changed.

## Passive X11 overlay and first PR checks (2026-11-04)

The insertion investigation still lacks a proven native-window-to-accessibility
mapping for an addressed Unicode transport. Process identity alone does not
distinguish two windows in one app. No field-identity requirement was added and
no clipboard/keymap workaround was enabled. Delivery remains explicit Copy.

Implemented the next user-visible native parity gap: an X11 status overlay using
its own XCB connection and Cairo/Pango rendering. It captures monitor geometry at
the first visible generation, respects work areas/RandR changes, and supports the
existing layouts, anchors, amplitude visualizers, countdown, phase glyphs and
bounded captions. Empty input shape is installed before mapping; input=False,
override-redirect and notification hints prevent interaction/activation. The
owner never modifies focus, input or clipboard ownership. Hidden surfaces have no
timer; shutdown revokes work, clears transient text and waits within a bound.
Wayland/XWayland reject creation. Transparency/compositor and GNOME preference
limits are documented in `internal/platform/overlay/README.linux.md`.

Verification:

- Native race/probe suites pass on Debian 13 and Ubuntu 24.04. The C fixture is
  compiled with optimization, fortification and warnings as errors. It verifies
  initial hidden state, all layouts/anchors/visualizers, unchanged focus and
  synthetic selection ownership, empty input shape, work-area changes, Unicode
  rendering, hide/destroy, and concurrent Go update/close ownership. Only fresh
  private Xvfb displays are used. No WebView or physical input is involved.
- Full Linux Go tests, vet, overlay/app/storage race suites, both production
  package builds and checksum-matched install/reinstall/versioned-fixture
  replacement/removal pass. Latest package hashes are below; earlier hashes
  identify the pre-overlay builds. No cloud app launch was retried.
- Both edited Svelte files pass the Svelte autofixer. Svelte checking has zero
  errors/warnings; 64 files / 532 unit tests pass. The new Linux browser scenario
  verifies Copy-only controls and the X11 overlay limits. Actionlint, 95 workflow/
  release tests and the 57-page site build pass.

```text
7f6c2081eeb0372b5789be1722a8d6e31c4d6602c203ae44be6eceb11bc105f3  freehand_0.1.0-1_debian13_amd64.deb
1e8d3b4a63a31a0519573a3aee0a009f3d75e2f1ce857279c7b3faa3cebaa88f  freehand_0.1.0-1_ubuntu24.04_amd64.deb
```

Opened experimental PR #102 after authorization. Its first run, `41747682166`,
passes Windows amd64/ARM64, macOS, Go, frontend, site and selection. Storage fails
because its integration fixtures now need GTK/WebKit development packages; those
dependencies are added. Both distro packages build, then their metadata guard
falls into Git's no-index mode because root does not trust the runner-owned
checkout. Reproduced this locally and scoped Git trust to that exact checkout,
with an explicit repository assertion before diff. Removal now checks whether
installation occurred. CI also adds separate unprivileged passive-overlay probes
without a WebView before the existing normal-sandbox frontend smoke.

Evidence uses `overlay-*` and `ci-git-ownership.log` under
`/workspace/freehand-qualification/`; `overlay-native.png` contains only synthetic
fixture status. Normal desktop/compositor/scaling, physical audio/keyboard,
Secret Service, login, Wayland and live model acquisition still need acceptance.
No model load, sandbox relaxation, merge or release was performed.

Run `41750212042` at `782cd33` confirms the CI fixes: storage, Windows
amd64/ARM64, macOS, Go, frontend and site all pass. Both Linux lanes build and
install their packages, pass the unprivileged native-overlay probes, and remove
the package successfully. Their sole failure is the installed frontend smoke:
`Application exited before readiness stability check`. The original harness
discards child stderr, so that log alone does not prove the exit's underlying
cause or a namespace denial on the GitHub runner.

The harness now preserves only bounded fixed failure classifications and the
OS exit status, never raw child output. Tests cover a split namespace-error line
followed by a longer native trace and a synthetic private draft, proving the
classification survives while neither private content nor trace is exposed.
Package/harness tests pass 15/15 on both distros; launch/sandbox settings and
readiness requirements are unchanged. This enables one diagnostic CI attempt
on the otherwise unexplained GitHub failure without retrying the denied cloud
operation. A confirmed runner namespace denial must remain blocked rather than
being retried or bypassed.
