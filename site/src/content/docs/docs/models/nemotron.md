---
title: Nemotron 3.5 ASR streaming
description: Completed and live transcription with language selection and shared vocabulary boosting.
---

The **Nemotron 3.5 ASR streaming** model profile supports NVIDIA's
**Nemotron 3.5 ASR streaming 0.6B** through **[NeMo-Speech.cpp](../../backends/nemo-speech/)**.
NeMo-Speech.cpp is the server; Nemotron is the model it runs.

For local use on Windows or macOS, follow [managed NeMo setup](../../guides/local-runtime/#set-up-local-transcription).
Nemotron is the recommended managed speech model. Freehand supplies its profile
when you select the built-in Connection; you enable live mode in Voice.

## What the profile adds

For a manual Connection, choose **Nemotron 3.5 ASR streaming** beneath the model picker in
**Voice transcription → Transcription** or **Audio file → Transcription** options.

| Setting                | Completed recordings and audio files        | Realtime microphone               |
| ---------------------- | ------------------------------------------- | --------------------------------- |
| Spoken language        | Automatic or one of 32 base-model locales   | Same language choices             |
| Shared vocabulary      | Recognition phrases with a strength setting | Same vocabulary support           |
| Realtime transcription | Voice can switch to live mode               | Live text in the results pane     |
| Live overlay captions  | Not used                                    | Optional single-row caption strip |

Selecting this profile shows the supported language list and vocabulary
controls. In Voice, it also reveals **Realtime transcription** inside the
Transcription panel. Enabling it uses your selected connection and model;
turning it off restores completed recording and checkpoint behavior.
Audio-file settings remain independent.

## Connect and select the model

For a manually configured NeMo service:

1. Follow the [NeMo-Speech.cpp setup guide](../../backends/nemo-speech/#connect-freehand) to add the server connection.
2. Open the workflow's **Settings** cog and **Transcription** options, then select that connection and the loaded model ID.
3. Select **Nemotron 3.5 ASR streaming** as the model profile, then choose automatic detection or your spoken language.
4. For live dictation, enable **Realtime transcription**. Enable **Live overlay captions** for the caption strip; the main Overlay preference must also be on.
5. Choose **Save** in the options sidebar.

## Vocabulary and output

Keep names and terminology in **Settings → Vocabulary**, then enable the list
for Voice, audio files, or both. Nemotron accepts up to 32 phrases, each up to
128 UTF-8 bytes, with 2048 bytes total. **Vocabulary strength** runs from 0 to 5.
Reuse the list with other supported models; see [Vocabulary](../../guides/vocabulary/).

Freehand requests verbatim output with the model's native punctuation and
removes the terminal language tag from displayed text. Vocabulary guides
recognition; use the separate Cleanup stage for rewrite instructions.

Live results are provisional. The final transcript replaces the preview when
you stop recording, then follows your normal cleanup and delivery settings.
NeMo-Speech.cpp v0.1.0 can revise hypotheses without marking replacements, so
provisional words may repeat before finalization. The overlay stays one row
tall and follows the newest text. See [live transcription](../../guides/live-transcription/)
for recording, cancellation, and delivery behavior.
