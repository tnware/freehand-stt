---
title: Live transcription
description: See words while speaking with Nemotron and a user-managed NeMo-Speech.cpp server.
---

Live transcription is an optional microphone mode. Words appear in the results pane while you speak; an optional single-row overlay shows the newest words without taking focus. Stop recording to finalize the transcript, run any enabled cleanup, and insert the result when the original target is still safe.

## Connect your server

1. Run **NeMo-Speech.cpp v0.1.0** with **Nemotron 3.5 ASR streaming 0.6B** loaded on a machine you choose. Freehand does not host the model. Follow the runtime's [installation guide](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/install.md).
2. In Voice's **Transcription** quick controls, or **Settings → Voice transcription**, add a connection. The separate Connection Manager opens. Choose **NeMo-Speech.cpp** and enter its HTTP API base URL, such as `http://127.0.0.1:8088/v1` for a server listening locally on port 8088. Allow HTTP explicitly when appropriate for your server; HTTPS uses a secure WebSocket.
3. Save and use the connection. Choose **Check**, then select the exact loaded model ID reported by the server. This only reads metadata. The server must actually host the qualified Nemotron model; a model name alone does not establish compatibility.
4. Choose **Nemotron 3.5 ASR streaming** under **Model profile**. Choose the spoken language or automatic detection and enable **Realtime transcription** in that same panel. Keep **Live overlay captions** enabled to see the caption strip; the main overlay preference must also be enabled.

The client connects to `/v1/realtime` beneath the selected API root. Local servers configured without authentication need no key. A remotely authenticated deployment can use a stored API key, sent in the WebSocket upgrade header.

## Recognition controls

**Vocabulary hints** favor names and terminology: one phrase per line, up to 32 phrases, with 128 UTF-8 bytes per phrase and 2048 total. **Vocabulary strength** ranges from 0 to 5; start with 2–3. These are recognition hints, not commands or fine-tuning. Use Cleanup for rewrite instructions. The profile exposes only languages supported by the base model, without requiring adaptation packages.

The server owns GPU selection, loaded models, chunk latency, and optional extra processing models. Freehand preserves the model's native punctuation and requests verbatim output; it does not load optional ITN, punctuation, diarization, or VAD models.

## While recording

The preview can change as speech is recognized. NeMo-Speech.cpp v0.1.0 can emit revised hypotheses without marking replacements, so provisional words may occasionally repeat; the final transcript replaces the preview. The overlay stays one row tall and reveals the latest text as older words leave the visible area.

Only finalized text can be cleaned up, retained in optional memory history, copied, or inserted. Cancellation or a disconnected stream discards its preview. Start a new recording after a connection failure; Freehand does not replay captured audio.

Live mode uses the existing toggle/hold shortcut and recording duration limit. Silence trimming, checkpoints, and automatic stop are for completed transcription and are bypassed in live mode. Turning realtime off keeps the same connection and model and uses completed recording/checkpoints with your saved capture preferences. Vocabulary hints and live captions apply only in realtime mode. Audio-file transcription has its own connection, model, language, and options; changing Voice does not change it.
