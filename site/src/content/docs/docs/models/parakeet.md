---
title: Parakeet TDT v3
description: Transcribe completed recordings with NVIDIA Parakeet through NeMo-Speech.cpp.
---

Use **Parakeet TDT v3** with the **NeMo-Speech.cpp** backend for microphone
recordings, pause-aware checkpoints, and audio-file transcription. The model
detects the spoken language automatically and produces punctuated text across
25 European languages.

## Set up the model

For local use on Windows or macOS, follow [managed NeMo setup](../../guides/local-runtime/#set-up-local-transcription),
but choose **Parakeet TDT v3** from the catalog instead of Nemotron. Download and
start it, then select the built-in NeMo Connection for Voice, Audio file, or both.
Freehand supplies the model profile; Parakeet uses completed transcription only.

For a manually configured service, install [NeMo-Speech.cpp v0.1.0](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/install.md)
on your inference machine. Its `parakeet-tdt` model entry selects
[nvidia/parakeet-tdt-0.6b-v3](https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3).
Follow the runtime's [model and server instructions](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/asr/models.md)
to load it and expose the HTTP API.

Connect that manual service in Freehand:

1. Add a **NeMo-Speech.cpp** [connection](../../backends/nemo-speech/).
2. Open the **Settings** cog in Voice transcription or Audio file and select that connection
   in **Transcription** options. Configure each workflow independently if you use both.
3. Refresh models and select the loaded Parakeet model.
4. Choose **Parakeet TDT v3** as the model profile, then save.

## What appears in Freehand

Microphone transcription completes when you stop recording or at a pause-aware
checkpoint. You can use cleanup and focus-safe insertion as usual. Select the
model separately for audio-file transcription, then copy the result when ready.

**Spoken language** shows **Automatic detection** with an explanation instead of
a language dropdown. Parakeet recognizes its 25 supported languages automatically;
it cannot be forced to use a particular language like Nemotron. See
[NVIDIA's model card](https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3)
for the supported languages.

The profile does not offer language hints,
context, vocabulary boosting, temperature overrides, or realtime microphone
streaming with NeMo-Speech.cpp.
Shared vocabulary remains saved for other compatible selections.

This profile describes the **TDT v3** checkpoint. Parakeet CTC and RNNT variants
have different behavior and should not use this profile.
