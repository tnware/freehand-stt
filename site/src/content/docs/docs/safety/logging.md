---
title: Engineering logging contract
description: Contributor rules for structured diagnostics without sensitive content.
---

This document defines application logging contract version 1. Logs are operational metadata for diagnosing the local desktop process; they are not an audit trail, transcript store, or protocol trace.

## Ownership and output

`internal/app` creates one logger hierarchy from Wails' `application.DefaultLogger`. Child loggers identify the owning component (`app`, `wails`, `service`, `dictation`, `postprocess`, `insertion`, or `overlay`) and are injected into packages that emit diagnostics. Runtime packages must not construct another emitting logger or print directly to stdout/stderr. `internal/diagnostics` provides one non-emitting fallback for isolated tests and incomplete construction only.

The single exception is root `main.go`: construction can fail before the Wails logger exists. That isolated bootstrap path prints only a bounded `error_kind` before exiting. After initialization, `App.Run` owns the structured terminal record and root exits silently on its returned error. The bootstrap path must never format the underlying error.

Development runs show the default logger in the terminal. The application does not create or retain a log file. Production output follows Wails' default logger behavior; changing destinations or adding retention is a separate privacy and support decision.

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
| `generation` | Opaque operation correlation number |
| `segment` | Opaque segment number within a generation |
| `duration_ms` / `latency_ms` | Whole milliseconds; do not introduce competing elapsed-time names |
| `timeout_seconds` | Captured capability request budget; use a capability-prefixed field only when two distinct budgets appear on one start record |
| `outcome` | Small operation-specific result vocabulary |
| `phase` / `stage` / `probe` | Small state vocabulary |
| `error_kind` | Content-free category from `internal/diagnostics` |
| `server` | Parsed host and optional port only |
| counts and sizes | Bounded numeric metadata such as characters, bytes, or model count |

Provider/model identifiers are deliberately absent from diagnostics. A user can inspect the active model in settings and bounded run details in opt-in history without copying those identifiers into terminal output.

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
