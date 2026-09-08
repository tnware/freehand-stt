---
provider: vllm
title: vLLM
description: Configure vLLM for completed transcription, Qwen realtime, and text cleanup.
---

The **vLLM** profiles cover speech transcription and text cleanup independently.
Choose a speech model for transcription and a text model for cleanup; each
operation has its own endpoint, model ID, and credential settings. Your servers
can run locally, on another machine, or behind a compatible hosted deployment.

For **Qwen3-ASR-1.7B**, including optional realtime microphone results and
captions, use the [Qwen3-ASR setup guide](../../models/qwen3-asr/). Select its explicit
model profile in feature settings; the connection remains vLLM.

For completed recognition, choose [Cohere Transcribe](../../models/cohere-transcribe/).
For Mistral’s live microphone model, choose [Voxtral Mini Realtime](../../models/voxtral-realtime/).
The [vLLM-Omni backend](../vllm-omni/) handles Qwen3-TTS speech generation separately.

## Run vLLM with Docker

Use PowerShell with Docker Desktop's WSL2 Linux backend and an NVIDIA GPU.
This recipe uses vLLM v0.28.0 with the audio packages needed for transcription.

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
also has a 1.7B checkpoint, covered in the [Qwen3-ASR guide](../../models/qwen3-asr/).
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
2,048-token context in this example is intended for short transcripts.

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
model ID advertised by that server. In **Settings → Connections**, create an
entry with the **vLLM** profile and enable Transcription and/or Post-processing
under **Used for**, according to the routes your deployment exposes. Set its URL,
authentication, and HTTP permission, then **Save connection**. Select that entry
on its feature page, choose the model, and save feature settings. Connection
tests read `/models` beneath the base URL without inference.
An explicit transcription health path retains the existing base-relative rules.

On Windows, upstream recommends WSL for vLLM's Linux runtime;
Freehand itself remains a native Windows application. See the
[upstream installation guide](https://docs.vllm.ai/en/latest/getting_started/installation/gpu/).

## Local runtime setup notes

The pinned v0.28.0 image needs the optional audio packages for
transcription. Install the matching release's audio extras when building your
server image; keep its existing CUDA/PyTorch dependencies pinned. A running
metadata endpoint alone does not establish that audio decoding is installed.

The Docker examples select the V1 runner and synchronous scheduling for WSL2
compatibility. Keep these options in the server launch configuration.

## Transcription

Microphone requests use completed `POST /audio/transcriptions` JSON. Files can
use completed JSON or vLLM's server-sent transcription chunks. The file is
uploaded once; streaming describes the arriving result, not realtime microphone
transcription.

Language, context (`prompt`), and optional temperature are supported request
fields. v0.28.0's Whisper and Qwen3-ASR implementations consume the context;
model-specific interpretation and language support still vary. Dedicated
hotwords are not available with this profile. Sampling, VAD, translation,
timestamp, and diarization options are not exposed.

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

## Server version

This profile uses the APIs in **vLLM v0.28.0**:

- [Transcription request and response schemas](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/transcription/protocol.py).
- [Speech stream implementation](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/base/serving.py), including per-chunk finish reasons and whole-file completion.
- [Chat request mapping](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/openai/chat_completion/protocol.py), which maps reasoning effort `none` to template thinking disabled.

[vLLM-Omni](../vllm-omni/) handles speech generation through a separate backend
profile. Realtime microphone transcription requires the explicit Qwen3-ASR or
Voxtral Mini Realtime model profile. Model management remains outside Freehand.

## Language selection

The [language guide](../../guides/languages/) explains Server default, Automatic
detection, named language codes, and custom values. The chosen model determines
which languages work; profile availability is not a multilingual guarantee.
S1-mini cleanup is English only. It keeps raw text when a non-English input
language is selected or reported, and assumes English when no language is known.
