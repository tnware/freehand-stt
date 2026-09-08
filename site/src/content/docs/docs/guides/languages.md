---
title: Choose a transcription language
description: Select input languages, understand automatic detection, and use English-only S1-mini cleanup safely.
---

Under **Settings → Voice transcription** or **Audio-file transcription**, choose the spoken language. The two tasks keep independent selections. Searchable pickers accept language names or codes. Long lists fit the available window space and scroll; use the mouse wheel or arrow keys to reach additional options. This setting requests transcription in the source language; it does not request translation.

## Choose a mode

| Choice | Behavior |
| --- | --- |
| Server default | Leaves the language field out. The server may detect a language or use its configured default. Existing blank settings keep this behavior. |
| Automatic detection | Uses the selected provider's detection contract. Detection still depends on the server and model. |
| A named language | Sends its code, such as `en`, `es`, or `ja`. |
| Custom server value | Preserves an explicit value required by a compatible server. Existing unlisted values appear here rather than being replaced. |

The picker includes ISO 639-1 languages and the speech-oriented `haw` and `yue`
choices. Its list identifies languages; it does **not** establish that your model
supports all of them. Select a model with the required language support. In
particular, an English-only recognition model does not become multilingual when
you choose Automatic detection or another language.

A custom value is limited to 32 UTF-8 bytes without control characters. Ordinary
custom values are sent unchanged. The reserved `auto` value uses Freehand's
Automatic detection behavior.

## Provider behavior

| Profile | Server default | Automatic detection | Named language |
| --- | --- | --- | --- |
| Generic OpenAI-compatible | Omit `language` | Omit `language`; relies on the server's detection behavior | Send the selected code |
| Speaches | Omit `language` | Omit `language`; the qualified Whisper path detects when supported | Send the selected code |
| whisper.cpp | Omit `language`; keeps the server's configured language | Send `language=auto` | Send the selected code |
| vLLM | Omit `language` | Omit `language`; model-dependent detection | Send the selected code |

For Generic, Speaches, and vLLM, Server default and Automatic detection currently
produce the same request field omission. The separate choices preserve intent
when switching to whisper.cpp, whose default can be a fixed language. Freehand
does not use `detect_language=true` with whisper.cpp: that flag asks the native
server to stop after detection instead of completing transcription.

The mappings follow [Speaches v0.8.3](https://github.com/speaches-ai/speaches/blob/v0.8.3/src/speaches/routers/stt.py),
[whisper.cpp's qualified server source](https://github.com/ggml-org/whisper.cpp/blob/52a939a2a762224e255d366c1182b2af4dd1a032/examples/server/server.cpp),
and [vLLM v0.28.0's request contract](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/transcription/protocol.py).
For Qwen3-ASR served by vLLM, send the code through vLLM; its model implementation
handles the internal language-name prompt formatting. Freehand does not inject
Qwen prompt tokens. See [vLLM's model implementation](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/model_executor/models/qwen3_asr.py).

These are source and client-fixture qualifications, not live acceptance of every
language. The earlier vLLM Whisper test encountered automatic-detection failures;
selecting Automatic detection here does not fix an underlying server/model bug.

## S1-mini cleanup is English only

S1-mini's existing preset declares **English** as its fixed supported language.
There is no additional language dropdown or new language control in its trained
prompt. Its required reasoning-off behavior remains in place.

- If you explicitly select a non-English transcription language, Freehand skips
  S1-mini and keeps the raw transcript.
- If the server reports a non-English language, including a mixed-language
  result, Freehand also skips S1-mini, even if English was selected.
- With Server default or Automatic detection and no known language, S1-mini runs
  **assuming English**. This assumption is shown in the preset's controls. It is
  not local language detection or a guarantee about the recording.

A language mismatch makes no cleanup request. The workflow shows an English-only
notice and retains raw text for normal safe insertion or explicit file-result
copying. If session history is enabled, the entry records the raw fallback and
`unsupported_language` reason. History remains optional.

For non-English cleanup, choose **Custom instruction** with a suitable model and
instructions, or turn cleanup off. Selecting S1-mini never changes your saved
transcription language. Freehand does not infer a model's languages from its
name, probe models to discover support, or silently translate text.
