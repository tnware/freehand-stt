---
title: "ADR 0015: Optional managed local speech runtime"
description: Own a bounded Windows NeMo lifecycle while preserving existing speech transports and manual connections.
---

- Status: Accepted
- Superseded in part by [ADR 0016](../0016-managed-runtime-connections/): runtime instances and ordinary Connections replace the singleton managed-mode routing and persistence described below. Acquisition and native safety requirements remain in force.
- Supersedes: ADR 0005's prohibition on managed inference and ADR 0008's user-managed-only runtime requirement. ADR 0009's manual Voice/file selections remain independent; managed mode supplies one local model to both workflows.

## Decision

Offer an optional Windows x64 managed runtime using official prebuilt
NeMo-Speech.cpp v0.1.0 archives. Freehand owns installation, a speech-model
catalog, explicit model downloads, process supervision, and removal inside its
per-user application-data directory. Nothing is installed globally or added to
PATH. Freehand does not build NeMo from source or depend on Docker, WSL, Python,
or an external runtime manager.

Recommend Nemotron 3.5 ASR streaming with realtime microphone transcription
selected when the user chooses the recommended managed setup. Completed Voice
transcription remains available; audio files use the existing completed
transcription flow. The manual configuration and its default mode are unchanged.
Realtime partials remain presentation-only. Authoritative finals use existing
cleanup and focus-safe delivery, without automatic replay after a disconnect.

The UI expresses intent: use the managed runtime, select a speech model, choose
realtime when supported, or return to a manual endpoint. It does not expose
arbitrary command-line flags, environment variables, ports, GPU indices, or
backend tuning. The NeMo adapter selects an appropriate published artifact and
lets NeMo handle the supported backend's execution. Manual endpoints remain the
option for custom deployments and advanced flags.

## Ownership

`internal/managedruntime` owns the lifecycle state machine and NeMo adapter.
Windows-specific process creation and process-tree lifetime belong behind that
boundary; other platforms report unsupported without downloading or launching
anything. Wails owns the service context. Install, model pull, metadata commands,
and server work must be tracked children of that context.

`internal/settings` remains the transactional preference and credential owner.
Persist the managed enablement, selected model, and realtime preference through
a forward Goose migration and sqlc queries under ADR 0014. Do not persist live
ports, PIDs, readiness, errors, or progress as settings. Reconcile installation
and downloaded-model metadata on restart.

The settings owner resolves a coherent endpoint/model/profile snapshot for
new operations. Managed requests carry no manual provider key or custom headers.
Saved manual URLs, model choices, profiles, and credentials remain unchanged.
Cleanup and TTS keep their independent selections. The existing completed STT
client and qualified NeMo realtime transport do not acquire process-management
responsibilities.

## Installation and catalog

Pin the runtime version, official archive locations, and SHA-256 digests.
Download and verify before publishing an installation. Reject archive traversal,
links, excessive expansion, and incomplete extraction. Unsupported artifacts or
failed integrity checks are errors, never permission to build from source.

Read the installed binary's model index through its metadata-only CLI. Intersect
that index with Freehand's explicit qualified speech profiles. Browsing never
pulls or loads models. Model acquisition uses NeMo's own `pull` operation and
its pinned artifact validation. Set `NEMO_SPEECH_MODEL_DIR` to Freehand-owned
storage; do not reuse or delete a separate NeMo installation's cache. The model
manager uses the Windows-provided `curl` executable. Report a missing prerequisite
rather than installing another manager.

Selecting and starting one downloaded model authorizes loading that model only.
Do not enumerate the catalog through inference to discover hardware suitability.
Runtime archives may contain other capabilities; their presence does not enable
TTS, translation, diarization, or additional companion downloads.

## Process and network boundary

Bind the child server explicitly to `127.0.0.1` on a dynamically selected local
port. Do not expose a LAN listener or open the bundled playground. Poll `/ready`
with bounded requests and startup time; qualify the loaded model identity through
metadata before publishing an endpoint. A free-port probe alone is not a lasting
reservation: startup must handle bind failure and child exit, rather than accept
another listener's readiness as success.

Own the Windows process tree with a kill-on-close Job Object. Child creation
must not allow descendants to escape ownership before assignment. Stop accepting
work during shutdown, cancel installation and downloads, terminate owned
processes, and wait within a documented bound. Do not use process-name matching
or kill unrelated NeMo servers.

Continuously drain stdout and stderr into bounded private buffers. Do not expose
raw child output to Wails, persist it, or copy it into application diagnostics.
Publish safe phase, progress, and error categories through bounded status events.
Model-loader logs can contain paths or text even when Freehand did not request
verbose logging.

## Failure and privacy

Managed mode is off by default and performs no installation or model download
without an explicit action. Enabling a previously installed, downloaded selection
may start it on launch. Local recognition does not make independently configured
cleanup or speech generation local.

When managed mode fails, retain the manual configuration as a recovery choice,
but fail the active operation. Never silently send captured audio to a remote
fallback, rewrite an in-flight request, or replay realtime audio. Turning managed
mode off explicitly restores manual endpoint selection for subsequent work.
Stopping or switching a loaded model must not silently reroute an active request.

Removal is confined to Freehand-owned runtime/model storage, with explicit
confirmation in the UI. It does not delete manual connections, credentials,
source audio files, or other runtime installations. Existing audio deletion,
history opt-in, credential storage, and insertion rules remain in force.

## Validation

Use deterministic tests for state transitions, cancellation, installation
integrity, cache/path boundaries, model qualification, port/readiness admission,
process-tree shutdown, save/reopen, and manual credential isolation. Cover the
recommended realtime selection and completed file requests with their existing
protocol fixtures. Browser fixtures cover onboarding, catalog actions, progress,
failures, removal confirmation, and returning to manual connections.

Run real Windows process-ownership tests separately from fixture-only lifecycle
tests. User acceptance exercises the official binary, the explicitly selected
model, live captions/finals, completed transcription, exit, and restart. Do not
add automatic inference tests to CI or claim macOS/Linux support from stubs or
cross-compilation.

## Sources

- [NeMo v0.1.0 release archives](https://github.com/NVIDIA/NeMo-Speech.cpp/releases/tag/v0.1.0)
- [Installation and archive contract](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/install.md)
- [Model index, pulls, cache, and backend selection](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/cli.md)
- [HTTP readiness and realtime protocol](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/api.md)
