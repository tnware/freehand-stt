---
title: Release lifecycle
description: Versioning, Windows and macOS artifacts, integrity, and update boundaries.
---

Freehand uses Conventional Commits and Release Please. Changes merged to `main`
accumulate in one release pull request. That pull request updates
`CHANGELOG.md`, `.release-please-manifest.json`, and the human version in
`build/config.yml`. Merging it creates a draft release and a `v`-prefixed
SemVer tag. Use the version in the release manifest and tag; do not infer the current version from an example.

## Version policy

`0.1.0` is the first non-alpha baseline. It remains a pre-1.0 SemVer version,
not a promise of a frozen 1.0 compatibility contract. Subsequent release bumps
follow Conventional Commits: `fix` bumps the patch, `feat` bumps the minor, and
an explicit breaking change bumps the major. From `0.1.0`, those produce
`0.1.1`, `0.2.0`, and `1.0.0`, respectively.

Release Please uses `prerelease: false` with its `prerelease` versioning strategy.
Despite that strategy's name, this combination graduates an existing alpha to
its non-prerelease base version and then applies ordinary SemVer bumps to stable
versions. There is no `release-as` override pinning future releases. Keep the
release manifest and application version in the bot-owned release PR; changing
`initial-version` alone does not override an existing release's version.

The repository uses squash merges with the pull request title as the commit
subject and an empty commit body. Write the **PR title** as a Conventional
Commit, for example `fix(security): block inference redirects and sanitize response metadata`.
Release Please reads that title after merge; individual branch commit messages
and the PR description do not become separate release-note entries. Use `fix`
for corrective changes, reserve `feat` for new capabilities, and do not add `!`
or force a release version for an ordinary bug fix.

Merge reviewed product fixes before the pending release PR so Release Please can
refresh its changelog. Leave `CHANGELOG.md`, `.release-please-manifest.json`, and
the version in `build/config.yml` to that release PR rather than editing them in
fix branches. Merging a fix is not publication: merging the refreshed release PR
starts the draft/tag and Windows and macOS packaging flow described below.

The release workflow validates the tag through the same CI workflow used by PRs,
with every workload enabled and the expected version checked before packaging.
Only after validation succeeds does publication download the complete Windows and macOS artifacts
from that same release run, create checksums and attestations, and make the draft
public as a non-prerelease and GitHub's latest release. It rejects prerelease
tags before uploading and reads back both the published flags and latest tag.
It does not reuse PR artifacts or rebuild different bytes after tests.
A failed validation leaves publication blocked; rerun the failed jobs after
diagnosing the failure rather than making the draft public manually.

`build/config.yml` is the release identity source. The release build derives
Windows' required four-part numeric version from that SemVer value before it
generates the executable resources and installer metadata. About reads the same
embedded source. macOS build numbers are derived from that same SemVer value;
there is no separate counter to update. `CFBundleShortVersionString` uses the
three release components. `CFBundleVersion` is numeric `major.minor.build`,
where `build = patch × 262144 + stage × 65536 + revision`; stages are alpha (0),
beta (1), rc (2), and stable (3, revision 0). This keeps prerelease-to-stable
ordering while satisfying Apple's numeric bundle format. Supported prereleases
are `alpha.N`, `beta.N`, and `rc.N`, with the same positive uint16 revision
bound used by the Windows resources. For example, `0.1.0-alpha.5` produces
bundle build `0.1.5`, without changing its displayed release version.

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

## Public artifact contract

New cross-platform releases must contain the complete asset set below. Older
releases may lack macOS assets; never present a missing asset as downloadable.

- `freehand-windows-amd64.exe`, the bare executable selected by Wails updater;
- `freehand-windows-amd64-installer.exe`, the per-user NSIS installer;
- `freehand-darwin-arm64.zip`, the Apple Silicon app bundle;
- `freehand-darwin-amd64.zip`, the Intel app bundle; and
- one `SHA256SUMS`, covering all four assets with no missing or extra entries.

Publication must reject incomplete sets and use only the artifacts from the
same trusted, validated tagged workflow run. Do not rebuild or mix runs.

GitHub also records artifact attestations. The binaries are not currently
Authenticode-signed, so Windows may display an unfamiliar-publisher warning.
Download only from the official
[Releases page](https://github.com/tnware/freehand-stt/releases) and verify the
checksum when installing manually.

## macOS packaging and updates

Build on a native macOS runner with cgo, the pinned Wails CLI, and Xcode Command
Line Tools. Package with `wails3 task package ARCH=arm64` or `ARCH=amd64`.
The deployment target is macOS 13+. Validate architecture, bundle identity,
version, resources, ZIP contents, and ad-hoc signature before publication.
Build checks do not establish real microphone, keyboard, insertion, Intel
hardware, or update/relaunch acceptance.

The pinned Wails updater selects the exact Darwin architecture ZIP, verifies
`SHA256SUMS`, extracts its `.app`, and uses a helper to swap the app bundle on
Restart. Freehand configures no update public key. Ad-hoc signatures and checksums
are not Developer ID trust or notarization; native install/relaunch acceptance
is separate. Keep [manual ZIP replacement](../../guides/macos-setup/#update)
available as recovery.

## Windows in-app updates

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
