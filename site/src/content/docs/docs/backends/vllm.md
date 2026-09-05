---
provider: vllm
title: vLLM
description: Configure vLLM for completed or streamed transcription and text cleanup.
---

The **vLLM** profiles cover speech transcription and text cleanup independently.
Choose a speech model for transcription and a text model for cleanup; each
operation has its own endpoint, model ID, and credential settings. Your servers
can run locally, on another machine, or behind a compatible hosted deployment.

## Run vLLM with Docker

Use PowerShell with Docker Desktop's WSL2 Linux backend and an NVIDIA GPU.
This recipe reproduces the pinned v0.28.0 environment used for the scoped
[acceptance evidence](#contract-and-evidence). It favors compatibility on the
tested Windows/WSL setup; its scheduler and memory settings are not throughput
tuning recommendations for every machine.

Create a directory containing a file named `Dockerfile`:

```dockerfile
FROM vllm/vllm-openai@sha256:61fc8a896b0a4fbbbdc063bc4b0dbc25ce98e02b5050c24aeb7830ac02039b14
RUN python3 -m pip install --no-deps av==18.1.0 scipy==1.18.1 soundfile==0.14.0 soxr==1.1.0
```

These are the audio dependencies missing from that specific image; do not
apply this list blindly to a different release. For other versions, follow
[upstream installation](https://docs.vllm.ai/en/latest/getting_started/installation/gpu/)
and the release's audio extras instructions. In the directory containing the
Dockerfile, build the image and prepare its persistent model cache:

```powershell
docker build --tag freehand-vllm-audio:0.28.0 .
docker volume create freehand-vllm-models
```

### Start Qwen3-ASR transcription

```powershell
docker run --detach --name freehand-vllm-stt `
  --gpus device=0 --shm-size 2g `
  --publish 127.0.0.1:8052:8000 `
  --volume freehand-vllm-models:/models/hf `
  --env HF_HOME=/models/hf --env VLLM_USE_V2_MODEL_RUNNER=0 `
  --entrypoint python3 freehand-vllm-audio:0.28.0 `
  -m vllm.entrypoints.openai.api_server `
  --model Qwen/Qwen3-ASR-0.6B --host 0.0.0.0 --port 8000 `
  --max-model-len 4096 --max-num-batched-tokens 4096 `
  --gpu-memory-utilization 0.35 --max-num-seqs 1 `
  --enforce-eager --no-async-scheduling --no-enable-log-requests
```

The server downloads only the selected checkpoint on first startup. Qwen3-ASR
also has a 1.7B checkpoint; changing model size requires available memory and a
separate quality check. The 0.6B example has user-confirmed Freehand evidence.
The [official vLLM recipe](https://docs.vllm.ai/projects/recipes/en/latest/Qwen/Qwen3-ASR.html)
documents the same transcription API.

```powershell
docker logs --tail 30 freehand-vllm-stt
Invoke-RestMethod http://127.0.0.1:8052/health
Invoke-RestMethod http://127.0.0.1:8052/v1/models
```

In Freehand, choose **vLLM**, base URL **`http://127.0.0.1:8052/v1`**,
model **`Qwen/Qwen3-ASR-0.6B`**, authentication **None**, and allow local HTTP.
Save, then test a short recording. Automatic language behavior depends on the
model; explicit English is also available for an English test.

```powershell
docker stop freehand-vllm-stt
docker start freehand-vllm-stt
```

### Start S1-mini cleanup

This is a separate text model and endpoint. Stop the STT container first if
your GPU cannot hold both. To use STT and cleanup together, provide enough
capacity for both or use another speech server; a stopped STT endpoint cannot
transcribe a new recording.

```powershell
docker run --detach --name freehand-vllm-cleanup `
  --gpus device=0 --shm-size 2g `
  --publish 127.0.0.1:8053:8000 `
  --volume freehand-vllm-models:/models/hf `
  --env HF_HOME=/models/hf --env VLLM_USE_V2_MODEL_RUNNER=0 `
  --entrypoint python3 freehand-vllm-audio:0.28.0 `
  -m vllm.entrypoints.openai.api_server `
  --model superwhisper/s1-mini --host 0.0.0.0 --port 8000 `
  --max-model-len 2048 --max-num-batched-tokens 2048 `
  --gpu-memory-utilization 0.35 --max-num-seqs 1 `
  --enforce-eager --no-async-scheduling --no-enable-log-requests
```

Check `/health` and `/v1/models` at port **8053**. Enable Freehand
post-processing with profile **vLLM**, base URL **`http://127.0.0.1:8053/v1`**,
model **`superwhisper/s1-mini`**, authentication **None**, local HTTP allowed,
and the **S1-mini** prompt preset. The preset forces reasoning off. The
2,048-token context is a small test configuration; start with short transcripts.

```powershell
docker logs --tail 30 freehand-vllm-cleanup
docker stop freehand-vllm-cleanup
docker start freehand-vllm-cleanup
```

To change a container's launch options, stop and remove that named container,
then repeat its `docker run` command with the new options. The named model
volume remains. This workflow changes the server, not Freehand's saved
credentials or connections.

## Connect

Use a Base URL ending in `/v1`, such as `http://127.0.0.1:8000/v1`, and the
model ID advertised by that server. Select **vLLM** in the relevant Settings
section, choose the authentication appropriate to your deployment, test, and
save. Connection tests read `/models` beneath the base URL without inference.
An explicit transcription health path retains the existing base-relative rules.

`Qwen/Qwen3-ASR-0.6B` was confirmed working by a user in the native Freehand
application with vLLM v0.28.0. Use the exact model ID exposed by your deployment.
The [upstream Qwen3-ASR guide](https://docs.vllm.ai/projects/recipes/en/latest/Qwen/Qwen3-ASR.html)
documents the transcription API used by this profile.

The initial English-only `openai/whisper-tiny.en` test checkpoint had limitations:
automatic language detection returned HTTP 500, and some speech returned empty
text even with explicit English. Its passing fixed sample is limited transport
evidence, not a recommendation for dictation. Freehand preserves your language
choice; it does not silently force English for other models.

For Windows-hosted testing, upstream recommends WSL for vLLM's Linux runtime;
Freehand itself remains a native Windows application. See the
[upstream installation guide](https://docs.vllm.ai/en/latest/getting_started/installation/gpu/).

## Local runtime setup notes

The tested v0.28.0 base image needed the optional audio packages for
transcription. Install the matching release's audio extras when building your
server image; keep its existing CUDA/PyTorch dependencies pinned. A running
metadata endpoint alone does not establish that audio decoding is installed.

On the tested WSL2/RTX 5060 Ti system, the default V2 runner failed with
`UVA is not available`. Setting `VLLM_USE_V2_MODEL_RUNNER=0` selected the installed
older runner. Whisper's encoder also required `--max-num-batched-tokens 2048`
even with `--max-model-len 448` and one concurrent request. With that older
runner, the short sample stalled under asynchronous scheduling;
`--no-async-scheduling` allowed completed and streamed requests to finish. These are scoped
server setup observations, not settings Freehand sends in its API requests.

## Transcription

Microphone requests use completed `POST /audio/transcriptions` JSON. Files can
use completed JSON or vLLM's server-sent transcription chunks. The file is
uploaded once; streaming describes the arriving result, not realtime microphone
transcription.

Language, context (`prompt`), and optional temperature are supported request
fields. v0.28.0's Whisper and Qwen3-ASR implementations consume the context;
model-specific interpretation and language support still vary. Dedicated
hotwords remain unavailable: accepting a schema field is not evidence that the
qualified model paths use it. Freehand sends no unqualified model-specific
sampling, VAD, translation, timestamp, or diarization options.

A vLLM stream carries `object: "transcription.chunk"` and
`choices[].delta.content`. Each server-side audio chunk can finish separately.
Freehand preserves deltas exactly and requires a successful final chunk plus
`[DONE]` for the entire request. Length-limited or aborted chunks, malformed
payloads, provider errors, and premature disconnects remain failures. Accepted
partial text is available for manual recovery and never treated as a successful
transcript for automatic cleanup. There is no automatic replay. Completed JSON
returned to a streaming request uses the existing completed-result handling.

Supported audio formats, upload ceilings, server-side audio splitting, and
language behavior depend on the deployed vLLM/model combination. Existing
Freehand file and response bounds still apply.

## Cleanup

Cleanup uses non-streaming `POST /chat/completions`, with the configured model,
string system/user messages, and temperature zero.

- Optional output limits send `max_tokens` (1–65,536). Off omits the field.
- For **Custom instruction**, **Disable reasoning** optionally sends
  `reasoning_effort: "none"`.
- **S1-mini requires reasoning off.** Its preset always sends that override
  through this profile, independently of the saved custom-model switch.
- A compatible runtime and model template must honor the reasoning setting.
  Freehand does not rewrite arbitrary templates or infer support from model IDs.
- Rejected requests, empty output, and `finish_reason: "length"` use the
  existing raw-transcript fallback without retry.

The fixed S1-mini instruction and trained controls remain unchanged. An output
limit does not implement long-input chunking or enlarge the context window.

## Contract and evidence

This profile is qualified against **vLLM v0.28.0**:

- [Transcription request and response schemas](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/transcription/protocol.py).
- [Speech stream implementation](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/base/serving.py), including per-chunk finish reasons and whole-file completion.
- [Chat request mapping](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/openai/chat_completion/protocol.py), which maps reasoning effort `none` to template thinking disabled.

Client fixtures cover requests, framing, multiple audio chunks, usage, failures,
cancellation, credential reflection, and required S1-mini reasoning off. Source
and fixture evidence do not establish interoperability for every earlier
release or model/template. Record actual server versions and models when doing
live acceptance.

On September 5, 2026, the native Windows client HTTP adapter passed the public
11-second whisper.cpp JFK sample against v0.28.0 with
`openai/whisper-tiny.en`: completed microphone-shaped upload, completed file,
and streamed file (22 nonempty deltas). A 33-second repetition of that sample
returned two empty, successfully terminated server chunks; it is recorded as
an empty model result, not successful long-file recognition. Multiple-chunk
text assembly and incomplete-stream recovery are covered by fixtures. These
checks do not establish microphone capture, focus-safe insertion, recognition
quality, or support for every audio format.

On the same date, the user confirmed successful live transcription in Freehand
with **vLLM v0.28.0 and `Qwen/Qwen3-ASR-0.6B`**. GPU access and GPU model/cache
allocation were verified. The observed request was slow under the conservative
WSL test configuration; latency tuning remains separate. This confirmation does
not establish Qwen file streaming, all languages/formats, or exhaustive native
focus-safety acceptance.

The same v0.28.0 runtime also passed a fixed S1-mini cleanup request using
`superwhisper/s1-mini` and the native Windows processing adapter, with the
optional custom-model reasoning switch off: the S1-mini preset still enforced
`reasoning_effort: "none"`. A one-token output limit produced the expected
incomplete-response error rather than accepting truncated cleanup text.

vLLM-Omni speech playback is a separate, still-planned profile. These profiles
do not add realtime microphone transcription or model management.
