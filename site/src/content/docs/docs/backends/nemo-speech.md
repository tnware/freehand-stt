---
title: NeMo-Speech.cpp
description: Use NeMo-Speech.cpp for transcription and MagpieTTS speech generation on one server.
---

Freehand's **NeMo-Speech.cpp** backend profile supports the **v0.1.0** server API
for microphone recordings, audio-file uploads, realtime microphone audio, and
[MagpieTTS speech generation](../../models/magpie-tts/).
For Nemotron's language and vocabulary controls, use the separate
**[Nemotron 3.5 ASR streaming model profile](../../models/nemotron/)**.

For automatic language detection with completed recordings, choose the
[Parakeet TDT v3 model profile](../../models/parakeet/).

## Run the server

On Windows and macOS, [managed local setup](../../guides/local-runtime/) can install NeMo,
download supported models when you choose **Get**, and start the server for you. The instructions
below are for manually managed local or remote servers.

Follow NeMo-Speech.cpp's [v0.1.0 installation guide](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/install.md)
to install the runtime and load a supported transcription model, a MagpieTTS
speech model with its codec and tokenizer assets, or both.
The server controls model loading, GPU selection, and chunk latency.
It can run on the same PC or on another machine reachable from Freehand.

## Combined transcription and speech

NeMo can load **one transcription engine and one speech-generation engine in the
same process**. Its [YAML configuration](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/config/server.example.yaml)
sets the model paths for each engine. Enabled engines load before the HTTP
listener becomes ready, so memory use and startup time include both models.
The request's model field does not switch between multiple loaded transcription
models: the server has one engine for each configured capability.

Freehand's managed setup selects one Nemotron or Parakeet transcription model
and an optional MagpieTTS speech model independently. Starting, stopping, or
restarting NeMo affects both. For a manual server, follow the upstream
[engine and listener configuration](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/server.md#engine-and-listener-configuration).
Freehand's Connections select how each task uses that server; they do not load
additional models on a remote server.

## Connect Freehand

1. Open **Connections** and choose **Add connection**.
2. Choose **NeMo-Speech.cpp**, name the connection, and enable the uses you need: **Voice transcription**, **Audio-file transcription**, and **Text to speech**.
3. Enter the HTTP API root, such as `http://127.0.0.1:8088/v1` for a local server on port 8088. Enable **Allow HTTP for this connection** when using HTTP on a trusted network; add authentication if required by the server.
4. Save the connection, then open the **Settings** cog in a workflow
   and select it in **Transcription** or **Speech** options.
5. Check the connection and select the model loaded on the server. Choose **Nemotron 3.5 ASR streaming** or **Parakeet TDT v3** for transcription, and **MagpieTTS Multilingual 357M** for speech generation.

Connection checks read server metadata without invoking the model. Each workflow filters model choices by its advertised capability; transcription also retains entries whose capability is unknown. Connection diagnostics can show both transcription and speech models from a combined server. Diagnostics show the advertised capabilities and device, plus the server version when `/health` is available. These facts do not select a model behavior profile or prove inference support. The qualified server version is **0.1.0**.

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

Speech generation uses `/v1/audio/speech` with buffered WAV output. The selected
[MagpieTTS profile](../../models/magpie-tts/) supplies its voice and language
controls. **Refresh voices** reads `voices` and `languages` from the speech model
in `/v1/models`; it does not invoke synthesis. A connection check identifies the
loaded speech model independently of the transcription model.

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
