---
title: Keyboard shortcuts
description: Configure global recording, hold-to-talk, and show-window shortcuts.
---

Freehand exposes three global shortcut actions. Configure them under
**Settings → Shortcuts** by choosing a field and pressing the combination you
want to use.

| Action | Required | Accepted forms | Default |
| --- | --- | --- | --- |
| Toggle recording | Yes | One or more modifiers plus a supported primary key; or F13-F24 alone | `Ctrl+Shift+Space` |
| Show Freehand | Yes | One or more modifiers plus a supported primary key; or F13-F24 alone | `Ctrl+Shift+D` |
| Hold to talk | No | The global forms above; or two or more modifiers alone | Unassigned |

The supported primary-key groups are A-Z, 0-9, Space, F1-F11, and F13-F24.
F12 is rejected because Windows reserves it for the debugger. F13-F24 are the
only unmodified keys accepted: they are intended as dedicated programmable
keys and avoid taking over ordinary typing or navigation.

## Deliberately excluded inputs

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

## Conflicts and capture results

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
