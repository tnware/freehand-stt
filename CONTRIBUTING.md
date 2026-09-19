# Contributing to Freehand

Freehand is an early desktop alpha for Windows and macOS,
maintained by one person. Bug reports,
compatibility results, documentation fixes, and focused code contributions are
welcome. For a substantial change, please open an
[issue](https://github.com/tnware/freehand-stt/issues) before investing in an
implementation.

Never include credentials, transcripts, private endpoint URLs, machine names,
personal paths, or unredacted logs in a public report.

## Development

Shared development prerequisites:

- Go 1.27 or newer
- Node.js 22 or newer
- the Wails CLI version pinned by `go.mod`

### Windows

Use Windows 11, WebView2, and a compatible C toolchain.

```powershell
npm ci --prefix frontend
$wailsVersion = (go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3).Trim()
go install "github.com/wailsapp/wails/v3/cmd/wails3@$wailsVersion"
wails3 generate bindings -clean=true -ts -i
wails3 task dev
```

### macOS

Use macOS 13 or newer with Xcode Command Line Tools, Go 1.27, Node.js 22+
and cgo enabled. Build the pinned Wails CLI with the same Go toolchain as the
application, rather than an older auto-selected toolchain:

```sh
export PATH="$HOME/go/bin:$PATH"
GOTOOLCHAIN=go1.27.0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
npm ci --prefix frontend
wails3 dev -config ./build/config.yml -port 9245
```

Use the native `bin/freehand.dev.app` for interactive testing. Frontend edits hot
reload; Go and native `.m`, `.h`, and `.c` edits rebuild and restart the app.
The development bundle has a separate permission identity from the packaged app.
If it is absent from Accessibility or Input Monitoring, add that bundle using
the **+** button in System Settings. Its display name is still **Freehand**.

Build a distributable bundle separately with `wails3 task package ARCH=arm64`.

Use `ARCH=amd64` for an Intel build. Packaging produces an app bundle and a
matching-architecture ZIP; launch the bundle, not a bare binary, for native
permission tests. A local ad-hoc signature is not Developer ID signing or
notarization. See the [macOS setup guide](site/src/content/docs/docs/guides/macos-setup.md).
Record real microphone, keyboard, insertion, Keychain, overlay and login behavior
separately from unit/browser tests and cross-compilation. Never invoke model
inventories to qualify a build.

## Validation

Before opening a pull request, run the checks relevant to the change.
The normal baseline is:

```powershell
gofmt -w main.go internal build/scripts
go test ./...
npm --prefix frontend test
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix site run build
wails3 task build CGO_ENABLED=1 ARCH=amd64
git diff --check
```

Use `ARCH=arm64` for Apple Silicon native builds. Windows and macOS packaging
run on their respective native CI runners; deterministic checks do not establish
hardware acceptance. Release publication must use the complete Windows and macOS
artifacts from the same validated tagged run, with one complete `SHA256SUMS`.
See the [release contract](https://tnware.github.io/freehand-stt/docs/development/releases/).

Use a conventional commit prefix such as `feat:`, `fix:`, `docs:`, or
`refactor:` because release notes are generated from commit history. In the
pull request, explain what changed, how it was verified, and any native platform
behavior that still needs manual validation.

Test observable behavior at its owning boundary, including failure and cancellation
where relevant. Avoid tests that merely repeat implementation details.
For UI interactions, run relevant local Playwright cases with
`npm --prefix frontend run test:browser -- <spec-file>`. Browser fixtures use
synthetic services and cannot establish native desktop behavior.
Follow the [native acceptance procedure](site/src/content/docs/docs/safety/native-test-checklist.md)
for changes involving OS integration.

## Documentation

Update product guides when a change affects how someone uses Freehand. Keep
contributor pages focused on how to build, validate, generate, or release it.
Implementation details, decisions, test results, and remaining acceptance work
belong in the issue or pull request.
