---
title: "ADR 0017: One managed installation per provider"
description: List built-in runtimes directly and bound each provider to one installation and running model.
---

- Status: Accepted
- Extended by [ADR 0019](../0019-managed-ggml-gpu/): explicit NVIDIA CUDA execution replaces the initial CPU-only GGML qualification below.
- Extended by [ADR 0018](../0018-built-in-runtime-connections/): automatic built-in Connections and explicit acquisition outcomes.
- Supersedes: ADR 0016's user-created, named instances of the same provider. Its Connections routing, transactional publication, credential isolation, and lifecycle contracts remain in force.

## Decision

Show every implemented provider in the runtime settings list, including providers
not installed yet. Installation is an explicit action on that row, not an Add
runtime dialog. Keep one configured installation and one running process tree
per provider. Different providers may run concurrently; NeMo transcription and
llama.cpp cleanup are independent tasks, not a global managed mode.

The catalog remains first-class. Installing a runtime does not download a model,
and downloading a model does not start it. Stop the runtime before changing its
loaded model. Keep model selection, the active model, and process readiness
separate in status and controls. Saved Connections remain the authority for task
selection.

## Additional Windows adapters

Qualify llama.cpp for S1-mini by Superwhisper cleanup and whisper.cpp for
completed transcription. Reuse the existing model profiles and HTTP clients:
llama.cpp publishes an API base ending in `/v1`; whisper.cpp publishes its
origin for the native `/inference` route. Neither provides realtime or speech
synthesis through these managed adapters.

Use pinned official Windows x64 release archives, verified before extraction,
and a bounded model catalog with immutable revision URLs, byte sizes, and
SHA-256 checksums. These two adapters initially use CPU execution so their
installation does not require CUDA or consume NeMo's GPU allocation. NeMo keeps
its existing hardware-selection policy and upstream model manager. S1-mini
keeps ADR 0001's attribution, English-only policy, trained controls, reasoning
disabled, and raw-text fallback.

GGML acquisition is owned by the runtime adapter, not by an external runtime
manager. No Python, global installation, PATH changes, arbitrary flags, custom
model paths, remote fallback, or inventory inference probes are introduced.
Reuse the bounded worker, cancellation, archive/path validation, loopback
listener ownership, and Windows Job Object boundary. Pin launch arguments and
metadata endpoints against the selected binary rather than assuming all GGML
servers share a protocol.

## Existing installations

Retain stored opaque IDs, paths, model selections, and Connections, including
`nemo-default`. Do not merge or delete duplicate installations automatically:
that could change a task's model or remove downloaded data. Durable validation
continues to load these inventories. Mutation admission rejects new duplicate
identities; installation is blocked until duplicates are explicitly resolved.
The recovery controls expose existing entries so users can stop them, remove
unwanted files, reassign Connections, and delete redundant entries.

A provider-wide process guard covers explicit starts, start-at-launch, and
retiring workers. Retain child ownership until actual exit, not merely until
its endpoint is cleared. Competing starts fail through runtime status; they do
not evict the running model. Preserve existing start-at-launch preferences, but
only one process for a provider can be admitted.

No database migration rewrites IDs or task selections. The existing schema and
settings transaction remain authoritative. Removing runtime files retains the
saved reference for explicit repair.

## Verification boundary

Exercise real adapters with bounded HTTP/process fixtures for integrity,
request routes, model qualification, cancellation, and concurrent provider
ownership. Exercise the exact official archives separately for extraction and
CLI metadata. Neither fixture results nor `--help` establish live inference
acceptance. Native acceptance uses only a model explicitly selected for testing;
model inventories are never automatically invoked in CI.
