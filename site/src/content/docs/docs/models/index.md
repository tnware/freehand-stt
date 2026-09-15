---
title: Model profiles
description: Choose model behavior independently of your server connection.
---

A **connection** tells Freehand where your server is and which backend API it
uses. A **model** is the ID that server should run. A **model profile** tells
Freehand how to use that model, including its languages, recognition hints, and
output controls. For a manual Connection, select the profile yourself, even
when the server uses a familiar model name or a custom alias.

For installation inside Freehand, see [Local runtimes](../guides/local-runtime/#choose-a-runtime-and-model).
That guide lists the managed models and supported computers. A managed
Connection gets its model profile from the selected runtime model; you do not
need to enter a model ID or choose its profile manually.

## Choose a model profile

For a manual Connection, use the workflow page's **Settings** cog to open its right
options sidebar. Choose **Transcription** in Voice or Audio file, **Cleanup** in
either transcription workflow, or **Speech** in Text to speech.
Choose an active connection, choose or enter the model, and review **Model
profile** directly beneath it. Choose a specialized profile only when you know
that is the model your server is running, then choose **Save**.

| Feature            | Available model profiles                                                                                                                                                      | Behavior                                                                                                                                   |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Voice / Audio file | Generic; [Nemotron](./nemotron/), [Parakeet](./parakeet/), [Qwen3-ASR](./qwen3-asr/), [Cohere Transcribe](./cohere-transcribe/), [Voxtral Mini Realtime](./voxtral-realtime/) | Each profile shows the recognition controls supported by its backend. Nemotron, Qwen3-ASR, and Voxtral enable optional realtime for Voice. |
| Cleanup            | Generic; [S1-mini by Superwhisper](./s1-mini/)                                                                                                                                | Generic uses your cleanup instruction. S1-mini uses its fixed normalization prompt and trained output controls.                            |
| Text to speech     | Generic; [Qwen3-TTS](./qwen3-tts/) on vLLM-Omni                                                                                                                               | WAV speech with a voice ID and speed. Qwen3-TTS adds preset voices, language, and style instructions.                                      |

If a backend offers only Generic, Freehand uses it without a profile selector.

Use **Generic** for standard backend options, including Whisper transcription,
Kokoro speech, and cleanup with your own instructions. A dedicated model profile
adds controls for that model and shows the options its backend supports.
Explore the [model directory](../../models/) for what each profile adds in Freehand.

Model discovery reads metadata only. It does not infer a model profile from a
name, download a model, or run one to detect its capabilities. Manual whisper.cpp
connections use the model already loaded by the server; managed whisper.cpp
loads the model you select in Local runtime.

Voice's **Transcription** options keep completed and realtime configuration together.
With NeMo-Speech.cpp/Nemotron, vLLM/Qwen3-ASR, or vLLM/Voxtral Mini Realtime,
enable **Realtime transcription** for live results and optional overlay captions.
Turning realtime off keeps the same connection and model for completed
recordings. Audio-file transcription has its own selection.

## Dedicated model guides

- [S1-mini by Superwhisper](./s1-mini/): English cleanup with styling, structure, and context controls.
- [Nemotron 3.5 ASR streaming](./nemotron/): completed and live transcription with language selection and vocabulary boosting.
- [Qwen3-ASR](./qwen3-asr/): completed transcription with context hints, plus optional live dictation on vLLM.
- [Parakeet TDT v3](./parakeet/): completed transcription with automatic language detection.
- [Cohere Transcribe](./cohere-transcribe/): completed recordings and streamed file results with a selected language.
- [Voxtral Mini Realtime](./voxtral-realtime/): live or completed transcription with automatic language detection.
- [Qwen3-TTS](./qwen3-tts/): text to speech with preset voices, languages, and style instructions.

## Remember settings for each model

The model ID picker below applies to manual Connections. For a managed
Connection, choose a downloaded model in the runtime controls; every task using
that runtime shares its selected model.

Voice, Audio file, Cleanup, and Text to speech use the same searchable **Model**
picker in their options and first-run setup controls. **Saved** identifies remembered options,
**Server** identifies an advertised model, and **Edited** identifies a model with
draft options in this editing session. An ID can have more than one label;
**Manual ID** means it came from neither the saved list nor current discovery.
The selected model's profile appears below the picker. You can enter an exact ID
without refreshing models; discovery never chooses a profile for you.

Choose **Save** in the options sidebar to apply your edits together. A failed save
keeps the previously applied settings active and preserves the draft for correction.
First-run setup controls apply valid changes immediately and show their save status.

Freehand remembers your choices separately for each **connection, feature, and
model ID**. Choose a saved model from the searchable **Model** picker to restore those options
without listing models on the server. A new model ID starts with Generic and
Freehand's default model options; its name never selects a specialized profile.

| Feature        | Remembered model options                                            |
| -------------- | ------------------------------------------------------------------- |
| Transcription  | Model profile, prose context hint, and temperature override         |
| Cleanup        | Model profile, output limit, and reasoning override                 |
| Text to speech | Model profile, voice, speech language, and voice-style instructions |

Changing models or connections keeps your transcription language, custom cleanup
instruction, S1-mini style/structure/context choices, and speaking speed.
Context hints are remembered per model. Review language compatibility after a
switch: selecting S1-mini does not change the transcription language, and it
skips cleanup for known non-English input.

Search or enter a model ID in the **Model** picker, select it, then review its options.
**Save** saves the current selection and all edited model options together.
Switching models in options changes your draft; **Discard** restores the applied
selection. You can switch freely: unsaved options stay in this editing session for each
model, and returning to that model restores your edits. Discard clears all of
these model drafts. First-run setup model controls apply and save
immediately, including the restored options.

Switching connections restores that connection's last selected model for the
feature. A connection without a remembered selection starts with defaults and
needs a model chosen. Enable switches, recording behavior, shortcuts, and request
timeouts remain feature or application settings rather than model preferences.
Running jobs continue with the settings captured when they started.

The model actions menu’s **Forget saved settings** removes its saved preferences and clears the current model
selection. Save or discard edits first. You can enter the same ID again to start
from defaults. Freehand remembers up to 32 models per connection and feature;
forget an unused model if you reach that limit.

Renaming a connection or changing its authentication keeps remembered models.
Changing its URL or backend profile clears its remembered models and active model
choices. Select the model and review its options again after such a change.
Duplicating a connection starts a separate set of model preferences. Deleting a
connection, or removing one of its inactive uses, removes the corresponding
remembered preferences.

For whisper.cpp, options belong to the connection's **Server-loaded model** slot.
Freehand cannot identify a replacement model loaded by that server; review its
language and options yourself when changing it. The server still owns model loading.

No credentials or generated transcripts are included in model preferences.

Shared vocabulary terms, Voice/file opt-ins, and Nemotron vocabulary strength
live in [Vocabulary](../guides/vocabulary/), independently of remembered models.

For Qwen setup, supported languages, and completed versus realtime controls, see [Qwen3-ASR](./qwen3-asr/).
