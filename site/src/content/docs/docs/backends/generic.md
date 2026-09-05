---
title: Generic OpenAI-compatible
description: Configure the default Freehand contract for a compatible server.
---

**Status:** available for transcription, post-processing, and speech playback.
**Evidence:** automated client contract fixtures. No universal server or model
compatibility is implied.

## Configure a connection

1. Choose **Generic OpenAI-compatible** in the appropriate Settings section.
2. Enter the API base URL, normally ending in `/v1`, and the exact model ID.
3. Choose authentication and save any required key through Freehand's credential field.
4. Use **Test** for model-list or health metadata, then save settings.
5. Explicitly try one operation with the model you chose and review the result.

See [Connect a speech server](../../guides/connect-a-server/) for trusted HTTP,
independent endpoints, and credential configuration. Existing settings default
to Generic when no compatibility profile was previously stored.

## Implemented capabilities

| Operation | Freehand's contract |
| --- | --- |
| Microphone and audio-file transcription | Multipart `file`, `model`, `response_format=json`, and optional `language`, `prompt`, and `temperature`; completed JSON with string `text`. |
| Optional file streaming | `stream=true`; typed transcript delta/done events or legacy per-segment text events. A server may return completed JSON instead. |
| Transcript cleanup | Non-streaming text chat completions with system/user string messages and temperature zero. |
| Speech playback | `model`, `input`, string `voice`, `speed`, and `response_format=wav`; fully buffered PCM16 WAV audio. |

The base URL is a prefix: Freehand appends `audio/transcriptions`,
`chat/completions`, or `audio/speech`. Entering a complete operation URL would
append the path again. Generic describes these particular contracts, rather
than every feature of the OpenAI API or every implementation using that name.

## Limits and model qualifications

- The server must accept the required fields; Freehand does not infer request
  variants from model names or automatically send provider-only parameters.
- File streaming is progressive output for an uploaded file. It does not enable
  live microphone or Realtime API sessions.
- Typed streams require a final transcript. Legacy segment streams finish at EOF.
  Incompatible streams are not automatically resubmitted.
- Upload limits may be lower than Freehand's 2 GiB file ceiling. The client does
  not split oversized stored files automatically.
- Speech must be mono/stereo PCM16 WAV at 8–192 kHz. A WAV MIME type alone does
  not establish the sample encoding. Freehand buffers up to 32 MiB before playback.
- Failed or reported length-limited cleanup falls back to raw text under the
  normal delivery/cancellation rules.

The dedicated **OpenAI hosted** profile is planned. Hosted and self-hosted
endpoints can use Generic when the selected model accepts this exact contract.
See the [full protocol reference](../../reference/protocol/) for bounds and
failure handling.

## Optional transcription controls

In **Settings → Transcription → Transcription controls**, context is sent as
`prompt`; temperature is sent only when its override is enabled. These are
optional common request fields, not a guarantee that every compatible model
honors them. Leave them unset to keep the original request shape. Generic does
not send `hotwords`; that field requires the Speaches profile.

See [Transcription controls](../../guides/connect-a-server/#transcription-controls)
for limits, persistence, and the difference between recognition hints and cleanup.
