---
title: Engineering logging contract
description: Contributor rules for structured diagnostics without sensitive content.
---

This document defines application logging contract version 1. Logs are operational metadata for diagnosing the local desktop process; they are not an audit trail, transcript store, or protocol trace.

## Ownership and output

`internal/app` creates one logger hierarchy from Wails' `application.DefaultLogger`. Child loggers identify the owning component (`app`, `wails`, `service`, `dictation`, `postprocess`, `insertion`, or `overlay`) and are injected into packages that emit diagnostics. Runtime packages must not construct another emitting logger or print directly to stdout/stderr. `internal/diagnostics` provides one non-emitting fallback for isolated tests and incomplete construction only.

The single exception is root `main.go`: construction can fail before the Wails logger exists. That isolated bootstrap path prints only a bounded `error_kind` before exiting. After initialization, `App.Run` owns the structured terminal record and root exits silently on its returned error. The bootstrap path must never format the underlying error.

Development runs show the default logger in the terminal. The application does not create or retain a log file. In the pinned Wails version, `application.DefaultLogger` uses `io.Discard` with the `production` build tag, including normal Windows release builds. These records are therefore available for local development diagnostics, not as a retrievable release-build support log. Changing destinations or adding retention is a separate privacy and support decision.

## Levels and bridge debugging

The application and Wails system logger remain at `Info` in every normal build:

- `Info`: expected lifecycle boundaries and outcomes.
- `Warn`: degraded behavior with a safe fallback, rejection, or incomplete shutdown.
- `Error`: an operation failed without its expected result.

Do not enable Wails bridge debug logging during real use. In the pinned Wails version, debug call tracing serializes binding parameters and results. That can include transient credential drafts, settings, transcript text, and history results. A developer may opt in only with synthetic, non-sensitive data for a narrowly scoped bridge investigation, then restore `Info` before committing or packaging.

## Event and field vocabulary

Meaningful asynchronous operations use a stable noun-and-action message. A start record is paired with exactly one terminal `completed`, `failed`, or `cancelled` record when the process can observe that boundary. Synchronous state reads and high-frequency updates are not logged.

Preferred bounded fields are:

| Field | Meaning |
|---|---|
| `component` | Logger owner from the injected hierarchy |
| `generation` | Opaque operation correlation number, scoped to its workflow owner, not globally unique |
| `workflow` | Parent workflow (`dictation` or `file`) carried into shared post-processing records |
| `segment` | Opaque segment number within a generation |
| `duration_ms` / `latency_ms` | Whole milliseconds; do not introduce competing elapsed-time names |
| `timeout_seconds` | Captured capability request budget; use a capability-prefixed field only when two distinct budgets appear on one start record |
| `outcome` | Small operation-specific result vocabulary |
| `phase` / `stage` / `probe` | Small state vocabulary |
| `error_kind` | Content-free category from `internal/diagnostics` |
| `server` | Parsed host and optional port only |
| counts and sizes | Bounded numeric metadata such as characters, bytes, or model count |

Provider/model identifiers are deliberately absent from diagnostics. A user can inspect the active model in settings and bounded run details in opt-in history without copying those identifiers into terminal output.

### Reading operation outcomes

- Connection tests and voice discovery use `completed`, `failed`, or `cancelled`
  in both the terminal message and `outcome`. Rejected input/missing credentials
  use `Warn`; unsuccessful network/provider operations use `Error`; observed
  cancellation uses `Info`. `duration_ms` covers the whole call, while
  `latency_ms` is the metadata request measurement. A reachable server may still
  have capability/model warnings in the UI: probe completion is not inference
  qualification.
- Post-processing keeps its own `component` and receives the originating
  `workflow` and `generation` through an immutable context value. Match those
  to the dictation or file-transcription owner's generation; do not correlate
  solely by server name or timestamps. Cancellation is `Info`; failure remains
  `Warn` because the owning workflow preserves raw text as its fallback.
- Speech generation covers inference, WAV decoding, native loading, and the
  initial playback handoff. A failed terminal identifies the bounded `stage`
  (`inference`, `decode`, `load`, or `play`). Successful generation and playback
  are separate lifecycles; playback completion uses `outcome=played`. Restart
  creates a playback generation without another generation/inference lifecycle.
  Stop, replacement, Clear, and shutdown terminate an active playback monitor
  as `cancelled`, not as a new completion of already-finished audio. Device
  names are not emitted.
- A late file or speech worker may record cancellation after shutdown was
  requested, without publishing UI state or delivering its result. A deadline
  record means the owner stopped waiting, not that a blocked native call ended.
  No terminal record can be promised after process exit or for a call that
  never returns. A successfully completed stage can still be discarded by a
  subsequently cancelled parent workflow.

Direct-input terminal records use only `utf16_units`, `batch_count`, `duration_ms`, `strategy`, and, on failure, `stage` plus `error_kind`. They never contain the inserted text or any part of the captured target identity.

## Prohibited content

Never log:

- credentials, credential drafts, authorization state beyond `none`/`stored`/`draft`, or custom headers;
- raw, provisional, processed, cancelled, or historical transcript text;
- audio samples or multipart/protocol bodies;
- full file paths or selected file names;
- model identifiers;
- URL credentials, paths, queries, or fragments (log only the parsed server host);
- target window handles, titles, process identity, or clipboard contents;
- raw Go errors, provider response bodies, or unbounded renderer-controlled strings.

Classify errors with `diagnostics.ErrorKind`; do not attach `err`, `err.Error()`, or `%v` to a runtime record. User-facing error messages belong in the existing bounded status/error surfaces. A frontend failure that requires action must be shown there rather than existing only in `console.*` output.

## Noise limits

Do not log PCM callbacks, audio levels, every VAD frame/state oscillation, upload ticks, streamed transcript deltas, renderer events, or routine polling/snapshot reads. Log segment/checkpoint boundaries and whole-operation results instead. The UI and native overlay own live feedback.

## Review checklist

When adding a log:

1. Identify the operational question that the record answers.
2. Use the injected component logger and the established lifecycle message shape.
3. Prefer a bounded enum/count/correlation ID over content.
4. Pair starts with one observable terminal result.
5. Add a focused captured-record test when the boundary handles credentials, transcript content, URLs, files, or provider errors.
6. Re-run the prohibited-content search and keep Wails at `Info`.

SQLite failures use bounded configuration categories such as `locked`,
`newer_schema`, `migration_failed`, `backup_failed`, and `commit_uncertain`.
Storage exposes these through `DiagnosticKind()` so wrapped save/recovery
errors retain their category in `diagnostics.ErrorKind`. The shared allowlist
also preserves inference admission and empty/unexpected response categories;
unknown classifications remain `operation`, never arbitrary error text.
Never log raw driver/goose errors, SQL statements or parameters, database paths,
credential account references, or custom settings content. Settings-service
start/outcome logs cover saves and explicit recovery without query tracing.
