# macOS support: local verification and acceptance

## Status

Native implementation, frontend integration, and browser-test repairs have passed
scoped spec/quality reviews and final integration re-review. This is a **local
acceptance build**, not a notarized distribution or a claim of full hardware
acceptance. No push, pull request, release publication, or remote CI run was made.

Working branch: `feat/macos-support`.

## Manual-testing checkpoint: focus issue remains open

The user initially reported that everything seemed to work, then clarified:
“except the focus thing actually, seems to always think focus changed.”
Testing is continuing; this is not full acceptance or authorization to push.

- **Open defect:** reported repeated focus-changed rejection during native
  insertion. Root cause is not yet established; automated fixture passes do not
  disprove this runtime report.
- The exact target application, trigger sequence, and displayed error have not
  yet been captured. Investigate with a targeted reproduction and privacy-safe
  focus/identity diagnostics before changing the fail-closed insertion policy.
- Implementation and packaged binaries are unchanged at this checkpoint.
  Existing automated/build evidence below remains separate from manual results.

## Implemented boundaries

- CoreAudio capture/playback; explicit device selection, cancellation-aware
  microphone authorization, existing bounded PCM/WAV and playback behavior.
- Operation-scoped AX target identity; no target text reads, activation, or
  automatic clipboard fallback. Unicode insertion is generation/focus checked.
  Explicit copy distinguishes not-performed, failed, and ambiguous-running
  outcomes and bounds pending native work.
- Native keyboard observation/capture with injection filtering and bounded
  teardown. Startup/capture recovery preserves independent Carbon shortcuts
  when optional hold-to-talk is unavailable. Clear and explicit Retry remain
  usable; genuine settings/global-registration failures retain rollback.
- Passive Cocoa overlay with recent-caption fitting, accessibility overrides,
  amplitude/checkpoint rendering, bounded wakes, and visible-only timers.
- Direct nonprompting Keychain access, ownership-checked per-user startup,
  native menus/tray/Dock lifecycle, and separate development/production bundle
  identities with microphone usage text and macOS 13 deployment targets.
- Platform-aware settings/readiness, native shortcut glyphs with spoken labels,
  and explicit permission recovery, including rejected-denial refresh.

ADR: `site/src/content/docs/docs/decisions/0013-native-macos-boundary.md`.
User guide: `site/src/content/docs/docs/guides/macos-setup.md`.

## Automated evidence

Final source tests after the capture-resume recovery fix:

| Check | Observed result |
| --- | --- |
| Full Go suite with race detector | Passed |
| Frontend unit tests | 256 passed across 34 files |
| Svelte check | 0 errors, 0 warnings |
| Full Playwright browser suite | 128 passed |
| Native overlay drawing/visible timer fixtures | Passed |
| Native insertion lifecycle/cancellation fixtures | Passed without real clipboard writes |
| Repeated keyboard/cancellation/shortcut race regressions | Passed |
| Independent final integration re-review | Approved, no remaining reported security/logic findings |

Commands from repository root:

```sh
CGO_CFLAGS='-O2 -g -mmacosx-version-min=13.0' \
CGO_LDFLAGS='-O2 -g -mmacosx-version-min=13.0' \
go test -race ./... -count=1 -timeout=150s \
  -ldflags='-extldflags=-mmacosx-version-min=13.0'
npm --prefix frontend test
npm --prefix frontend run check
npm --prefix frontend run test:browser -- --reporter=line
```

Verification logs from the local run are under `/tmp` and are not durable CI:
`freehand-final-go-race.log`, `freehand-final-frontend.log`, and
`freehand-final-browser.log`.

The original browser failures were reproduced on baseline `60ee2a9` before
repair. Connection tests now retain the actual manager component across
hide/reopen, rather than masking stale state by recreating it. Native lifecycle
signals are browser proxies, not evidence of OS focus behavior. Playback geometry
waits for actual animation completion before an atomic measurement; its original
strict alignment bound remains sensitive to an injected geometry regression.

## Packaging and runtime boundary

Package commands, using the pinned Wails CLI from Go 1.27:

```sh
wails3 task package ARCH=amd64
wails3 task package ARCH=arm64
```

Outputs: `bin/freehand-darwin-amd64.zip`, `bin/freehand-darwin-arm64.zip`, and
`bin/freehand.app` (Apple Silicon after the commands above). Both architectures
were built as full applications. Extracted packages passed signature validation,
architecture checks, executable mode checks, and `otool` minimum-version checks
for macOS 13.0. Signing is **ad hoc**, not Developer ID signing/notarization.
The handoff packages were rebuilt from committed source `e6d6412`, including
the final reviewed shortcut recovery code. Extracted packages were verified
again after this rebuild. SHA-256:

```text
474cf5826d65ad061f77a3a3191dcc6729feaddf933129293444a9d202d76bb2  freehand-darwin-amd64.zip
5094443d3fa4c346c571b9ba9e7657836a903203c7ad74cd50a5e5fda4a1f64e  freehand-darwin-arm64.zip
```

Windows GUI cross-compilation also passed. Explicit Windows CGO flags avoid
inheriting the host's macOS deployment flags:

```sh
CGO_CFLAGS='-O2 -g' CGO_CXXFLAGS='-O2 -g' CGO_LDFLAGS='-O2 -g' \
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
go build -tags production -trimpath -ldflags='-s -w -H windowsgui' \
  -o bin/freehand-windows-amd64.exe .
```

The final packaged Apple Silicon smoke build launched using a temporary HOME.
Native process metadata reported completed application launch and an on-screen
main window. A native graceful-quit request exited with code 0. This proves
launch/window creation/quit, **not** successful rendering or dictation. The
inspection process had neither Accessibility nor Screen Recording access.

Nonblocking observed warnings: duplicate `-lobjc` linker entries, Vite's large
production chunk warning, and browser-fixture Svelte ownership warnings.

## Required user acceptance before any push

Use a final packaged build at a stable location. Do not provide credentials in
chat; configure the chosen speech service directly in Freehand.

- [ ] Inspect actual Mac windows/menus/tray, light/dark appearance, close/reopen,
  Dock and second-instance behavior, then Quit.
- [ ] Verify microphone allow/deny/recovery and cancel while authorization is
  pending. A denial should lead to Open Microphone settings, not another prompt.
- [ ] Record through the real microphone into the chosen service, stop, check
  transcription, and exercise TTS/playback/seek. Verify explicit audio devices.
- [ ] Verify toggle/show, hold press/release, Escape capture cancellation,
  unavailable-hold Clear, explicit Retry after Secure Input/session recovery,
  and independent globals after capture ends with modifiers still held.
- [ ] Verify insertion into the original target, switched-focus rejection,
  secure-field rejection, Unicode, and explicit Copy without automatic clipboard
  replacement. Review ambiguous-copy errors rather than assuming rollback.
- [ ] Verify overlay click-through/no focus theft, Spaces/full-screen,
  mixed-display placement, captions and accessibility preference changes.
- [ ] Exercise real Keychain persistence/error recovery and start-at-login.
  Disable startup before moving the bundle; re-enable at its new location.
- [ ] Run on actual Intel hardware and Windows if claiming runtime compatibility
  for those targets. Cross-compilation alone is not runtime acceptance.

No microphone recording, native typing, real clipboard mutation, Keychain
mutation, login-item registration, or permission grant was performed by the
verification agent.
