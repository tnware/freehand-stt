---
title: Local runtimes
description: Set up local transcription and cleanup with NeMo, whisper.cpp, and llama.cpp on Windows.
---

On Windows, Freehand can install and manage NeMo-Speech.cpp for local Voice and
audio-file transcription. The recommended model is **Nemotron 3.5 ASR streaming**
with realtime enabled: words appear while you speak, and the final transcript
uses the same cleanup and safe insertion rules as other dictation.

You can still connect to your own local, network, or hosted service. Each task
selects a Connection; a managed Connection uses a runtime installed by Freehand.
Your manual connections remain intact. macOS uses
[manual connections](../connect-a-server/); managed installation is Windows x64
only.

## Before installing

- Use Windows 11 x64. NeMo selects its backend automatically. llama.cpp and
  whisper.cpp initially install CPU binaries, with an explicit NVIDIA CUDA
  option. Performance depends on the selected model and PC.
- Allow internet access for the runtime and explicit model downloads. Model
  weights use additional disk space; the catalog shows download sizes when the
  runtime provides them.
- NeMo's model manager needs the `curl` executable supplied with current Windows.
  Freehand does not install Python, Docker, WSL, or a development toolchain.

Runtime binaries and downloaded weights stay under Freehand's per-user local
application-data directory. They are not installed globally or added to PATH.

## Set up local transcription

1. Open **Settings → Local runtime**, under **Connections & vocabulary**.
2. Find **NeMo-Speech.cpp** in the runtime list and choose **Install**. Keep the
   recommended Nemotron 3.5 Streaming model. Installation does not download it.
3. The row opens its setup controls. Progress and cancellation stay in that area;
   use the chevron beside the runtime to collapse or reopen its details.
4. When installation finishes, the action changes to **Download selected model**.
   Choose it to download Nemotron, or use the catalog below to choose an alternative.
   Browsing the catalog does not download or load any model.
5. After downloading, choose **Start runtime** in the same area. Wait for **Running**.
6. In Voice's connection picker, select the built-in **NeMo-Speech.cpp** Connection.
   It appears automatically; no URL, API key, or additional connection setup is needed.
   Complete the microphone and recording setup.
7. In Voice's transcription settings, enable **Realtime transcription** for the
   recommended live-preview workflow. Captions and language remain Voice options.

Start recording with the intended destination focused. The live preview can
change; only finalized text can be inserted, copied, or kept in optional
history. [Live transcription](../live-transcription/) explains captions,
stop/cancel behavior, and the difference from completed recording.

Audio-file transcription selects its own Connection. Choose the same managed
Connection to share this runtime, or keep a different connection. Files use
completed requests rather than a live microphone stream. Turning realtime off
in Voice keeps its connection and model selected and uses completed recording.

## Choose another model

With details collapsed, each runtime row shows its status and selected model.
Use **Download**, **Start**, **Stop**, or **Cancel** directly from that row; download
progress and its result remain visible there. The chevron opens the full setup,
model catalog, and runtime preferences. These actions never select a different
Connection for your tasks.

The runtime list also offers **whisper.cpp** for completed transcription and
**llama.cpp** for local cleanup. Each runtime is installed once and runs one
selected model at a time. Different runtimes can run together.

### whisper.cpp transcription

Choose **Install** on the whisper.cpp row, download a model from its catalog,
and choose **Start runtime**. Select its built-in Connection for Voice or audio
files. Voice uses completed transcription, not
realtime; file response streaming is also unavailable. CPU and NVIDIA CUDA
execution are available. Larger models need more memory and take longer to process.

### Local cleanup with S1-mini

Install **llama.cpp**, download **S1-mini by Superwhisper**, and start the runtime.
Select its built-in Connection in **Cleanup**, and enable cleanup.
NeMo can continue handling Voice while llama.cpp cleans up its completed text.
CPU is the default, leaving NeMo's GPU allocation alone. You can explicitly
switch llama.cpp to NVIDIA CUDA when your GPU has room for both models.

S1-mini is English-only and runs with reasoning disabled. If the input language
is unknown, Freehand assumes English for cleanup; explicitly non-English input
skips S1-mini. Failed cleanup keeps the raw transcript. The model is not a
general chat assistant or a speech-synthesis model. See
[transcript cleanup](../post-processing/) for its style and structure controls.

### Use NVIDIA GPU acceleration

For llama.cpp or whisper.cpp, open its details using the chevron. If it is
running, choose **Stop runtime**. Under **Runtime binary**, choose
**NVIDIA GPU (CUDA)** and wait for installation to finish, then choose
**Start runtime**. Runtime management, quick settings, and Connection details
show the installed backend separately from the Connection name. Built-in
llama.cpp and whisper.cpp Connections omit the old default **(CPU)** name suffix;
changing the backend does not rename custom Connections or alter task selections.

CUDA 12.4 requires a compatible NVIDIA GPU with compute capability 5.0 or newer
and driver 551.78 or newer; newer GPUs also need a driver that supports them.
Freehand downloads the runtime's required CUDA libraries, not a developer
toolkit or driver. AMD and Intel GPU acceleration are not offered by these
managed adapters.

Switching binaries keeps your downloaded models, selected model, built-in
Connections, and startup preference. A cancelled or failed download leaves the
previous installation in place. To return to CPU, stop the runtime and choose
**CPU** in the same controls. Do not use **Remove runtime files** to switch
backends: that action also deletes downloaded models.

GPU memory is shared with NeMo and other applications. If a GPU start fails,
check your NVIDIA driver and available memory, or switch back to CPU. The CUDA
label identifies the installed binary; it does not measure how much of a model
is currently running on the GPU. Backend changes never select a remote server.

### NeMo models

The catalog contains models from NeMo's installed index that match Freehand's
supported speech profiles. It is not a general Hugging Face browser. A model
must be downloaded before it can run. Realtime is available only for a qualified
streaming model; a completed-only selection cannot enable live mode.

Downloads show transferred bytes and a percentage while the total is known,
including NeMo downloads. Preparation and verification use an activity indicator.
A full transfer is not finished until verification succeeds. The setup area and
the model's catalog row show progress and offer cancellation. The row stays in
place with a completion, cancellation, or failure message. Successful downloads
also receive a **Downloaded** badge.

Downloads can be cancelled and retried. Stop active transcription before
switching or removing the loaded model. In task quick settings, choose the
**Connection** first, then choose one of that runtime's downloaded models under
**Selected model**. The runtime shares its selected model with every task using
it. **Manage runtime** opens installation, downloads, and runtime details.
Removing a downloaded model frees its
managed cache data; using it again requires another download.

## Stop, disable, or remove

Stopping releases the running server; starting it again reloads the selected
model. **Start when Freehand launches**, under **Runtime preferences**, enables
startup for an installed runtime without downloading missing files.
Quitting Freehand stops the processes it owns, including active downloads.

To switch back to a manual service, select its saved Connection in the task's
connection picker. Manual connections retain their URLs, models, and API keys.
Stopping a runtime does not change any task's selection. A local runtime failure
does not automatically send audio to another server.

**Remove runtime files**, under **Runtime preferences**, deletes that runtime's
managed binaries and model data after
confirmation. It does not remove manual connections, their credentials, source
audio files, or models installed by another application.

File removal leaves its Connections selected but unavailable until repaired;
it cannot silently switch transcription to a saved server. Other runtimes remain
untouched.

If an earlier setup left duplicate installations, **Manage** exposes each one
for review. Keep one, remove unwanted runtime files, and reassign or remove its
Connections before choosing **Delete duplicate entry**. Deleting the entry alone
does not remove downloaded files. Freehand does not choose which copy to keep
or run two copies of the same runtime.

## Privacy and recovery

Local recognition does not make optional cleanup local. If cleanup is enabled,
its independently configured server receives the transcript. TTS also keeps its
own endpoint. See [managed local recognition](../privacy-and-safety/#managed-local-recognition).

NeMo creates temporary model-download diagnostics. Freehand deletes those files
after a download or cancellation, and on the next runtime inspection after an
abrupt exit. It does not show or copy their contents into application logs.

If installation or download fails, read the reported stage, check disk space
and network access, then retry. An integrity failure must not be bypassed. If
startup fails, retry after freeing PC resources or return explicitly to a manual
endpoint. Freehand does not replay a failed realtime recording; start a new one.

Advanced backend flags, custom model files, alternative runtime versions, and
LAN serving belong in a separately managed server. Use a
[manual NeMo connection](../../backends/nemo-speech/) for those deployments.
