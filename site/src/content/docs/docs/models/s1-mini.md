---
title: S1-mini by Superwhisper
description: Shape English transcripts with styling, structure, and context controls.
---

S1-mini is a transcript cleanup model. Freehand sends it the completed text
from Voice or an audio file, then uses the cleaned result for delivery.
Your transcription connection and model stay separate.

## What the profile adds

Selecting **S1-mini by Superwhisper** in **Cleanup** options for Voice transcription
or Audio file replaces the
custom instruction editor with the model's trained controls:

| Control   | Choices                                  |
| --------- | ---------------------------------------- |
| Styling   | Casual, semi-casual, semi-formal, formal |
| Structure | Prose, lists                             |
| Context   | General, email                           |

Freehand builds the fixed instruction and control line for you. Settings shows
the effective prompt for review. Temperature stays at zero, and reasoning must
be off. The optional output-token limit and timeout remain available.

Open the workflow's **Settings** cog to show **Cleanup** options on the right.
Choose **Save** to apply your edits; closing or leaving unsaved options offers
**Save**, **Discard**, or **Keep editing**.
Switching models preserves your cleanup style, structure, and context choices.

## Choose a backend

For local cleanup on supported Windows and macOS computers, follow
[managed llama.cpp setup](../../guides/local-runtime/#local-cleanup-with-s1-mini).
Freehand installs the runtime and downloads S1-mini only when you choose those
actions. Select its built-in Connection for Cleanup and enable cleanup; the
runtime supplies the S1-mini profile and disables reasoning.

For a service you manage separately:

- **[llama.cpp](../../backends/llama-cpp/#run-llamacpp-on-windows)** provides a native Windows launch recipe. Freehand sends `reasoning_effort: "none"` for S1-mini; the server and model template must honor it. Keep `--reasoning off` in the launch command.
- **[vLLM](../../backends/vllm/#start-s1-mini-cleanup)** provides a Docker launch recipe. Freehand sends `reasoning_effort: "none"` for S1-mini; use the qualified server version and a template that honors it.
- **[Generic OpenAI-compatible](../../backends/generic/)** can connect an existing chat endpoint. Configure reasoning off on that server; the options show **Disable on server**.

For these manual services, select the saved connection in **Cleanup** options,
enable cleanup, choose the model ID exposed by the server, and select
**S1-mini by Superwhisper** as the model profile. Choose your output controls and save.

## Language and results

S1-mini supports **English**. An explicit or detected non-English language skips
cleanup and preserves the raw transcript. When the language is unknown,
Freehand assumes English for cleanup without changing your transcription language.

If cleanup fails, returns empty text, or reports an output-token limit,
Freehand uses the raw transcript. Cleanup runs once per completed transcript;
it does not split long inputs into chunks. Optional history can retain both
the raw and cleaned versions during the current session.

See [transcript cleanup](../../guides/post-processing/) for workflow and generation
settings, or [model profiles](../) for how saved model options work.
