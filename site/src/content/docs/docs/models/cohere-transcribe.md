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
Accept the model's access conditions on Hugging Face and authenticate the inference
runtime with an account that has access before downloading it. This download login
is separate from any API key used to connect Freehand to your running server.
Serve the model with the runtime's audio dependencies installed:

```sh
vllm serve CohereLabs/cohere-transcribe-03-2026
```

In Freehand, add a **vLLM** connection, select its model in **Voice transcription**
or **Audio-file transcription**, and choose **Cohere Transcribe** as the model
profile, then save. If your server uses a custom alias, select that ID and choose
the same model profile.

## Language and recognition controls

Choose English, French, German, Italian, Spanish, Portuguese, Greek, Dutch,
Polish, Chinese, Japanese, Korean, Vietnamese, or Arabic. **Server default
(English)** leaves the language field unset. It does not ask the model to detect
the language automatically.

You can override temperature. vLLM enables punctuation automatically for this
model; Freehand has no punctuation switch. Context hints and shared vocabulary
are unavailable because this vLLM integration does not apply them. Settings for
the direct Transformers API do not necessarily apply to this server.

Microphone requests complete after a recording or checkpoint. The profile does
not enable realtime microphone streaming. Cleanup and delivery use your existing
workflow settings.
