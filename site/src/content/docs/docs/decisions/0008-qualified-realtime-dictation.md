---
title: "ADR 0008: Qualified realtime dictation"
description: Add an optional Nemotron streaming transport without expanding Freehand into an inference runtime.
---

- Status: Accepted
- Date: 2026-09-07

> [ADR 0011](../0011-qwen-vllm-realtime/) adds a separately qualified vLLM/Qwen3-ASR transport. The NeMo-specific protocol below remains unchanged.
- Supersedes: ADR 0005's blanket deferral of realtime dictation, and ADR 0002's proposed first transport and event correlation requirements for this adapter only.

> The separate feature-selection and credential-slot design below is superseded by [ADR 0009](../0009-unified-voice-transcription/). The qualified transport and safety decisions remain in force.

## Decision

Offer opt-in realtime microphone dictation using the explicit **NeMo-Speech.cpp v0.1.0** backend and **Nemotron 3.5 ASR streaming 0.6B** model profile. Completed/file transcription remains independently configured; the existing pause-aware flow remains the default. Conversation mode remains deferred. Freehand never installs, starts, downloads, or manages inference models as a product feature.

The user value is immediate provisional text in the result pane and an optional passive, single-row native caption strip. New words replace the visible tail within fixed geometry. Partial text has no copy, cleanup, history, or insertion authority. Stopping capture commits the stream; authoritative final text then enters the existing cleanup, optional memory history, and focus-safe insertion path. Cancellation, transport loss, and missing finals discard previews without automatic replay or reconnect.

## Qualified contract

The versioned `nemo-speech-v1` adapter uses the project-specific `/v1/realtime` WebSocket protocol, not a claim of generic OpenAI Realtime compatibility. Configuration completes before capture: `session.created`, one `session.update`, and `session.updated`. The loaded model ID must match the user's explicit selection. Audio is binary 16 kHz mono little-endian PCM16. Native capture feeds a bounded 128-frame pipe; a slow transport fails rather than retaining unbounded audio.

Incremental events update provisional text. A completed event replaces that turn's preview. Local generation and sequential turn counters fence state; v0.1.0 supplies no item IDs, so this adapter does not fabricate transport correlation. `input_audio_buffer.commit` follows capture shutdown; completion requires an authoritative final and `input_audio_buffer.committed`. Setup is bounded to 10 seconds, frame writes to 5 seconds, finalization to 30 seconds, and text to 256 KiB. Workers derive from the recording/application context and drain/zero capture buffers on cancellation.

A v0.1.0 limitation is that a revised hypothesis can be sent as a full replacement in a delta event without a replacement marker. Preview text can therefore repeat or be revised. The authoritative final corrects it. Freehand does not guess replacement boundaries or deliver provisional text.

## Options and ownership

`internal/compatibility` owns the wire contract, `internal/modelprofile` the explicit Nemotron language/vocabulary contract, and `internal/realtime` the transport. `internal/dictation` retains capture, generation, finalization, cleanup, and delivery ownership. Realtime credentials are captured independently from completed STT through the existing settings owner; metadata discovery never invokes a model.

The profile accepts automatic language detection or the published base-model locales, newline-separated vocabulary phrases (32 phrases, 128 UTF-8 bytes each, 2048 total), and finite vocabulary strength from 0 to 5. Vocabulary is recognition biasing, not instruction following or training. Native punctuation is preserved; verbatim output avoids requiring an ITN model. No additional punctuation, diarization, VAD, or translation model is requested. Chunk latency, GPU choice, and model loading remain server configuration.

SQLite migration 00008 adds the independent capability and remembered model options through goose and sqlc. Model selection preserves task language; model preferences never contain transport settings or credentials. Live mode bypasses local silence trimming, checkpoints, and automatic stop without altering those saved preferences. Toggle/hold controls and the recording duration limit remain authoritative.

## Evidence and validation

Protocol authority: [NeMo-Speech.cpp v0.1.0 API](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/api.md), [ASR configuration](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/asr/configuration.md), and [NVIDIA model card](https://huggingface.co/nvidia/nemotron-3.5-asr-streaming-0.6b).

Deterministic fixtures cover configuration, binary audio, final replacement, missing finals, cancellation, credential isolation, and SQLite restart. Live model qualification is manual and specific to the selected deployment; never add inference probes to CI. Native acceptance must separately exercise caption focus behavior, stop/cancel, and safe insertion.
