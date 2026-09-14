---
title: "ADR 0018: Built-in runtime Connections"
description: Project configured runtimes into the ordinary connection catalog without a second setup or routing path.
---

- Status: Accepted
- Supersedes: ADR 0016's manually created managed Connections. ADR 0017's provider inventory, acquisition, and process limits remain in force.

## Decision

Every configured managed runtime supplies a built-in Connection in the existing
connection catalog. Users install, download, and start a runtime, then select
its Connection directly in a compatible task. There is no separate managed
connection creation form or "Connect to a task" setup step. Runtime management
lives under Connections & vocabulary in Settings.

The built-in connection identity is stable and derived from the saved runtime
identity, not the current port, loaded model, or running state. Keep the row
visible when stopped or missing files so selected tasks remain explicit and
repairable. Running makes it usable; stopping never redirects a task to a
manual server. Selection itself remains an explicit user action.

Runtime-owned name, model, backend, transport, authentication, and qualified
uses are not editable connection fields. The Connections detail view exposes
status, runtime management, and supported task navigation instead. Task-owned
language, realtime mode, cleanup behavior, and model-owned options retain their
existing owners and validation. The runtime catalog controls the loaded model.

## Persistence and compatibility

Project built-in rows into the single storage connection state used for catalog
reads, selection, request capture, and credentials. Materialize them through
the existing sqlc queries in ordinary settings transactions to preserve selection
and remembered-model foreign keys. Do not add a second routing registry or a
runtime-event persistence path. Built-in rows do not consume the manual
connection quota. Their identity and ownership are derived by Go, never trusted
from a renderer flag.

Preserve existing managed connection aliases, opaque IDs, task selections, and
remembered options. Do not silently merge, rename, or delete them. New managed
aliases are rejected; new manual server connections are unchanged. Built-in
rows cannot be renamed, duplicated, edited, or deleted through the connection
mutation boundary. Removing a runtime requires explicit deselection and
resolution of any legacy references first.

## Acquisition feedback

An asynchronous request receipt is not completion. The runtime owner publishes
a bounded operation identity, operation kind, acquisition phase, byte counters,
and terminal outcome. Only the verified operation result can report success;
100% transferred still permits verification failure. Cancellation and failure
remain distinguishable from success.

GGML adapters count bytes written during acquisition. NeMo remains on its own
model manager; the pinned release suppresses progress output for redirected
execution, so Freehand observes file-size metadata only for that selected
model's exact owned partial/final paths while the child runs. No diagnostic text,
paths, or child output cross into renderer progress events. No model inventory
is downloaded or invoked for progress qualification. Unknown-size phases use an
activity indicator, not invented percentages.

## Verification

Exercise automatic catalog membership, task selection and save/reopen, runtime
model changes, stale or unavailable request rejection, immutable built-in
connection fields, legacy aliases, manual connection limits, and credential
isolation at the real settings/storage boundaries. Progress tests cover transfer,
verification, cancellation, and failure through worker notifications. Browser
checks prove built-in task selection without entering a URL and progress that
changes with events without claiming success at full transfer. Native download
and inference acceptance remain separate from deterministic fixtures and builds.
