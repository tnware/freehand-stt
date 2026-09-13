---
title: macOS setup and permissions
description: Run a macOS source build and recover native microphone, keyboard and insertion access.
---

The macOS build requires macOS 13 or newer. Use a packaged `Freehand.app`
built for your Mac's architecture; running a bare executable from Terminal gives
macOS a different permission/signing context. See the contributor build instructions
for source builds. Windows release downloads are not macOS applications.

A local ad-hoc-signed build is not notarized or a trusted Developer ID release.
Do not disable Gatekeeper globally. Keep the bundle in a stable location before
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
being granted does not guarantee every app/control permits insertion: password
fields, Secure Input, protected apps, changed focus or a closed target can still
require explicit copy.

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
will not bring that app back to the foreground. If you move to another field or
window, the result may remain **Copy required** rather than being typed elsewhere.
Release physical modifiers before delivery. Use explicit **Copy** to recover the
transcript; normal Unicode delivery does not replace unrelated clipboard content.

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

## Reporting a native problem

Include macOS version, Apple Silicon or Intel, app version, affected workflow,
permission states, whether the app was moved/rebuilt, and exact non-secret steps.
Distinguish local build success from native behavior you actually observed. Do not
attach API keys, transcripts, microphone recordings, endpoint URLs or unredacted
window/target identifiers.
