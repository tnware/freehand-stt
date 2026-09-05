---
title: whisper.cpp
description: Connect the native whisper.cpp HTTP server for microphone and file transcription.
---

The **whisper.cpp** transcription profile supports completed microphone and
file uploads to the native HTTP server. Model loading and acceleration belong
to your server. Freehand never calls its model-loading route.

## Connect

1. Start `whisper-server` with the model you want to use.
2. In **Settings → Server**, select **whisper.cpp**.
3. Set the Base URL to the server root, for example `http://127.0.0.1:8081`.
   Omit `/v1` and `/inference`. A reverse-proxy prefix can be included.
4. Choose the authentication mode required by your deployment and permit HTTP
   only where appropriate. The native server may need a proxy for authentication.
5. Run **Test**, then save. The model field shows **Server-loaded model**;
   no client model ID is required or sent.

The default test reads `/health` beneath that root/prefix. An explicit custom
health path overrides the default. A successful health check establishes server
availability, not model identity or transcription quality. Freehand does not
invent a model inventory. A model ID retained from another profile remains saved
but is ignored by this adapter.

The default `/inference` route is qualified. If your server changes
`--inference-path`, expose the default route through a proxy. The base URL is a
root/prefix setting, not a full request URL.

## Supported controls and files

- Language, transcription context (`prompt`), and optional temperature use the
  server's multipart fields. Blank hints and disabled temperature are omitted,
  preserving server defaults. Model/language compatibility remains server-owned.
- Dedicated `hotwords` are unavailable. You can supply context through `prompt`.
- File transcription uses completed JSON. The streaming control and retry are
  unavailable for this profile; a stored streaming preference cannot cause a
  failed first upload or automatic resubmission.
- Microphone audio is Freehand's normalized WAV. WAV is the conservative file
  baseline; other formats depend on the server's decoder build or its optional
  FFmpeg conversion. Freehand does not transcode selected files for this adapter.
- Both workflows retain their existing size limits, timeouts, cancellation,
  optional cleanup, history policy, and delivery behavior.

## Contract and evidence

Requests are multipart `POST /inference`, including `file` and
`response_format=json`, without `model` or `stream`. Responses require a JSON
object with a string `text`. Non-success statuses and malformed responses fail
without replay. The configured credential and permitted custom headers apply
as they do for other transcription profiles.

Source qualification pins whisper.cpp
[`52a939a2a762`](https://github.com/ggml-org/whisper.cpp/blob/52a939a2a762224e255d366c1182b2af4dd1a032/examples/server/server.cpp).
Client fixtures cover prefixed routing, health defaults/overrides, omitted
model fields, bounded multipart upload, hints, and completed-only behavior.
This is not a claim that every build, audio format, or model has been tested
interactively on Windows.


### Scoped live acceptance — 2026-09-05

Freehand's Windows Go adapters completed both microphone-request and file-upload
paths against the existing CUDA image (`sha256:2c42506808d7546ea3440c0053dd6543373cc4252c525b7972ab96554a533837`)
using `ggml-tiny.en.bin` and the public 11-second whisper.cpp JFK sample. The
server's `/health` probe succeeded. This exercises HTTP/inference behavior from
Windows; interactive capture, focus-safe insertion, and every file format were
not part of that fixed-sample run. The source pin above records inspected
contract evidence independently of the tested image digest.
