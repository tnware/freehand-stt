---
title: S1-mini by Superwhisper
description: Shape English transcripts with styling, structure, and context controls.
---

S1-mini is a transcript cleanup model. Freehand sends it the completed text
from Voice or an audio file, then uses the cleaned result for delivery.
Your transcription connection and model stay separate.

## What the profile adds

Selecting **S1-mini by Superwhisper** in **Settings → Cleanup** replaces the
custom instruction editor with the model's trained controls:

| Control   | Choices                                  |
| --------- | ---------------------------------------- |
| Styling   | Casual, semi-casual, semi-formal, formal |
| Structure | Prose, lists                             |
| Context   | General, email                           |

Freehand builds the fixed instruction and control line for you. Settings shows
the effective prompt for review. Temperature stays at zero, and reasoning must
be off. The optional output-token limit and timeout remain available.

These controls also appear in the **Cleanup** quick settings popover. Quick
changes apply immediately; full Settings edits apply when you choose **Save settings**.
Switching models preserves your cleanup style, structure, and context choices.

## Choose a backend

- **[llama.cpp](../../backends/llama-cpp/#run-llamacpp-on-windows)** provides a native Windows launch recipe. Freehand enforces reasoning off for S1-mini requests.
- **[vLLM](../../backends/vllm/#start-s1-mini-cleanup)** provides a Docker launch recipe. Freehand also enforces reasoning off for S1-mini requests.
- **[Generic OpenAI-compatible](../../backends/generic/)** can connect an existing chat endpoint. Configure reasoning off on that server; Settings shows **Disable on server**.

Select the saved connection in **Settings → Cleanup**, enable cleanup, choose
the model ID exposed by the server, and select **S1-mini by Superwhisper** as
the model profile. Choose your output controls and save.

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
