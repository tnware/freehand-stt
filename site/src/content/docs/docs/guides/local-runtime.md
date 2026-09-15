---
title: Local runtimes
description: Set up local transcription and cleanup with NeMo, whisper.cpp, and llama.cpp on Windows and macOS.
---

Freehand can install local runtimes for speech recognition and transcript cleanup.
Choose managed setup to let Freehand install the runtime, download a supported
model when you ask, and start or stop it. You do not need to enter a server URL
or API key. For live dictation, start with NeMo and its recommended Nemotron model.

## Choose a runtime and model

| Runtime | Managed models and tasks | Supported computers |
| --- | --- | --- |
| NeMo-Speech.cpp | [Nemotron 3.5 Streaming](../../models/nemotron/) (recommended): live and completed transcription. [Parakeet TDT v3](../../models/parakeet/): completed transcription only. | Windows 11 x64; macOS 13+ on Apple Silicon or Intel |
| whisper.cpp | Whisper Base (recommended), Small, or Medium: completed transcription only. | Windows 11 x64 only |
| llama.cpp | [S1-mini by Superwhisper](../../models/s1-mini/) v1 Q4_K_M: English transcript cleanup only. | Windows 11 x64; macOS 13.3+ on Apple Silicon or Intel |

Runtime binaries and models are **not bundled with Freehand**. Installation and
model downloads are separate, explicit actions; browsing the catalog downloads
nothing. Managed setup supports only the models above, not arbitrary Whisper
checkpoints or llama.cpp models. There is no managed text-to-speech runtime.

You can instead [configure a service manually](../connect-a-server/) on this
computer, your network, or a hosted provider. Freehand connects to that service
but does not install or manage it. Each task selects its own Connection; using
a managed runtime leaves your manual connections intact and never falls back
to them after a local failure. Different runtimes can run together, with one
installation and one selected model at a time per runtime.

Managed whisper.cpp is unavailable on macOS because upstream does not publish
a macOS server executable. Use NeMo for managed transcription or a
[manual whisper.cpp connection](../../backends/whisper-cpp/).

## Before installing

- Use Windows 11 x64 or macOS 13 or later on Apple Silicon or Intel. Managed
  llama.cpp requires macOS 13.3 or later. NeMo selects Metal on Apple Silicon,
  CPU on Intel Macs, and a compatible backend on Windows. For llama.cpp and
  Windows whisper.cpp, review the recommended binary before installation.
  Performance depends on your computer and selected model.
- Allow internet access for the runtime and explicit model downloads. Model
  weights use additional disk space; the catalog shows download sizes when the
  runtime provides them.
- NeMo's model manager uses the `curl` executable supplied with Windows or macOS.
  Freehand does not install Python, Docker, WSL, or a development toolchain.

Runtime binaries and downloaded weights stay under Freehand's per-user local
application-data directory. They are not installed globally or added to PATH.

### Download sources

Runtime details show the source of the binary and each model. Review these
before downloading; viewing source information does not download or load a
model. Source links open in your browser.

- **NeMo-Speech.cpp:** official releases from `NVIDIA/NeMo-Speech.cpp` on GitHub.
  Model downloads use NeMo's built-in model manager and its model index.
- **llama.cpp:** official releases from `ggml-org/llama.cpp` on GitHub.
  Freehand downloads S1-mini directly from `superwhisper/s1-mini-GGUF` on
  Hugging Face, not through a llama.cpp model downloader.
- **whisper.cpp:** official releases from `ggml-org/whisper.cpp` on GitHub.
  Freehand downloads the qualified Whisper models directly from
  `ggerganov/whisper.cpp` on Hugging Face.

Freehand uses specific versions rather than fetching the latest release
automatically. Expand **Binary download details** or **Model download details**
to see filenames and checksums. NeMo model details come from its bundled model
index; NeMo's model manager handles the download.

## Set up local transcription

1. Open **Local runtime** from the activity rail on the left.
2. Find **NeMo-Speech.cpp** in the runtime list and choose **Install**. Keep the
   recommended Nemotron 3.5 Streaming model. Installation does not download it.
3. Select the runtime in the inventory sidebar to view its setup controls,
   progress, and cancellation action.
4. When installation finishes, the action changes to **Download**.
   Choose it to download Nemotron, or use the catalog below to choose an alternative.
   Browsing the catalog does not download or load any model.
5. After downloading, choose **Start** in the runtime header. Wait for **Running**.
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

Select a runtime in the inventory sidebar. Its detail pane shows state, setup
actions, and a model table with download and selection controls. Open
**Runtime preferences** for startup and binary options, or **Manage runtime**
for removal and recovery actions. These actions never select a different
Connection for your tasks.

Choose a model from the runtime's catalog for the task you need.

### whisper.cpp transcription

On Windows, choose **Install** on the whisper.cpp row, review and confirm the binary choice,
download a model from its catalog, and choose **Start**. Select its
built-in Connection for Voice or audio files. Voice uses completed transcription, not
realtime; file response streaming is also unavailable. CPU and NVIDIA CUDA
execution are available. Larger models need more memory and take longer to process.

### Local cleanup with S1-mini

Choose **Install** for **llama.cpp**, review and confirm the binary choice,
download **S1-mini by Superwhisper**, and start the runtime.
Select its built-in Connection in **Cleanup**, and enable cleanup.
NeMo can continue handling Voice while llama.cpp cleans up its completed text.
Choose CPU if you want to leave GPU memory for NeMo. A GPU recommendation
checks compatibility, not whether your GPU has room for both models.

S1-mini is English-only and runs with reasoning disabled. If the input language
is unknown, Freehand assumes English for cleanup; explicitly non-English input
skips S1-mini. Failed cleanup keeps the raw transcript. The model is not a
general chat assistant or a speech-synthesis model. See
[transcript cleanup](../post-processing/) for its style and structure controls.

### Use Apple GPU acceleration

On Apple Silicon, NeMo installs its Metal package automatically. For a new
llama.cpp installation, **Auto (recommended)** selects **Apple GPU (Metal)**.
Choose **CPU** to disable GPU offload, then **Download and install**.
Intel Macs use CPU; CUDA is not offered on macOS.

To change llama.cpp later, stop it and choose **CPU** or **Apple GPU (Metal)**
under **Runtime binary**, then start it again. Switching preserves downloaded
models, saved Connections, and startup preferences. Metal shares your Mac's
memory with NeMo and other apps; a recommendation does not reserve memory.

### Use NVIDIA GPU acceleration

On Windows, for a new llama.cpp or whisper.cpp installation, **Install** first shows a
binary recommendation and its reason without downloading files. Keep
**Auto (recommended)** or explicitly choose **CPU** or **NVIDIA GPU (CUDA)**,
then choose **Download and install**. Unknown or unsupported NVIDIA hardware or
driver information produces a CPU recommendation. The choice is between Freehand’s
pinned CPU and CUDA 12.4 packages, not an upgrade to the latest upstream release.

Existing installations do not change automatically. Recommendations do not
reserve GPU memory, stop other runtimes, or change as free GPU memory fluctuates.

For llama.cpp or whisper.cpp, select it in the runtime sidebar. If it is
running, choose **Stop**. Under **Runtime binary**, choose
**NVIDIA GPU (CUDA)** and wait for installation to finish, then choose
**Start**. Runtime management, quick settings, and Connection details
show the installed backend separately from the Connection name. Changing the
backend does not rename custom Connections or alter task selections.

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
also show **Downloaded** beside their size.

Downloads can be cancelled and retried. Stop active transcription before
switching or removing the loaded model. In task quick settings, choose the
**Connection** first, then choose one of that runtime's downloaded models under
**Selected model**. The runtime shares its selected model with every task using
it. **Manage runtime** opens installation, downloads, and runtime details.
Removing a downloaded model frees its
managed cache data; using it again requires another download.

## Startup and process output

Starting verifies the installed files, launches the selected runtime, and waits
for the selected model to be ready. Status shows the current phase and its
elapsed time rather than an estimated percentage. GPU startup includes warm-up;
loading and warm-up may appear as one phase when the runtime cannot report them
separately. Wait for **Running** before using the runtime.

Warm-up uses only the selected installed model. For CUDA whisper.cpp, Freehand
sends one second of synthetic silence to the local runtime after it is ready
and discards the response. It never records your microphone, warms other catalog
models, or sends the warm-up to a remote server. GPU llama.cpp and NeMo use their
built-in startup warm-up. This also applies when **Start when Freehand launches**
is enabled; browsing models and connection checks remain metadata-only.

If startup fails or takes too long, use **Cancel**, check resources, and retry.
You can inspect recent process output while startup is still in progress:

1. Choose **View output** in Local runtime or the task’s runtime quick controls.
2. The separate **Process output** window opens with a blank viewer and disabled
   output controls. Read its warning banner and choose **Show output** only if
   displaying it on your screen is safe.
3. Use **Search** to find text, **Follow** to follow new output, or **Clear** to
   discard the captured output. Collection continues when Follow is off.
4. Select text and choose **Copy selection** if you want it on your clipboard.
   Copied text can remain there after the viewer closes.

The read-only viewer supports colors and in-place progress updates when the
runtime emits them. It does not accept commands or save log files. Closing it
does not stop the runtime. Each opening requires consent again; closing or
switching runtimes clears the displayed text and revokes access, but the private
bounded tail remains until cleared, the next start attempt, runtime removal, or
Quit. Older output is discarded as the buffer fills. llama.cpp captures normal
informational, warning, and error output without debug logging. This can still
include prompts or other sensitive text. Output can be sparse or absent; an
empty viewer is not proof of a failed start.
See [diagnostics privacy](../privacy-and-safety/#diagnostics) before displaying
output during screen sharing.

## Stop, disable, or remove

Stopping releases the running server; starting it again reloads the selected
model. **Start when Freehand launches**, under **Runtime preferences**, enables
startup for an installed runtime without downloading missing files.
Quitting Freehand stops the processes it owns, including active downloads.

To switch back to a manual service, select its saved Connection in the task's
connection picker. Manual connections retain their URLs, models, and API keys.
Stopping a runtime does not change any task's selection. A local runtime failure
does not automatically send audio to another server.

The recording controls remain available while the runtime is stopped or starting.
The recording area shows its current availability; use transcription quick settings
to start it. An attempt to record before it is ready reports the failure in Freehand
and through the status overlay when error feedback is enabled. Start the runtime,
wait for **Ready**, then try recording again. Previous results remain available to copy.

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
startup fails, retry after freeing computer resources or return explicitly to a manual
endpoint. Freehand does not replay a failed realtime recording; start a new one.

Advanced backend flags, custom model files, alternative runtime versions, and
LAN serving belong in a separately managed server. Follow the
[backend guides](../../backends/) for those deployments.
