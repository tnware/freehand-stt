---
title: Release lifecycle
description: Versioning, Windows artifacts, integrity, and the in-app updater.
---

Freehand uses Conventional Commits and Release Please. Changes merged to `main`
accumulate in one release pull request. That pull request updates
`CHANGELOG.md`, `.release-please-manifest.json`, and the human version in
`build/config.yml`. Merging it creates a draft prerelease and a `v`-prefixed
SemVer tag. The first intended public version is `v0.1.0-alpha.1`.

`build/config.yml` is the release identity source. The release build derives
Windows' required four-part numeric version from that SemVer value before it
generates the executable resources and installer metadata. About reads the same
embedded source.

Release Please uses its generic line updater and the
`x-release-please-version` annotation on that one field. Do not switch it to a
YAML updater: serializing the complete Wails configuration strips comments and
creates unrelated formatting churn.

After changing product identity or a version manually, synchronize the derived
assets:

```powershell
wails3 task common:update:build-assets
```

Review the generated diff. `build/windows/nsis/project.nsi` contains
application-specific policy that must survive an upstream Wails asset refresh.
Do not repair derived Windows version fields individually.

Prerelease versions end in a positive numeric revision. For example,
`0.1.0-alpha.1` maps to the four-part Windows version `0.1.0.1`; a stable
release uses a zero revision.

## Build the Windows package

Freehand requires CGo for native audio, so production artifacts are built on a
native Windows GitHub runner. This follows Wails' CI guidance to use a native
runner for the target platform rather than treating a cross-compiled binary as
release acceptance.

Build the default per-user package locally with:

```powershell
wails3 task package CGO_ENABLED=1 ARCH=amd64
```

An intentional all-users package can be built with
`INSTALL_SCOPE=machine`; it requires elevation and is not the public default.
Both paths write `bin/freehand-amd64-installer.exe`.

The installer creates one Start Menu shortcut and does not launch Freehand or
create a Desktop shortcut. Install and uninstall stop with an instruction when
the tray process is running rather than terminating it. Uninstall deliberately
retains settings, WebView state, and Credential Manager entries.

Each alpha release contains:

- `freehand-windows-amd64.exe`, the bare executable selected by Wails updater;
- `freehand-windows-amd64-installer.exe`, the per-user NSIS installer; and
- `SHA256SUMS`, covering both files.

GitHub also records artifact attestations. The binaries are not currently
Authenticode-signed, so Windows may display an unfamiliar-publisher warning.
Download only from the official
[Releases page](https://github.com/tnware/freehand-stt/releases) and verify the
checksum when installing manually.

Automatic update checks are enabled by default and can be disabled under
**Settings → General**. A quiet check runs shortly after startup and then once
per day. A check only reads GitHub release metadata when Freehand is current.
If it discovers an update, Wails opens its first-party update window, downloads
and verifies the selected executable, and waits for the user to restart.
**Check now** opens the same flow on demand. Freehand does not restart into an
update without user action.

The current Wails v3 updater stages a verified bare executable by replacing the
installed executable. It does not run the NSIS installer. Installer-level
migrations or additional packaged files would require a future updater policy
change.

There is no Ed25519 release key in the first public lifecycle. The standard
Wails GitHub provider verifies the downloaded executable with the release's
`SHA256SUMS`; adding an unused private signing key would not strengthen that
path. Authenticode is the separate future mechanism for Windows publisher
identity.

When Authenticode is introduced, both the executable and final installer must
be signed with a trusted timestamp. Wails reads a configured certificate from
`wails3 setup signing` or the `SIGN_CERTIFICATE`, `SIGN_THUMBPRINT`, and
`TIMESTAMP_SERVER` task variables. The ordered local task is:

```powershell
wails3 task package:signed CGO_ENABLED=1 ARCH=amd64
```

No signing credential is configured today. Never commit certificates,
passwords, or thumbprints.
