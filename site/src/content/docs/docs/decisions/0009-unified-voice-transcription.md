---
title: "ADR 0009: Unified Voice transcription"
description: One microphone provider/model selection with an optional qualified realtime mode, independent of audio files.
---

- Status: Accepted
- Date: 2026-09-07
- Supersedes: ADR 0008's separate realtime feature selection and ADR 0002's separate completed-microphone/realtime settings slots. Their transport and final-text safety contracts remain applicable.

## Decision

Voice owns one connection, model, explicit model profile, language, and request-option set. Realtime is a mode of that selection, not another top-level feature or a second active microphone provider. Audio-file transcription owns an independent selection. Cleanup and speech generation remain separate capabilities.

`internal/compatibility` declares backend realtime support, and `internal/modelprofile` intersects it with explicit model behavior. Only a qualified combination exposes the Realtime transcription switch. Changing to an ineligible model or connection disables that mode. Model selection preserves task language; explicitly choosing a profile in quick settings can reset an unsupported language to Automatic with a visible notice.

NeMo-Speech.cpp v0.1.0 supports completed multipart `/v1/audio/transcriptions` as well as its qualified WebSocket adapter. Turning realtime off retains the same connection/model and uses completed recording or checkpoints. Both request forms request native punctuation and verbatim output. Nemotron vocabulary boosting and caption controls are exposed in realtime mode; completed transcription does not apply those hints. Generic does not grant realtime support, and model IDs never select a profile implicitly.

## Ownership and migration

`config.Settings.VoiceTranscription` owns microphone settings. Existing root transcription fields own audio files. `settings.DictationProfiles` captures only the selected Voice credential and adapts an immutable Voice snapshot to the completed request pipeline when needed. `settings.RequestProfiles` captures only the file credential. Neither capture mutates persisted settings; headers are copied. Native capture, finalization, cleanup, and safe insertion keep their existing owners.

SQLite migration 00009 preserves the previously active microphone configuration: the old realtime slot when enabled, otherwise the old shared completed-STT configuration. The file selection stays unchanged. Former completed connections gain a Voice use; former realtime uses become Voice uses. Remembered models and credential references survive, including inactive realtime choices. Legacy JSON import seeds Voice once from its completed settings. Subsequent selections are independent, while a reused connection still deliberately shares its server address and credential.

Voice setup completion and recording readiness no longer depend on an audio-file connection. Selecting an audio-file server does not reset Voice setup. Request admission still requires a model unless the backend owns a server-loaded model. The quick controls and Settings share the same Voice component; the footer reports metadata only for the current task.

## Validation

Deterministic tests cover v8 migration with realtime on and off, retained inactive choices, completed Voice snapshot and credential isolation, mode eligibility, both completed request forms, and file/Voice selection independence. Svelte checks, frontend tests, generated bindings/SQL checks, and a Windows build are required. Native review separately checks connection selection, mode switching, capture, one-row captions, and focus-safe delivery. No automatic inference tests are introduced.

Vocabulary ownership is superseded by [ADR 0010](../0010-shared-vocabulary/): phrase lists and use preferences are shared task settings, including completed NeMo requests.
