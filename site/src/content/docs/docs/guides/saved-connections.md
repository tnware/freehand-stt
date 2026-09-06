---
title: Saved connections
description: Create server connections in one place, then choose which connection each feature uses.
---

Open **Settings → Connections** to create and manage the servers Freehand can
use. Connections is under **Application**; Transcription, Post-processing, and
Speech playback are grouped under **Features**. New installations start with an empty list. Existing configured endpoints
are retained when upgrading. A connection represents one reusable server: its
URL, backend profile, authentication, and HTTP permission are shared. Enable
Transcription, Post-processing, and/or Speech playback under **Used for**, then
select the same connection independently in each feature.

## Create a connection

1. Choose **New connection**, enter a name, and choose the server's **Compatibility profile**.
2. Enable its **Used for** switches and enter its base URL. Only uses with an
   implemented contract for that profile can be enabled. Choose the operations
   your deployed server actually exposes; a backend name does not prove this.
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
| Connection name and supported uses | Active connection selection |
| Compatibility profile and base URL | Model, model profile, and language |
| Authentication and stored API key | Cleanup preset and instructions |
| Allow insecure HTTP | Voice, speed, and feature enable switches |
| Transcription health path and headers | Timeouts and provider-specific options |

**Save connection** updates only that connection. Editing an active connection
applies its endpoint settings and key to new requests from **every feature using
that server** in one save. Renames and key changes retain model preferences;
changing the URL or backend profile clears them and the active model choices. **Save feature settings** does not edit the saved
connection. The home screen has the same active connection selectors. Its separate settings
links open feature settings; model and other quick controls save runtime settings.

## Reuse a server

A Speaches connection can enable both Transcription and Speech playback, while
vLLM can enable Transcription and Post-processing. Generic offers all three uses;
llama.cpp currently offers Post-processing, and whisper.cpp offers Transcription.
These choices describe Freehand's implemented contracts, not detected server
capabilities. A particular vLLM deployment may expose only one operation.

To extend an existing connection, **Edit** it, enable another supported use, and
**Save connection**. Then select it on the other feature page and choose that
feature's model. Use separate connections when URLs, credentials, or backend
profiles differ. Transcription's custom health path and headers apply only to
transcription; they do not get sent through cleanup or playback adapters.

## Switch, edit, duplicate, or delete

- Choose an **Active connection** on a feature page to switch immediately.
  A different connection restores its last selected model and remembered options
  for that feature. A connection without a remembered model starts with defaults;
  cleanup and speech playback turn off until configured and enabled again, and
  transcription needs its setup completed again. Running jobs keep their captured settings and keys.
- Choose **Edit connection** to open the selected entry in Connections. Save or
  discard feature edits first. The **Back** button at the top returns to the
  feature you came from, or to the connection list when editing there. Saving
  returns to that feature too. Simply viewing a connection needs no save or
  discard. If you changed something, Back, Cancel, or choosing another section
  lets you keep editing or discard the connection edits.
- **Duplicate** creates an inactive copy with no remembered model preferences. Later edits and key replacements affect
  only that copy. Rename it through **Edit** if needed.
- **Delete** removes an inactive entry after confirmation. To delete an active
  entry, select another connection or **None** in every feature using it first.
  The same rule applies before removing an enabled use from a connection.
  Deleting the last inactive entry is allowed.

New names must be unique across the connection library, with up to 32 available
connections per feature. Upgrades preserve existing names and entries; duplicate
URLs are not automatically merged because their authentication or uses may differ.
Language, voice, instructions, and provider options follow the remembered model
for the selected connection and feature. See [Model profiles](../model-profiles/)
for choosing, saving, and forgetting those preferences.

## Test and protect credentials

**Test connection** in Connections checks that saved entry, including its own
credential, without selecting it. It reads health or model-list metadata only.
Results distinguish metadata access and authentication. They do not assess every
feature that can use this connection. Open a feature and choose **Refresh models**
to check its selected model and local option requirements too. See
[connection-check results](../troubleshooting/#understand-connection-check-results)
for interpreting advertised models, health-only checks, and stale results.
A successful metadata check does not establish inference compatibility or
inference authorization.

Keys stay in Windows Credential Manager. The app never displays a stored key;
leave its password field blank to keep it, enter a replacement, or explicitly
remove it. Canceling or leaving settings clears the transient key draft.
SQLite and backups contain opaque credential references only. A duplicate
initially shares that reference; replacing its key creates an independent one.
Deletion reclaims a key only when no saved connection uses it. Failed saves
preserve the previously committed settings and credentials.

See [connect a server](../connect-a-server/), [backend guides](../../backends/),
and [settings recovery](../troubleshooting/#saved-settings-need-attention).
