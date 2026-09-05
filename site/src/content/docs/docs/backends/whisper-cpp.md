---
title: whisper.cpp
description: Connect the native whisper.cpp HTTP server for microphone and file transcription.
---

The **whisper.cpp** transcription profile supports completed microphone and
file uploads to the native HTTP server. Model loading and acceleration belong
to your server. Freehand never calls its model-loading route.

## Choose a deployment

whisper.cpp runs natively on Windows. Docker is optional. Both expose the same
HTTP contract to Freehand, so choose the deployment that suits your server.
The instructions below use PowerShell and the same model folder and local port.

## Download a model

This example uses full `large-v3` for an accuracy-oriented test. Its download is
about 3 GB. Upstream lists approximately 3.9 GB model memory for large; actual
GPU use also depends on runtime buffers and the workload. Choose a smaller
checkpoint when the available memory or desired latency calls for it. See the
[upstream model and memory table](https://github.com/ggml-org/whisper.cpp#memory-usage).

```powershell
New-Item -ItemType Directory -Force freehand-whisper-models | Out-Null
$modelDirectory = (Resolve-Path freehand-whisper-models).Path
curl.exe --fail --location `
  https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin `
  --output "$modelDirectory/ggml-large-v3.bin"
if ($LASTEXITCODE -ne 0) { throw 'Model download failed.' }
```

If you already have this GGML model, set `$modelDirectory` to its existing
folder instead. You do not need to download it again for a different server
executable or container.

## Run natively on Windows

If you already have `whisper-server.exe`, use that installation. Otherwise,
download and extract an appropriate x64 package from the
[official releases](https://github.com/ggml-org/whisper.cpp/releases). Keep the
executable with its DLLs. The inspected **b4938** Windows archive contains
`Release/whisper-server.exe`; run from that extracted `Release` folder.

- `whisper-bin-x64.zip` provides the CPU build.
- Packages named `whisper-cublas-...-bin-x64.zip` provide CUDA builds; select one
  compatible with your NVIDIA GPU and driver.

A CUDA package label alone does not establish support for every GPU generation.
The inspected release's CUDA archives are labeled 11.8 and 12.4; for an RTX 50
series system, use a build known to support that GPU or build with a suitable
current toolkit as described below. The inspected GGML build configuration
identifies CUDA 12.8 or later for native Blackwell architecture support.

In the PowerShell session with `$modelDirectory` set, change to the folder
containing your executable and launch:

```powershell
.\whisper-server.exe `
  -m "$modelDirectory/ggml-large-v3.bin" `
  --host 127.0.0.1 --port 8051 -t 4
```

Leave that terminal running. In another PowerShell window, check readiness:

```powershell
Invoke-RestMethod http://127.0.0.1:8051/health
```

Check startup output for the selected backend: a GPU-capable build should
identify CUDA and the chosen device. A successful health check alone does not
prove GPU offload. In Freehand, choose **whisper.cpp**, base URL
**`http://127.0.0.1:8051`**, authentication **None**, and allow local HTTP.
The model is **Server-loaded**. Save, then test a short recording.

Press **Ctrl+C** in the server terminal to stop. Run the same command to
restart. To change models, restart with a different `-m` path; Freehand does
not load or switch server models. If another server already occupies 8051,
stop it or use a different port and update Freehand's base URL to match.

### Build a native CUDA server when needed

Use Git, CMake 3.24 or later, Visual Studio C++ build tools, and an NVIDIA CUDA
toolkit compatible with your GPU and compiler. For native RTX 50-series
architecture support, use CUDA 12.8 or later. From a developer PowerShell
configured for your C++ toolchain:

```powershell
git clone https://github.com/ggml-org/whisper.cpp.git
Set-Location whisper.cpp
cmake -B build -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=native
cmake --build build --config Release --target whisper-server --parallel
```

With the Visual Studio generator, the executable is normally under
`build/bin/Release`. Keep its generated DLLs alongside it. Follow the
[upstream CUDA build instructions](https://github.com/ggml-org/whisper.cpp#nvidia-gpu-support)
and [GPU architecture configuration](https://github.com/ggml-org/whisper.cpp/blob/master/ggml/src/ggml-cuda/CMakeLists.txt)
for your toolchain. Package contents and source instructions were checked;
this native installation recipe has not received separate live acceptance.

## Run whisper.cpp with Docker

This alternative uses Docker Desktop's WSL2 Linux backend and an NVIDIA GPU.
Use the model directory from [Download a model](#download-a-model). The pinned
CUDA image below is the one recorded in [compatibility evidence](#contract-and-evidence).
It runs the same HTTP server inside a container.

Start the server:

```powershell
$whisperImage = 'ghcr.io/ggml-org/whisper.cpp@sha256:2285844e0c38744d90eed59ce5b90fe68cd2dfc6ecb07bb0b68b8ff800528be4'
docker run --detach --name freehand-whisper `
  --gpus device=0 `
  --publish 127.0.0.1:8051:8080 `
  --mount "type=bind,source=$modelDirectory,target=/models,readonly" `
  --entrypoint /app/build/bin/whisper-server `
  $whisperImage `
  -m /models/ggml-large-v3.bin --host 0.0.0.0 --port 8080 -t 4
```

Inspect startup, then check readiness without transcribing anything:

```powershell
docker logs --tail 30 freehand-whisper
Invoke-RestMethod http://127.0.0.1:8051/health
```

Connect with profile **whisper.cpp**, base URL **`http://127.0.0.1:8051`**,
and authentication **None**. Allow local HTTP. The model is **Server-loaded**;
do not enter a Hugging Face model ID. For an English test, choose English in
Freehand's existing Language setting. Save, then try your own short dictation.
A healthy server is not yet proof of recognition quality.

```powershell
docker stop freehand-whisper
docker start freehand-whisper
```

To change the model, download its whisper.cpp GGML file, stop and remove this
container with `docker rm freehand-whisper`, and rerun the launch command with
the new `-m` filename. The host model folder is retained. Freehand's connection
settings stay the same. Refer to [upstream server setup](https://github.com/ggml-org/whisper.cpp/tree/master/examples/server)
for native builds, other accelerators, and optional format conversion.

## Connect

1. Start `whisper-server` with the model you want to use.
2. In **Settings → Server**, select **whisper.cpp**.
3. Set the Base URL to the server root, for example `http://127.0.0.1:8081`.
   Omit `/v1` and `/inference`. A reverse-proxy prefix can be included.
4. Choose the authentication mode required by your deployment and permit HTTP
   only where appropriate. The native server may need a proxy for authentication.
5. Run **Test**, then save. The model field shows **Server-loaded model**;
   no client model ID is required or sent.

The default test reads `/health` beneath that root/prefix. An explicit custom
health path overrides the default. A successful health check establishes server
availability, not model identity or transcription quality. Freehand does not
invent a model inventory. A model ID retained from another profile remains saved
but is ignored by this adapter.

The default `/inference` route is qualified. If your server changes
`--inference-path`, expose the default route through a proxy. The base URL is a
root/prefix setting, not a full request URL.

## Supported controls and files

- Language, transcription context (`prompt`), and optional temperature use the
  server's multipart fields. Blank hints and disabled temperature are omitted,
  preserving server defaults. Model/language compatibility remains server-owned.
- Dedicated `hotwords` are unavailable. You can supply context through `prompt`.
- File transcription uses completed JSON. The streaming control and retry are
  unavailable for this profile; a stored streaming preference cannot cause a
  failed first upload or automatic resubmission.
- Microphone audio is Freehand's normalized WAV. WAV is the conservative file
  baseline; other formats depend on the server's decoder build or its optional
  FFmpeg conversion. Freehand does not transcode selected files for this adapter.
- Both workflows retain their existing size limits, timeouts, cancellation,
  optional cleanup, history policy, and delivery behavior.

## Contract and evidence

Requests are multipart `POST /inference`, including `file` and
`response_format=json`, without `model` or `stream`. Responses require a JSON
object with a string `text`. Non-success statuses and malformed responses fail
without replay. The configured credential and permitted custom headers apply
as they do for other transcription profiles.

Source qualification pins whisper.cpp
[`52a939a2a762`](https://github.com/ggml-org/whisper.cpp/blob/52a939a2a762224e255d366c1182b2af4dd1a032/examples/server/server.cpp).
Client fixtures cover prefixed routing, health defaults/overrides, omitted
model fields, bounded multipart upload, hints, and completed-only behavior.
This is not a claim that every build, audio format, or model has been tested
interactively on Windows.


### Scoped live acceptance — 2026-09-05

Freehand's Windows Go adapters completed both microphone-request and file-upload
paths against the existing CUDA image (`sha256:2c42506808d7546ea3440c0053dd6543373cc4252c525b7972ab96554a533837`)
using `ggml-tiny.en.bin` and the public 11-second whisper.cpp JFK sample. The
server's `/health` probe succeeded. This exercises HTTP/inference behavior from
Windows; interactive capture, focus-safe insertion, and every file format were
not part of that fixed-sample run. The source pin above records inspected
contract evidence independently of the tested image digest.
