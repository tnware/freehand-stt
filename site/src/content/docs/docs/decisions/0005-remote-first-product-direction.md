---
title: "ADR 0005: Remote-first client product direction"
description: Keep Freehand focused on native access to user-selected speech infrastructure.
---

- Status: Accepted
- Date: 2026-09-03
- Supersedes: ADR 0002 as an active roadmap commitment; retains its protocol and safety research

## Context

Freehand began because the desired speech models and gateways did not need to
run on the Windows client. The useful desktop problem was native capture,
configuration, visibility, and safe delivery against infrastructure already
owned or deliberately selected by the user.

As the application matured, its roadmap accumulated local-runtime research,
realtime transcription, conversation, TTS sequencing, and other capabilities
common to broader dictation and voice-workspace products. Realtime was
implemented experimentally but did not improve final post-processing or safe
insertion enough to justify its transport, configuration, and state surface.

The surrounding open-source category also contains mature local-first model
runners and much broader meeting, notes, and agent products. Feature parity with
all of them is neither a clear identity nor a bounded delivery strategy.

## Decision

Freehand is a lightweight native desktop client for self-hosted and
OpenAI-compatible speech infrastructure.

The desktop application owns:

- audio capture and local VAD/silence policy;
- native shortcuts, status surfaces, and platform permissions;
- independent STT, processing, and TTS capability configuration;
- observable connection and request state;
- bounded recovery/history and safe text delivery.

The configured infrastructure owns:

- model acquisition and lifecycle;
- accelerator selection and setup;
- inference process supervision, scheduling, and batching;
- provider accounts and server-side retention policy.

Localhost, a private LAN server, a user-managed remote host, and a hosted
compatible provider are equal endpoint choices. Freehand does not bundle or
manage inference merely to make localhost easier.

Realtime transcription, conversation, continuous listening, meetings, notes,
agents, and workspace features are not active roadmap milestones. They may be
reconsidered only with product evidence that cannot be satisfied by deeper
interoperability, checkpointed dictation, or reliability work.

Windows remains the only supported runtime. Future macOS and Linux clients must
reuse portable workflows and protocol contracts while adding native adapters
for privileged operating-system behavior. Cross-platform support must not move
capture, credentials, insertion, or other native authorities into the WebView.

## Consequences

- Remote-server setup, evidence-labeled compatibility, diagnostics, native
  reliability, and release quality receive roadmap priority.
- OpenAI compatibility remains a set of independently tested capability
  contracts rather than a claim that every route works on every named server.
- Existing stored-audio STT and explicit TTS remain bounded companion workflows;
  they do not imply a media workspace or voice assistant.
- ADR 0002 remains useful transport and safety research, but no implementation
  should be inferred from it without a new accepted decision.
- Products that bundle a local model may provide an easier first-run experience
  for users who do not already have speech infrastructure. Freehand accepts that
  tradeoff rather than absorbing model management by default.
