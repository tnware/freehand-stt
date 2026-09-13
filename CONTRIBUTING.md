# Contributing to Freehand

Freehand is an early Windows alpha maintained by one person. Bug reports,
compatibility results, documentation fixes, and focused code contributions are
welcome. For a substantial change, please open an
[issue](https://github.com/tnware/freehand-stt/issues) before investing in an
implementation.

Never include credentials, transcripts, private endpoint URLs, machine names,
personal paths, or unredacted logs in a public report.

## Development

Development currently requires:

- Go 1.27 or newer
- Node.js 22 or newer
- the Wails CLI version pinned by `go.mod`
- Windows 11, WebView2, and a compatible C toolchain

```powershell
npm ci --prefix frontend
$wailsVersion = (go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3).Trim()
go install "github.com/wailsapp/wails/v3/cmd/wails3@$wailsVersion"
wails3 generate bindings -clean=true -ts -i
wails3 task dev
```

### macOS source build

Use macOS 13 or newer with Xcode Command Line Tools, Go 1.27, Node.js 22+
and cgo enabled. Build the pinned Wails CLI with the same Go toolchain as the
application, rather than an older auto-selected toolchain:

```sh
export PATH="$HOME/go/bin:$PATH"
GOTOOLCHAIN=go1.27.0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
npm ci --prefix frontend
wails3 task package ARCH=arm64
```

Use `ARCH=amd64` for an Intel build. Packaging produces an app bundle and a
matching-architecture ZIP; launch the bundle, not a bare binary, for native
permission tests. A local ad-hoc signature is not Developer ID signing or
notarization. See the [macOS setup guide](site/src/content/docs/docs/guides/macos-setup.md)
and [ADR 0013](site/src/content/docs/docs/decisions/0013-native-macos-boundary.md).
Record real microphone, keyboard, insertion, Keychain, overlay and login behavior
separately from unit/browser tests and cross-compilation. Never invoke model
inventories to qualify a build.

See the [contributor documentation](https://tnware.github.io/freehand-stt/docs/development/)
for architecture, testing, and native Windows acceptance. Before opening a
pull request, run the checks relevant to the change. The normal baseline is:

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

Use a conventional commit prefix such as `feat:`, `fix:`, `docs:`, or
`refactor:` because release notes are generated from commit history. In the
pull request, explain what changed, how it was verified, and any native Windows
behavior that still needs manual validation.
