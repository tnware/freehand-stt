---
title: Speaches
description: Connect Speaches for transcription and on-demand speech playback.
---

**Status:** available for transcription and speech playback.
**Evidence:** automated client contract fixtures plus a reported working
Whisper-family STT and Kokoro TTS setup. The runtime version for that reported
setup was not recorded; it does not establish compatibility for every model
in the server inventory.

## Configure Freehand

For transcription, choose **Speaches** in the speech-to-text connection,
enter the server's base URL including `/v1`, and select an installed
transcription model. Set authentication to match the server.

For speech playback, enable it in its own Settings section, choose **Speaches**,
and provide the speech base URL, installed TTS model, and an appropriate voice
ID. Save settings before explicitly previewing a voice. The two operations may
share a server but retain separate credentials and settings.

The [Speaches setup example](../../guides/connect-a-server/#example-speaches-on-your-pc)
explains one deployment path. Freehand neither installs models nor loads all
models during a connection test.

## Implemented capabilities

| Capability | Scope |
| --- | --- |
| Microphone transcription | Completed JSON transcription, including local checkpoint requests. |
| Stored audio files | Completed JSON or optional streamed transcript results. |
| Streaming dialects | Typed transcript delta/done events and legacy untyped text segments. |
| Language hint | Optional `language` request field; effect depends on the model. |
| Recognition context | Optional `prompt`, at most 8,192 UTF-8 bytes. |
| Hotwords | Optional `hotwords`, at most 2,048 UTF-8 bytes; Speaches-specific field. |
| Decoding temperature | Optional `temperature` from 0 to 1; explicit zero is supported. |
| Speech playback | Voice ID, speed request, and buffered PCM16 WAV. |
| Transcript cleanup | Configure a separate Generic or llama.cpp chat connection. |

Generic retains these compatible response shapes for older Freehand settings.
Selecting Speaches identifies the dedicated contract without changing the model,
URL or authentication. Optional controls are omitted until configured; selecting
Speaches alone adds no fields.

## Version and model evidence

The compatibility audit inspected **v0.8.3** and **v0.9.0-rc.3**:

- The older transcription route emits a JSON text segment per SSE event and
  completes by closing the response.
- The inspected newer Whisper executor emits typed delta/done events. Freehand
  requires final text for a typed stream; EOF alone is insufficient.
- The inspected binary speech path can produce compatible PCM16 WAV. Freehand
  does not consume speech SSE events or perform progressive playback.

These are source and client-fixture qualifications, not live tests of every
release or executor. The RC label is retained here intentionally. Actual model
and voice IDs, decoder support, and speed behavior depend on the served setup.

## Current limits

The profile does not add timestamps, a translation workflow, server VAD
controls, voice instructions, cloning inputs, or voice catalog discovery.
Provider limits and Freehand's bounded-buffer limits still apply; consult the
[protocol reference](../../reference/protocol/).

## Upstream references

- [Speaches repository](https://github.com/speaches-ai/speaches)
- [v0.8.3 transcription route](https://github.com/speaches-ai/speaches/blob/v0.8.3/src/speaches/routers/stt.py)
- [v0.9.0-rc.3 Whisper executor](https://github.com/speaches-ai/speaches/blob/v0.9.0-rc.3/src/speaches/executors/whisper.py)
- [v0.9.0-rc.3 speech route](https://github.com/speaches-ai/speaches/blob/v0.9.0-rc.3/src/speaches/routers/speech.py)

## Recognition controls and qualification

The inspected **v0.8.3** transcription route accepts `prompt`, `hotwords`, and
`temperature` and forwards them to faster-whisper. **v0.9.0-rc.3** forwards the
same controls in both its completed and streaming Whisper executor paths.
This qualification covers the Whisper-family contract, not every Speaches
executor or all model behavior. These new fields have request fixtures and
source evidence; the earlier working-setup report does not establish live
acceptance of the new controls.

Context supplies expected subject matter or wording. Hotwords supply terms to
favor; neither is a strict replacement dictionary. Both may be supplied, but
model prompt budgets and decoding behavior can limit their effect. Freehand's
byte limits bound requests; they are not model token allowances. Temperature
is a decoding request, not a guaranteed quality or determinism control.

Keep these options unset if the selected model does not support them. A rejected
request fails normally; Freehand does not silently drop hints and repeat inference.
See [Transcription controls](../../guides/connect-a-server/#transcription-controls).

Server-side VAD is intentionally not exposed: v0.8.3 accepts `vad_filter`, while
the inspected v0.9.0-rc.3 route runs VAD internally with fixed options. Local
microphone VAD settings remain independent. Library settings such as beam size
are not automatically fields on the Speaches HTTP request.
