---
title: Choose a transcription language
description: Select input languages, understand automatic detection, and use English-only S1-mini cleanup safely.
---

Open **Voice transcription** or **Audio file**, use the page's **Settings** cog,
and choose **Transcription → Spoken language** in the right options sidebar.
Search by language name or code, then choose **Save**. Voice and audio files keep independent language selections
when you change models or connections. This setting requests transcription in
the source language, not translation.

## Choose a mode

| Choice              | Behavior                                                                                                                      |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Server default      | Lets the server detect a language or use its configured default.                                                              |
| Automatic detection | Asks the server to detect the language, where supported by the selected model.                                                |
| A named language    | Sends its code, such as `en`, `es`, or `ja`.                                                                                  |
| Custom server value | Preserves an explicit value required by a compatible server. Existing unlisted values appear here rather than being replaced. |

The available choices depend on the model profile. The **Generic** list includes
ISO 639-1 languages, Hawaiian (`haw`), and Cantonese (`yue`), but does **not** prove
your model supports them. An English-only model does not become multilingual
when you choose Automatic detection or another language.

Specialized profiles have narrower choices:

- **Nemotron 3.5 ASR streaming** supports automatic detection or a listed locale,
  such as `en-US`, in completed and realtime mode.
- **Qwen3-ASR** supports listed languages for completed recordings and audio
  files. Realtime uses automatic detection. Cantonese and Filipino cannot be
  selected as explicit language hints with the supported vLLM backend.
- **Cohere Transcribe** requires the spoken language for non-English audio;
  **Server default (English)** does not request automatic detection.
- **Parakeet TDT v3** and **Voxtral Mini Realtime** use automatic detection.

See the [model guides](../../models/) for each model's language support.

A custom value, where available, is limited to 32 UTF-8 bytes without control characters. Ordinary
custom values are sent unchanged. The reserved `auto` value uses Freehand's
Automatic detection behavior.

## Provider behavior

| Backend with Generic model profile | Server default                        | Automatic detection                                  | Named language                |
| ---------------------------------- | ------------------------------------- | ---------------------------------------------------- | ----------------------------- |
| Generic OpenAI-compatible          | Use the server's default              | Rely on the server's detection behavior              | Request the selected language |
| Speaches                           | Use the server's default              | Detect with a supported model, such as Whisper       | Request the selected language |
| whisper.cpp                        | Keep the server's configured language | Override the server default with automatic detection | Request the selected language |
| vLLM                               | Use the server's default              | Detect if the model supports it                      | Request the selected language |

For Generic, Speaches, and vLLM, Server default and Automatic detection send the
same request. With whisper.cpp, choose Automatic detection if you want to
override a fixed server language. If detection fails or chooses the wrong
language, select the spoken language explicitly where the model supports it.

## S1-mini cleanup is English only

S1-mini supports **English** cleanup only. Its language cannot be changed.

- If you explicitly select a non-English transcription language, Freehand skips
  S1-mini and keeps the raw transcript.
- If the server reports a non-English language, including a mixed-language
  result, Freehand also skips S1-mini, even if English was selected.
- With Server default or Automatic detection and no known language, S1-mini runs
  **assuming English**. This assumption is shown in the preset's controls. It is
  not local language detection or a guarantee about the recording.

A language mismatch makes no cleanup request. The workflow shows an English-only
notice and retains raw text for normal safe insertion or explicit file-result
copying. If session history is enabled, the entry records why cleanup was skipped.

For non-English cleanup, choose the **Generic** cleanup model profile with a
suitable model and instructions, or turn cleanup off. Selecting S1-mini never changes your saved
transcription language. Freehand does not infer a model's languages from its
name, probe models to discover support, or silently translate text.
