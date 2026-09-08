---
title: "ADR 0011: Qwen3-ASR on vLLM realtime"
description: Qualify a second realtime transport under the existing unified Voice selection.
---

- Status: Accepted
- Date: 2026-09-07
- Extends: ADR 0008's qualified realtime transports; preserves ADR 0009's unified selection and ADR 0010's shared vocabulary ownership.

## Decision

Add the explicit `qwen3-asr` model profile, qualified for vLLM 0.28.0 completed
transcription and its Qwen realtime architecture. Reuse the existing `vllm`
backend ID with a separately implemented realtime role. Generic model behavior
does not gain realtime eligibility. Inference remains user-managed; no runtime
installation or model loading is added to the application.

Completed language hints are the intersection of the model's published
languages and vLLM's language map. Context, shared vocabulary through prompt,
temperature, and file response streaming use the completed adapter. Realtime
accepts only the model and PCM16 audio; unsupported completed preferences are
preserved but never transmitted. There are no invented session controls.

## Wire and lifecycle

`internal/realtime` shares bounded capture-pipe drainage and shutdown with NeMo,
but dispatches separate wire behavior after configuration validation. vLLM's
`session.created` is followed by model-only `session.update` and a non-final
`input_audio_buffer.commit` that starts generation. There is no configuration
acknowledgement; model admission errors arrive asynchronously and fail capture.
This is a specific vLLM protocol, not generic OpenAI Realtime compatibility.

Audio uses JSON `input_audio_buffer.append` with base64 little-endian mono
16 kHz PCM16. The temporary encoded buffer is cleared after writing. Stopping
capture sends `input_audio_buffer.commit` with `final:true`. Delta events are
presentation-only. Only `transcription.done` with an explicit text field after
the local stop request can produce a result. Premature done, missing text,
disconnect, server error, cancellation, and size overflow fail closed.

The adapter bounds setup to 10 seconds, writes to 5 seconds, finalization to
30 seconds, and raw/returned text to 256 KiB. Authentication stays in the
upgrade header, redirects are refused, and untrusted peer errors are not
returned or logged. Existing recording ownership, immutable credentials,
cleanup fallback, focus-safe insertion, and single-row captions remain intact.

The model layer strips recognized Qwen language headers across fragmented
deltas and independently generated segments. It preserves ordinary text and
reports mixed detected languages as multilingual rather than declaring an
arbitrary single language. Repeated content is not heuristically deduplicated.

## Evidence

See the [Qwen setup guide](../../models/qwen3-asr/) for pinned source links,
runtime setup, supported controls, and scoped transport evidence. Recognition
quality and runtime performance are deployment concerns, not Freehand contracts.

Deterministic tests exercise JSON audio, model-only configuration, fragmented
previews, final replacement, malformed/missing finals, disconnect, cancellation,
peer-error redaction, transcript limits, and capability/language intersection.
Inference tests are manual and limited to an explicitly selected model. Native
microphone, overlay, and focus-safe insertion acceptance remains separate.
