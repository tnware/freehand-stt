---
title: Keyboard shortcuts
description: Configure global recording, hold-to-talk, and show-window shortcuts.
---

Freehand exposes three global shortcut actions. Configure them under
**Settings → Shortcuts** by choosing a field and pressing the combination you
want to use.

All three actions may be left unassigned. Use **Clear** and save to disable a
shortcut; saving an empty toggle does not restore the default. You can finish
Voice setup and use the on-screen recording controls without a global shortcut.
For insertion into another application, a working shortcut lets you start while
that destination is focused; otherwise review the result and copy it explicitly.

## Tray quick view

Click the tray or menu-bar icon to see dictation status, your recording shortcut,
the selected Voice connection, and your latest transcript. Use **Copy** for an
explicit clipboard action. **Voice settings** opens configuration in the main
window; **Open Freehand** opens the workspace.

To record into another app, close quick view, focus the destination text field,
and use your recording shortcut. Quick view does not start or stop recordings.
During recording or processing, **Cancel recording** or **Cancel dictation**
discards the active operation. Closing quick view leaves work running. Right-click
the tray icon for the native menu, including **Quit Freehand**.

## Windows

| Action | Required | Accepted forms | Default |
| --- | --- | --- | --- |
| Toggle recording | No | One or more modifiers plus a supported primary key; or F13-F24 alone | `Ctrl+Shift+Space` |
| Show Freehand | No | One or more modifiers plus a supported primary key; or F13-F24 alone | Unassigned |
| Hold to talk | No | The global forms above; or two or more modifiers alone | Unassigned |

The supported primary-key groups are A-Z, 0-9, Space, F1-F11, and F13-F24.
F12 is rejected because Windows reserves it for the debugger. F13-F24 are the
only unmodified keys accepted: they are intended as dedicated programmable
keys and avoid taking over ordinary typing or navigation.

## macOS

Toggle recording defaults to **Control+Shift+Space**. Show Freehand and Hold to
talk are optional and initially unassigned. Labels use **Command** and **Option**.
Supported primary keys are A–Z, 0–9, Space, and **F1–F20, including F12**.
**F21–F24 are not mapped on macOS**. Toggle/Show accept a modifier plus a
supported primary key, or an unmodified F13–F20 key. Hold to talk also accepts
two or more modifiers alone. Native mappings use physical ANSI/QWERTY positions;
check your layout and avoid system-reserved combinations. Fn itself is not a
recordable modifier; hardware may use it to produce a function-key event.

Toggle/Show use Carbon global shortcuts without Accessibility or Input Monitoring.
Capturing a new shortcut needs both permissions; hold-to-talk needs keyboard
observation. After restoring access or leaving Secure Input, release all keys
and choose **Retry hold-to-talk**. See [macOS permissions and shortcut recovery](../../guides/macos-setup/).

## Deliberately excluded inputs (Windows)

- Navigation and editing keys are not accepted because a failed or delayed
  global registration could interfere with ordinary document use.
- OEM punctuation is not accepted because its virtual-key meaning varies with
  keyboard layout and AltGr behavior.
- Media, browser, Caps Lock, Num Lock, and Scroll Lock keys retain their system
  or hardware purpose.
- `Fn` is not a standalone Windows virtual key on typical keyboards. A device
  may translate an Fn combination into an ordinary supported key, including a
  programmable F13-F24 key, but Freehand does not interpret Fn itself.
- Left and right versions of Ctrl, Alt, Shift, and Win normalize to the same
  logical modifier. This matches `RegisterHotKey` and keeps persisted chords
  portable between keyboards.

Ctrl+Alt chords may overlap AltGr on international layouts. Freehand therefore
does not describe a locally valid chord as universally safe or available. The
settings UI explains the accepted shape, while native acceptance testing must
cover the user's representative layouts and accessibility-key configuration.

## Conflicts and capture results (Windows)

Freehand can identify incomplete, unsupported, Windows-reserved, duplicate,
timed-out, and locally unavailable capture outcomes before a settings save.
These are returned as structured categories with bounded UI messages.

Windows does not expose a supported inventory of global shortcuts owned by
other processes. Toggle and show availability is therefore known only when
`RegisterHotKey` runs during Save. A Windows rejection is reported separately
from an in-app duplicate, and the settings transaction restores the complete
previous shortcut set rather than persisting a partially applied replacement.

Shortcut capture temporarily pauses Freehand's working shortcuts. Press Escape
to cancel. A captured, rejected, cancelled, or timed-out attempt restores the
previous bindings.

If a saved shortcut is unavailable at startup, Freehand keeps your saved choice
but leaves that binding inactive. Other available shortcuts still work. No
fallback key is selected automatically. Open **Settings → Shortcuts** to record
a replacement directly, or **Clear**, save, and record one later. Capture only
pauses and restores bindings Freehand successfully registered; an existing
startup conflict does not prevent recording a replacement.
