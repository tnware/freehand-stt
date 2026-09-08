---
title: Model profiles
description: Choose model behavior independently of your server connection.
---

A **connection** tells Freehand where your server is and which backend API it
uses. A **model** is the ID that server should run. A **model profile** tells
Freehand how to use that model, including its languages, recognition hints, and output controls. These choices stay separate because servers can expose custom
model names and the same model can run behind different backends.

## Choose a model profile

Open **Settings → Voice transcription**, **Audio-file transcription**, **Cleanup**, or **Text to speech**.
Choose an active connection, choose or enter the model, and review **Model
profile** directly beneath it. Choose a specialized profile only when you know
that is the model your server is running, then **Save settings**.

| Feature            | Available model profiles                                                               | Behavior                                                                                                                                                                                    |
| ------------------ | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Voice / Audio file | Generic; [Nemotron](./nemotron/), [Parakeet](./parakeet/), [Qwen3-ASR](./qwen3-asr/), [Cohere Transcribe](./cohere-transcribe/), [Voxtral Mini Realtime](./voxtral-realtime/) | Each profile shows the recognition controls supported by its backend. Nemotron, Qwen3-ASR, and Voxtral enable optional realtime for Voice. |
| Cleanup            | Generic; [S1-mini by Superwhisper](./s1-mini/)                                         | Generic uses your cleanup instruction. S1-mini uses its fixed normalization prompt and trained output controls.                                                                             |
| Text to speech | Generic; [Qwen3-TTS](./qwen3-tts/) on vLLM-Omni | WAV speech with a voice ID and speed. Qwen3-TTS adds preset voices, language, and style instructions. |

Where only Generic is available, the redundant profile row is hidden. The standard
backend contract still applies; there is no extra choice to make.

Use **Generic** for standard backend options, including Whisper transcription,
Kokoro speech, and cleanup with your own instructions. A dedicated model profile
adds controls for that model and shows the options its backend supports.
Explore the [model directory](../../models/) for what each profile adds in Freehand.

Model discovery reads metadata only. Freehand does not infer a model profile
from a name, download a model, or run one to detect its capabilities. For
whisper.cpp, the model remains the one already loaded by the server.

Voice’s **Transcription** quick controls also expose the model profile. With NeMo-Speech.cpp/Nemotron or vLLM/Qwen3-ASR or Voxtral Mini Realtime selected, **Realtime transcription** appears inside that panel. Its caption control appears when enabled; shared terminology is managed in **Settings → Vocabulary**, and turning it off keeps the same connection/model. Audio file remains independent.

## Dedicated model guides

- [S1-mini by Superwhisper](./s1-mini/): English cleanup with styling, structure, and context controls.
- [Nemotron 3.5 ASR streaming](./nemotron/): completed and live transcription with language selection and vocabulary boosting.
- [Qwen3-ASR](./qwen3-asr/): completed transcription with context hints, plus optional live dictation on vLLM.

## Remember settings for each model

Voice, Audio file, Cleanup, and Text to speech use the same searchable **Model**
picker in quick controls and Settings. **Saved** identifies remembered options,
**Server** identifies an advertised model, and **Edited** identifies a model with
draft options in this Settings session. An ID can have more than one label;
**Manual ID** means it came from neither the saved list nor current discovery.
The selected model's profile appears below the picker. You can enter an exact ID
without refreshing models; discovery never chooses a profile for you.

Quick panels show **Saving…**, **Saved**, or a failed-save message beside their
controls. A failed save keeps the previously applied settings active; retry the
change to save it. Full Settings pages continue to use **Save settings**.

Freehand remembers your choices separately for each **connection, feature, and
model ID**. Choose a saved model from the searchable **Model** picker to restore those options
without listing models on the server. A new model ID starts with Generic and
Freehand's default engine options; its name never selects a specialized profile.

| Feature        | Remembered model options                                    |
| -------------- | ----------------------------------------------------------- |
| Transcription  | Model profile, prose context hint, and temperature override |
| Cleanup        | Model profile, output limit, and reasoning override         |
| Text to speech | Model profile, voice, speech language, and voice-style instructions |

**Task intent stays in place when you change models or connections:** transcription
language, the custom cleanup instruction, S1-mini style/structure/context choices,
and speaking speed. Switching engines does not replace these with an older model's
values. Prose context hints remain model options because their meaning
and availability depend on the backend. Specialized profiles still constrain what
can run: S1-mini's English-only behavior never changes the transcription language.

Search or enter a model ID in the **Model** picker, select it, then review its options.
**Save feature settings** saves the current selection and all edited model options together.
Switching models in Settings changes your draft; **Discard** restores the applied
selection. You can switch freely: unsaved options stay in this editing session for each
model, and returning to that model restores your edits. Discard clears all of
these model drafts. Home-screen model controls apply and save
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
choices, because the previous options may describe a different server contract.
Duplicating a connection starts a separate set of model preferences. Deleting a
connection, or removing one of its inactive uses, removes the corresponding
remembered preferences.

For whisper.cpp, options belong to the connection's **Server-loaded model** slot.
Freehand cannot identify a replacement model loaded by that server; review its
language and options yourself when changing it. The server still owns model loading.

Upgrades retain current model selections and seed their remembered preferences.
No credentials or generated transcripts are included in model preferences.

Shared vocabulary terms and Voice/file opt-ins live in [Vocabulary](../guides/vocabulary/), independently of remembered models.

For Qwen setup, supported languages, and completed versus realtime controls, see [Qwen3-ASR](./qwen3-asr/).
