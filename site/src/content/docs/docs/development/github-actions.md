---
title: GitHub Actions
description: Public continuous integration, Windows packaging, and Pages deployment.
---

Freehand uses small, explicit GitHub Actions workflows rather than a custom CI
container:

- **CI** always starts for pull requests, pushes to `main`, and manual dispatches.
  It selects application and site workloads from the complete Git diff, then
  emits one stable **Validation** result. Linux jobs validate Go, the SQLite
  contract, generated branding, Wails bindings, and the Svelte frontend. The
  Windows job runs native Go tests, builds the CGo executable and per-user NSIS
  installer, and verifies the packaged artifacts. Browser regressions remain
  available as optional local checks and do not gate packaging or release.
- **Site validation** is a reusable workflow shared by CI and Pages. Relevant
  PR changes run it inside CI; they do not start a deployment workflow.
- **Pages** builds and publishes `site/dist` after matching changes land on `main`
  or a maintainer dispatches it on `main`. Its concurrency is separate from PR
  validation, and a newer push does not interrupt an active deployment. The
  production Astro base path is `/freehand-stt/`.
- **Release** lets Release Please maintain the release pull request, changelog,
  SemVer tag, and draft GitHub release. When a release is created, it calls the
  complete CI workflow for that tag. Selection resolves the tag to one commit
  used by every validation job; release version checks precede packaging.
  Publication downloads the Windows artifacts from that same trusted workflow
  run, emits `SHA256SUMS`, creates GitHub artifact attestations, and only then
  makes the release public. It neither rebuilds the validated executable nor
  downloads artifacts from a PR or another workflow run. Ordinary main pushes
  run Release Please bookkeeping without building release artifacts.

## Workload selection and required checks

Site files and the root/branding README prose do not trigger application jobs.
Application, build, and unknown paths do. Shared backend-catalog inputs, the
shared mark/provider assets, Go manifests, and GitHub configuration also select
site validation. Renames include both paths, and a missing Git baseline fails
closed. Manual runs, initial pushes, and release validation run all workloads.

The final **Validation** job runs even when a dependency fails or is skipped.
It requires every selected job to succeed and accepts only explicitly unselected
jobs as skipped. Missing selection outputs, cancellation, unexpected skips, and
failed jobs cannot produce a green gate. Do not require individual conditional
jobs or add workflow-level path filters to CI: those can leave required checks
pending on documentation-only changes.

Configure `main` to require pull requests and the GitHub Actions **Validation**
check, with the branch required to be up to date. Enable the rule only after the
check has actually run. Keep main-push validation until that gate is enforced and
release validation has been verified. Main runs currently provide integrated
commit evidence and warm trusted caches; removing them must preserve a trusted
cache-warming path rather than relying on PR caches that other PRs cannot read.

## Toolchains and caches

Third-party actions are pinned to immutable commit SHAs. Dependabot proposes
weekly grouped updates for Actions, Go modules, and both npm lockfiles so a
fresh repository does not produce one pull request per action.

The local `setup-go` composite action disables the shared default Go cache and
uses distinct workload/platform/toolchain namespaces. Immutable dependency/tool
caches include the Go manifests and pinned tool inputs; compiled caches refresh
with source inputs and restore only within the same workload and toolchain.
Pages cannot populate the application cache with its smaller dependency set.
GitHub's cache scoping lets PRs read trusted main caches without writing to main.
Wails binaries are reused only from an exact dependency/tool cache; changed pins
do not fall back to an older installed CLI.

Measure cold and warm runs separately. A cache hit does not establish a speedup,
particularly on Windows where extracting a large cache can dominate setup.
Source-keyed compiled caches consume repository quota until GitHub evicts them.

The Ubuntu jobs install Wails' documented GTK 4 and WebKitGTK 6.0 development
packages before compiling packages that import Wails or installing the CLI.
The Windows runner installs the Wails CLI at the exact module version from
`go.mod`, an MSYS2 MinGW-w64 C compiler for malgo/miniaudio, and NSIS 3.12. It
does not rely on a private runner or registry. CI packaging uses `npm ci` with
the committed lockfile; ordinary local package builds retain `npm install`.

Production Windows artifacts are always built on a native Windows runner. This
follows Wails' CI guidance and keeps cross-compilation separate from native
release acceptance. See Wails' official
[cross-platform build and CI/CD guide](https://v3.wails.io/guides/build/cross-platform/#cicd-integration)
for the upstream dependency matrix and runner guidance.

CI never invokes a configured speech, post-processing, or text-to-speech model.
Those operations could consume private resources or unexpectedly load large
models. Native runtime acceptance remains a deliberate local Windows step.

## Storage enforcement

The storage job runs the pinned sqlc command, rejects stale or untracked generated
queries, compares existing migrations against the PR base (or previous main
revision), and checks import/query ownership. Real SQLite tests cover migrations,
recovery, constraints, and file locking; fixtures never use personal settings or
credentials. See [SQLite storage](../storage/) for the matching local commands.
