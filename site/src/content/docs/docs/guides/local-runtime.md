---
title: Local speech runtime
description: Install a managed Windows speech runtime and use Nemotron live transcription on this PC.
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
2. Choose **Add runtime**, give it a name, and keep the recommended Nemotron 3.5
   Streaming model. Adding it does not download or start anything.
3. Choose **Install runtime** at the top of the page. Progress and cancellation
   stay in that same area.
4. When installation finishes, the action changes to **Download selected model**.
   Choose it to download Nemotron, or use the catalog below to choose an alternative.
   Browsing the catalog does not download or load any model.
5. After downloading, choose **Start runtime** in the same area. Wait for **Running**.
6. Open **Manage connections**, add a managed Connection referencing this runtime,
   and select it for Voice. Complete the microphone and recording setup.
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

The catalog contains models from NeMo's installed index that match Freehand's
supported speech profiles. It is not a general Hugging Face browser. A model
must be downloaded before it can run. Realtime is available only for a qualified
streaming model; a completed-only selection cannot enable live mode.

Downloads can be cancelled and retried. Stop active transcription before
switching or removing the loaded model. Removing a downloaded model frees its
managed cache data; using it again requires another download.

## Stop, disable, or remove

Stopping releases the running server; starting it again reloads the selected
model. **Start when Freehand launches**, under **Instance preferences**, enables
startup for an installed instance without downloading missing files.
Quitting Freehand stops the processes it owns, including active downloads.

To switch back to a manual service, select its saved Connection in the task's
connection picker. Manual connections retain their URLs, models, and API keys.
Stopping a runtime does not change any task's selection. A local runtime failure
does not automatically send audio to another server.

**Remove runtime files**, under **Instance preferences**, deletes that instance's
managed binaries and model data after
confirmation. It does not remove manual connections, their credentials, source
audio files, or models installed by another application.

File removal leaves its Connections selected but unavailable until repaired;
it cannot silently switch transcription to a saved server. **Delete instance**
removes the inventory entry only, not downloaded files. Reassign or remove its
Connections first. Other runtime instances remain untouched.

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
