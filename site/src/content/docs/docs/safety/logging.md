---
title: Engineering logging contract
description: Contributor rules for structured diagnostics without sensitive content.
---

Use logging contract **version 1** when adding diagnostics or reviewing a new
output surface. Application logs contain operational metadata for diagnosing the
local process.

:::caution[Keep Wails at Info]
Bridge debug tracing can serialize credential drafts, transcripts, history, and
runtime output. Investigate the bridge only with synthetic, non-sensitive data,
then restore `Info` before committing or packaging.
:::

## Ownership and output

| Owner | Contract |
| --- | --- |
| `internal/app` | Creates one hierarchy from Wails' `application.DefaultLogger` |
| Runtime packages | Receive an injected child logger; never create another emitting logger or print to stdout/stderr |
| Component child | Identifies `app`, `wails`, `service`, `dictation`, `postprocess`, `insertion`, or `overlay` |
| `internal/diagnostics` | Provides a non-emitting fallback only for isolated tests or incomplete construction |
| Root `main.go` before logger construction | May print a bounded `error_kind` and exit; never formats the underlying error |

After initialization, `App.Run` owns the structured terminal record; root exits
silently on its returned error.

| Output surface | Availability and retention |
| --- | --- |
| Development application logger | Visible in the terminal; Freehand creates no log file |
| Production application logger | Pinned Wails uses `io.Discard`, including normal Windows releases; no retrievable support log |
| Wails detached update helper | Writes `wails-update-<pid>.log` in the OS temp directory and to stderr; closes but does not delete the file |

Changing application log destinations or adding retention requires a separate
privacy and support decision.

:::caution[Update-helper exception]
The upstream helper is outside this hierarchy. Its swap diagnostics include
installation/staging paths, process IDs, and raw filesystem errors. It receives
no transcripts, audio, or inference credentials, but does not satisfy Freehand's
path/error filtering rules. Review its log before including it in a support
bundle, and account for it in native update acceptance. Changing helper logging
requires a separate upstream integration change.
:::

## Levels and bridge debugging

The application and Wails system logger remain at `Info` in every normal build:

| Level | Use for |
| --- | --- |
| `Info` | Expected lifecycle boundaries and outcomes |
| `Warn` | Degraded behavior with a safe fallback, rejection, or incomplete shutdown |
| `Error` | Failure without the operation's expected result |

Never enable Wails bridge debug logging during real use. A synthetic-data
exception must remain narrowly scoped to the bridge investigation.

## Event and field vocabulary

Meaningful asynchronous operations use a stable noun-and-action message:

```text title="Observable operation lifecycle"
started → exactly one of: completed | failed | cancelled
```

Pair starts with the terminal boundary when the process can observe it. Do not
log synchronous state reads or high-frequency updates.

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

Direct-input terminal records use only `utf16_units`, `batch_count`, `duration_ms`, `strategy`, and, on failure, `stage` plus `error_kind`. They never contain the inserted text or any part of the captured target identity.

## Prohibited content

| Never log | Use instead, where needed |
| --- | --- |
| Credentials, drafts, custom headers | Only authorization state `none` / `stored` / `draft` |
| Raw, provisional, processed, cancelled, or historical transcript text | Bounded counts or sizes |
| Audio, multipart bodies, protocol bodies | Bounded counts or sizes |
| Full paths or selected filenames | A stable operation category |
| Model IDs | Inspect Settings or bounded opt-in history details outside logs |
| URL credentials, paths, queries, or fragments | Parsed host and optional port only |
| Target handles, titles, process identity, clipboard contents | Content-free outcome metadata |
| Raw Go errors, provider bodies, unbounded renderer strings | `diagnostics.ErrorKind` |

Classify errors with `diagnostics.ErrorKind`; do not attach `err`, `err.Error()`, or `%v` to a runtime record. User-facing error messages belong in the existing bounded status/error surfaces. A frontend failure that requires action must be shown there rather than existing only in `console.*` output.

## Managed process output

| Viewer | Admission |
| --- | --- |
| Embedded **Runtime output** tab | Displays bounded child output immediately when opened |
| Standalone **Process output** window | Requires **Show output** consent for each opening or runtime switch |

:::caution[Process output can contain sensitive text]
Both viewers are separate from application logs. Upstream output may contain
prompts, transcripts, paths, and other sensitive content. Terminal filtering and
severity highlighting are not redaction.
:::

Keep raw process output in bounded private memory, with bounded renderer reads.
Only read while the selected viewer and workspace are visible; hiding, switching
runtime, or teardown must revoke reads, clear displayed text, and reject late
results. Do not publish raw output events or forward output to application logs.

| Allowed | Forbidden |
| --- | --- |
| Backend-approved color/progress controls; explicit Copy selection | HTML, output-triggered clipboard writes, links, title changes, or process input |
| Bounded private memory and bounded visible-viewer reads | Raw output events, application logging, file retention, export, or crash-report attachment |
| Viewer controls | Ownership of the runtime process |

Keep Wails at `Info`: bridge debug tracing can serialize sensitive output bindings.
Do not enable upstream trace/debug logging or file/prompt sinks. Preserve the
runtime environment allowlist so inherited configuration cannot enable them.
Sparse output does not establish startup failure. Check launcher flags and
capture limits in the owning source rather than duplicating their values here.

## Noise limits

Do not log PCM callbacks, audio levels, every VAD frame/state oscillation, upload ticks, streamed transcript deltas, renderer events, or routine polling/snapshot reads. Log segment/checkpoint boundaries and whole-operation results instead. The UI and native overlay own live feedback.

## Review checklist

Before merging a new log:

- [ ] Identify the operational question the record answers.
- [ ] Use the injected component logger and established lifecycle message shape.
- [ ] Prefer bounded enums, counts, and correlation IDs over content.
- [ ] Pair starts with one observable terminal result.
- [ ] Add a focused captured-record test for boundaries handling credentials,
  transcripts, URLs, files, or provider errors.
- [ ] Re-run the prohibited-content search and keep Wails at `Info`.

Storage failures follow the same rules: classify them through `DiagnosticKind()`
and `diagnostics.ErrorKind`; never emit raw driver errors, SQL statements or
parameters, database paths, credential account references, or settings content.
