---
title: Model profiles
description: Choose model behavior independently of your server connection.
---

A **connection** tells Freehand where your server is and which backend API it
uses. A **model** is the ID that server should run. A **model profile** tells
Freehand how to use that model, including any verified restrictions or special
instructions. These choices stay separate because servers can expose custom
model names and the same model can run behind different backends.

## Choose a model profile

Open **Settings → Voice transcription**, **Audio-file transcription**, **Cleanup**, or **Text to speech**.
Choose an active connection, choose or enter the model, and review **Model
profile** directly beneath it. Choose a specialized profile only when you know
that is the model your server is running, then **Save feature settings**.

| Feature | Available model profiles | Behavior |
| --- | --- | --- |
| Voice / Audio file | Generic; Nemotron 3.5 ASR streaming on NeMo-Speech.cpp | Generic offers the selected backend’s completed transcription options. Nemotron restricts languages to qualified base-model locales and enables optional realtime mode for Voice. |
| Cleanup | Generic; S1-mini by Superwhisper | Generic uses your cleanup instruction. S1-mini uses its fixed normalization prompt and trained output controls. |
| Text to speech | Generic | Standard WAV speech generation with a provider voice ID. |

Where only Generic is available, the redundant profile row is hidden. The standard
backend contract still applies; there is no extra choice to make.

**Generic is a baseline, not a claim that every model supports every option.**
Language support, voice IDs, context hints, temperature, and generation controls
still depend on the deployed model. Freehand exposes only options permitted by
both its backend contract and its model profile. More specialized transcription
and speech profiles can be added after their differences have been verified.

Model discovery reads metadata only. Freehand does not infer a model profile
from a name, download a model, or run one to detect its capabilities. For
whisper.cpp, the model remains the one already loaded by the server.

Voice’s **Transcription** quick controls also expose the model profile. With NeMo-Speech.cpp and the Nemotron profile selected, **Realtime transcription** appears inside that panel. Its vocabulary and caption controls appear when enabled; turning it off keeps the same connection/model. Audio file remains independent.

## S1-mini

Selecting S1-mini keeps its specialized prompt and output controls, regardless
of the model ID your server exposes. Its requirements are shown near the picker:

- **English only.** Selected or reported non-English input skips cleanup and
  preserves the raw transcript. When the language is unknown, Freehand runs
  S1-mini assuming English; it does not change transcription's language setting.
- **Reasoning off.** Qualified llama.cpp and vLLM adapters enforce this on every
  cleanup request. With Generic, you must disable reasoning on the server.
- **Fixed instructions.** The model's trained styling, structure, and context
  controls replace the custom instruction editor. Temperature remains zero.

See [post-processing](../post-processing/) and [languages](../languages/).

## Remember settings for each model

Freehand remembers your choices separately for each **connection, feature, and
model ID**. Choose a saved model from the searchable **Model** picker to restore those options
without listing models on the server. A new model ID starts with Generic and
Freehand's default engine options; its name never selects a specialized profile.

| Feature | Remembered model options |
| --- | --- |
| Transcription | Model profile, context hint, hotwords, and temperature override |
| Cleanup | Model profile, output limit, and reasoning override |
| Text to speech | Model profile and voice |

**Task intent stays in place when you change models or connections:** transcription
language, the custom cleanup instruction, S1-mini style/structure/context choices,
and speaking speed. Switching engines does not replace these with an older model's
values. Recognition hints and hotwords remain model options because their meaning
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

The model actions menu’s **Forget saved settings for this model** removes its saved preferences and clears the current model
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
