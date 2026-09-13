---
title: macOS setup and permissions
description: Install and verify the macOS ZIP, manage permissions, update, and uninstall Freehand.
---

## Download, verify, and install

Freehand requires macOS 13 or newer. In **Apple menu → About This Mac**, check
whether your Mac has an Apple chip (Apple Silicon) or an Intel processor.

1. Open the official [GitHub Releases](https://github.com/tnware/freehand-stt/releases).
   Choose `freehand-darwin-arm64.zip` for Apple Silicon or
   `freehand-darwin-amd64.zip` for Intel. Download `SHA256SUMS` from that **same
   release**, not another tag or a third-party download site. Older releases may
   have no macOS assets; choose a release that includes your architecture.
2. Before opening the ZIP, open Terminal in its download folder and compute its
   SHA-256 hash:

   ```sh
   shasum -a 256 freehand-darwin-arm64.zip
   ```

   For Intel, substitute `freehand-darwin-amd64.zip`. Compare the entire hash
   with the entry for that exact filename in `SHA256SUMS`. If the entry is
   missing or the hashes differ, stop and download again from the official
   release. A checksum verifies matching bytes, not publisher identity.
3. Double-click the verified ZIP in Finder. Move the extracted **Freehand.app**
   to **Applications** (or your account's Applications folder) before enabling
   permissions or login startup. Run the app bundle, not its internal executable.
4. Open **Freehand** from Applications. The alpha is **ad-hoc-signed, not
   Developer ID-signed or notarized**, so Gatekeeper may block its first launch.
   If you trust the verified official download, use **System Settings → Privacy
   & Security → Open Anyway** after the blocked attempt, then confirm macOS's
   prompt. If that option is unavailable or macOS reports malware, stop rather
   than bypassing the protection. Never disable Gatekeeper globally or remove
   quarantine recursively as a routine installation step.
5. Review the permissions below, then follow [Get started](../../getting-started/).
   Open About to confirm the installed version matches the chosen release.

For source builds, see the
[contributor instructions](https://github.com/tnware/freehand-stt/blob/main/CONTRIBUTING.md).
The development bundle has a separate `.dev` permission identity but still
appears as **Freehand**; grant permissions to the bundle you actually run.

Keep the bundle in a stable location before
enabling permissions or **Start at login**. Rebuilding, changing identity/signature,
or moving a bundle can require permission review. macOS, not Freehand, owns these
permissions and any password or approval prompts.

## Choose a workflow

- **Voice dictation** needs a microphone and a selected transcription service.
- **Audio file transcription** needs its selected service, but no microphone,
  Accessibility permission, or recording shortcut.
- **Text to speech** needs its own enabled speech service. It does not require
  microphone, transcription, or keyboard permissions.

API credentials are kept in your macOS Keychain. The app does not return stored
keys to the renderer, store them in its settings database, or fall back to plaintext
if Keychain access is denied. Enter credentials only in Freehand's connection editor;
never include them in logs, commands or bug reports.

## Native permissions

Open **Settings → General → macOS permissions** to check access. Permission
checks do not prompt. Use the explicit permission or System Settings action when
needed, and complete macOS's own approval flow yourself.

| Permission | What uses it | Recovery |
| --- | --- | --- |
| Microphone | Recording and microphone preview | Privacy & Security → Microphone |
| Accessibility | Safe target inspection, Unicode delivery and shortcut capture | Privacy & Security → Accessibility |
| Input Monitoring | Hold-to-talk press/release and shortcut capture | Privacy & Security → Input Monitoring |

After changing keyboard permissions, quit and reopen Freehand if macOS requests
it. Refresh the permission status after returning from System Settings. A permission
being granted does not guarantee every app/control accepts insertion. Secure
Input, protected apps, changed app/window focus or a closed target can still
require explicit copy. Freehand does not classify password fields; custom secure
fields that do not enable macOS Secure Input are not guaranteed to be detected.

If microphone access is denied, use the Microphone settings pane to enable it;
repeatedly pressing Record cannot override a denied OS decision. A restricted
permission may be controlled by device management. No audio capture is started by
merely displaying permission status.

## Shortcuts and delivery

Mac shortcut labels use **Command** and **Option**. The native shortcut backend
uses physical ANSI/QWERTY key positions; check behavior on your keyboard layout.
Avoid system-reserved combinations. Hold-to-talk needs both press and release
observation, not just a toggle shortcut.

Toggle and Show Freehand use Carbon global shortcuts and do not require
Accessibility or Input Monitoring. Recording a new chord needs both permissions;
clearing an optional shortcut does not. If a saved hold shortcut cannot start,
Freehand keeps the saved preference and the independent Toggle/Show bindings.

After unlocking the session, leaving Secure Input, or restoring Input Monitoring,
release all keys and choose **Settings → Shortcuts → Retry hold-to-talk**. This
explicit, bounded action rearms the saved chord even when settings are unchanged;
it does not save or discard your draft. Save shortcut edits first if you want to
retry a different chord. Failed attempts remain visible and can be retried after
resolving the cause. Merely opening settings or refreshing permissions does not
rearm the hook or request access.

Start dictation while the intended app and editable field are focused. Freehand
will not bring that app back to the foreground. It checks that the same app and
window are still frontmost, not that the same editor field is focused. Moving to
another field in that window delivers there; moving to another app or window
leaves the result **Copy required**. macOS Secure Input blocks delivery while
active; Freehand does not classify individual editor fields.
Release physical modifiers before delivery. Use explicit **Copy** to recover the
transcript; normal Unicode delivery does not replace unrelated clipboard content.

**Copy required** does not always mean focus changed. When available, the message
includes a bounded capture, validation or send reason. Delivery can be partial;
check the target for existing text before pasting to avoid duplicates. Freehand
does not automatically retry ambiguous typing.

The status overlay is passive and click-through. It is not a transcript editor or
an insertion destination. Use the main app or menu-bar item for actions.

## Closing, reopening and login startup

Closing an interactive window hides it. Use the menu-bar item or Dock to reopen
the main window; use **Quit Freehand** or **Command-Q** to exit. A second launch
reveals the existing instance rather than starting another recorder.

**Start at login** registers the current bundle executable for your next login; it
does not immediately start another instance. **Disable Start at login before
moving the bundle**, then open it at the new location and enable the setting
again. Freehand deliberately refuses to overwrite a registration pointing to a
different executable. If you already moved the bundle, move it back temporarily
and disable startup there before relocating it. macOS can also disable
background/login items in System Settings. Disabling startup in Freehand removes only its own safe,
per-user registration, not unrelated login items.

## Update

The in-app updater selects the matching-architecture ZIP, verifies it against
the release’s `SHA256SUMS`, extracts the app, and uses the Wails helper to replace
the app bundle when you choose Restart. This is checksum verification, not
Developer ID trust or notarization. If an in-app update fails, use manual
replacement:

1. Read the newer release notes and download the matching-architecture ZIP plus
   its same-release `SHA256SUMS`. Verify it using the installation steps above.
2. Disable **Start at login** before replacing or moving the app, then choose
   **Quit Freehand** or **Command-Q**. Closing its window is not quitting.
3. Extract the new ZIP and replace the old app in the same Applications location.
   Keep private settings backups before an upgrade; do not delete application data.
4. Open the replacement, check its version in About, and review microphone,
   Accessibility, and Input Monitoring access. Signature changes can require
   permission approval again. Re-enable **Start at login** if desired.

An older binary cannot downgrade a newer settings database. Review release notes
and [settings recovery](../troubleshooting/#saved-settings-need-attention) before
attempting a downgrade.

## Uninstall

1. Disable **Start at login** while the app is still at its registered location.
2. Choose **Quit Freehand**, then move **Freehand.app** to Trash in Finder.
3. Optionally remove Freehand from Accessibility and Input Monitoring in
   **System Settings → Privacy & Security**.

Removing the app does not remove saved settings, backups, or Keychain entries.
For a deliberate data reset, first remove saved connections/credentials in the
app, then quit it and remove only `~/Library/Application Support/Freehand` in
Finder (**Go → Go to Folder**). Keep any configuration you want to restore;
removal is destructive and history is already memory-only. If credentials
remain after removing the app, review only Freehand's entries in Keychain Access;
do not delete unrelated credentials.

## Reporting a native problem

Include macOS version, Apple Silicon or Intel, app version, affected workflow,
permission states, whether the app was moved/rebuilt, and exact non-secret steps.
Distinguish local build success from native behavior you actually observed. Do not
attach API keys, transcripts, microphone recordings, endpoint URLs or unredacted
window/target identifiers.
