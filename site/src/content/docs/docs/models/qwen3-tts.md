---
title: Qwen3-TTS
description: Choose preset voices, speech language, and delivery instructions with Qwen3-TTS.
---

Freehand's **Qwen3-TTS 1.7B CustomVoice** profile uses
[Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice](https://huggingface.co/Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice)
through **vLLM-Omni v0.18.0**.

## Choose the model and voice

Follow the [vLLM-Omni connection guide](../../backends/vllm-omni/), then open
**Settings → Text to speech**. Select the connection, choose the served model,
and select **Qwen3-TTS 1.7B CustomVoice** as the model profile.

The voice picker offers the checkpoint's nine preset speakers: Vivian, Serena,
Uncle Fu, Dylan, Eric, Ryan, Aiden, Ono Anna, and Sohee. **Refresh voices** reads
server metadata without generating audio.

## Shape the delivery

- **Speech language:** Automatic, Chinese, English, Japanese, Korean, German,
  French, Russian, Portuguese, Spanish, or Italian.
- **Voice style:** describe the delivery, such as “Speak warmly and clearly,
  with a relaxed pace.” Use up to 500 characters, or leave it empty for the voice's usual style.
- **Speaking speed:** controls the speed of the returned speech.

Choose **Preview** to hear the current edits before saving. The preview uses
your draft voice, model profile, language, style, speed, and timeout together.
It does not save those edits. After saving, the composer and explicit Listen
actions use the same options. Speech quick settings also offer language and style.

Language and style are remembered with this model's voice settings. Switching
models keeps speaking speed in place. Generated audio stays in memory until you
clear it, replace it, quit, or explicitly save a WAV file.

This profile selects the **1.7B CustomVoice** task. Qwen's Base voice-cloning,
VoiceDesign, and 0.6B variants are separate model behaviors; their controls are not
part of this profile.
