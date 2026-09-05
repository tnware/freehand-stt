---
title: Troubleshooting
description: Diagnose setup, connection, recording, delivery, cleanup, and playback problems.
---

Start with the status shown in Freehand. Connection checks are metadata-only:
they can confirm that a server, credentials, and model listing are reachable,
but they do not submit audio or prove that inference will succeed.

## Setup does not complete

**Open the unfinished readiness item, correct its settings, and choose Save
changes.** Return to the readiness screen, select **Test connection**, then
choose **Finish setup** once all requirements are ready.

:::note[Initial setup requires a microphone]
This alpha requires a usable microphone even if you plan to transcribe stored
audio files. Connect or enable a microphone, then select it under **Settings → Audio**.
:::

- Confirm the speech-to-text base URL includes the server's API prefix,
  normally `/v1`.
- Enter the exact transcription model ID expected by the server. If model
  discovery is available, select an ID from the returned list.
- If the endpoint requires a key, select **Authentication → API key** under
  **Settings → Transcription**, then enter it. Leave authentication at **None**
  only for an endpoint that does not require a key.
- Enable insecure HTTP only when you intentionally use a trusted plaintext
  local or LAN endpoint.
- If an explicitly selected microphone is missing, choose another device or
  return to **System default**.

<details>
<summary>Freehand reports an invalid settings file</summary>

Retry loading it or deliberately reset it from the recovery screen. Freehand
does not silently replace invalid saved settings.

</details>

## Connection and request failures

**Match the visible failure to the table below, correct the cause, and retry
deliberately.** Test speech, cleanup, and playback connections independently.

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

**Check the microphone under Settings → Audio, then try your configured shortcut.**

- Open **Settings → Audio** and confirm the intended microphone is available.
- Re-select the device after unplugging, disabling, or replacing it.
- Check the shortcut shown under **Settings → Shortcuts**. A conflicting global
  shortcut is rejected when settings are saved, leaving the previous working
  shortcut in place.
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

Freehand inserts voice text only when the application and focused control that
were active at recording start are still the destination at completion. If the
target changed, Freehand keeps the result available for explicit copying
instead of typing into another window.

Stored-audio results always require an explicit **Copy** action. Also check the
configured delivery mode: manual copy never inserts automatically.

## Cleanup was skipped or failed

**Use the raw transcript, then check Settings → Post-processing.** Confirm
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
uses the raw transcript and shows an output-limit notice. The alpha does not
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

**Check the Windows output device and volume, then review Settings → Speech playback.**

- Confirm text-to-speech is enabled and its endpoint implements
  `POST /v1/audio/speech`.
- Check the configured model and voice ID expected by that endpoint.
- Verify the Windows default output device and system volume.
- Generate the speech again after changing endpoint or output settings.

## Report a problem

Search the [GitHub issues](https://github.com/tnware/freehand-stt/issues) before
opening a report. Include the Freehand version, the visible failure category,
the operation you attempted, and reproducible steps.

:::caution[Keep private data out of reports]
Do not include API keys, private endpoint URLs, transcripts, audio, full file
paths, machine names, or unredacted diagnostic output in a public issue.
:::
