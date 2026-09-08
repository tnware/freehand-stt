---
title: ADR 0010 — Shared vocabulary
description: Task-owned terminology and capability-gated adapter projections.
---

Status: Accepted.

## Decision

Names and phrases belong to one Vocabulary settings area, independently of reusable connections and remembered model behavior. `config.VocabularySettings` owns the non-secret list, separate Voice/file opt-ins, and the qualified Nemotron boost. Prose context and cleanup instructions retain their existing owners. This supersedes the per-model ownership of vocabulary/hotword text in ADRs 0007–0009.

`modelprofile.VocabularyMode` resolves the selected explicit model and backend contract. `config.PreviewVocabulary` and `config.WithVocabulary` use the same validation for renderer feedback and request admission. Speaches uses hotwords, prompt-capable transcription uses context appended after existing prose, and the qualified NeMo/Nemotron combination uses `speech_contexts` in completed multipart requests or realtime session configuration. The NeMo v0.1.0 API and configuration references in the Vocabulary guide qualify the completed request field. Unsupported combinations omit the shared hints, retaining the user's preference. Supported combinations reject excessive text before audio is sent; no truncation or inventory-based model inference occurs.

`settings` projects hints only into the coherent immutable request snapshot. Existing in-flight jobs, checkpoint sequences, credentials, focus-safe insertion, and cleanup fallback retain their contracts. Request-only completed NeMo fields are excluded from JSON and persistence.

SQLite migration 00010 introduces the singleton vocabulary table. It combines active Voice vocabulary/hotwords and file hotwords, preserves existing use intent and strength, and clears the old active text fields. Historical model rows remain readable for compatibility but model selection cannot restore their phrase fields. Legacy JSON import seeds the same task-owned settings. The library supports 16,384 bytes while each adapter retains its qualified limits. Saving the list, opt-ins, connection settings, and model preferences remains one transaction.

No glossary is injected into S1-mini's fixed training prompt, generic cleanup instructions, or TTS. Supporting those roles later requires a qualified role-specific behavior contract.
