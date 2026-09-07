---
title: Saved connections
description: Create server connections in one place, then choose which connection each feature uses.
---

Start from the task you want to use. Its connection picker offers **Add connection…**;
feature Settings pages also have an **Add connection** button when none is selected.
The same editor is available in **Settings → Connections** for managing your library.
New installations start with an empty list; upgrading retains existing connections.

A connection represents one reusable server: its URL, backend profile, authentication,
and HTTP permission. Transcription, Cleanup, and Text to speech select connections
independently and can share the same server.

## Add a connection while setting up a task

1. Open the task's connection picker and choose **Add connection…**.
2. Enter a recognizable name, choose the server's **Compatibility profile**, and
   enter its base URL. The task you came from is already selected under supported uses.
3. Configure authentication and explicitly allow HTTP if your trusted server uses it.
   **Also use this server for other tasks** lets you declare additional operations
   the deployment exposes; this does not select it for those tasks.
4. Choose **Save and use connection**. Saving and selecting happen together. On
   failure, the previous selection stays active and the form remains available to retry.
5. You return to the same task or Settings page. Discover models or enter an exact model
   ID; whisper.cpp uses its server-loaded model. Home's quick controls apply immediately.
   In Settings, model and task edits apply with **Save settings**.

**Cancel** returns without changing the active connection. If you edited the form,
you can keep editing or discard those changes. Closing the window clears its transient key draft.

Dictation's first-run screen includes connection and model controls. Run **Test connection**
and **Finish setup** after reviewing the microphone and shortcut. Audio-file transcription
and Text to speech have independent setup; neither requires dictation setup to be complete.
See [Get started](../../getting-started/).

## Create a library entry without using it yet

Open **Settings → Connections → New connection**, enter the server details and supported
uses, then choose **Save connection**. This creates an inactive entry. Select it later
from any task it supports. The library remains the place to edit, duplicate, or delete servers.

While editing a library entry, **Save connection** and **Cancel** stay in the
bottom action bar, including in narrow windows. A disabled save explains what
the form still needs. Only available backend profiles appear as new choices;
an unavailable saved profile remains visible until you explicitly replace it.

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
changing the URL or backend profile clears them and the active model choices. **Save settings** does not edit the saved
connection. The home screen has the same active connection selectors. Its separate settings
links open feature settings; model and other quick controls save runtime settings.

Changing Settings sections opens the destination at the top without discarding
your edits. Rejected settings saves take you to the relevant section and, where
available, the invalid control. Correct the value and save again; other drafts
and the previously applied settings remain intact.

## Reuse a server

A Speaches connection can enable both Transcription and Text to speech, while
vLLM can enable Transcription and Cleanup. Generic offers all three uses;
llama.cpp currently offers Cleanup, and whisper.cpp offers Transcription.
These choices describe Freehand's implemented contracts, not detected server
capabilities. A particular vLLM deployment may expose only one operation.

To extend an existing connection, **Edit** it, enable another supported use, and
**Save connection**. Then select it on the other feature page and choose that
feature's model. Use separate connections when URLs, credentials, or backend
profiles differ. Transcription's custom health path and headers apply only to
transcription; they do not get sent through cleanup or playback adapters.

## Switch, edit, duplicate, or delete

- Choose an **Active connection** on a feature page to switch immediately.
  Unsaved Settings edits must be saved or discarded before the switch; **Keep editing** cancels it.
  A different connection restores its last selected model and remembered options
  for that feature. A connection without a remembered model starts with defaults;
  cleanup and text to speech turn off until configured and enabled again, and
  transcription needs its setup completed again. Running jobs keep their captured settings and keys.
- Choose **Edit connection** to open the selected entry in Connections. Save or
  discard feature edits first using **Save and continue**, **Discard and continue**, or **Keep editing** when prompted. The **Back** button at the top returns to the
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
Voice and provider-specific engine options follow the remembered model. Language,
cleanup instructions and style, and speaking speed stay with the current task. See [Model profiles](../model-profiles/)
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


### Discover speech voices

Speaches and Kokoro-FastAPI connections offer **Refresh voices** in Text to speech settings. Search the voice field or type an ID. Speaches may identify
voices for the selected model; server-wide lists are labelled accordingly.
Changing the connection or model hides results from another selection. Refreshing
voices neither changes the selected voice nor generates audio. Save feature
settings to remember the chosen voice for that connection and model.
