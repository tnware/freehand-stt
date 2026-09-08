---
title: Qwen3-ASR
description: Use Qwen3-ASR on vLLM for completed audio and optional realtime microphone transcription.
---

Choose **vLLM** for the connection and **Qwen3-ASR** for the model profile.
They describe different things: vLLM supplies the server API; the explicit model
profile limits controls to Qwen's qualified behavior. Freehand does not infer
the profile from a model name or download models itself.

## Connect Freehand

1. Create a connection with backend **vLLM**, base URL such as
   `http://127.0.0.1:8089/v1`, and Transcription enabled under **Used for**.
   A local unauthenticated server uses **None** and requires permission for HTTP.
   Remote deployments use their own address and authentication.
2. In **Voice → Transcription**, select that connection, check its model list,
   choose **Qwen/Qwen3-ASR-1.7B**, then choose the **Qwen3-ASR** model profile.
3. Enable **Realtime transcription** inside this same panel for live results.
   **Live overlay captions** shows the latest text in the existing single-row
   overlay. Stop recording to finalize, optionally clean up, and safely insert.
4. Turn realtime off for completed recordings and pause-aware checkpoints.
   Select the connection and model separately in **Audio file** to use files.

Metadata checks only read the server's health/model endpoints. A successful
check confirms reachability; it does not establish recognition quality.

## Supported controls

| Setting | Completed recording / audio file | Realtime microphone |
| --- | --- | --- |
| Model profile | Qwen3-ASR | Same profile and selected model |
| Language | Automatic or a qualified language hint | Automatic only |
| Context hint | Recognition context, sent as `prompt` | Unavailable |
| Shared Vocabulary | Appended to context when enabled | Unavailable; list and preference preserved |
| Temperature | Optional 0–1 request override | Unavailable |
| File response streaming | Supported | Separate WebSocket audio transport |
| Caption preview | After completed transcription | Provisional live text; never inserted directly |

The model card lists 30 languages and Chinese dialect recognition. The vLLM
0.28.0 language map supports only 28 matching explicit hints: Cantonese and
Filipino must use automatic detection. Freehand does not substitute an
unqualified Tagalog prompt. Dialects, translation, forced alignment, timestamps,
diarization, and model training have no controls in this profile.

Context and vocabulary are recognition hints, not a chat instruction or a
guarantee of spelling. The shared library remains in **Settings → Vocabulary**.
Completed settings stay saved when realtime is enabled and apply again when
it is disabled. Realtime sends no language, prompt, vocabulary, or temperature.

Freehand displays provisional text as it arrives and removes Qwen's structured
language headers. Stopping recording asks the server to finalize. Disconnects,
cancellation, and missing finals discard previews. Finalization has a 30-second
budget after capture stops.

## Run the qualified server

Use a user-managed Linux vLLM **0.28.0** runtime with its audio dependencies.
For Windows, Docker Desktop's WSL2 GPU backend is one deployment option.
The [vLLM Docker setup](../vllm/#run-vllm-with-docker) documents the pinned
image and additional audio packages used for this qualification.

Serve the requested 1.7B checkpoint with the realtime architecture override.
The same loaded model also serves completed audio; a second model is unnecessary.
For example, in the Linux runtime:

```sh
vllm serve Qwen/Qwen3-ASR-1.7B \
  --revision 7278e1e70fe206f11671096ffdd38061171dd6e5 \
  --hf-overrides '{"architectures":["Qwen3ASRRealtimeGeneration"]}' \
  --host 0.0.0.0 --port 8000 \
  --gpu-memory-utilization 0.55 --max-model-len 4096 --max-num-seqs 1 \
  --enforce-eager --no-enable-log-requests --disable-log-stats
```

These memory and concurrency values are a small local test configuration,
not universal sizing advice. Publish the container port as
`127.0.0.1:8089:8000` for local testing. A remotely accessible server needs the
deployment's authentication and network protections. Keep request-content
logging disabled. No forced-aligner model is needed.

Check `http://127.0.0.1:8089/health` and
`http://127.0.0.1:8089/v1/models` after startup. Freehand derives
`ws://127.0.0.1:8089/v1/realtime` from the base URL; enter the HTTP base URL,
not the WebSocket URL, in Connections. An HTTPS base uses WSS automatically.
Stop a competing GPU inference model before starting this one if capacity is
limited. Model installation, startup, shutdown, and tuning remain server tasks.

## Qualification

This integration targets vLLM 0.28.0 and the original
`Qwen/Qwen3-ASR-1.7B` checkpoint at the revision above. It does not automatically
qualify other runtimes, modified weights, quantizations, or a future vLLM protocol.
The completed API and realtime endpoint were exercised with synthetic English
audio through Docker/WSL2, including Freehand's Go realtime adapter. This is
transport evidence; native microphone, overlay, and insertion acceptance is
separate. Runtime performance and recognition quality belong to the chosen
model and deployment.

Sources: [Qwen model card](https://huggingface.co/Qwen/Qwen3-ASR-1.7B),
[vLLM realtime protocol](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/realtime/protocol.py),
[Qwen realtime implementation](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/model_executor/models/qwen3_asr_realtime.py),
and [completed transcription implementation](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/model_executor/models/qwen3_asr.py).
