---
title: Saved connections
description: Create server connections in one place, then choose which connection each feature uses.
---

Open **Settings → Connections** to create and manage the servers Freehand can
use. New installations start with an empty list. Existing configured endpoints
are retained when upgrading. Each connection belongs to Transcription,
Post-processing, or Speech playback; those features have independent selections.

## Create a connection

1. Choose **New connection**, enter a name, and choose **Used for**.
2. Choose the server's **Compatibility profile** and enter its base URL.
3. Configure authentication and explicitly allow HTTP if your trusted server
   uses it. Transcription also offers a custom health path and non-secret headers.
4. Choose **Save connection**. This saves the entry without activating it.
5. Open the corresponding feature page and choose its **Active connection**.
6. List models or enter the exact model ID, configure the feature's options,
   and choose **Save feature settings**. whisper.cpp uses its server-loaded model.

For first-time transcription setup, return to the readiness screen, explicitly
**Test connection**, and finish setup. See [Get started](../../getting-started/).

## What goes where?

| Settings → Connections | Feature settings pages |
| --- | --- |
| Connection name and purpose | Active connection selection |
| Compatibility profile and base URL | Model and language |
| Authentication and stored API key | Cleanup preset and instructions |
| Allow insecure HTTP | Voice, speed, and feature enable switches |
| Transcription health path and headers | Timeouts and provider-specific options |

**Save connection** updates only that connection. Editing an active connection
applies its endpoint settings to new requests; its model and feature options
remain on the feature page. **Save feature settings** does not edit the saved
connection. The home-screen connection labels link to feature settings; model
and other quick controls still save their own runtime settings.

## Switch, edit, duplicate, or delete

- Choose an **Active connection** on a feature page to switch immediately.
  A different connection clears that feature's model choice; cleanup and speech
  playback turn off until configured and enabled again. Transcription needs its
  setup completed again. Running jobs keep their captured settings and keys.
- Choose **Edit connection** to open the selected entry in Connections. Save or
  discard feature edits first. Save or cancel a connection form before navigating.
- **Duplicate** creates an inactive copy. Later edits and key replacements affect
  only that copy. Rename it through **Edit** if needed.
- **Delete** removes an inactive entry after confirmation. To delete an active
  entry, select another connection or **None** on its feature page first.
  Deleting the last inactive entry is allowed.

Names are unique within each feature, with up to 32 connections per feature.
Switching preserves language, voice, instructions, and provider options. If the
new connection cannot accept an enabled provider option, the switch fails and
keeps your current setup; adjust and save that option before switching.
Separate saved preferences for each model are future work.

## Test and protect credentials

**Test connection** in Connections checks that saved entry, including its own
credential, without selecting it. It reads health or model-list metadata only.
A successful check establishes reachability, not inference compatibility.

Keys stay in Windows Credential Manager. The app never displays a stored key;
leave its password field blank to keep it, enter a replacement, or explicitly
remove it. Canceling or leaving settings clears the transient key draft.
SQLite and backups contain opaque credential references only. A duplicate
initially shares that reference; replacing its key creates an independent one.
Deletion reclaims a key only when no saved connection uses it. Failed saves
preserve the previously committed settings and credentials.

See [connect a server](../connect-a-server/), [backend guides](../../backends/),
and [settings recovery](../troubleshooting/#saved-settings-need-attention).
