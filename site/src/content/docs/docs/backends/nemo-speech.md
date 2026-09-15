---
title: NeMo-Speech.cpp
description: Connect the NeMo-Speech.cpp server for completed audio and realtime microphone transcription.
---

Freehand's **NeMo-Speech.cpp** backend profile supports the **v0.1.0** server API
for microphone recordings, audio-file uploads, and realtime microphone audio.
For Nemotron's language and vocabulary controls, use the separate
**[Nemotron 3.5 ASR streaming model profile](../../models/nemotron/)**.

For automatic language detection with completed recordings, choose the
[Parakeet TDT v3 model profile](../../models/parakeet/).

## Run the server

On Windows and macOS, [managed local setup](../../guides/local-runtime/) can install NeMo,
download a supported speech model, and start the server for you. The instructions
below are for manually managed local or remote servers.

Follow NeMo-Speech.cpp's [v0.1.0 installation guide](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/install.md)
to install the runtime and load **Nemotron 3.5 ASR streaming 0.6B**.
The server controls model loading, GPU selection, and chunk latency.
It can run on the same PC or on another machine reachable from Freehand.

## Connect Freehand

1. Open **Connections** and choose **Add connection**.
2. Choose **NeMo-Speech.cpp**, name the connection, and enable **Voice transcription**, **Audio-file transcription**, or both.
3. Enter the HTTP API root, such as `http://127.0.0.1:8088/v1` for a local server on port 8088. Enable **Allow HTTP for this connection** when using HTTP on a trusted network; add authentication if required by the server.
4. Save the connection, then open the **Settings** cog in Voice transcription or Audio file
   and select it in that workflow's **Transcription** options.
5. Check the connection and select the model loaded on the server. Choose **Nemotron 3.5 ASR streaming** as the model profile for its language, vocabulary, and realtime controls.

Connection checks read server metadata without invoking the model. NeMo model choices include advertised transcription models and entries whose capability is unknown; advertised speech-generation and translation models are excluded. Diagnostics show the advertised capabilities and device, plus the server version when `/health` is available. These facts do not select a model behavior profile or prove inference support. The qualified server version is **0.1.0**.

Generic
model behavior offers completed transcription. Select the explicit Nemotron
profile to enable **Realtime transcription** in Voice's Transcription panel.

## Supported API

Completed audio uploads use `/v1/audio/transcriptions` with `verbose_json` to receive text, language, and duration.
Realtime uses `/v1/audio/transcriptions/realtime`, the explicit NeMo transcription WebSocket endpoint. Freehand does not fall back to the general realtime route.
Freehand derives the WebSocket address from your HTTP base URL; HTTPS uses WSS.
An API key, when configured, is sent in the upgrade header.

This backend returns completed file results; it does not stream partial results
from file uploads. Enable live microphone transcription in Voice; turning it
off keeps the same connection and model for completed recordings.

## Transcription controls

Choose the qualified **Nemotron 3.5 ASR streaming** or **Parakeet TDT v3** model
profile, then open **NeMo transcription controls** in the workflow's
Transcription options. Voice and audio files remember their controls separately
for each connection and model. Save applies changes to the next request.

| Control                       | Default           | Effect and prerequisites                                                                                                                                                                        |
| ----------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Automatic punctuation         | On                | Keeps model punctuation and server formatting. Off asks NeMo to strip formatting.                                                                                                               |
| Normalize numbers and dates   | Off               | Requests inverse text normalization (ITN), such as “twenty one” → “21”. Requires a server build with ITN support and configured language grammars.                                              |
| Profanity filter              | Off               | Requests masking using a word list configured on the server.                                                                                                                                    |
| Server endpointing delay (ms) | 0: server default | Realtime only. A value from 100 to 10,000 adjusts the pause threshold when server endpointing is already enabled. It does not enable endpointing or change Freehand's microphone stop controls. |

The server's metadata does not report whether these optional assets or
endpointing are configured. Managed NeMo does not install ITN grammars or a
profanity word list, and uses the default server endpointing configuration.
Enabling a request control cannot supply missing assets or enable server
endpointing. No extra models or files are downloaded by these controls.

These controls follow the pinned [NeMo server API](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/api.md)
and [ASR configuration](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/asr/configuration.md).
