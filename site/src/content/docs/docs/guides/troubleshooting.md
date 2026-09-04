---
title: Troubleshooting
description: Diagnose setup, connection, recording, delivery, cleanup, and playback problems.
---

Start with the status shown in Freehand. Connection checks are metadata-only:
they can confirm that a server, credentials, and model listing are reachable,
but they do not submit audio or prove that inference will succeed.

## Setup does not complete

- Confirm the speech-to-text base URL includes the server's API prefix,
  normally `/v1`.
- Enter the exact transcription model ID expected by the server. If model
  discovery is available, select an ID from the returned list.
- Check whether the endpoint requires an API key.
- Enable insecure HTTP only when you intentionally use a trusted plaintext
  local or LAN endpoint.
- If an explicitly selected microphone is missing, choose another device or
  return to **System default**.
- If Freehand reports an invalid settings file, retry loading it or deliberately
  reset it from the recovery screen. It will not silently replace invalid saved
  settings.

## Connection and request failures

| Result | What it usually means | What to check |
| --- | --- | --- |
| Invalid settings | Freehand rejected the configuration before networking | API prefix, required fields, and plaintext HTTP policy |
| Connection failed | No usable HTTP response arrived | Server process, hostname, port, firewall, TLS, and reverse proxy |
| Unauthorized or forbidden | The server or gateway rejected authentication | Authentication mode, current API key, and gateway policy |
| Model not found | Metadata worked but the configured ID was absent | Select a discovered model or enter the exact routed ID |
| Request too large | The server or proxy rejected the upload | Proxy body limit, server upload limit, and selected file size |
| Timed out | The capability's configured request budget expired | Request budget, server load, model warmup, and network path |
| Route unsupported | The server is reachable but lacks that capability | Confirm the specific STT, chat, or TTS route |

Freehand does not automatically retry an ordinary inference failure because a
retry can duplicate work or billing. Correct the configuration or server
condition, then retry deliberately.

## Recording does not start

- Open **Settings → Audio** and confirm the intended microphone is available.
- Re-select the device after unplugging, disabling, or replacing it.
- Check the shortcut shown under **Settings → Shortcuts**. A conflicting global
  shortcut is rejected when settings are saved, leaving the previous working
  shortcut in place.
- Finish or cancel active audio-file transcription or speech playback before
  starting another recording.

If startup is slower after the device has been idle, wait for Freehand's ready
state before speaking. The ready state means native capture has started, not
merely that the shortcut was received.

## A transcript was not inserted

Freehand inserts voice text only when the application and focused control that
were active at recording start are still the destination at completion. If the
target changed, Freehand keeps the result available for explicit copying
instead of typing into another window.

Stored-audio results always require an explicit **Copy** action. Also check the
configured delivery mode: manual copy never inserts automatically.

## Cleanup was skipped or failed

Transcript cleanup has its own endpoint, model, credentials, and timeout. Test
that connection independently and confirm the selected processing profile
matches the server and model.

A cleanup failure never discards a successful speech-to-text result. Freehand
keeps and delivers the raw transcript and records the cleanup outcome in
history when history is enabled.

## Audio-file transcription stops or is incomplete

- A `413` response means the server or a reverse proxy rejected the upload
  size. Increase that server-side limit or choose a smaller file.
- Increase the visible stored-file request budget when a large file or cold
  model legitimately needs more time.
- When compatible streamed output fails after partial text arrived, Freehand
  preserves the available partial result rather than presenting it as a
  completed transcript.

Automatic long-file segmentation is not currently provided.

## Speech playback produces no sound

- Confirm text-to-speech is enabled and its endpoint implements
  `POST /v1/audio/speech`.
- Check the configured model and voice ID expected by that endpoint.
- Verify the Windows default output device and system volume.
- Generate the speech again after changing endpoint or output settings.

## Report a problem

Search the [GitHub issues](https://github.com/tnware/freehand-stt/issues) before
opening a report. Include the Freehand version, the visible failure category,
the operation you attempted, and reproducible steps.

Do not include API keys, private endpoint URLs, transcripts, audio, full file
paths, machine names, or unredacted diagnostic output in a public issue.
