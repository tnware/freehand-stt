---
title: Saved connections
description: Create server connections in one place, then choose which connection each feature uses.
---

A saved connection contains a server's address, backend profile, and
authentication details. Voice, Audio file, Cleanup, and Text to speech each
choose their own connection and can share the same server.

Open **Settings → Connections** or **Manage connections…** in a task's connection
picker. Search by name, backend, or address, then select an entry to edit it.

## Built-in local connections

On supported Windows and macOS computers, each configured [local runtime](../local-runtime/) appears automatically
as a **Built-in** Connection. Install its runtime and model, start it, then select
it directly from a compatible task's connection picker. No separate connection
creation, URL, or API key is needed. llama.cpp with S1-mini appears for Cleanup;
NeMo and whisper.cpp appear for their qualified transcription tasks.

The runtime owns its name, endpoint, model, and supported uses. These are not
editable connection fields. Manage the runtime to change its downloaded/loaded
model; use the task's settings for language, realtime mode, and cleanup options.
Built-in Connections cannot be duplicated or deleted from the connection editor.
They remain visible but unavailable when stopped, without changing any task's
selection or falling back to a server. Previously saved local connection aliases
remain intact. Built-in rows do not count against the manual connection limit.

## Add a connection while setting up a task

1. Open the task's connection picker and choose **Add connection…**.
2. Enter a recognizable name, choose the server's **Backend**, and
   enter its base URL. The task you came from is already selected under supported uses.
3. Configure authentication and explicitly allow HTTP if your trusted server uses it.
   Expand **Available in … workflows** to declare additional operations the deployment
   exposes; this does not select it for those tasks.
4. Choose **Save and return** to save the connection and select it for the task.
5. Choose a listed model or enter its exact ID; whisper.cpp uses the model already
   loaded by its server. Review the model profile and options, then choose
   **Save and return** to resume the task.

If you cancel with unsaved edits, choose **Discard** to leave the active
connection unchanged or **Keep editing** to continue. A failed save leaves your
previous connection active; correct the error and try again.

For microphone and shortcut setup, see [Get started](../../getting-started/).

## Create a library entry without using it yet

Open **Settings → Connections → Add connection**, enter the server details and supported
uses, set **After saving** to **Save for later**, then choose **Save connection**. This creates an inactive entry. Select it later
from any task it supports. To configure it immediately instead, choose a workflow
under **After saving** and use **Save and return**.

Choose **Save connection** to save edits to an existing entry, or **Cancel** to
leave it. If saving is unavailable, check the message for a missing name, URL,
or supported use.

## What goes where?

| Settings → Connections                | Feature settings pages                    |
| ------------------------------------- | ----------------------------------------- |
| Connection name and supported uses    | Active connection selection               |
| Backend profile and base URL          | Model, model profile, and language        |
| Authentication and stored API key     | Cleanup preset and instructions           |
| Allow HTTP for this connection        | Voice, speed, and feature enable switches |
| Transcription health path and headers | Timeouts and provider-specific options    |

Editing a saved connection affects **every task using it** on its next request.
Renaming it or replacing its key keeps remembered model options. Changing its URL
or backend clears its model selections and remembered options; choose the models
again for the new server. Requests already running use their original settings.

## Reuse a server

A transcription server can be used for Voice, Audio file, or both. Some servers
also provide cleanup or speech generation. Check the [backend guides](../../backends/)
and enable only the operations your server actually provides.

To reuse an existing connection, edit it, enable another supported use, and
**Save connection**. Then select it on the other feature page and choose that
feature's model. Use separate connections when URLs, credentials, or backend
profiles differ. Custom transcription health paths and headers apply only to
Voice and Audio file, not cleanup or speech generation.

## Switch, edit, duplicate, or delete

- Choose an **Active connection** on a feature page to switch immediately.
  Unsaved Settings edits must be saved or discarded before the switch; **Keep editing** cancels it.
  Freehand restores that connection's last model and remembered options for the
  task. If none are saved, choose a model and configure it. Cleanup and text to
  speech need to be enabled again after selecting an unconfigured connection.
- **Manage connections…** opens the list; **Edit connection** opens the selected
  entry. **All connections** returns to the list.
- The editor's **Use for…** menu selects this server for a supported workflow and
  opens its model settings. A checkmark identifies tasks already using it.
- The **…** menu contains **Duplicate** and **Delete**. **Duplicate** creates an inactive copy with no remembered model preferences. Later edits and key replacements affect
  only that copy. Rename it through **Edit** if needed.
- **Delete** removes an inactive entry after confirmation. To delete an active
  entry, select another connection or **None selected** in every task using it first.
  The same rule applies before removing an enabled use from a connection.

Use a unique name for each connection. You can save up to 32 connections for
each use: Voice, Audio file, Cleanup, and Text to speech, including separate
entries for the same address with different credentials. A reusable connection
counts toward the limit for each of its enabled uses.
See [Model profiles](../../models/#remember-settings-for-each-model) for the
settings remembered when you switch models or connections.

## Test and protect credentials

The list shows which tasks use each connection. Select an entry and
expand **Connection check** for **Check connection** and its diagnostics. It checks
that saved entry, including its own credential, without selecting it or using
unsaved edits. It reads health or model-list metadata only, without sending audio
or prompts. Configure custom headers and health paths under **Transcription
connection options** when your gateway requires them.
Results distinguish metadata access and authentication. They do not assess every
task that can use this connection. Open a task's model picker and choose
**Refresh models**, or **Check server** for whisper.cpp, to check that selection.
See
[connection-check results](../troubleshooting/#understand-connection-check-results)
for interpreting advertised models, health-only checks, and stale results.
A successful check does not guarantee that a transcription or speech request
will work, or that your key has permission to run the selected model.

Keys stay in Windows Credential Manager or macOS Keychain. The app never displays a stored key;
leave its password field blank to keep it, enter a replacement, or explicitly
remove it. Leaving settings clears any unsaved key from the form.
Duplicating a connection lets the copy use the same stored key; replacing the
copy's key does not change the original. Deleting a connection does not remove
a key still used by another saved connection. See
[Privacy and safety](../privacy-and-safety/#credentials-and-transport) for storage
and transport details.

See [connect a server](../connect-a-server/), [backend guides](../../backends/),
and [settings recovery](../troubleshooting/#saved-settings-need-attention).

### Discover speech voices

Speaches, Kokoro-FastAPI, and vLLM-Omni connections offer **Refresh voices** in
Text to speech settings. Search the voice field or type an ID. Speaches may identify
voices for the selected model; server-wide lists are labelled accordingly.
Refreshing voices neither changes the selected voice nor generates audio. Save
your settings to remember the chosen voice for that connection and model.
