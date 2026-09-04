---
title: Protocol profile
description: The OpenAI-compatible request and response contracts Freehand supports.
---

## Validated implementations

Freehand targets capability-specific OpenAI-compatible routes rather than requiring one particular server. Compatibility claims use three evidence levels:

- **Validated** — exercised end to end in the native Windows application.
- **Contract-compatible** — the client implements the documented route and
  shape, but a named backend is not claimed as tested.
- **Unsupported or unknown** — a required route is absent, or available evidence
  is insufficient to make a compatibility claim.

The following combinations have been exercised end to end in the Windows app:

| Freehand capability | Tested backend | Compatible route | Evidence |
| --- | --- | --- | --- |
| Microphone and stored-file speech to text | [Speaches](https://github.com/speaches-ai/speaches) | `POST /v1/audio/transcriptions` | Validated in native Windows use |
| Text to speech | [Speaches](https://github.com/speaches-ai/speaches) | `POST /v1/audio/speech` | Validated in native Windows use |
| S1-mini transcript post-processing | [llama.cpp](https://github.com/ggml-org/llama.cpp) `llama-server` | `POST /v1/chat/completions` | Validated in native Windows use |

These are known-working implementations, not product dependencies or an exhaustive compatibility list. “OpenAI-compatible” does not guarantee that a server implements every optional audio and chat route, so Freehand configures and tests STT, TTS, and post-processing independently. A project name or model listing is never sufficient evidence to claim end-to-end support.

See [Connect a speech server](../../guides/connect-a-server/) for topology,
configuration, and failure guidance.

## STT request

```http
POST {base_url}/audio/transcriptions
Authorization: Bearer <credential>
Content-Type: multipart/form-data

file=<recording.wav>
model=speech/stt
language=<optional>
```

Expected response:

```json
{
  "text": "transcribed text"
}
```

Stored-audio transcription uses this same endpoint and configured model. The native picker accepts `flac`, `mp3`, `mp4`, `mpeg`, `mpga`, `m4a`, `ogg`, `wav`, and `webm`; Go revalidates the selected regular file and streams it from disk as multipart data without sending audio through the Wails bridge.

The user can choose either response contract:

- Completed: `response_format=json`, followed by one bounded `{ "text": ... }` response.
- Streamed: `response_format=json` and `stream=true`, followed by `text/event-stream` events.

Streamed mode requests `Accept: text/event-stream` and accepts current typed `transcript.text.delta` and `transcript.text.done` events, plus the untyped `{ "text": ... }` segment events used by older Speaches releases. It also accepts a completed JSON response from peers that ignore `stream=true`, and cleans up an older Speaches SSE body when an intermediary buffers and wraps that body inside the JSON `text` field. The UI identifies that fallback as buffered because client-side parsing cannot recover progressive timing once an intermediary has collected the response. A rejected streaming request is never retried automatically because that could duplicate inference or billing. Provider and reverse-proxy upload limits still apply; client-side splitting of stored files is deferred.

While a stored file is uploading or streaming, its backend-owned status is rendered as an ephemeral live row in the main History panel. Copy remains unavailable while work is active. Once Go finalizes the transcript, the live row is replaced in place by the terminal result. If the 8 MiB transcript-response ceiling is reached, already accepted text remains available under an explicit failed-partial state rather than being silently dropped. When history is enabled and that partial text fits its separate 2 MiB total budget, it may be retained as a failed run for recovery.

The client accepts an OpenAI-compatible base URL ending in `/v1` and joins endpoint paths without duplicating or removing that prefix.
HTTPS is required by default. Plain HTTP is accepted only when **Allow insecure HTTP** is explicitly enabled in the saved or currently tested settings; this sends credentials and audio without transport encryption.

## Example deployment

```text
Base URL: https://speech.example.com/v1
STT model: speech/stt
Language: auto/unset
```

The LiteLLM key must be entered by the user and stored in Windows Credential Manager.

## Headers

- `Authorization: Bearer ...` is generated from the credential store.
- Arbitrary extra headers are supported for compatible private gateways.
- Secret-looking extra headers must be stored with credentials, not non-secret config.
- Hop-by-hop headers, `Host`, `Content-Length`, and a second `Authorization` header are rejected.

## Metadata-only connection check

Preferred order:

1. If the user configured a health path, call it.
2. Otherwise call `{base_url}/models` with the configured credential.
3. Confirm the configured STT model appears when the response has an OpenAI model-list shape.

The check uses the currently displayed endpoint/model values and an optional bounded credential draft without persisting the draft. It returns a structured, window-lifetime result containing the probe URL, reachability, HTTP status, latency, checked time, stable failure kind, bounded model IDs, and configured-model presence. Returned model IDs are metadata only; choosing one updates the settings draft and performs no request.

This check does not submit audio or inference requests and stops after 15 seconds. Model inventory is capped at 200 distinct IDs of at most 200 bytes each, and the full response remains subject to the 1 MiB metadata limit.

## Request budgets and safety ceilings

The shared HTTP transport does not impose a response-header deadline. The owning capability applies the validated budget captured at operation start: microphone and each silence checkpoint default to 120 seconds, a stored-file request defaults to 360 minutes, post-processing defaults to 120 seconds, and speech generation defaults to 180 seconds. These values are user-configurable in Settings. A streamed-file fallback request receives a fresh stored-file budget, but Freehand never retries an ordinary failed request automatically.

Fixed safety ceilings are not user-configurable: 8 MiB per microphone WAV, 2 GiB per stored audio file, 1 MiB per completed transcription/metadata/chat response, 8 MiB per stored-file transcript response, 2 MiB per chat request, 4,096 characters per speech input, and 32 MiB per generated WAV.

## Text to speech

On-demand speech uses an independent `POST /audio/speech` capability profile and requests PCM16 WAV for native playback. It has its own endpoint, model, voice, authentication, plaintext-HTTP opt-in, credential, and request budget even when it shares a server with another capability. The compatible API defines no portable voice-list endpoint, so model discovery uses `GET /models` and the voice ID remains explicit.

Example self-hosted values:

```text
LLM model: one user-selected ollama/<model> route
TTS model: speech/tts
TTS voice: bf_alice
```

The client must never iterate across the LLM catalog.

## Optional transcript post-processing

The client can pass the raw STT result to `POST /chat/completions` through an independently configured endpoint, model, and credential. This capability can be disabled for verbatim transcription. Processing failure and empty processor output fall back to the raw transcript without failing delivery.

S1-mini v1 requires its exact documented system prompt and control line. Valid values are:

```text
Styling: casual | semi-casual | semi-formal | formal
Structure: prose | lists
Context: general | email
```

The default is `semi-casual/prose/general`; `balanced` is not a trained S1-mini v1 value. Thinking must be disabled by the backend route. See [ADR 0001](../../decisions/0001-s1-mini-post-processing/) and the [post-processing setup guide](../../guides/post-processing/).

## Shelved realtime microphone STT research

Realtime microphone STT is not an active product milestone. The existing
pause-aware checkpoint flow continues to use the ordinary transcription route,
supports optional cleanup, and produces one stable delivery result. The
following contract is retained only as research for a future live-caption or
provisional-editing use case:

Realtime STT has an independent endpoint and credential because LiteLLM is not a required transport:

```yaml
realtime_stt:
  transport: websocket
  url: ws://speaches.internal:8000/v1/realtime
  transcription_model: Systran/faster-distil-whisper-small.en
  credential_ref: optional-separate-reference
```

Speaches v0.8.2 accepts 24 kHz mono PCM16 through `input_audio_buffer.append`, supports server VAD, and emits finalized `conversation.item.input_audio_transcription.completed` events. The application also models the OpenAI `delta` event for providers that support true incremental transcription. It never inserts provisional text. See [ADR 0002](../../decisions/0002-realtime-transcription/).
