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
4. Open **Freehand** from Applications. The app is **ad-hoc-signed, not
   Developer ID-signed or notarized**, so Gatekeeper may block its first launch.
   If you trust the verified official download, use **System Settings → Privacy
   & Security → Open Anyway** after the blocked attempt, then confirm macOS's
   prompt. If that option is unavailable or macOS reports malware, stop rather
   than bypassing the protection. Never disable Gatekeeper globally or remove
   quarantine recursively as a routine installation step.
5. Review the permissions below, then follow [Get started](../../getting-started/).
   Open About to confirm the installed version matches the chosen release.

Keep the app in a stable location before enabling permissions or **Start at
login**. Moving or replacing it can require permission approval again. If you
also use a development build, it has separate permissions even though both
apps display **Freehand**. Grant access to the app you actually run.

## Choose a workflow

- **Voice dictation** needs a microphone and a selected transcription service.
- **Audio file transcription** needs its selected service, but no microphone,
  Accessibility permission, or recording shortcut.
- **Text to speech** needs its own enabled speech service. It does not require
  microphone, transcription, or keyboard permissions.

API credentials are kept in your macOS Keychain, not in the settings database.
Freehand does not display saved keys or store them in plaintext if Keychain access
is denied. Enter credentials only in Freehand's connection editor;
never include them in logs, commands or bug reports.

## Native permissions

Open **Settings → General → macOS permissions** to check access. Permission
checks do not prompt. Use the explicit permission or System Settings action when
needed, and complete macOS's own approval flow yourself.

| Permission | What uses it | Recovery |
| --- | --- | --- |
| Microphone | Recording and microphone preview | Privacy & Security → Microphone |
| Accessibility | Checking the destination, inserting text, and capturing shortcuts | Privacy & Security → Accessibility |
| Input Monitoring | Hold-to-talk and capturing shortcuts | Privacy & Security → Input Monitoring |

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

Mac shortcut labels use **Command** and **Option**. Shortcuts use physical
ANSI/QWERTY key positions; check the combination on your keyboard layout and
avoid system-reserved combinations.

The saved **Toggle** and **Show Freehand** shortcuts do not require Accessibility
or Input Monitoring. Capturing a new shortcut needs both permissions; clearing
an optional shortcut does not. If hold-to-talk is unavailable, Freehand keeps
your saved shortcut and the separate Toggle and Show Freehand shortcuts.

After unlocking the session, leaving Secure Input, or restoring Input Monitoring,
release all keys and choose **Settings → Shortcuts → Retry hold-to-talk**. This
action restores the saved shortcut without saving or discarding draft edits.
Save shortcut changes first if you want to retry a different combination.
If it fails, resolve the reported cause and try again. Opening Settings or
refreshing permissions alone does not restore hold-to-talk.

Start dictation while the intended app and editable field are focused. Freehand
will not bring that app back to the foreground. It checks that the same app and
window are still frontmost, not that the same editor field is focused. Moving to
another field in that window delivers there; moving to another app or window
leaves the result **Copy required**. macOS Secure Input blocks delivery while
active; Freehand does not classify individual editor fields.
Release physical modifiers before delivery. Use explicit **Copy** to recover the
transcript; normal Unicode delivery does not replace unrelated clipboard content.

**Copy required** does not always mean focus changed. When available, the message
explains why delivery stopped. Delivery can be partial;
check the target for existing text before pasting to avoid duplicates. Freehand
does not automatically retry ambiguous typing.

The status overlay is passive and click-through. It is not a transcript editor or
an insertion destination. Use the main app or menu-bar item for actions.

## Closing, reopening and login startup

Closing an interactive window hides it. Click the menu-bar item for the compact
panel, then **Open Freehand** for the full workspace, or reopen Main from the Dock.
Right-click the menu-bar item for **Quit Freehand**, or use **Command-Q** to exit.
A second launch reveals the existing instance rather than starting another recorder.

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
the release's `SHA256SUMS`, and replaces the app when you choose **Restart**.
This is checksum verification, not
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
permission states, whether the app was moved or replaced, and steps to reproduce
what happened. Do not attach API keys, transcripts, microphone recordings,
private endpoint URLs, full file paths, or unredacted diagnostics.
