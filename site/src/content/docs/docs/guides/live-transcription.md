---
title: Live transcription
description: Set up realtime microphone transcription and review words while you speak.
---

Live transcription streams microphone audio to your selected server while you
speak. Review the words in Freehand or turn on overlay captions to see them
without leaving your app. Stop recording to finalize the transcript and run any
enabled cleanup. Automatic insertion still requires the original destination
to be focused; manual-copy mode leaves the result ready to copy.

## Choose a model and backend

| Model profile                                        | Backend                                        | Recognition controls in live mode                              |
| ---------------------------------------------------- | ---------------------------------------------- | -------------------------------------------------------------- |
| [Nemotron 3.5 ASR streaming](../../models/nemotron/) | [NeMo-Speech.cpp](../../backends/nemo-speech/) | Automatic or explicit language; shared vocabulary and strength |
| [Qwen3-ASR](../../models/qwen3-asr/)                 | [vLLM](../../backends/vllm/)                   | Automatic language detection                                   |
| [Voxtral Mini Realtime](../../models/voxtral-realtime/) | [vLLM](../../backends/vllm/) | Automatic language detection |

Follow the model guide for server setup and supported versions. Freehand does
not install or start the server for you.

## Enable live mode

1. Open Voice's **Transcription** quick settings, or **Settings → Voice transcription**.
2. Select your connection, the loaded model, and its model profile.
3. Enable **Realtime transcription**, which appears for a compatible combination.
4. Enable **Live overlay captions** if you want the caption strip. The main **Overlay** preference must also be enabled.
5. In full Settings, choose **Save** or **Save and return**. Quick settings changes apply immediately.

Language and vocabulary controls follow the selected model profile. Completed
settings remain saved when a control is unavailable in realtime. Shared terms
stay in **Settings → Vocabulary**.

## While recording

Keep the intended destination focused and use your recording shortcut to begin.
The preview can change as speech is recognized. The final transcript replaces
it when recording stops. Overlay captions show the newest words, not the full
transcript.

Only finalized text can be cleaned up, kept in optional history, copied, or
inserted. Cancellation or a disconnected stream discards its preview. Start a
new recording after a connection failure; Freehand does not replay captured audio.

Live mode uses the existing toggle/hold shortcut and recording duration limit.
Silence trimming, checkpoints, and automatic stop apply to completed recording
and are bypassed in live mode. Turning realtime off keeps the same connection
and model and restores your completed capture preferences.

Audio-file transcription has its own connection, model, language, and options;
changing Voice does not change it. Streaming results from an uploaded file is
a separate feature from live microphone transcription.
