---
title: vLLM-Omni
description: Connect vLLM-Omni for text to speech with Qwen3-TTS preset voices and style instructions.
---

Freehand supports **vLLM-Omni v0.18.0** for buffered WAV speech generation.
Choose this backend separately from the **vLLM** transcription or cleanup
backend. For text to speech, select **vLLM-Omni** in Connections.

## Run the server

Follow the runtime's [installation and Qwen3-TTS serving guide](https://docs.vllm.ai/projects/vllm-omni/en/v0.18.0/user_guide/examples/online_serving/qwen3_tts/)
on your inference machine. Load **Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice** and
expose its HTTP API. Configure model loading and the GPU on that server.

## Connect Freehand

1. Open **Connections → Add connection**.
2. Choose **vLLM-Omni**, select **Text to speech** under **Used for**, and enter the HTTP API root,
   such as `http://127.0.0.1:8091/v1` for a local service on port 8091.
3. Configure authentication. If using HTTP on a trusted network, enable
   **Allow HTTP for this connection**, then save.
4. Open **Text to speech** and its **Settings** cog, then select the connection and model in **Speech** options.
5. Choose the [Qwen3-TTS model profile](../../models/qwen3-tts/) to use its preset
   speakers, ten languages, and voice-style instructions.
6. Turn on **Enable text to speech**, choose a preset voice, and use **Preview**
   to hear it. Choose **Save**.

Connection checks use model metadata. **Refresh voices** reads `/v1/audio/voices`.
Speech generation uses `POST /v1/audio/speech` with `response_format: "wav"` and
`stream: false` for native playback. The Qwen profile additionally sends its
`task_type: "CustomVoice"`, language name, and optional `instructions`.

Generic model behavior offers the standard voice-ID and speed fields when
your loaded speech model accepts them. Choose the dedicated Qwen profile only
for its matching checkpoint. Freehand does not upload reference voices or select
voice-cloning tasks through this integration.
