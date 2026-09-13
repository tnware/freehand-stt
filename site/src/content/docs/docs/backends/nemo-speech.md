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

Follow NeMo-Speech.cpp's [v0.1.0 installation guide](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/install.md)
to install the runtime and load **Nemotron 3.5 ASR streaming 0.6B**.
The server controls model loading, GPU selection, and chunk latency.
It can run on the same PC or on another machine reachable from Freehand.

## Connect Freehand

1. Open **Settings → Connections** and choose **Add connection**.
2. Choose **NeMo-Speech.cpp**, name the connection, and enable **Voice transcription**, **Audio-file transcription**, or both.
3. Enter the HTTP API root, such as `http://127.0.0.1:8088/v1` for a local server on port 8088. Enable **Allow HTTP for this connection** when using HTTP on a trusted network; add authentication if required by the server.
4. Save the connection, then select it in each workflow you want to use.
5. Check the connection and select the model loaded on the server. Choose **Nemotron 3.5 ASR streaming** as the model profile for its language, vocabulary, and realtime controls.

Connection checks read server metadata without invoking the model. Generic
model behavior offers completed transcription. Select the explicit Nemotron
profile to enable **Realtime transcription** in Voice's Transcription panel.

## Supported API

Completed audio uploads use `/v1/audio/transcriptions` and return JSON text.
Realtime uses `/v1/realtime`, NeMo-Speech.cpp's versioned WebSocket protocol.
Freehand derives the WebSocket address from your HTTP base URL; HTTPS uses WSS.
An API key, when configured, is sent in the upgrade header.

This backend returns completed file results; it does not stream partial results
from file uploads. Enable live microphone transcription in Voice; turning it
off keeps the same connection and model for completed recordings.
