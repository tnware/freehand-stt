# Linux insertion experiment boundary

The Linux adapter currently preserves transcripts for explicit Copy. This is not
because field identity is mandatory: app/window identity can be sufficient under
the existing delivery contract. Capturing a credible X11 target and delivering
arbitrary Unicode to that target are separate problems.

## Candidate: qualified accessibility transport

A future, disabled-by-default experiment can support controls with AT-SPI
EditableText without installing an IME, changing the keyboard map, using the
clipboard, or synthesizing a generic paste shortcut:

1. Capture the original local X11 window with XRes client PID, process starttime,
   XID, and a capture generation invalidated by destruction. Never trust the
   client-provided `_NET_WM_PID` property as process provenance.
2. Authenticate the accessibility bus unique connection's process against that
   native owner. Retain an accessible top-level identity only after establishing
   an unambiguous relationship to the captured native window.
3. Select a currently focused editable object *inside that retained top-level*.
   Original field identity is optional. If a reliable original field identity
   exists, apply the same confirmed-change rejection rule as other platforms;
   its absence alone must not rule out app/window-bound delivery.
4. Require a qualified editable text role, enabled/editable/focused state, a
   usable caret, and no selection. Reject password/protected/unknown controls.
   Read no text contents, names, descriptions, or bulk accessibility caches.
5. Immediately before one bounded UTF-8 InsertText call, revalidate the original
   X11 window, PID/starttime, accessible ancestry, modifier state and cancellation.
   Address the RPC to the exact bus unique name and object path. Do not activate
   a window, delete a selection, replace all text, or redirect to another object.
6. A timeout or disconnect after dispatch means insertion may have happened.
   Preserve an ambiguous terminal result and never retry automatically. Initially
   exclude realtime insertion; additional chunks require fresh validation.

This does not require a universal field-identity fallback. Accessibility is a
possible Unicode transport for a qualified subset of controls, not a new global
requirement on all native insertion implementations.

## Precise missing evidence

Standard AT-SPI Accessible and Component interfaces do not expose a standardized
native X11 window ID. A PID match does not distinguish multiple windows in the
same process. An accessible ACTIVE flag or unique active top-level may be a useful
signal, but has not been qualified as an exact native-window mapping here. Do not
replace mapping with titles, geometry, enumeration order, or a presumed first
window. A toolkit-specific mapping or demonstrated, retained accessibility-window
identity contract must be established before enabling this path.

AT-SPI addressed insertion cannot be redirected to a newly focused application's
object, but may modify the original object after a last-moment focus change.
There is no general atomic focus-check-and-UTF-8-insert operation. This race is
not itself a reason to demand more than other platforms: macOS currently validates
then calls CGEventPostToPid separately. Compare a candidate against that bounded
preflight guarantee, while reporting post-dispatch ambiguity honestly.

Clipboard opt-in does not solve these problems. Wails exposes a text clipboard,
not a transactional snapshot of all lazy selection formats, INCR transfers and
ownership changes. Current-focus paste has no trustworthy completion acknowledgment.
Do not add clipboard-paste or IBus injection as an allegedly universal fallback.

## Design acceptance tests

These are requirements, not tests claimed to have run:

- Ignore spoofed `_NET_WM_PID`; missing XRes/starttime fails closed.
- Reject PID reuse, XID destruction/reuse, and a different window in the same PID.
- Reject accessibility process mismatch, ambiguous top-level association and
  unsupported transport without reading any field content.
- Exercise two windows and several fields in one real GTK application. A field
  change within the retained window follows optional-field policy; a different
  top-level always rejects. Repeat against each explicitly supported toolkit.
- Reject password/unknown roles, active modifiers, selection and cancellation
  before dispatch, leaving the clipboard and target contents untouched.
- Change focus at every preflight boundary and prove no dispatch occurs. Inject
  a focus change after final preflight to document the remaining race explicitly.
- Delay a successful InsertText reply beyond timeout: outcome is ambiguous, with
  zero retry and no later chunk. Repeat for disconnect and object destruction.
- Check UTF-8 including non-Latin text and emoji, caret semantics, bounded chunks,
  and cancellation without reading existing private text.
- Assert no clipboard writes, keymap changes, activation, GetText, bulk GetAll,
  or content-cache requests. Native tests use only dedicated synthetic fields.

## Primary contracts

- [AT-SPI Accessible](https://raw.githubusercontent.com/GNOME/at-spi2-core/main/xml/Accessible.xml)
- [AT-SPI Component](https://raw.githubusercontent.com/GNOME/at-spi2-core/main/xml/Component.xml)
- [AT-SPI EditableText](https://raw.githubusercontent.com/GNOME/at-spi2-core/main/xml/EditableText.xml)
- [X11 ICCCM selections](https://xorg.freedesktop.org/releases/current/doc/xorg-docs/icccm/icccm.html)

No automatic insertion implementation is enabled by this design note.
