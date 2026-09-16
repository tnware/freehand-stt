---
title: Speech model families
description: Understand speech model families, their serving runtimes, and their place in Freehand.
---

A model family groups related recognition or speech-generation models. Choose
both a model and a backend that can serve it, then select the matching Freehand
profile from the table below. A different size, quantization, or server alias
does not automatically change the profile; check the model guide's requirements.

:::tip[Want Freehand to install the model?]
Use the [managed runtime catalog](../../guides/local-runtime/#choose-a-runtime-and-model).
The families below also include models that require a service you manage yourself.
:::

## Families with a Freehand path

| Family | Backend | Freehand model profile |
| --- | --- | --- |
| Whisper, Distil-Whisper | Speaches, whisper.cpp, compatible transcription servers | Generic |
| Nemotron 3.5 streaming | NeMo-Speech.cpp | [Nemotron](../nemotron/) |
| Parakeet TDT v3 | NeMo-Speech.cpp | [Parakeet TDT v3](../parakeet/) |
| Qwen3-ASR | vLLM | [Qwen3-ASR](../qwen3-asr/) |
| Cohere Transcribe | vLLM | [Cohere Transcribe](../cohere-transcribe/) |
| Voxtral Mini 4B Realtime | vLLM | [Voxtral Mini Realtime](../voxtral-realtime/) |
| Kokoro | Speaches, Kokoro-FastAPI | Generic |
| MagpieTTS Multilingual 357M v2602 | NeMo-Speech.cpp | [MagpieTTS](../magpie-tts/) |
| Qwen3-TTS 1.7B CustomVoice | vLLM-Omni | [Qwen3-TTS](../qwen3-tts/) |
| S1-mini | llama.cpp, vLLM, compatible chat servers | [S1-mini](../s1-mini/) |

<details>
<summary>Family, runtime, or model format?</summary>

Whisper is a model family; faster-whisper and whisper.cpp are different
implementations. GGUF, ONNX, and MLX describe packaging rather than new Freehand
behavior profiles. A different size, quantization, or alias still needs a
compatible backend and model profile. The server can run on any machine
reachable from the Windows or macOS client.

</details>

## Other families to consider

:::caution[Exploration, not a compatibility list]
These families have **no dedicated Freehand model profile**. Check the serving
API before choosing a connection; a model download is not an API endpoint.
:::

| Recognition family | Distinct behavior to explore |
| --- | --- |
| [Microsoft VibeVoice ASR Streaming](https://huggingface.co/microsoft/VibeVoice-ASR-Streaming-1.5B) | Streaming recognition with speaker-attributed output and hotword context. Its plugin server uses a different protocol from Freehand's vLLM adapter. |
| [NVIDIA Canary](https://huggingface.co/nvidia/canary-1b-v2) | Multilingual transcription and speech translation. |
| [SenseVoice](https://huggingface.co/FunAudioLLM/SenseVoiceSmall) | Language identification, text normalization, and additional audio annotations in the FunASR ecosystem. |
| [Kyutai STT](https://huggingface.co/kyutai/stt-2.6b-en) | Streaming recognition with dedicated English and English/French variants. |
| [IBM Granite Speech](https://huggingface.co/ibm-granite/granite-speech-5.0-470m-turboctc) | A speech family with distinct checkpoint architectures, including English TurboCTC. |
| [IndicConformer](https://huggingface.co/ai4bharat/indic-conformer-600m-multilingual) | Dedicated recognition across 22 Indian languages. |

For speech generation, [Chatterbox](https://huggingface.co/ResembleAI/chatterbox),
[Fish Audio S2](https://huggingface.co/fishaudio/s2-pro), and
[OmniVoice](https://huggingface.co/k2-fsa/OmniVoice) offer different expression,
voice, and language controls. XTTS, F5-TTS, CosyVoice, Piper, VoxCPM, and Higgs are
also distinct families worth considering when choosing a speech server.

Some servers can already provide ordinary transcription or speech generation
through a [Generic connection](../../backends/generic/) if they accept its request
fields and response formats. Generic does not enable a model's specialized
controls or an unrelated streaming API.

## Reading model directories

Use Hugging Face's [recognition](https://huggingface.co/models?pipeline_tag=automatic-speech-recognition&sort=trending)
and [speech-generation](https://huggingface.co/models?pipeline_tag=text-to-speech&sort=trending)
directories to discover families, then check:

- **The task:** general transcription, alignment, diarization, and language-specific derivatives can appear together.
- **The serving API:** popularity and download counts do not establish compatibility.
- **The controls:** choose the model and backend together for the workflow you need.
