---
title: Native acceptance
description: Validate desktop integration on Windows and macOS without confusing fixtures with native evidence.
---

Run the checks relevant to the change on each affected OS and architecture.
Use disposable text, credentials, and isolated test data. Record the application
revision, OS, architecture, actions, results, and unverified cases in the issue
or pull request.

Compilation, browser fixtures, and cross-compilation do not establish native
behavior. On macOS, permission acceptance requires a packaged app with a stable
bundle identity. A local ad-hoc signature does not establish signing or notarization
acceptance. Opt-in `FREEHAND_NATIVE_*` tests can use real devices or OS resources;
inspect the test before enabling it. Skipped tests establish no native evidence.

## Capture and delivery

- Test toggle and hold shortcuts, real key release, repeated activation,
  cancellation, conflicting bindings, and permission denial. Normal input must
  remain usable. On macOS, include Secure Input and event-tap interruption.
- Exercise default and explicitly selected microphones, device removal, and
  repeated start/stop/cancel. Permission denial must leave file transcription and
  text to speech independently usable.
- Use disposable targets to check Unicode, emoji, multiline and long text.
  Change application/window, close the target, and cancel during processing or
  insertion. Delivery must stop when the captured target becomes invalid;
  never reactivate it or insert into a different window. On macOS, changing
  fields within the same window delivers to its currently focused field.
- Check explicit-copy recovery and clipboard contention. A rejected delivery
  may have inserted some text already; verify the warning and inspect the target
  before pasting. Automatic delivery must not mutate unrelated clipboard content.
- If touching realtime, verify that provisional captions never authorize
  insertion or retention. Test disconnects and cancellation without replay.
  Cleanup failure must preserve the finalized raw result.

## Desktop lifecycle and presentation

- Check tray/menu-bar behavior, close-to-hide, second launch, and reopening the
  main window; include Dock reopen on macOS. Opening the tray must not start
  capture. Start dictation from the destination using its global shortcut.
- Verify passive overlays never take focus or intercept input. Check changed UI
  in both themes, with keyboard navigation, OS scaling, and multiple monitors.
- Quit during affected work: capture, upload, cleanup, playback, runtime startup,
  or native dialogs. Confirm actual process exit, resource release, and no late
  insertion, playback, or export. A service timeout is not proof of native cleanup.
- Test changed startup, installer, or updater behavior with the packaged app and
  actual next launch/login. Follow the [release procedure](../../development/releases/)
  for artifact, signature, and update checks.

## Settings, credentials, and local runtimes

- Test save/reopen, failure recovery, and native vault replacement with disposable
  credentials. Secrets must not reach SQLite, logs, argv, events, or returned
  renderer snapshots. Clear password drafts after save, navigation, and hide.
- Use isolated databases for corruption, newer-schema, reset, and backup recovery.
  Keep personal settings and legacy files untouched. History stays off by default;
  captured audio must not survive completion, failure, cancellation, or shutdown.
- For runtime changes, exercise explicit install/download/cancel/start/stop and
  owned descendant cleanup, including parent termination. Unrelated servers and
  manual connections must remain untouched. Local failure must not trigger remote
  fallback or leak manual credentials to a managed endpoint.
- Use only an explicitly selected endpoint/model for live inference. Catalog and
  health checks remain metadata-only. Record actual CPU/GPU execution separately
  from installation, CLI help, health responses, and browser fixtures.
- For output-viewer changes, verify bounded, read-only display and revocation when
  hidden. Standalone output requires consent; terminal content must never trigger
  input, links, clipboard writes, files, or logging. See the [logging rules](../logging/).
