---
title: Parakeet TDT v3
description: Transcribe completed recordings with NVIDIA Parakeet through NeMo-Speech.cpp.
---

Use **Parakeet TDT v3** with the **NeMo-Speech.cpp** backend for microphone
recordings, pause-aware checkpoints, and audio-file transcription. The model
detects the spoken language automatically and produces punctuated text across
25 European languages.

## Set up the model

Install [NeMo-Speech.cpp v0.1.0](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/install.md)
on your inference machine. Its `parakeet-tdt` model entry selects
[nvidia/parakeet-tdt-0.6b-v3](https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3).
Follow the runtime's [model and server instructions](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/asr/models.md)
to load it and expose the HTTP API.

In Freehand:

1. Add a **NeMo-Speech.cpp** [connection](../../backends/nemo-speech/).
2. Select that connection in **Voice transcription**, **Audio-file transcription**, or both.
3. Refresh models and select the loaded Parakeet model.
4. Choose **Parakeet TDT v3** as the model profile, then save.

## What appears in Freehand

This pairing uses completed transcription. Voice retains its recording,
checkpoint, cleanup, and insertion controls. Audio files retain their separate
selection and explicit copy workflow.

The profile uses automatic language detection. It does not offer language hints,
context, vocabulary boosting, temperature overrides, or the realtime toggle:
those controls are not part of this model's NeMo-Speech.cpp API contract.
Shared vocabulary remains saved for other compatible selections.

This profile describes the **TDT v3** checkpoint. Parakeet CTC and RNNT variants
have different behavior and should not use this profile.
