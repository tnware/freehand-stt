# Linux qualification record

Branch implementation is experimental, not a supported release. Test date:
2026-11-03. Cloud host: Debian 13, x86_64, XFCE/X11. Native dependencies were
extracted from signed official Debian packages into a private workspace prefix.
No system services, security settings, keyboard maps, or user machine were changed.

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

The final implementation passes `go test ./...`; focused Go race tests include
app/input, keyboard, autostart, hotkey policy, audio, managed runtime, and owned
process lifecycle. Frontend checks report zero Svelte errors/warnings and
64 test files / 532 tests pass. Production frontend build succeeds. Packaging,
CI-contract, and site tests are separate from the unrun native package acceptance.
Independent review verified monotonic shutdown cancellation and the Linux
window-reachability paths. None of these replace the unresolved gates above.
