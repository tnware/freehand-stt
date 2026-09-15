---
title: MagpieTTS Multilingual 357M
description: Generate local speech with MagpieTTS voices and language selection on NeMo-Speech.cpp.
---

Freehand's **MagpieTTS Multilingual 357M** profile supports the **v2602**
checkpoint through **NeMo-Speech.cpp v0.1.0**. It generates spoken audio from
text with a choice of voices and languages.

## Set up speech

For installation inside Freehand, follow
[Local speech with MagpieTTS](../../guides/local-runtime/#local-speech-with-magpietts).
NeMo can load Magpie alongside your selected transcription model. Both share
one runtime's **Start** and **Stop** controls.

For a manually managed server, follow the [NeMo connection guide](../../backends/nemo-speech/)
and enable **Text to speech** on the connection. Open **Text to speech** and its
**Settings** cog, choose that connection and the loaded speech model, then select
**MagpieTTS Multilingual 357M** as its model profile.

Turn on **Enable text to speech**, choose a voice and language, and **Save**.
Choose **Preview** to hear your current edits without saving them. Enter text in
the composer and choose **Speak** to use the saved settings. These actions
request audio; opening settings and refreshing metadata do not.

## Choose a voice and language

The checkpoint provides five speaker identities: **John, Sofia, Aria, Jason,
and Leo**. Each supports all of the checkpoint's languages; there is no separate
preset list for each language. **Refresh voices** reads the loaded speech
model's metadata. **default** uses the server's configured speaker.

The v2602 model supports English, Spanish, German, French, Italian, Vietnamese,
Hindi, Mandarin Chinese, and Japanese. Japanese and Chinese require NeMo builds
with the corresponding language frontends enabled. After a successful metadata
refresh, Freehand limits the language choices to those advertised by the server.
Choose **Server default** to use its configured language; this does not request
automatic language detection.

## Available controls

This profile sends the chosen voice and language and requests complete WAV audio.
NeMo v0.1.0 accepts only normal speaking speed, so Freehand disables speed
adjustment for this profile. If another speech setup has a different speed, set
**Speaking speed** to **1.0** before choosing the Magpie model or connection.
Freehand keeps your previous selection until this is corrected.

Voice-style instructions, voice cloning, and streamed
speech generation are unavailable through this integration. Text normalization
also depends on separately configured server support and grammar assets; managed
setup does not install those assets.

Your speech voice and language are saved with this model's options. Generated
audio stays in memory until cleared, replaced, recording starts, or Freehand
quits. You can explicitly save a WAV file from the player.

## Checkpoint version

Managed setup downloads the versioned Magpie model, NanoCodec decoder, and
tokenizer assets qualified with NeMo v0.1.0. The
[newer v2607 model card](https://huggingface.co/nvidia/magpie_tts_multilingual_357m)
adds Arabic, Korean, and Portuguese for twelve languages. Those additions are
outside Freehand's v2602 profile and managed download; installing or refreshing
the catalog does not upgrade the checkpoint.

See the [v2602 model card](https://huggingface.co/nvidia/magpie_tts_multilingual_357m/blob/v2602/README.md),
[NeMo speech API](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/api.md#post-v1audiospeech),
and [language frontend requirements](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/tts/configuration.md)
for the qualified server behavior.
