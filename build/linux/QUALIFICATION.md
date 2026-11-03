# Linux qualification record

Branch implementation is experimental, not a supported release. Test date:
2026-11-03. Original cloud host: Debian 13, x86_64, XFCE/X11. Native dependencies were
extracted from signed official Debian packages into a private workspace prefix.
No system services, security settings, keyboard maps, or user machine were changed.
The managed-cloud follow-up below records the later environment separately.

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

## Unresolved qualification gates

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

## Packaged frontend bridge smoke (pending runner execution)

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
pass locally after restoring official native development dependencies. The new
packaged launch remains unexecuted
until a manually dispatched CI run completes; a sandbox failure must remain a
failure, never be worked around with sandbox-disabling options.

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
