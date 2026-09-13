---
title: Local speech runtime
description: Install a managed Windows speech runtime and use Nemotron live transcription on this PC.
---

On Windows, Freehand can install and manage NeMo-Speech.cpp for local Voice and
audio-file transcription. The recommended model is **Nemotron 3.5 ASR streaming**
with realtime enabled: words appear while you speak, and the final transcript
uses the same cleanup and safe insertion rules as other dictation.

You can still connect to your own local, network, or hosted service. Managed
mode keeps those saved connections intact. macOS uses
[manual connections](../connect-a-server/); managed installation is Windows x64
only.

## Before installing

- Use Windows 11 x64. Freehand selects a supported runtime artifact and prefers
  GPU execution where available; CPU operation remains an option through its
  automatic hardware policy. Performance depends on the selected model and PC.
- Allow internet access for the runtime and explicit model downloads. Model
  weights use additional disk space; the catalog shows download sizes when the
  runtime provides them.
- NeMo's model manager needs the `curl` executable supplied with current Windows.
  Freehand does not install Python, Docker, WSL, or a development toolchain.

Runtime binaries and downloaded weights stay under Freehand's per-user local
application-data directory. They are not installed globally or added to PATH.

## Set up local transcription

1. Open **Settings → Local runtime**.
2. Choose **Enable local transcription** to select Nemotron 3.5 with realtime on.
3. Choose **Install runtime** at the top of the page. Progress and cancellation
   stay in that same area.
4. When installation finishes, the action changes to **Download selected model**.
   Choose it to download Nemotron, or use the catalog below to choose an alternative.
   Browsing the catalog does not download or load any model.
5. After downloading, choose **Start runtime** in the same area. Wait for **Running**, then return to Voice and complete
   the microphone and recording setup.

Start recording with the intended destination focused. The live preview can
change; only finalized text can be inserted, copied, or kept in optional
history. [Live transcription](../live-transcription/) explains captions,
stop/cancel behavior, and the difference from completed recording.

Your selected managed model also handles audio-file transcription. Files use
completed requests rather than a live microphone stream. Turning realtime off
under **Runtime options** keeps the model selected and restores completed Voice behavior.

## Choose another model

The catalog contains models from NeMo's installed index that match Freehand's
supported speech profiles. It is not a general Hugging Face browser. A model
must be downloaded before it can run. Realtime is available only for a qualified
streaming model; a completed-only selection cannot enable live mode.

Downloads can be cancelled and retried. Stop active transcription before
switching or removing the loaded model. Removing a downloaded model frees its
managed cache data; using it again requires another download.

## Stop, disable, or remove

Stopping releases the running server; starting it again reloads the selected
model. An enabled, installed selection can start when Freehand launches.
Quitting Freehand stops the processes it owns, including active downloads.

Disable managed mode to return to your saved manual Voice and audio-file
connections. They retain their URLs, selected models, and stored API keys.
A local runtime failure does not automatically send audio to those servers.

**Remove runtime**, under **Runtime options**, deletes Freehand's managed binaries and model data after
confirmation. It does not remove manual connections, their credentials, source
audio files, or models installed by another application.

Removal leaves managed mode selected, so it cannot silently switch transcription
to a saved server. Choose **Use my own server** separately if you want to switch.

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
