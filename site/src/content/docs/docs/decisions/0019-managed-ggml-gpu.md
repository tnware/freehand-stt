---
title: "ADR 0019: Explicit Windows GPU execution for managed GGML"
description: Add pinned NVIDIA CUDA runtime binaries without changing model catalogs, Connections, or NeMo policy.
---

- Status: Accepted
- Extends [ADR 0017](../0017-managed-provider-installations/), replacing its CPU-only qualification for llama.cpp and whisper.cpp.
- Retains [ADR 0018](../0018-built-in-runtime-connections/)'s built-in Connections and acquisition ownership.
- [ADR 0020](../0020-managed-runtime-startup-and-diagnostics/) supersedes the initial CPU-only recommendation policy with host-aware qualified recommendations and adds selected-model startup preparation and explicit private diagnostics.

## Decision

Offer CPU and NVIDIA CUDA execution for the Windows x64 managed llama.cpp and
whisper.cpp adapters. Keep CPU as the initial installation choice and preserve
existing installations. GPU use is explicit: different providers can share a
GPU, but Freehand does not evict another provider's model, reserve all available
VRAM, or change NeMo's hardware-selection policy.

Qualify official prebuilt server binaries and required companion libraries as
one pinned installation. Every archive has a fixed URL, size, and SHA-256 digest;
verify all required files before launching. Keep driver/runtime compatibility
checks metadata-only and bounded. Do not install drivers, toolkits, global DLLs,
Python, or external runtime managers. macOS, AMD/Intel acceleration, arbitrary
GPU flags, and user-selected binary paths are outside this decision.

## Installation and ownership

A narrow backend-change request identifies the existing runtime instance and an
allowlisted backend. It does not change the selected model, model cache, task
Connections, credentials, or startup preference. The verified installation
records its backend; this is installation metadata, not a second task-routing
setting or a database migration.

Changing backend requires a stopped runtime and the existing worker's exclusive
operation admission and active-work guards. Download and verify the replacement
before publishing it. Cancellation, integrity failures, and incomplete downloads
must leave the previous installation recoverable. Do not require users to delete
model weights or rebuild their Connections to change execution backend.

CPU launch arguments explicitly disable GPU offload. CUDA launch arguments allow
the qualified GPU path while retaining loopback binding, bounded concurrency and
context, offline model ownership, S1-mini reasoning disabled, readiness checks,
and Job Object supervision. An unavailable selected local runtime never falls
back to a remote Connection. Cleanup keeps its existing raw-transcript fallback.

## Status and validation

Display the installed binary backend separately from provider identity and model
selection. A CUDA label identifies the installed execution capability; it is not
measurement or proof that every model operation is GPU-resident.

Deterministic acceptance covers CPU/GPU replacement, preservation of model data,
cancellation/integrity recovery, rejection while running or busy, and production
launch arguments. Native binary qualification verifies exact archives and
metadata commands with the sanitized production environment. GPU inference
acceptance additionally requires execution of an explicitly selected model on
supported hardware; neither browser fixtures nor successful CLI help establish
that evidence. Do not add model inventory inference tests to CI.
