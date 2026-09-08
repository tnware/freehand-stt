---
title: Cohere Transcribe
description: Use Cohere Transcribe with vLLM for microphone recordings and audio files.
---

The **Cohere Transcribe** profile supports
[CohereLabs/cohere-transcribe-03-2026](https://huggingface.co/CohereLabs/cohere-transcribe-03-2026)
through **vLLM v0.28.0**. It provides completed microphone transcription,
checkpoints, and audio-file results, including streamed file responses.

## Connect the model

Prepare your inference machine using the [vLLM backend guide](../../backends/vllm/).
Serve the model with the runtime's audio dependencies installed:

```sh
vllm serve CohereLabs/cohere-transcribe-03-2026
```

In Freehand, add a **vLLM** connection, select its model in **Voice transcription**
or **Audio-file transcription**, and choose **Cohere Transcribe** as the model
profile. Custom server model aliases work too; profile selection is explicit.

## Language and recognition controls

Choose English, French, German, Italian, Spanish, Portuguese, Greek, Dutch,
Polish, Chinese, Japanese, Korean, Vietnamese, or Arabic. **Server default
(English)** leaves the language field unset. It does not ask the model to detect
the language automatically.

The temperature override is available. vLLM's Cohere adapter builds its own
recognition prefix with punctuation enabled. Freehand does not show a punctuation
switch, context box, or vocabulary toggle for fields that this adapter does not
apply. Its punctuation behavior differs from the direct Transformers API.

Microphone requests complete after a recording or checkpoint. The profile does
not enable realtime microphone streaming. Cleanup and delivery use your existing
workflow settings.
