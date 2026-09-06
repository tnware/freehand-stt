---
title: Kokoro-FastAPI
description: Connect Kokoro-FastAPI for speech playback and searchable server voices.
---

Choose **Kokoro-FastAPI** for on-demand speech playback. The dedicated profile
supports voice discovery, manual voice IDs, speed control, and buffered PCM16
WAV audio. It does not provide transcription or transcript cleanup.

## Connect an existing server

1. Open **Settings → Connections**, create a connection, and select
   **Kokoro-FastAPI** with **Speech playback** enabled.
2. Enter the server's API base URL, including `/v1`. For the local Docker example
   below, use `http://127.0.0.1:8880/v1`. For a remote server, use its HTTPS URL
   and the authentication supplied by its administrator.
3. Save the connection. In **Settings → Speech playback**, select it and choose
   the model ID advertised by your server, normally `kokoro`.
4. Click **Refresh voices**, then search or choose a voice such as `af_heart`.
   Manual IDs remain available, including administrator-provided aliases.
5. Enable speech playback, save feature settings, and explicitly preview the voice.

Voice and speed are remembered for the selected connection and model. Refreshing
models or voices performs metadata requests only; it does not synthesize samples.
A server-wide voice list does not establish that every voice works with every
model ID. Failed discovery leaves the selected voice and manual entry intact.

## Run Kokoro-FastAPI with Docker

These PowerShell examples require Docker Desktop using Linux containers. The
CPU image is enough to get started and requires no dedicated GPU. Images include
model assets; first use downloads the image. The upstream `latest` tag changes,
so pin a release tag or image digest for a repeatable deployment.

```powershell
docker run -d --name freehand-kokoro --restart unless-stopped `
  -p 127.0.0.1:8880:8880 `
  ghcr.io/remsky/kokoro-fastapi-cpu:latest
```

For an NVIDIA GPU, replace the final image with
`ghcr.io/remsky/kokoro-fastapi-gpu:latest` and add `--gpus all` to the Docker
options. RTX 50-series hardware needs the upstream CUDA 12.8 image
`ghcr.io/remsky/kokoro-fastapi-gpu:latest-cu128`; see the
[upstream hardware-specific recipes](https://github.com/remsky/Kokoro-FastAPI#quick-start).
Choose one container recipe for this name and port.

```powershell
docker logs -f freehand-kokoro
docker stop freehand-kokoro
docker start freehand-kokoro
```

The bundled API documentation is at `http://127.0.0.1:8880/docs`. This recipe
binds to loopback for a client on the same machine; see
[connection topologies](../../guides/connect-a-server/#choose-a-topology) for a
separate server. Freehand does not start or manage the container.

## Contract and evidence

| Capability | Contract |
| --- | --- |
| Model discovery | `GET /v1/models`; listed IDs may be compatibility aliases. |
| Voice discovery | `GET /v1/audio/voices`; current ID/name objects and legacy strings. |
| Speech | `POST /v1/audio/speech` with `model`, `input`, `voice`, `speed`, `response_format: "wav"`, and `stream: false`. |
| Playback | Fully buffered PCM16 WAV; speed requests from 0.25× through 4×. |

A manually requested sample against a deployment reporting API **0.6.0** returned
68 voice entries and a playable-format 24 kHz mono PCM16 WAV for `kokoro` with
`af_heart` at speed 1.0. Freehand's real request adapter and WAV decoder handled
that response on Windows. This verifies the response format, not speaker-device
playback or every voice, model alias, speed, image tag, or server version.

Local fixtures cover both voice-list shapes, Speaches isolation, authentication,
malformed and oversized metadata, redirects, and the Kokoro buffering override.
The upstream router was inspected at
[`5fb71ea`](https://github.com/remsky/Kokoro-FastAPI/blob/5fb71ea6e75379f95dee0f4a42c12152f4ea0e1a/api/src/routers/openai_compatible.py).

## Limits

Language overrides, server normalization, voice-blend creation, cloning inputs,
captioned audio, and progressive playback are not exposed by this profile.
These need separate request contracts and model qualifications. The existing
request timeout and 32 MiB speech response limit still apply.
