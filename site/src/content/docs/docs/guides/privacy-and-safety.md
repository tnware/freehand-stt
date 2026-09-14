---
title: Privacy and safety
description: What Freehand sends, retains, stores, and inserts on Windows and macOS.
---

Freehand is a lightweight Windows and macOS client for speech-to-text and text-to-speech
services you choose. Audio and text are sent for the workflows you use:
transcription, optional transcript cleanup, or on-demand speech
generation from your text or retained transcripts.
The destination may be localhost, a private server, or a hosted provider, so
that server's own privacy and retention policy still applies.

## What goes where?

| Data | Destination | What Freehand keeps |
| --- | --- | --- |
| Microphone or selected-file audio | Your configured speech-to-text endpoint | Audio for the active request; released afterward. Existing source files are unchanged. |
| Transcript sent for cleanup | Your separate cleanup endpoint, when enabled | Keeping both versions after successful cleanup requires enabled session history. Raw failure fallback does not require history. |
| API keys | The configured capability endpoint when authentication is enabled | Saved keys in Windows Credential Manager or macOS Keychain. |
| Transcript history | Memory on your computer | Off by default; at most 20 entries and 2 MiB, cleared on exit. |
| Speech playback text and audio | Your playback endpoint receives text and returns audio | Generated audio in memory until cleared, replaced, a recording begins, or Freehand exits; saving a file is explicit. |
| Update checks | GitHub release service | Update metadata and any downloaded update; no recordings or transcripts are sent. |
| Managed runtime installation (Windows/macOS, optional) | Official pinned NeMo, llama.cpp, or whisper.cpp releases and the selected model’s download host | Runtime binaries and selected model weights in Freehand's application-data directory, until removed. |
| Managed process output | Private memory on this computer; shown only after sensitive-output consent | A bounded rolling tail, cleared explicitly or on the next start attempt, runtime removal, or exit. Closing the viewer revokes access but does not erase the private tail. |

## Managed local recognition

The optional managed runtime recognizes speech on this computer. Its listener is
restricted to this computer, not your LAN. Installing binaries and downloading
models requires internet access; browsing the catalog does not run inference.
Other software running on the same computer can potentially access a loopback
service, so a local listener is not a sandbox against other local programs.

Each task selects its own Connection. Local Voice or audio-file recognition does
not make cleanup local: if cleanup is enabled, its selected connection receives
the recognized text. Choose [managed llama.cpp with S1-mini](../local-runtime/#local-cleanup-with-s1-mini)
or another local cleanup service to keep that stage on this computer. Text to
speech needs a manually configured endpoint; Freehand has no managed TTS runtime.
Turn cleanup and speech off or configure them locally if you do not want their
text sent to a remote service.

Freehand keeps your manual connections and their API keys when you enable local
recognition. It does not send those keys to the managed runtime. A runtime
failure does not silently switch captured audio to a remote server: retry local
setup or explicitly return to your manual connection for new work.

Runtime/model removal does not remove source recordings, manual connections,
or another application's model cache. Model weights are retained installation
data, not retained microphone audio or transcript history.

Starting a selected GPU runtime includes warm-up, also when you enable its
start-at-launch preference. llama.cpp and NeMo use built-in warm-up. CUDA
whisper.cpp receives one second of synthetic silence on this computer, using
only the selected loaded model; Freehand discards its response. This does not
capture your microphone, enter transcript history, run other catalog models, or
send a request to a remote service. Connection checks remain metadata-only.

## Audio

Microphone and selected-file audio is kept only for the active transcription
request. Freehand does not retain audio in history. It releases active audio and
deletes temporary audio after completion, failure, or cancellation. Selecting an
existing audio file does not delete or modify the original file.

Optional text-to-speech is separate: generated playback audio remains in
memory until cleared or replaced, a recording begins, or Freehand exits. You
can explicitly save generated audio. Your chosen inference server may retain
audio or text according to its own policy.

## Transcripts and history

The latest dictation result stays in memory for inspection and explicit copying,
until the next recording, Clear, or exit. File transcription keeps its current
result until you clear or replace the selected file, start another transcription,
or exit. These single current results are available with history off. Failed
insertion recovery also works without history.

Unsent text in the Text to speech composer stays in the current window session
across task and settings navigation. It is never written to browser storage or
the settings database; reloading the window or quitting clears it.

Transcript history is disabled by default. When enabled, it is memory-only,
bounded to 20 entries and 2 MiB, and cleared when Freehand exits. It stores
raw and cleaned transcript text and limited non-secret run details—not audio,
credentials, request headers, full file paths, or destination-window identity.

The **Transcription details** for a history entry are removed with that entry,
including when you clear or disable history or the entry reaches its retention
limit. An open details window does not preserve a separate copy.

Stored-audio results require an explicit Copy action. Voice dictation can use
focus-safe direct insertion or manual copy, according to your settings.

## Safe text insertion

Start dictation with the intended app and text field focused, and keep them
focused until delivery finishes. Freehand will not bring an app to the foreground
to insert text.

- On Windows, the original app, window, and focused control must still match.
- On macOS, the same app and window must remain focused. If you move to another
  field in that window, Freehand delivers to the currently focused field.

macOS Secure Input blocks delivery while active. Freehand does not identify
every password field: custom secure fields that do not enable Secure Input may
not be detected. Avoid dictating sensitive text into an uncertain destination.

If the destination changes or Freehand cannot safely deliver, the transcript
stays available for explicit **Copy**. A failed delivery may have inserted some
text already; check before pasting to avoid duplicates.

Clipboard-paste insertion is not enabled. Freehand does not silently replace
the clipboard as part of automatic delivery.

## Saved settings and backups

On Windows, non-secret settings are stored in
`%LOCALAPPDATA%\Freehand\freehand.db`. They include server addresses, model
choices, vocabulary, custom instructions, and request headers. Database access is restricted
to your Windows user and SYSTEM; the database is not encrypted. Treat it and its
backups as private configuration. API keys remain in Windows Credential Manager or macOS Keychain,
and transcript history remains memory-only. Window size and position are kept
separately in `%APPDATA%\Freehand\window-state.json`.

On macOS, settings and backups are under `~/Library/Application Support/Freehand`,
with `freehand.db` and the separate `window-state.json`. Files use per-user
permissions, not Windows ACLs; the database is not encrypted. Keychain denial
does not trigger plaintext credential storage.

Earlier alpha settings and native credentials are left untouched and are not
imported or reused on either platform. See the [first-launch reset notice](../../getting-started/#first-launch).
Future schema upgrades retain up to three database backups; explicit recovery
resets retain an archive of the replaced current database.
See [settings recovery](../troubleshooting/#saved-settings-need-attention) before
restoring or removing these files.

## Credentials and transport

API keys are stored in Windows Credential Manager or macOS Keychain. They are not written to the
settings database or displayed again after saving. Inactive connections keep
their saved keys. Deleting a connection removes its key only when no saved
connection still uses it. See [saved connections](../saved-connections/).

HTTPS is required by default. You can explicitly allow HTTP for a trusted local
or LAN endpoint, but doing so sends audio, transcript text, and credentials
without transport encryption. Do not enable it across an untrusted network.

Inference and connection-check requests never follow HTTP redirects, even to
another path on the same server. Configure the final base URL instead of a
redirecting alias; Freehand will not forward your key, audio, or text to the
redirect destination.

Freehand filters literal copies of your API key from server response details
and rejects transcript text containing the key. This does not make an untrusted
server safe: the server has already received the key and could misuse or
transform it. Only connect to services you trust.

## Connection checks

Finishing the initial **Voice** dictation setup requires an explicit connection
test. That setup does not gate the **Audio file** or **Text to speech** tabs.
After dictation setup, Freehand checks the saved STT connection automatically on
launch and after relevant connection settings change. Automatic and manual
checks request a health route or
`GET /v1/models`. They do not submit audio, prompts, or synthetic inference
jobs, and they do not cycle through discovered models. Model selection itself
does not invoke the model.

## Update checks

Automatic update checks are on by default. Freehand checks GitHub release
metadata shortly after startup and once per day. If an update is available,
the updater can download and checksum-verify the platform asset, then waits for
you to restart. macOS uses an app-bundle ZIP; checksum verification does not
establish Developer ID trust or notarization. See [macOS updates and manual fallback](../macos-setup/#update). You can
disable automatic checks under **Settings → General**. These checks do not
send recordings or transcripts to GitHub.

## Diagnostics

Operational logs include status, timing, and failure categories. They
exclude audio, transcript text, credentials, private headers, full paths,
model IDs, URL paths and queries, and destination-window identity.

Managed runtime **View output** is separate from those logs. Freehand privately
captures recent process output in memory even with the viewer closed. The tail
is limited to 256 KiB and 1,024 chunks, with older text discarded as it fills.
For llama.cpp this includes normal informational, warning, and error output,
not debug logging. It can still contain transcripts, prompts, file paths, or
other sensitive upstream text; Freehand does not promise complete redaction.
Transcript history being off does not prevent such text appearing in process
output. This private capture is not saved as a log file.

The separate viewer requires **Show output** consent each time it opens or
switches runtime. Avoid displaying it during screen sharing. It is read-only,
with search, colors, and progress updates but no command input or file logging.
**Copy selection** puts only the text you select on the clipboard when you ask;
other applications may read it, and it can remain after closing the viewer.
Closing/switching clears displayed text
and revokes access without stopping the runtime or erasing its private tail.
**Clear**, the next start attempt, runtime removal, and Quit discard the tail.
Pausing scrolling does not pause collection. Freehand does not forward this
output to application logs, events, or crash reports. See
[startup and process output](../local-runtime/#startup-and-process-output) for
viewer controls.
