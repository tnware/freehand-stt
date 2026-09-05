---
title: GitHub Actions
description: Public continuous integration, Windows packaging, and Pages deployment.
---

Freehand uses small, explicit GitHub Actions workflows rather than a custom CI
container:

- **CI** runs application checks for pull requests, pushes to `main`, and manual
  dispatches. Linux jobs validate Go, the SQLite contract, generated branding, Wails bindings, and
  the Svelte frontend. A Windows job builds the native executable and per-user
  NSIS installer with the real CGo audio dependency, after native storage fixtures pass. Site-only changes skip
  these application jobs.
- **Pages** builds and validates the site for pull requests that change site or
  shared branding files, then publishes `site/dist` after matching changes land
  on `main` or a maintainer dispatches the workflow. The production Astro base
  path is `/freehand-stt/`.
- **Release** lets Release Please maintain the release pull request, changelog,
  SemVer tag, and draft GitHub release. Once that release exists, the Windows
  job builds the tagged source, publishes the bare updater executable and
  installer, emits `SHA256SUMS`, creates GitHub artifact attestations, and only
  then makes the release public.

Third-party actions are pinned to immutable commit SHAs. Dependabot proposes
weekly grouped updates for Actions, Go modules, and both npm lockfiles so a
fresh repository does not produce one pull request per action.

The Ubuntu jobs install Wails' documented GTK 4 and WebKitGTK 6.0 development
packages before compiling packages that import Wails or installing the CLI.
The Windows runner installs the Wails CLI at the exact module version from
`go.mod`, an MSYS2 MinGW-w64 C compiler for malgo/miniaudio, and NSIS 3.12. It
does not rely on a private runner, registry, or package cache.

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
