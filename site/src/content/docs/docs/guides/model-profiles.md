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

Open **Settings → Transcription**, **Post-processing**, or **Speech playback**.
Choose an active connection, choose or enter the model, and review **Model
profile** directly beneath it. Choose a specialized profile only when you know
that is the model your server is running, then **Save feature settings**.

| Feature | Available model profiles | Behavior |
| --- | --- | --- |
| Transcription | Generic | Standard transcription with the options available through the selected backend. |
| Post-processing | Generic; S1-mini by Superwhisper | Generic uses your cleanup instruction. S1-mini uses its fixed normalization prompt and trained output controls. |
| Speech playback | Generic | Standard WAV speech generation with a provider voice ID. |

Where only Generic is available, Freehand shows its name and explanation instead
of a dropdown with a single choice. The home screen also identifies the
transcription model profile and lets you choose a cleanup model profile.

**Generic is a baseline, not a claim that every model supports every option.**
Language support, voice IDs, context hints, temperature, and generation controls
still depend on the deployed model. Freehand exposes only options permitted by
both its backend contract and its model profile. More specialized transcription
and speech profiles can be added after their differences have been verified.

Model discovery reads metadata only. Freehand does not infer a model profile
from a name, download a model, or run one to detect its capabilities. For
whisper.cpp, the model remains the one already loaded by the server.

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

## What gets saved?

Each feature saves its current model profile and options independently. Editing
a shared connection does not replace those choices. Model profiles do not own
URLs, credentials, shortcuts, or recording behavior.

Changing a model ID does not automatically switch its profile. Review the
profile when selecting another model. Separately remembered settings for each
model are future work; the current feature settings remain explicit.

Existing installations retain their cleanup profile, instructions, and output
controls. Transcription and speech playback start with Generic. Running jobs
keep the profile captured when they began.
