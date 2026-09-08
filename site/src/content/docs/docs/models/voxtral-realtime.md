---
title: Voxtral Mini Realtime
description: Follow live dictation with Mistral Voxtral Mini 4B Realtime through vLLM.
---

Use [mistralai/Voxtral-Mini-4B-Realtime-2602](https://huggingface.co/mistralai/Voxtral-Mini-4B-Realtime-2602)
with **vLLM v0.28.0** and Freehand's **Voxtral Mini Realtime** model profile.

## Connect Freehand

Set up [vLLM](../../backends/vllm/) on your inference machine with its audio
dependencies, then serve the checkpoint:

```sh
vllm serve mistralai/Voxtral-Mini-4B-Realtime-2602
```

1. Add a **vLLM** connection using the server's HTTP API root, ending in `/v1`.
2. In **Voice transcription**, choose the connection and the served model ID.
3. Choose **Voxtral Mini Realtime** as the model profile.
4. Enable **Realtime transcription** and optionally **Live overlay captions**.

Freehand derives `/v1/realtime` from the same connection. The recording shortcut
starts microphone streaming; provisional words appear in Current result and,
when enabled, the single-row overlay caption. Stop recording to receive the final
text and apply your usual cleanup and focus-safe insertion.

## Completed audio

Turning realtime off uses the same model for completed recordings and checkpoints.
You can also select this pairing independently for **Audio-file transcription**;
file uploads return completed text. This profile uses automatic language detection
and does not expose context, vocabulary, or temperature controls.

Interrupted realtime sessions do not insert their provisional text. This profile
is specifically for the Mini 4B Realtime checkpoint; other Voxtral variants have
different contracts.

See [vLLM's realtime example](https://docs.vllm.ai/en/v0.28.0/examples/speech_to_text/realtime/)
for the server's microphone protocol.
