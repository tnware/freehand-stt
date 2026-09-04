---
title: Contributor documentation
description: Build, test, understand, and release Freehand.
---

This section is for contributors and maintainers. If you want to install,
configure, or use Freehand, start with the [user guide](../getting-started/).

## Start contributing

Read the repository's
[contribution guide](https://github.com/tnware/freehand-stt/blob/main/CONTRIBUTING.md)
before beginning substantial work. It covers prerequisites, local setup,
required checks, pull-request expectations, and the native Windows acceptance
boundary.

Then use these references as needed:

- [Architecture](architecture/) explains runtime ownership, package
  boundaries, Wails services, and data flow.
- [Testing contract](testing/) separates deterministic checks from native
  Windows acceptance.
- [GitHub Actions](github-actions/) documents CI, Pages, and release
  automation.
- [Release lifecycle](releases/) covers versions, packaging, checksums,
  attestations, and updates.
- [Brand asset pipeline](brand-assets/) explains canonical artwork and
  deterministic platform assets.

## Maintainer references

These records are intentionally absent from the user-guide navigation:

- [Windows safety invariants](../safety/windows/)
- [Logging contract](../safety/logging/)
- [Native Windows acceptance checklist](../safety/native-test-checklist/)
- [S1-mini post-processing decision](../decisions/0001-s1-mini-post-processing/)
- [Realtime transcription research](../decisions/0002-realtime-transcription/)
- [Windows platform boundaries](../decisions/0003-windows-platform-boundaries/)
- [Native overlay and Wails boundary](../decisions/0004-native-overlay-and-wails-boundary/)
- [Remote-first product direction](../decisions/0005-remote-first-product-direction/)
