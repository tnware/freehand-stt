---
title: Saved connections
description: Create server connections in one place, then choose which connection each feature uses.
---

**Settings → Connections** opens the Connection Manager as a searchable list.
The same list is available through **Manage connections…** in each workflow's
connection picker. Search by name, backend, or address; select a row to edit it.
The list stays beside the editor in wider windows. **All connections** returns to
the full list at any size, and Save and Cancel stay visible below the form.

New installations start with an empty list; upgrading retains existing connections.
Reopening the manager preserves unfinished edits. Switching rows, returning to the
list, or closing with changes offers **Keep editing**, **Discard**, or **Save and
continue**. Closing clears the transient credential draft.

A connection represents one reusable server: its URL, backend profile, authentication,
and HTTP permission. Voice transcription, Audio-file transcription, Cleanup, and Text to speech select connections
independently and can share the same server.

All four workflow Settings pages put the active connection in a separate card
above model options, with its endpoint and **Edit connection** action. Voice’s
quick popover keeps its compact connection selector.

## Add a connection while setting up a task

1. Open the task's connection picker and choose **Add connection…**.
2. Enter a recognizable name, choose the server's **Backend**, and
   enter its base URL. The task you came from is already selected under supported uses.
3. Configure authentication and explicitly allow HTTP if your trusted server uses it.
   Expand **Available in … workflows** to declare additional operations the deployment
   exposes; this does not select it for those tasks.
4. Choose **Save and set up**. Saving and selecting happen together. On
   failure, the previous selection stays active and the form remains available to retry.
5. The workflow Settings page opens and loads metadata for its selected connection.
   Choose a discovered model or enter an exact model ID; whisper.cpp uses its server-loaded model. Home's quick controls apply immediately.
   In Settings, model and task edits apply with **Save settings**.

**Cancel** returns without changing the active connection. If you edited the form,
you can keep editing, discard those changes, or save before returning to the list. Closing the window clears its transient key draft.

Dictation's first-run screen includes connection and model controls. Run **Test connection**
and **Finish setup** after reviewing the microphone and shortcut. Audio-file transcription
and Text to speech have independent setup; neither requires dictation setup to be complete.
See [Get started](../../getting-started/).

## Create a library entry without using it yet

Open **Settings → Connections → Add connection**, enter the server details and supported
uses, set **After saving** to **Save for later**, then choose **Save connection**. This creates an inactive entry. Select it later
from any task it supports. To configure it immediately instead, choose a workflow
under **After saving** and use **Save and set up**. The library remains the place to edit, duplicate, or delete servers.

While editing an entry in the Connection Manager, use **Save connection** or **Cancel**. A disabled save explains what
the form still needs. Only available backend profiles appear as new choices;
an unavailable saved profile remains visible until you explicitly replace it.

## What goes where?

| Settings → Connections                | Feature settings pages                    |
| ------------------------------------- | ----------------------------------------- |
| Connection name and supported uses    | Active connection selection               |
| Compatibility profile and base URL    | Model, model profile, and language        |
| Authentication and stored API key     | Cleanup preset and instructions           |
| Allow insecure HTTP                   | Voice, speed, and feature enable switches |
| Transcription health path and headers | Timeouts and provider-specific options    |

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

A transcription server can be enabled for Voice, Audio file, or both. Each task keeps its own model and options. Speaches also supports Text to speech; vLLM supports Cleanup and qualified realtime Voice profiles. vLLM-Omni offers Text to speech, including the Qwen3-TTS CustomVoice profile. Generic offers completed transcription, cleanup, and speech generation. NeMo-Speech.cpp offers completed transcription and qualified [realtime Voice mode](../live-transcription/) with the explicit Nemotron profile. llama.cpp currently offers Cleanup, and whisper.cpp offers completed transcription.
These choices describe Freehand's implemented contracts, not detected server
capabilities. A particular vLLM deployment may expose only one operation.

To extend an existing connection, **Edit** it, enable another supported use, and
**Save connection**. Then select it on the other feature page and choose that
feature's model. Use separate connections when URLs, credentials, or backend
profiles differ. Custom transcription health paths and headers apply to Voice and Audio file; they do not get sent through cleanup or playback adapters.

## Switch, edit, duplicate, or delete

- Choose an **Active connection** on a feature page to switch immediately.
  Unsaved Settings edits must be saved or discarded before the switch; **Keep editing** cancels it.
  A different connection restores its last selected model and remembered options
  for that feature. A connection without a remembered model starts with defaults;
  cleanup and text to speech turn off until configured and enabled again, and
  transcription needs its setup completed again. Running jobs keep their captured settings and keys.
- **Manage connections…** opens the list; **Edit connection** opens the selected
  entry. Save or discard feature edits first when prompted. **All connections**
  and **Cancel** return to the list. Saving an existing entry keeps its editor open.
- The editor's **Use for…** menu selects this server for a supported workflow and
  opens that workflow's model settings. A checkmark identifies workflows already
  using it; choosing one opens those settings without changing the selection.
- The **…** menu contains **Duplicate** and **Delete**. **Duplicate** creates an inactive copy with no remembered model preferences. Later edits and key replacements affect
  only that copy. Rename it through **Edit** if needed.
- **Delete** removes an inactive entry after confirmation. To delete an active
  entry, select another connection or **None** in every feature using it first.
  The same rule applies before removing an enabled use from a connection.
  Deleting the last inactive entry is allowed.

New names must be unique across the connection library, with up to 32 available
connections per feature. Upgrades preserve existing names and entries; duplicate
URLs are not automatically merged because their authentication or uses may differ.
Voice and provider-specific engine options follow the remembered model. Language,
cleanup instructions and style, and speaking speed stay with the current task. See [Model profiles](../../models/)
for choosing, saving, and forgetting those preferences.

## Test and protect credentials

The list shows which workflows currently use each connection. Select an entry and
expand **Connection check** for **Check connection** and its diagnostics. It checks
that saved entry, including its own credential, without selecting it or using
unsaved edits. It reads health or model-list metadata only. Results stay with their
entries while browsing and clear when confirmed settings change or the manager
closes. Advanced transcription headers and health paths have a separate collapsed
section in the editor.
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
