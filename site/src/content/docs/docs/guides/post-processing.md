---
title: Transcript post-processing
description: Configure optional transcript cleanup without risking the original transcription.
---

The app can optionally pass each successful raw speech-to-text result to a separate OpenAI-compatible `POST /chat/completions` endpoint before delivery. Speech recognition and cleanup have independent endpoints, models, and credentials. The raw transcript is retained first when history is enabled.

If post-processing fails, is unavailable, or returns no usable text, the run falls back to the raw transcript. A processor failure never turns a successful transcription into a failed transcription and never prevents raw dictation from being inserted.

## Custom-instruction processor

In **Settings > Processing**:

1. Enable transcript post-processing.
2. Enter the processor's OpenAI-compatible base URL, normally ending in `/v1`.
3. Enable **Allow insecure HTTP** only for a trusted local or LAN server using `http://`.
4. Test the connection and select a discovered model.
5. Choose **Custom instruction** and write the system instruction for the selected model. Freehand shows the UTF-8 storage limit, validates empty or oversized instructions, and can restore the recommended meaning-preserving instruction.
6. Add a separate processor API key only when that endpoint requires one.

The system instruction is stored with the rest of the ordinary Freehand settings; the API key remains in Windows Credential Manager. The app sends a non-streaming chat completion with temperature `0`. The raw transcript is the separate user message, so the instruction does not need a transcript placeholder. Freehand does not send the speech-to-text credential to the processor.

Endpoint model IDs and request behavior are deliberately independent. A local or remote server may assign any model ID, so Freehand never guesses that a model needs the S1-mini contract from its name. Switching profiles preserves each profile's stored controls so switching back does not discard the custom instruction or S1-mini choices.

## S1-mini by Superwhisper with llama.cpp

[S1-mini by Superwhisper](https://huggingface.co/superwhisper/s1-mini) is an English transcript normalizer rather than a general chat model. Its prompt format and non-thinking template are exact requirements. A normal transcript repeatedly producing an empty response usually means the inference runtime did not disable Qwen thinking mode.

Install the current llama.cpp Windows package from PowerShell:

```powershell
winget install --exact --id ggml.llamacpp
```

Open a new PowerShell window after installation, then start the server with reasoning disabled:

```powershell
llama-server.exe `
  -hf superwhisper/s1-mini-GGUF:Q4_K_M `
  --jinja `
  --reasoning off `
  --temp 0 `
  -ngl 99 `
  --sleep-idle-seconds 60
```

This is a known-working Freehand setup. `-hf` downloads and caches the Q4_K_M model from Hugging Face on first use. `-ngl 99` requests GPU offload; omit it for CPU-only inference. [`--sleep-idle-seconds 60`](https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md#sleeping-on-idle) unloads the model and its associated memory after one idle minute, then reloads it for the next request. Omit that option when consistently warm response time is more important than releasing idle resources. The default OpenAI-compatible base URL is:

```text
http://127.0.0.1:8080/v1
```

Configure **Settings > Processing** as follows:

```text
Enabled:             On
Endpoint:            http://127.0.0.1:8080/v1
Allow insecure HTTP: On
Model:               select the model returned by Test
Profile:             S1-mini
API key:             empty
```

The S1-mini profile sends the model's exact system prompt and control line and sets temperature `0`. Full Processing settings display the effective instruction and generated control line without allowing the fixed S1 contract to be edited. Reasoning is a backend concern rather than a portable OpenAI chat-completions field, so the runtime must disable it; current llama.cpp releases use `--reasoning off`.

The styling, structure, and context controls are limited to values on which S1-mini was trained. See [ADR 0001](../../decisions/0001-s1-mini-post-processing/) for the request contract and design constraints.
