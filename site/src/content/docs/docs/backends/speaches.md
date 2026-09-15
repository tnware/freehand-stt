---
title: Speaches
description: Connect Speaches for transcription and on-demand speech playback.
---

Use **Speaches** for microphone and audio-file transcription, or for text to
speech when the server has a TTS model installed.

## Run Speaches with Docker

Use PowerShell and Docker Desktop with Linux containers. This NVIDIA example
follows [upstream installation](https://speaches.ai/installation/). The
`latest-cuda` tag changes; pin a release tag or image digest for a repeatable
deployment. For CPU-only setup, use the alternative
below instead of starting a second server.

```powershell
docker run --detach --name freehand-speaches `
  --gpus device=0 `
  --publish 127.0.0.1:8000:8000 `
  --volume freehand-speech-models:/home/ubuntu/.cache/huggingface/hub `
  ghcr.io/speaches-ai/speaches:latest-cuda
```

Wait for startup, then explicitly download one model suitable for the GPU's
available memory. This example selects full Whisper large-v3:

```powershell
docker logs --tail 30 freehand-speaches
Invoke-RestMethod -Method Post `
  -Uri 'http://127.0.0.1:8000/v1/models/Systran/faster-whisper-large-v3'
Invoke-RestMethod http://127.0.0.1:8000/v1/models
```

Model installation is an administrator action in this recipe. Freehand's
connection test never calls that POST or loads the model inventory. See the
[upstream model download instructions](https://speaches.ai/usage/model-discovery/)
for the models supported by your running release.

Set Freehand's transcription profile to **Speaches**, base URL to
**`http://127.0.0.1:8000/v1`**, model to **`Systran/faster-whisper-large-v3`**,
authentication to **None**, and enable **Allow HTTP for this connection**.
Save, then try a short recording.

### CPU alternative

Without an NVIDIA GPU, use this launch command instead:

```powershell
docker run --detach --name freehand-speaches `
  --publish 127.0.0.1:8000:8000 `
  --volume freehand-speech-models:/home/ubuntu/.cache/huggingface/hub `
  ghcr.io/speaches-ai/speaches:latest-cpu
```

Wait for startup in `docker logs --tail 30 freehand-speaches`, then download
the selected model:

```powershell
Invoke-RestMethod -Method Post `
  -Uri 'http://127.0.0.1:8000/v1/models/Systran/faster-distil-whisper-small.en'
```

Set the Freehand model to
`Systran/faster-distil-whisper-small.en` and the spoken language to English; the
remaining connection values stay the same. Compare results with a larger model
if you have the memory and processing capacity for it. These upstream
launch recipes have not been separately tested on Windows with every image.

### Stop, restart, and optional playback

```powershell
docker stop freehand-speaches
docker start freehand-speaches
```

The named volume retains downloaded models. Stop and remove this named
container before recreating it to change between CPU and CUDA images.
For speech playback, complete the separate
[upstream TTS installation](https://speaches.ai/usage/text-to-speech/), install a
selected TTS model, then configure Freehand's speech playback connection below.
Installing an STT model alone does not provision voices or a TTS model.

## Configure Freehand

Under **Connections**, create a **Speaches** connection and enable
**Voice transcription**, **Audio-file transcription**, or both under **Used for**.
Enter the base URL including `/v1`, authentication, and HTTP permission.
Choose **Save connection**, then open the **Settings** cog in Voice transcription
or Audio file. Select it under **Transcription**, choose an installed model, and save your changes.

For playback on the same server, edit that connection and enable **Text to speech**
under **Used for**. Save it, then select the same entry in **Text to speech → Speech**,
choose the installed TTS model and voice, and enable text to speech. Preview the
voice and save your changes. Both features share the connection
and key while retaining separate models and options. Create another connection
when the playback endpoint or credentials differ.

The [connection guide](../../guides/connect-a-server/) explains deployment
topologies and shared settings. Freehand's connection checks read metadata only;
they do not install, load, or run models.

## Implemented capabilities

| Capability               | Scope                                                                                               |
| ------------------------ | --------------------------------------------------------------------------------------------------- |
| Microphone transcription | Completed JSON transcription, including local checkpoint requests.                                  |
| Stored audio files       | Completed JSON or optional streamed transcript results.                                             |
| Streaming dialects       | Typed transcript delta/done events and legacy untyped text segments.                                |
| Language hint            | Optional `language` request field; effect depends on the model.                                     |
| Recognition context      | Optional `prompt`, at most 8,192 UTF-8 bytes.                                                       |
| Shared vocabulary        | Sent as `hotwords`, at most 2,048 UTF-8 bytes; Speaches-specific field.                             |
| Decoding temperature     | Optional `temperature` from 0 to 1; explicit zero is supported.                                     |
| Speech playback          | Voice ID, speed request, and buffered PCM16 WAV.                                                    |
| Voice discovery          | Model-associated voices from `/v1/models`, with a labelled server-wide `/v1/audio/voices` fallback. |
| Transcript cleanup       | Configure a separate Generic, llama.cpp, or vLLM chat connection.                                   |

Context, vocabulary, and temperature are optional. Freehand omits their request
fields until you configure or enable them.

## Streaming formats

Freehand supports the segment streams used by Speaches v0.8.3 and the typed
text events used by its v0.9.0-rc.3 Whisper executor. Speech playback buffers
PCM16 WAV before playing.

## Choose a voice

In **Text to speech → Speech**, select the Speaches connection and TTS model,
then use **Refresh voices** beside the voice field. Search the list by ID, name,
or language when the server supplies it. You can also type a custom voice ID.
The voice selection is remembered for this connection and model.

Freehand first reads `/v1/models` and uses the selected model's `voices` field.
If it is absent, it reads `/v1/audio/voices` and labels the result as server-wide;
those voices are not guaranteed to work with every model. An explicitly empty
model voice list remains empty. Discovery does not synthesize previews or load
models. Changing connection or model hides results from a different selection.
Errors and older servers leave manual voice entry available.

## Current limits

The profile does not add timestamps, a translation workflow, server VAD
controls, voice instructions, or cloning inputs.
Provider limits and Freehand's bounded-buffer limits still apply; consult the
[protocol reference](../../reference/protocol/).

## Upstream references

- [Speaches repository](https://github.com/speaches-ai/speaches)
- [v0.8.3 transcription route](https://github.com/speaches-ai/speaches/blob/v0.8.3/src/speaches/routers/stt.py)
- [v0.9.0-rc.3 Whisper executor](https://github.com/speaches-ai/speaches/blob/v0.9.0-rc.3/src/speaches/executors/whisper.py)
- [v0.9.0-rc.3 speech route](https://github.com/speaches-ai/speaches/blob/v0.9.0-rc.3/src/speaches/routers/speech.py)

## Recognition controls

Whisper-family models support context hints, hotwords, and temperature through
Speaches. These controls are available for completed and streaming transcription.

For Voice, set **Context hint** in **Voice transcription → Transcription**. For files,
set context in **Audio file → Transcription → Transcription controls**. Keep shared
terms in **Settings → Vocabulary**, and enable them for Voice, audio files, or
both. Freehand sends those terms as `hotwords`.

Context supplies expected subject matter or wording. Hotwords supply terms to
favor; neither is a strict replacement dictionary. Both may be supplied, but
model prompt budgets and decoding behavior can limit their effect. Freehand's
byte limits bound requests; they are not model token allowances. Temperature
is a decoding request, not a guaranteed quality or determinism control.

Keep these options unset if the selected model does not support them. A rejected
request fails normally; Freehand does not silently drop hints and repeat inference.
See [Transcription controls](../../guides/connect-a-server/#transcription-controls).

Freehand does not offer a server-side VAD control: v0.8.3 accepts `vad_filter`,
while the v0.9.0-rc.3 route runs VAD internally with fixed options. Local
microphone VAD settings remain independent. Library settings such as beam size
are not automatically fields on the Speaches HTTP request.

## Language selection

The [language guide](../../guides/languages/) explains Server default, Automatic
detection, named language codes, and custom values. The chosen model determines
which languages work; profile availability is not a multilingual guarantee.
S1-mini cleanup is English only. It keeps raw text when a non-English input
language is selected or reported, and assumes English when no language is known.
