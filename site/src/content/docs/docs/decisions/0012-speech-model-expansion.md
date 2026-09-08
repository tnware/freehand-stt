---
title: "ADR 0012: Explicit speech family contracts"
description: Extend existing transcription transports and add Qwen3-TTS speech options without conflating models and backends.
---

- Status: Accepted
- Date: 2026-09-08
- Extends: ADRs 0007, 0009, and 0011; preserves the remote-first boundary.

## Decision

Add explicit Parakeet TDT v3, Cohere Transcribe, Voxtral Mini Realtime, and
Qwen3-TTS 1.7B CustomVoice profiles. Keep selection independent of server IDs,
URLs, and model inventories. Existing inference servers remain user-managed.

Parakeet uses the NeMo-Speech.cpp v0.1.0 completed transcription path with
automatic detection and no language hint or vocabulary projection. Cohere uses
vLLM v0.28.0 completed transcription and file responses, its 14-language set, and
the adapter's English default. Its prompt builder fixes punctuation on and
ignores general context, so neither control is offered.

Voxtral uses vLLM v0.28.0 for completed audio and realtime microphone sessions.
It shares ADR 0011's session setup, PCM16/base64 transport, stop, bounded drain,
and authoritative-final rules. Parsing is selected by model profile: only Qwen
strips recognized Qwen language headers. Voxtral preserves those literal words.
Its initial Freehand contract uses automatic language detection without extra
recognition hints or file-response streaming.

Enable vLLM-Omni v0.18.0 as a speech backend. The standard speech path requests
buffered PCM16 WAV. Its voice metadata endpoint returns string IDs. The explicit
Qwen profile sends `task_type: CustomVoice`, a canonical language name, and optional
`instructions`. The profile owns the nine preset voice IDs exposed to the UI and
enforced by Go. Base, VoiceDesign, uploaded voices, and other checkpoint variants
are separate contracts.

## State and persistence

`modelprofile.SpeechOptions` contains bounded non-secret language and delivery
instructions. They are model options because these fields and their interpretation
depend on the selected speech profile. Speaking speed remains task-owned under
ADR 0007. SQLite migration 11 adds the options to active speech settings and
remembered model rows. The existing settings transaction saves both together;
released migrations are unchanged.

The renderer preview DTO carries the same speech options as the saved workflow.
Go validates them before credential access, captures the saved connection and
credential coherently, and passes an immutable request snapshot to inference.
Preview does not save. Discovery stays metadata-only and never invokes models.

## Evidence and acceptance

Protocol sources: [NeMo model contracts](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/asr/models.md),
[Cohere prompt construction](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/model_executor/models/cohere_asr.py),
[Voxtral realtime implementation](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/model_executor/models/voxtral_realtime.py),
[vLLM-Omni speech request schema](https://github.com/vllm-project/vllm-omni/blob/v0.18.0/vllm_omni/entrypoints/openai/protocol/audio.py),
and [Qwen3-TTS checkpoint behavior](https://huggingface.co/Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice).

Fixtures assert multipart fields, speech JSON, voice metadata, unsupported-option
rejection before network, realtime final authority and failure behavior, preview
snapshot isolation, and real SQLite upgrades/rollback. Native runtime acceptance
and actual model inference are recorded separately from fixture and build results.
No inference tests are added to CI.
