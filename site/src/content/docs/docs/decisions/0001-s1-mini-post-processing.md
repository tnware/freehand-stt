---
title: "ADR 0001: Optional S1-mini transcript post-processing"
description: Keep speech recognition and optional transcript normalization as separate capabilities.
---

- Status: Implemented
- Date: 2026-08-29
- Authority: [superwhisper/s1-mini v1 model card](https://huggingface.co/superwhisper/s1-mini/tree/v1)

## Protocol reliability clarification — 2026-09-05

The client now rejects cleanup explicitly marked `finish_reason: "length"` and
uses the raw transcript with an output-limit notice. This supersedes the
length-limited-output acceptance described in the historical alpha.1 note
below. The rule belongs to the generic chat contract and applies to both S1-mini
and custom processing profiles. It does not change S1-mini's trained prompts or
control values. Automatic sentence chunking and input-relative output limits
remain unimplemented, and omissions without a reported length limit cannot be
detected by this check.

## Alpha implementation clarification — 2026-09-04

The separate processing stage, trained S1-mini by Superwhisper profile, and raw
failure fallback are implemented. The accepted generation requirements below
remain the design intent, but **input-relative output limits and sentence
chunking near 1,000 tokens are not implemented in v0.1.0-alpha.1**. The client
sends one cleanup request per input and accepts nonempty output even when the
provider reports a length-limited completion. This note qualifies the
implementation status without replacing the decision or its requirements.

Raw failure fallback is independent of history. Retaining both versions after
successful cleanup, or retaining a raw result after cancellation, requires
history to be enabled and the text to fit its memory limits. History is
session-only and disabled by default.

## Decision

Keep speech recognition and transcript normalization as separate capabilities:

```text
microphone
  -> client capture
  -> LiteLLM /audio/transcriptions
  -> Speaches / Whisper
  -> raw transcript
  -> optional OpenAI-compatible /chat/completions
  -> S1-mini by Superwhisper
  -> clean transcript
  -> focus-safe insertion
```

S1-mini is never hidden inside the STT service. Raw transcription remains available and can be selected independently.

S1-mini may run behind LiteLLM or through a directly configured local OpenAI-compatible runtime such as llama.cpp. The Windows executable does not bundle llama.cpp, model weights, Python, or a local inference runtime.

## Why

Raw transcripts are required for verbatim records, subtitles, logs, diagnostics, and analysis. Cleanup deliberately removes fillers, resolves corrections, and changes formatting, so it is not semantically interchangeable with ASR.

The model is purpose-built for ASR normalization rather than general chat. The v1 model card describes 596M unique parameters, English-only support, a 462 MiB quantized build, and recommended inputs under roughly 1,000 tokens. Those properties make it attractive as a backend stage, but not a reason to expand the tray client into a model host.

## Client profile

```yaml
post_processing:
  enabled: true
  model: transcript/s1-mini
  styling: semi-formal
  structure: prose
  context: general
```

Raw mode sets `enabled: false`.

The initial design discussion used `balanced`; S1-mini v1 does not train that value. The valid styling values are exactly:

```text
casual | semi-casual | semi-formal | formal
```

Valid structure values:

```text
prose | lists
```

Valid context values:

```text
general | email
```

Do not send untrained values.

## Request contract

Endpoint:

```text
POST {base_url}/chat/completions
```

System message, exactly as documented by the model card:

```text
You are a text normalizer for speech-to-text transcripts. The input begins with a control line specifying the styling, structure, and context settings; clean the transcript to match those settings and output only the cleaned text.
```

User message:

```text
[Styling: semi-formal] [Structure: prose] [Context: general]
<raw transcript>
```

Generation requirements:

- Deterministic decoding; temperature 0.
- Thinking disabled by the configured inference runtime. Current llama.cpp releases use the server option `--reasoning off`.
- Plain cleaned text only.
- The model may return empty output for filler/noise-only input. The client deliberately treats an empty processor response as cleanup failure and falls back to the non-empty raw transcript so optional processing cannot suppress delivery.
- Bound output length relative to input.
- Chunk inputs approaching 1,000 tokens at sentence boundaries.

The configured runtime must forward or enforce thinking-disabled template behavior. Do not assume generic OpenAI compatibility preserves this provider-specific control. See the [local llama.cpp setup](../../guides/post-processing/#s1-mini-by-superwhisper-with-llamacpp) for a known-good Windows configuration.

## Hotkey/profile direction

Potential bindings after the dictation MVP:

```text
Normal dictation  -> STT -> S1-mini semi-formal/prose/general -> paste
Raw dictation     -> STT -> paste
Email dictation   -> STT -> S1-mini formal/prose/email -> paste
List dictation    -> STT -> S1-mini semi-formal/lists/general -> paste
```

Hotkeys choose named processing profiles rather than embedding model-specific branches in the platform adapter.

## Failure policy

Post-processing uses `fallback_raw`: retain the raw result before cleanup begins, mark cleanup failure in history and diagnostics, and continue delivery using the raw text. Cancellation after raw transcription retains the raw history entry but does not insert it. Raw text is never presented as successfully cleaned.

A stale cleanup response carries the same operation generation as its raw transcript and cannot insert after cancellation or a newer recording.

## Deployment constraints

Do not automatically place S1-mini on the desktop Ollama GPU. That GPU can hold only one model at a time, and using S1-mini there could evict the user's selected LLM on every dictation. Choose a backend placement deliberately—CPU, a separate worker, or a host with proven headroom—without broad model smoke testing.

## Consequences

- Post-processing remains a separately configurable, composable capability.
- Raw and polished text remain distinct in state and diagnostics.
- Future conversation mode can use `STT -> optional S1-mini -> LLM -> TTS` without changing STT semantics.
- The repository must retain attribution as **S1-mini by Superwhisper** wherever the model is referenced, consistent with its license naming clause.
