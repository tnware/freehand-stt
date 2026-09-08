---
provider: vllm-omni
title: vLLM-Omni
description: Connect vLLM-Omni for text to speech with Qwen3-TTS preset voices and style instructions.
---

Freehand supports **vLLM-Omni v0.18.0** for buffered WAV speech generation.
Choose this backend separately from the **vLLM** transcription or cleanup
backend; it serves a different speech API implementation.

## Run the server

Follow the runtime's [installation and Qwen3-TTS serving guide](https://docs.vllm.ai/projects/vllm-omni/en/v0.18.0/user_guide/examples/online_serving/qwen3_tts/)
on your inference machine. Load **Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice** and
expose its HTTP API. The runtime owns model loading and device configuration.

## Connect Freehand

1. Open **Settings → Connections → New connection**.
2. Choose **vLLM-Omni**, enable **Text to speech**, and enter the HTTP API root,
   such as `http://127.0.0.1:8091/v1` for a local service on port 8091.
3. Configure HTTP permission and authentication for your endpoint, then save.
4. In **Text to speech**, select the connection and model.
5. Choose the [Qwen3-TTS model profile](../../models/qwen3-tts/) to use its preset
   speakers, ten languages, and voice-style instructions.

Connection checks use model metadata. **Refresh voices** reads `/v1/audio/voices`.
Speech generation sends `/v1/audio/speech` with `response_format: "wav"` and
`stream: false` for native playback. The Qwen profile additionally sends its
CustomVoice task, language name, and optional style instructions.

Generic model behavior offers the standard voice-ID and speed fields when
your loaded speech model accepts them. Choose the dedicated Qwen profile only
for its matching checkpoint. Freehand does not upload reference voices or select
voice-cloning tasks through this integration.
