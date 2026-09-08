---
title: NeMo-Speech.cpp
description: Use a user-managed Nemotron server for completed or realtime transcription.
---

Freehand supports **NeMo-Speech.cpp v0.1.0** for completed microphone and audio-file transcription. With the explicit **Nemotron 3.5 ASR streaming** model profile, Voice can also use its qualified realtime protocol. Freehand does not install or manage this server as a product feature.

Add a **NeMo-Speech.cpp** connection and enable the uses you need: **Voice transcription**, **Audio-file transcription**, or both. Enter the HTTP API root, for example `http://127.0.0.1:8088/v1` for a locally configured server, and explicitly allow insecure HTTP when appropriate. Select the connection in each task independently. Metadata checks read server information without invoking a model.

Choose the model loaded on the server and its explicit model profile. Generic offers completed transcription without assuming Nemotron-specific behavior. The Nemotron profile restricts spoken languages to supported base-model locales. In Voice, it also exposes **Realtime transcription** inside the Transcription panel. See the [live transcription guide](../../guides/live-transcription/) for language, vocabulary, and caption controls.

Completed requests upload audio to `/v1/audio/transcriptions` and expect JSON. The server owns the loaded model; Freehand requests native punctuation and verbatim output without optional ITN processing. Realtime uses `/v1/realtime`, a versioned project-specific WebSocket protocol. It is separate from streamed responses to a completed audio-file upload; this backend does not expose file-response streaming in Freehand.

Turning realtime off retains the same Voice connection/model and returns to completed recording or checkpoints. Audio file keeps its own settings. Native capture, cancellation, cleanup, and focus-safe insertion remain Freehand's responsibility; GPU selection and model loading stay on your chosen server.
