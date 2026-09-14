---
title: Troubleshooting
description: Diagnose setup, connection, recording, delivery, cleanup, and playback problems.
---

Start with the status shown in Freehand. Connection checks read server metadata
and check your settings without submitting audio or text to a model. A successful
check does not guarantee that a transcription, cleanup, or speech request will work.

## macOS installation or permissions

Use the matching-architecture ZIP and same-release checksums in the
[macOS setup guide](../macos-setup/). Launch the installed app bundle from a
stable location. For microphone denial, review Privacy & Security → Microphone;
for text insertion, review Accessibility; for shortcut capture or hold-to-talk,
review Accessibility and Input Monitoring. Quit and reopen after permission
changes when requested. Development and installed apps have separate permissions
even though both display **Freehand**.
Release all keys and use **Retry hold-to-talk** after recovering keyboard access.
Keychain denial requires the OS approval/recovery flow, not plaintext storage.

## Setup does not complete

**Check the requirements for the task you selected.** The **Voice** readiness
screen and **Finish setup** are for dictation. You can switch to **Audio file**
or **Text to speech** without finishing dictation setup.

| Task | What must be ready |
| --- | --- |
| Voice | STT connection and model, authentication if required, microphone, and the initial metadata connection check; a recording shortcut is optional |
| Audio file | STT connection and model, authentication if required, and a supported file; no microphone or recording shortcut |
| Text to speech | Its own connection, model and voice ID, authentication if required, and **Enable text to speech**; no STT connection required |

For **Voice**, open the unfinished readiness item, correct its settings, and
choose **Save** (or **Save and return** when opened from a task).
Return to **Voice**, select
**Check connection** or **Check again**, then choose **Finish setup** once all dictation requirements
are ready. Home quick controls apply immediately.

For transcription setup:

- Confirm the speech-to-text base URL includes the server's API prefix,
  normally `/v1`. For whisper.cpp, use the server root without `/v1`.
- Enter the exact transcription model ID expected by the server. If model
  discovery is available, select an ID from the returned list. whisper.cpp
  uses its server-loaded model instead.
- If the endpoint requires a key, open **Settings → Connections**, edit the
  selected connection, then choose **Authentication → API key** and enter it. Leave authentication at **None**
  only for an endpoint that does not require a key.
- Enable insecure HTTP only when you intentionally use a trusted plaintext
  local or LAN endpoint.
- For dictation only, if an explicitly selected microphone is missing, choose
  another device or return to **System default**.

## Understand connection-check results

Check the connection for the task you are using: Voice, Audio file, Cleanup, or
Text to speech. Success for one does not check the others. **Ready to generate** on the composer describes
local configuration, not server reachability. **Not checked** and **Settings
changed** are not successful connection checks; refresh metadata explicitly.

On a feature page, **Refresh models** (or **Check server** for whisper.cpp) runs
one metadata request and shows a **Connection check** panel. **Check again**
repeats it explicitly. The panel distinguishes four things:

| Check | What it establishes |
| --- | --- |
| Connection | Whether the configured metadata route returned a usable response. Failures include the next setting or server condition to check. |
| Authentication | Whether that metadata request was accepted. A public health endpoint does not prove that the key works on inference routes. |
| Selected model | Whether the requested ID appears in the model list. Health-only checks cannot identify models; whisper.cpp reports a server-loaded model. |
| Configuration | Whether the selected model profile and options are supported by Freehand. Speech needs a voice ID; Generic S1-mini connections need reasoning disabled on the server. |

An unlisted model is a reason to review the ID, not proof that it cannot run: some
servers accept aliases they do not advertise. A listed model may also be unsuitable
for the selected feature. Metadata cannot verify transcription quality, supported
languages or voices, cleanup behavior, or inference permissions.

When you change the connection, model, or relevant model options, previous results
are marked as applying to older settings or cleared. Run another check to assess
the current settings.

The **Check connection** action on the Connections page checks only that saved
server and its authentication, without selecting it. Open a feature page to check
its model and options. Actual audio or text requests happen only through your
explicit transcription, cleanup, or speech workflow.

## Saved settings need attention

### An edited value was rejected

If saving fails, correct the marked setting using its error message, then save
again. Freehand keeps your other draft edits and leaves the applied settings
unchanged. For example, **Settings → Audio → Maximum duration** shows the limit
for the current recording mode: 1–262 seconds without splitting, or 1–3,600 seconds
with splitting.

### Saved configuration cannot be loaded

Freehand pauses new transcription and settings changes when saved configuration
cannot be loaded or a save's outcome cannot be confirmed. It does not silently
replace your settings with defaults.

- **In use or inaccessible:** close other Freehand instances, check file access
  and available disk space, then choose **Retry loading**.
- **Newer database:** update Freehand to a compatible version. Older builds do
  not downgrade a newer settings database.
- **Reset:** choose **Reset to defaults** only when you want a fresh setup.
  Freehand archives an existing database and its sidecars in a
  `freehand-recovery-*` folder, then starts with safe defaults. Native credentials
  are not automatically purged. Reconfigure your connections and enter keys again.

Earlier alpha settings are not imported. If your connections are missing after
upgrading, follow the [first-launch reset notice](../../getting-started/#first-launch).
Do not rename an old alpha database to use it with this version.

To restore a backup made by this new settings lineage:

1. **Quit Freehand from the tray or macOS menu bar.** Closing a window alone leaves it running.
2. Open `%LOCALAPPDATA%\Freehand` in File Explorer (Windows), or
   `~/Library/Application Support/Freehand` using Finder → Go to Folder (macOS). Copy `freehand.db` and any
   matching `freehand.db-journal`, `freehand.db-wal`, or `freehand.db-shm` files
   together into a separate recovery folder before removing them from this folder.
3. Copy a known-good `.db` file from `freehand-backups` into the Freehand folder and name
   the copy `freehand.db`. Keep the original backup. Do not mix old sidecars with
   the restored database.
4. Reopen Freehand and review settings and authentication. Backups contain
   configuration, not keys; replaced keys may need entering again.

The app retains the newest three backups made before schema upgrades. Explicit
reset archives are retained until you remove them. Keep any recovery files private:
they can contain endpoint addresses, headers, and custom instructions.

Downgrading does not convert settings to an older format. An older alpha may
read stale settings or refuse to open the newer database. Review release notes
before returning to an older version.

## Connection and request failures

Match the visible failure to the table below, correct the cause, and retry.
Check transcription, cleanup, and text-to-speech connections independently.

| Result | What it usually means | What to check |
| --- | --- | --- |
| Invalid settings | Freehand rejected the configuration before networking | API prefix, required fields, and plaintext HTTP policy |
| Connection failed | No usable HTTP response arrived | Server process, hostname, port, firewall, TLS, and reverse proxy |
| Unauthorized or forbidden | The server or gateway rejected authentication | Authentication mode, current API key, and gateway policy |
| Model not advertised | The model list does not include the configured ID | Check the ID or alias with your server; an unlisted alias may still work |
| Request too large | The server or proxy rejected the upload | Proxy body limit, server upload limit, and selected file size |
| Timed out | The capability's configured request budget expired | Request budget, server load, model warmup, and network path |
| Route unsupported | The server is reachable but lacks that capability | Confirm the specific STT, chat, or TTS route |

Freehand does not automatically retry an ordinary inference failure because a
retry can duplicate work or billing. Correct the configuration or server
condition, then retry deliberately.

## Recording does not start

**Check the microphone under Settings → Audio, then try your configured shortcut.**

- Open **Settings → Audio** and confirm the intended microphone is available.
- Re-select the device after unplugging, disabling, or replacing it.
- Check the shortcut shown under **Settings → Shortcuts**. A conflicting global
  shortcut is rejected when settings are saved, leaving the previous working
  shortcut in place.
- If the saved shortcut was already unavailable at startup, record a replacement
  or choose **Clear** and save it unassigned. There is no automatic fallback key,
  and an unavailable startup binding does not block replacement capture. Use the
  on-screen recording controls while unassigned; copy the result if the intended
  destination was not focused when recording began.
- Finish or cancel active audio-file transcription before starting a recording.
- Recording normally stops speech playback automatically before opening the
  microphone. If playback cannot be stopped, capture does not start. Stop
  playback explicitly and retry; if that fails, quit Freehand through the tray
  and relaunch.

<details>
<summary>Recording starts slowly after the microphone has been idle</summary>

Wait for Freehand to show that recording has started before speaking. Receiving
the shortcut does not itself mean that the microphone is ready.

</details>

## A transcript was not inserted

**Copy the available transcript into the intended text field.** For your next
voice dictation, keep the original field focused until processing finishes.

Windows checks the original app, window, and focused control. macOS checks the
app and focused window, not the editor field: changing fields within that window
delivers to the currently focused field. macOS Secure Input blocks delivery, but not all
custom secure fields enable it. A stale target or unavailable permission requires
Copy; a generic failure does not prove focus changed or that no text was typed.
Check for partial delivery before pasting to avoid duplicates.

Stored-audio results always require an explicit **Copy** action. Also check the
configured delivery mode: manual copy never inserts automatically.

## Cleanup was skipped or failed

**Use the raw transcript, then check Settings → Cleanup.** Confirm
cleanup is enabled and test its connection before another attempt.

Transcript cleanup has its own endpoint, model, credentials, and timeout. Test
that connection independently and confirm the selected processing profile
matches the server and model.

A cleanup failure falls back to the successful raw transcript. Delivery still
follows your copy setting, destination checks, and cancellation. The cleanup
outcome is retained in history when history is enabled and within its limits.

<details>
<summary>S1-mini returns empty text, or cleaned output is incomplete</summary>

Confirm that the **S1-mini by Superwhisper** profile is selected and its server
has reasoning disabled. If the server reports an output length limit, Freehand
uses the raw transcript and shows an output-limit notice. Freehand does not
automatically chunk long cleanup inputs and cannot detect omissions the server
does not report. Follow the
[S1-mini setup and input limits](../post-processing/#s1-mini-by-superwhisper-with-llamacpp)
and review output against the raw version when history is enabled.

</details>

## Audio-file transcription stops or is incomplete

**Copy any available partial result before retrying.** Check the failure
category and adjust the file, server limit, or request budget as appropriate.

- A `413` response means the server or a reverse proxy rejected the upload
  size. Increase that server-side limit or choose a smaller file.
- Increase the visible stored-file request budget when a large file or cold
  model legitimately needs more time.
- When compatible streamed output fails after partial text arrived, Freehand
  preserves the available partial result rather than presenting it as a
  completed transcript.

Automatic long-file segmentation is not currently provided.

## Speech playback produces no sound

**Check the system output device and volume, then review Settings → Text to speech.**

- Turn on **Enable text to speech**, save your settings, and confirm its
  endpoint implements `POST /v1/audio/speech`.
- Check the configured model and voice ID expected by that endpoint.
- Verify the system default output device and system volume.
- Choose **Speak** for text you enter, or **Listen** on a retained completed
  transcript. Playback never starts automatically after transcription.
- Generate the speech again after changing endpoint or output settings.

## Report a problem

Search the [GitHub issues](https://github.com/tnware/freehand-stt/issues) before
opening a report. Include the Freehand version, the visible failure category,
the operation you attempted, and reproducible steps.

:::caution[Keep private data out of reports]
Do not include API keys, private endpoint URLs, transcripts, audio, full file
paths, machine names, or unredacted diagnostic output in a public issue.
:::
