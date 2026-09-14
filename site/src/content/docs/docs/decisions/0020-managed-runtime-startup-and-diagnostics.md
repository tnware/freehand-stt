---
title: "ADR 0020: Managed runtime selection, startup, and private diagnostics"
description: Separate platform recipes from model contracts, prepare selected GPU models during startup, and expose process output only through an explicit diagnostics viewer.
---

- Status: Accepted; implemented checkpoint, native acceptance pending
- Extends [ADR 0019](../0019-managed-ggml-gpu/), superseding CPU as the only initial recommendation for managed GGML installations.
- Extends [ADR 0015](../0015-managed-local-speech-runtime/)'s private-output boundary with the narrow, user-authorized viewer described below.
- Preserves [ADR 0018](../0018-built-in-runtime-connections/)'s identity, selection, and ownership contracts.

## Platform recipes and installation intent

Keep provider/model behavior separate from executable deployment. A provider's
qualified recipes identify operating system, architecture, accelerator, pinned
archives, executable layout, and required dependencies. Platform adapters own
host metadata and native process supervision. Model catalogs and saved task
Connections do not select native binaries.

For a new installation, provide a metadata-only recommendation with an explanation
before the user downloads anything. Intersect detected capabilities with qualified
recipes; detection is not permission to execute models or obtain an unpinned
upstream release. Unknown hardware must produce a conservative supported choice
or an explicit unsupported result, not an optimistic GPU claim. Preserve explicit
backend overrides and use the same compatibility rules for recommendation and
installation admission.

An installed backend remains authoritative until the user explicitly replaces it.
Do not auto-switch on startup, overwrite an existing CPU installation because a
GPU becomes visible, or change backend as available memory fluctuates. Existing
binary replacement remains stopped, transactional, and model-cache preserving.
No GPU memory reservation or eviction of another provider is introduced.

The initial implementation qualifies Windows x64 only. Future macOS work supplies
verified platform recipes, hardware metadata, process ownership and native
acceptance. It must not require changing Connection identity or task routing.
A recipe is not platform acceptance: do not enable macOS with placeholder assets,
assumed Metal support, or successful cross-compilation alone. NeMo retains its
provider-specific model-manager and hardware policy.

The implemented Windows GGML recommendation uses metadata for NVIDIA device 0:
driver >=551.78 and compute capability >=5.0 qualify the pinned CUDA 12.4 recipe;
unknown or unsupported metadata recommends CPU. `GetBinaryOptions` does not
inspect or mutate existing installations. The UI presents Auto/CPU/CUDA and
requires acceptance before `InstallBackend`; the legacy `Install` operation
retains its CPU default. This selects among pinned recipes, not the latest
upstream release. NeMo binary selection is unchanged.

## Preparing a selected runtime

Starting a GPU runtime includes its qualified warm-up work before publishing an
endpoint for user requests. Prefer upstream initialization that precedes readiness.
If a provider requires an explicit warm-up operation, it is limited to the selected,
already-installed model on the owned loopback process, with bounded synthetic
input, discarded output, cancellation and a deadline. It never uses user audio,
enters history or cleanup, downloads another model, or falls back to another server.

This work belongs to process startup, including an explicitly saved start-at-launch
preference. It does not belong to metadata health checks, catalog browsing or model
discovery. Never iterate model inventories for warm-up. A failed or cancelled
warm-up must not leave the instance marked running or release process ownership
before the child has actually stopped.

Report actual startup boundaries and elapsed time: verification, process launch,
waiting for readiness, and warm-up where observable. An upstream readiness interval
may encompass loading and warm-up without exposing their individual boundaries;
label that interval honestly. Do not infer completion percentages from elapsed time
or arbitrary process text. First-use performance remains a native measurement,
not a guarantee derived from an HTTP health response.

The implemented GPU llama.cpp and NeMo paths enable upstream built-in warm-up;
CPU retains `--no-warmup`. CUDA whisper.cpp performs one multipart `/inference`
request with one second of synthetic silence after readiness, against the
already-loaded selected model, and discards its bounded response. Runtime/model
hashing before launch is cancellable. After process creation, readiness and warm-up share a
120-second deadline plus a four-second owned-process drain on failure or
cancellation; process ownership cannot be released merely because a wait expires.

`startupProgress` exposes phase-specific elapsed time for `verifying_runtime`,
`verifying_model`, `launching`, `waiting_ready`, `loading_warming`, and
`warming_up` as applicable. Combined upstream loading/warm-up remains one phase
when the boundary is unobservable.

## Explicit process-output viewer

Offer a read-only auxiliary window from local runtime management, available during
startup as well as while running. It observes the existing process and never owns
its lifetime: opening, closing, clearing, or pausing scrolling cannot start, stop,
restart or orphan the runtime.

Keep a bounded private rolling stdout/stderr tail for each runtime process. Preserve
separate strict bounded capture where command parsers require complete metadata;
a rolling tail must not turn truncated command output into valid metadata.
Scope output to a process generation and discard stale callbacks across restarts.
Retain only bounded process-tail data after exit for diagnosis; release it on
the next start attempt, runtime removal, and application shutdown.

Raw output may contain prompts, transcripts, paths or other sensitive values.
Display a clear warning and require explicit viewer access before exposing it.
This is a narrow exception to the usual prohibition on raw child output crossing
the renderer boundary, not permission to add it to status events, Wails bridge
tracing, application logs, crash reports, clipboard automation, or persistent files.
Do not promise comprehensive redaction of arbitrary upstream text.

Use bounded request/response reads while the authorized viewer is active, with
limited polling and no overlapping reads or unbounded renderer accumulation.
Revoke access and clear visible output when the viewer closes. Render text only;
never interpret HTML, terminal escape commands or clickable instructions. No shell
input, remote shell, automatic export or disk-log retention is introduced.

Upstream logging must be qualified separately from application diagnostics. Keep
Wails at Info and application logs content-free. Enable only the provider output
needed for this explicit private diagnostics capability; broad bridge debug
logging is not a substitute.

The implemented viewer is reusable and available from runtime management and
quick controls, including during startup. Private capture is enabled by default,
with limits of 256 KiB, 1,024 chunks, and 4 KiB per chunk. A separate bounded
prefix remains for existing diagnostic parsers, not for viewer reads. Cursor
deltas require consent per opening or runtime switch. Closing/switching clears
visible data and revokes retrieval; disabling access does not erase private
memory. Clear discards the tail. Follow/pause controls scrolling only, and no
Copy/export is offered.

### Superseding clarification: normal llama.cpp logging

The initial checkpoint retained llama.cpp `--log-disable` and treated sparse
output as expected. That **logging-suppression decision is superseded** by the
user's explicit authorization to capture normal, non-debug upstream logs in the
existing bounded private memory. It is not authorization for persistent logs,
application-log forwarding, raw events, or consent-free viewer reads.

For the pinned Windows CPU and CUDA llama.cpp `b10809` recipes, replace
`--log-disable` with `--log-verbosity 3 --log-colors off`. Upstream level 3 admits
normal info, warnings, and errors, not trace/debug. The common logger's file
sink defaults to null; neither `--log-file` nor `--log-prompts-dir` is supplied.
The fixed sanitized environment remains mandatory: it excludes logging
overrides and `APPDATA`/`PROGRAMDATA` config discovery, which upstream processes
before command-line arguments. See the repository's
`internal/managedruntime/testdata/ggml-startup-source.md` for pinned source and
model-free owned-process qualification.

Normal output can still include prompts, transcripts, paths, and upstream
exception text. Capture remains private even while the viewer is closed;
consent, capacity, lifetime, inert rendering, and all prohibited destinations
above are unchanged. Wails remains at Info and application logs content-free.
Whisper verbosity is not enabled. Output volume is never a readiness signal;
level 3 omits low-level trace details and upstream exit can lose unflushed logs.
These flags apply on subsequent ordinary launches; the change does not restart
or modify an already-running user runtime.

## Validation boundary

Exercise host/recipe selection, unsupported and unknown metadata, explicit
replacement preservation, selected-model warm-up admission/cancellation, process
ownership, output capacity/cursors/generation fencing, opt-in reads, and viewer
teardown with deterministic tests at the real ownership boundaries. Browser
fixtures establish presentation and interaction only. Native checks establish
startup behavior, first and subsequent selected-model request timings, window
lifecycle and shutdown on the supported OS. No automatic inference runs belong
in CI.

The prior CPU/CUDA checkpoint has user-reported Windows testing, not native
acceptance of these additions. Host recommendation, GPU warm-up and first-use
behavior, startup-phase presentation, and native viewer/window shutdown remain
pending. Implementation-agent checks did not perform live inference. Keep
results in the issue or pull request, distinguishing deterministic tests/builds
from the existing [native runtime checks](../../safety/native-test-checklist/#managed-local-runtime).
macOS has no qualified managed packages or native acceptance in this checkpoint.
