---
title: Get started
description: Install Freehand, connect a transcription endpoint, and complete your first dictation.
---

This guide takes you from installation to a working voice dictation. Freehand
connects to a speech-to-text service you provide; it does not install or run a
model for you.

## What you need

- Windows 11 with WebView2.
- A microphone, or a supported stored-audio file.
- A reachable OpenAI-compatible transcription endpoint.
- The endpoint's base URL, transcription model ID, and API key when authentication is required.

:::caution[Alpha distribution]
Freehand's GitHub Releases provide an unsigned executable and per-user NSIS installer. Download only from the official project, verify the corresponding entry in `SHA256SUMS`, and expect Windows to identify the publisher as unknown until Authenticode signing is introduced.
:::

Download the latest Windows alpha from the
[Releases page](https://github.com/tnware/freehand-stt/releases). The installer
is the normal first-install path. The bare executable exists for portable use
and for Freehand's Wails updater.

## First launch

Freehand opens a readiness screen the first time it starts. Work through each required item:

1. **Speech-to-text server** — enter the base URL and model ID under **Settings → Server**.
2. **Authentication** — enter an API key if the endpoint requires one. Freehand stores it in Windows Credential Manager, not in the JSON settings file.
3. **Microphone** — select **System default** or a specific device under **Settings → Audio**.
4. **Recording shortcut** — capture a toggle or hold-to-talk chord under **Settings → Shortcuts**.
5. **Connection check** — let Freehand perform its single metadata-only health/model-list request.

The connection check never submits audio and never invokes a discovered model.

## Configure the speech endpoint

Freehand expects a base URL whose OpenAI API prefix is already present, typically:

```text
https://speech.example.com/v1
```

It joins the transcription route as:

```text
POST https://speech.example.com/v1/audio/transcriptions
```

Set the exact model ID expected by your gateway. Model discovery is bounded metadata; selecting a discovered ID does not run it.

HTTPS is required by default. **Allow insecure HTTP** is an explicit opt-in for a trusted local or LAN service. It sends credentials and audio without transport encryption, so do not enable it for an untrusted network.

See [Connect a speech server](../guides/connect-a-server/) for shared-gateway
and separate-service layouts, the evidence-labeled compatibility list, and a
failure guide.

## Run your first dictation

1. Focus the text field where the result should go.
2. Start the configured toggle shortcut, or hold the configured hold-to-talk chord.
3. Speak.
4. Stop the toggle recording, or release the hold-to-talk chord.
5. Wait while Freehand transcribes the audio.

If the same safe target still owns focus, Freehand inserts the text with Unicode input. If focus or process identity changed, it fails closed and leaves the transcript available for explicit copying.

## Transcribe a stored audio file

Choose **Audio file** in the main workspace and select FLAC, MP3, MP4, MPEG, MPGA, M4A, OGG, WAV, or WebM audio through the native picker.

Stored-audio transcription never inserts automatically because no destination was captured at recording start. Copy the completed result explicitly. The selected full path and audio bytes remain owned by Go and do not cross the Wails bridge.

## Optional features

All of these are opt-in:

- Local voice activity detection, silence trimming, automatic stop, and pause-aware checkpoints.
- Bounded memory-only transcript history.
- A separate OpenAI-compatible transcript cleanup stage.
- Completed or compatible streamed responses for stored audio.
- On-demand transcript playback and text-to-speech through an independent endpoint.
- The passive native recording overlay and optional Windows Mica shell material.

See [Use Freehand](../guides/using-freehand/) for workflow details and [Transcript post-processing](../guides/post-processing/) for cleanup profiles.

If setup does not complete, use [Troubleshooting](../guides/troubleshooting/).
For exact request and response contracts, see
[Protocol compatibility](../reference/protocol/).
