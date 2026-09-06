---
title: "0007 — Task state and preference ownership"
description: Current work, optional retention, and which settings follow a model.
---

Status: Accepted. Date: 2026-09-06.

## Decision

Freehand remains dictation-first, with independently usable file transcription and
text-to-speech companions. Readiness and controls belong to the task selected.
A missing microphone cannot block file work or text-to-speech.

Unfinished composer text belongs to the current WebView session. Each transcription
owner keeps one current result for inspection and copying, independently of optional
recent history. These are in-memory lifetimes, not new persistent workspaces.
Current dictation is released on the next recording, Clear, cancellation, or exit;
current file work is released on clear/replacement, a new transcription, or exit.

Connections own endpoints, authentication, and backend API contracts. Model
preferences restore explicit behavior profiles, provider decoding/generation
options, recognition hints/hotwords, and model voice IDs. Task settings own language,
cleanup instructions, S1-mini style/structure/context, and speaking speed. A model or
connection switch preserves task settings; model constraints remain enforced at
request admission and preserve raw transcription fallback.

## Storage compatibility

The existing SQLite remembered-model snapshot contains some task fields. Preserve
the released migrations and readable format. Those historical fields are not
selection authority: `modelsettings.Select` restores engine fields and preserves
active task settings. `Apply` remains the full snapshot decoder used in validation.
The renderer follows the same selection contract. No new preset layer or database
reset is needed, and opening this version does not change active task preferences.

## Consequences

Selecting another engine cannot unexpectedly switch transcription language or
cleanup intent. Provider-specific hints remain scoped because sending unsupported
hotwords or request fields would violate the qualified backend contract. A generic
cleanup instruction and S1-mini's trained controls remain distinct representations;
Freehand does not translate arbitrary instructions into trained controls.

Quick changes on Home and active connection selection apply immediately. Feature
settings use explicit Save/Discard. In-flight requests remain immutable, and
simplifying presentation never weakens backend validation or insertion safety.
