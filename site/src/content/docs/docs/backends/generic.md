---
title: Generic OpenAI-compatible
description: Connect an OpenAI-compatible server for transcription, cleanup, or text to speech.
---

Use **Generic OpenAI-compatible** when your server supports the request and
response formats below and has no matching dedicated backend profile.

## Obtain an endpoint

Generic is a protocol baseline, not a server program to install. Use the
URL, model ID, and authentication supplied by your server operator or hosted
provider. If you want to run a server yourself, start with the
[backend launch guides](../#run-a-backend) and choose its qualified profile.
Check that the selected model supports the operation you want to use.

## Configure a connection

1. Open **Connections → Add connection**, choose its purpose and **Generic OpenAI-compatible** profile.
2. Name it and enter the API base URL, normally ending in `/v1`.
3. Configure authentication. For an HTTP URL, enable **Allow HTTP for this connection** only if you trust the network. Choose **Save connection**.
4. Open the workflow's **Settings** cog and choose **Transcription** for Voice or Audio file,
   **Cleanup** for either transcription workflow, or **Speech** for Text to speech.
   Select the connection, choose **Refresh models** or enter the exact model ID, then save. Model discovery reads metadata only.
5. Explicitly try one operation with the model you chose and review the result.

See [Connect a speech server](../../guides/connect-a-server/) for HTTP permission,
separate servers, and credential configuration.

## Implemented capabilities

| Operation                               | Request and response                                                                                                                        |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Microphone and audio-file transcription | Multipart `file`, `model`, `response_format=json`, and optional `language`, `prompt`, and `temperature`; completed JSON with string `text`. |
| Optional file streaming                 | `stream=true`; typed transcript delta/done events or legacy per-segment text events. A server may return completed JSON instead.            |
| Transcript cleanup                      | Non-streaming text chat completions with system/user string messages, temperature zero, and optional `max_tokens`.                          |
| Speech playback                         | `model`, `input`, string `voice`, `speed`, and `response_format=wav`; fully buffered PCM16 WAV audio.                                       |

The base URL is a prefix: Freehand appends `audio/transcriptions`,
`chat/completions`, or `audio/speech`. Entering a complete operation URL would
append the path again. An OpenAI-compatible label alone does not guarantee that
the server accepts these fields or returns these formats.

## Available options

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

In **Voice transcription → Transcription**, use **Context hint** and the temperature
override under **Request settings**. For **Audio file → Transcription**, open
**Transcription controls**. Context is sent as
`prompt`; temperature is sent only when its override is enabled. These are
optional common request fields, not a guarantee that every compatible model
honors them. Leave them unset to use server defaults. Generic does not send
`hotwords`; when you enable [shared vocabulary](../../guides/vocabulary/),
Freehand appends those terms to `prompt` instead.

See [Transcription controls](../../guides/connect-a-server/#transcription-controls)
for limits, persistence, and the difference between recognition hints and cleanup.

## Cleanup generation controls

Generic offers an optional output-token limit using `max_tokens`. Leave it off
when the selected endpoint/model does not accept that field. Freehand does not
substitute `max_completion_tokens`, send both fields, or retry after rejection.
The supported UI range is 1–65,536; the model can impose a smaller limit.

Generic does not send reasoning controls. S1-mini still requires thinking to be
disabled by the server. Choose a qualified llama.cpp or vLLM profile for automatic
S1-mini request enforcement. An explicit custom reasoning override must be
turned off before switching from llama.cpp or vLLM to Generic.

See [generation controls](../../guides/post-processing/#generation-controls).

## Language selection

The [language guide](../../guides/languages/) explains Server default, Automatic
detection, named language codes, and custom values. The chosen model determines
which languages work; profile availability is not a multilingual guarantee.
S1-mini cleanup is English only. It keeps raw text when a non-English input
language is selected or reported, and assumes English when no language is known.
