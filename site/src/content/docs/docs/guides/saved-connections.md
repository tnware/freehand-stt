---
title: Saved connections
description: Keep named server connections and switch them independently for transcription, cleanup, and speech playback.
---

Each capability has a **Saved connection** selector at the top of its Settings
section. Transcription, Post-processing, and Speech playback have independent
lists and selections. A saved connection is your named endpoint configuration;
its **Compatibility profile** describes the server's request contract.

Your existing setup becomes one saved connection per capability automatically,
including its credential reference. You can rename these entries immediately.
No server or model is started when a connection is created or selected.

## Save and switch

- **Edit a connection:** change the fields below the selector and choose
  **Save changes**. This updates the selected connection.
- **Save as new:** edit the connection values, choose **Save as new**, and enter
  a name. The new entry becomes selected. Only a newly entered API key is attached;
  an existing stored key is not copied to a different endpoint automatically.
- **Duplicate:** copy the selected saved connection, give it a new name, and
  select **Duplicate and use**. The copy initially uses the same credential
  reference. Later key replacements or other edits affect only the edited entry.
- **Switch:** choose a saved entry. Selection applies immediately to new requests;
  requests already running keep their captured endpoint, model, and credentials.
- **Rename:** change the display name without changing connection behavior.
- **Delete:** choose another saved connection in the confirmation dialog, then
  select **Delete and switch**. At least one entry must remain for that capability.

Save or discard ordinary edits before switching, duplicating, renaming, or
deleting. **Save as new** intentionally accepts your current draft. Names must be
unique within a capability; each capability supports up to 32 saved connections.

## What belongs to a connection?

Connections remember endpoint URL, HTTP permission, compatibility profile, last
selected model, and their credential reference. Transcription connections also
remember authentication mode, custom headers, and health path. Speech playback
connections remember authentication mode. Cleanup connections remember the
selected cleanup preset, so an explicitly chosen S1-mini preset stays associated
with that connection's model selection.

Language, timeouts, provider overrides, custom cleanup instructions, voice,
speed, and feature enable switches remain settings of their respective operation.
If a connection cannot accept a currently enabled provider option, Freehand
rejects the switch and keeps the existing setup. Adjust that option and save
before switching; Freehand does not silently erase it. Selecting an incomplete
connection can require setup again or disable an unconfigured optional capability.
Remembering a separate set of options for every model is future work.

## Credentials and privacy

Keys stay in Windows Credential Manager. Saved connections and SQLite backups
contain only opaque references, and the interface never receives stored keys.
Deleting a connection reclaims its key only when no other saved connection uses
that reference. A failed save preserves the previously committed selection and
credentials. Reset and backup behavior is described in
[settings recovery](../troubleshooting/#saved-settings-need-attention).

Connection tests remain metadata-only. They can read health or model listings,
but do not invoke models or establish inference compatibility. See
[connect a server](../connect-a-server/) and the [backend guides](../../backends/).
