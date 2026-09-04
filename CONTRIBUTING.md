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
