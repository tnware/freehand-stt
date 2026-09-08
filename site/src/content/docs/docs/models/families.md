---
title: Speech model families
description: Understand speech model families, their serving runtimes, and their place in Freehand.
---

A model family describes speech behavior. A backend describes the server API
Freehand connects to. Different sizes, quantizations, and server aliases do not
automatically need separate Freehand profiles.

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
| Qwen3-TTS 1.7B CustomVoice | vLLM-Omni | [Qwen3-TTS](../qwen3-tts/) |
| S1-mini | llama.cpp, vLLM, compatible chat servers | [S1-mini](../s1-mini/) |

Whisper-family runtimes such as faster-whisper and whisper.cpp are different
implementations of the same recognition family. A GGUF, ONNX, or MLX conversion
also describes packaging rather than a new Freehand behavior profile. The server
can run on any machine reachable from the Windows client.

## Other families to consider

These are pointers into the broader ecosystem, **not additional implemented
Freehand model profiles**. Check the serving API before choosing a connection;
a Hugging Face model download is not itself an API endpoint.

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
through a [Generic connection](../../backends/generic/). Model-specific behavior
needs an adapter that actually transmits the relevant fields and handles the
response. Selecting Generic does not add support for an unrelated streaming API.

## Reading model directories

[Hugging Face's recognition](https://huggingface.co/models?pipeline_tag=automatic-speech-recognition&sort=trending)
and [speech-generation](https://huggingface.co/models?pipeline_tag=text-to-speech&sort=trending)
directories help discover families. Trending and monthly download counts answer
different questions; neither is a complete inventory of compatible servers.
Language-specific derivatives, conversions, alignment, and diarization models
can appear beside general transcription models. Start with the workflow and
controls you want, then choose the model and its serving backend together.
