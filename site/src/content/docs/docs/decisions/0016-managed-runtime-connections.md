---
title: "ADR 0016: Managed runtime instances and Connections"
description: Separate built-in runtime lifecycle from per-task connection selection while preserving the qualified NeMo adapter.
---

- Status: Accepted
- Partly superseded by [ADR 0018](../0018-built-in-runtime-connections/): configured runtimes supply built-in Connections automatically rather than requiring managed connection creation.
- Partly superseded by [ADR 0017](../0017-managed-provider-installations/): one installation and running model per provider replaces user-created duplicate instances. Connections routing and safety contracts remain in force.
- Supersedes: ADR 0015's singleton managed-mode routing and persistence decisions. Its acquisition, process ownership, privacy, and native qualification requirements remain in force.

## Decision

Represent managed runtimes as independently supervised instances. Use saved
Connections as the sole authority for selecting the endpoint for Voice, audio
files, cleanup, and speech generation. A managed connection references an
instance; it does not store a localhost URL, dynamic port, or credentials.

Keep runtime installation, the model catalog, downloads, loading, Start/Stop,
and removal in a dedicated settings area. Connections links to that area and
allows a managed instance or a manually configured endpoint to be selected for
an appropriate task. The ordinary quick-settings connection picker uses the
same saved identities. Stopping a selected instance leaves that connection
selected and unavailable; it does not select a remote replacement.

This decision defines the multi-adapter cutover. It does not qualify additional
runtime binaries. NeMo-Speech.cpp remains the first implementation. A future
whisper.cpp transcription instance and a llama.cpp cleanup instance must fit
without adding a second routing authority. S1-mini is a cleanup model profile
served by llama.cpp, not a runtime adapter of its own.

## Runtime ownership

An instance definition contains a stable opaque ID, display name, built-in
provider ID, provider-qualified catalog model key, and explicit start-at-launch
preference. One instance owns one loaded model and one process tree. Bound the
inventory and each worker's operations; supporting multiple running instances
does not introduce a general inference scheduler.

The managed core owns operation admission, cancellation, status publication,
generation fencing, process ownership, and shutdown. Built-in adapters own
release artifacts and checksums, acquisition, model metadata, supported
platforms, launch arguments, readiness, and qualified per-role contracts.
Register only implemented adapters. Do not introduce dynamic plugins, arbitrary
launch commands, or a flags editor.

Model acquisition remains adapter-specific. NeMo uses its own model manager;
that requirement is not imposed on future whisper.cpp or llama.cpp adapters.
Downloads and model loading remain explicit. Catalog inspection never starts a
server or invokes inference.

Provider identity, API compatibility, and model behavior are separate values.
Qualification resolves all three for a particular role. A binary's unrelated
capabilities do not authorize offering those roles in Freehand. In particular,
the initial NeMo adapter supports qualified transcription, not cleanup or TTS.
Catalog model keys remain distinct from the exact model identity advertised by
the running API.

Each instance has an isolated owned directory. Preserve the original NeMo
installation at `managed-runtime` for the migrated `nemo-default` instance.
Allocate additional instances in sibling `managed-runtimes/<instance-id>`
directories, never beneath the old removable root. Removing one installation
must not remove another instance's files or affect a user-managed installation.

All workers derive their contexts from the Wails application lifetime. During
shutdown, reject new operations, cancel and terminate every owned worker, then
wait against one overall deadline. Do not multiply the shutdown budget by the
number of instances. Reserve mutation admission before checking active work,
and reject stale definitions or generations when resolving an endpoint.

## Connections and task settings

A saved connection contains either manual transport details or a managed
instance reference. Validate the alternatives exclusively in Go and SQLite.
Managed connections cannot contain a URL, health path, custom headers,
credential reference, or credential draft. Reject a credential draft before
entering the native credential transaction, not after storing it.

Connections retain their declared supported uses and independent task
selections. Validate managed uses against the referenced instance's qualified
model and adapter. Referenced instances cannot be deleted until their
connections are removed or explicitly reassigned. Removing runtime files may
leave an instance and its connections present but unavailable for repair.

The instance chooses what model to load. Task settings retain language,
timeout, vocabulary opt-ins, captions, cleanup intent, and supported task/model
options under their existing ownership contracts. Realtime is a capability-
gated Voice mode, not a runtime enablement preference. Remembered model options
must not change an instance's load target through task-side restoration.

Project managed references and qualified behavior into editable settings
without inventing a URL or persisting a resolved endpoint. Separate durable
reference validation from request-time transport validation. Manual connection
changes retain the existing model-selection and credential behavior.

## Request admission and failures

The settings transaction owner captures the selected connection, referenced
instance definition, and ready endpoint coherently with each request. Resolve
only the roles the operation will use: an unavailable managed file connection
must not block independently configured manual Voice.

Build managed transport snapshots from fresh credential-free values. Admit
only the exact owned loopback endpoint and qualified role/model contract. Do
not overlay a localhost URL onto a manual transport with headers or a key.
Resolve independently selected cleanup in the same capture; never fetch a new
cleanup selection or credential after transcription finishes. Unavailable
cleanup preserves the existing durable raw-text fallback rather than silently
choosing another peer.

Connection health checks use the same settings-owned resolution boundary.
They may inspect readiness and model metadata, but must not implicitly start
an instance, acquire a model, or run inference. Distinguish unavailable runtime
state from unavailable credentials.

Present selected model, active model, installation state, loading/readiness,
and request failure separately. A ready server is not proof of successful
transcription, and a selected model is not necessarily loaded. Report device
or memory information only when actually measured. Failures never authorize
remote fallback, audio replay, model substitution, or application activation.

## Persistence transition

Use a forward Goose migration in the existing `freehand.db` lineage with
sqlc-generated queries. Keep prior migrations unchanged. Do not import alpha
persistence or inspect legacy native credentials.

For an initialized singleton-runtime database, retain its selected NeMo model
as an instance. Map prior enablement to start-at-launch. If managed mode was
enabled, create and select a managed connection for Voice and files and carry
the old realtime preference into Voice. Retain all manual connections,
credentials, languages, and remembered model options. If disabled, retain the
inactive instance without changing task selections or adding an automatically
selected connection.

Fresh initialization still has an empty instance and connection inventory. The
unconditional default row introduced by migration 00002 alone is not evidence
of user setup; use the existing initialized preferences record to distinguish
an upgrade. Remove singleton preference persistence after conversion rather
than maintaining two configuration authorities. Reuse verified existing assets
without downloading them as part of migration.

## Validation boundary

Tests must detect concrete routing, ownership, or recovery failures. Exercise
real SQLite upgrades and transactions, managed credential rejection before
native storage, wrong-role admission, independent instance stop/change,
stale-definition fencing, and adapter endpoints through production clients.
Use external-boundary fixtures where live infrastructure would otherwise be
required. Do not assert source strings, private call sequences, or whole markup
as evidence of these contracts.

Browser checks exercise selecting local and remote connections, opening the
linked runtime manager, starting a stopped instance from quick settings, and
recovering without an automatic connection change. Official-binary installation,
model integrity/loading, live transcription, and native process-tree acceptance
remain separate Windows evidence. No automatic inference belongs in CI.
