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

```http title="Speech-to-text request"
POST {base_url}/audio/transcriptions
Authorization: Bearer <credential>
Content-Type: multipart/form-data

file=<recording.wav>
model=speech/stt
language=<optional>
```

Expected response:

```json title="Completed transcription response"
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

```text title="Example speech endpoint settings"
Base URL: https://speech.example.com/v1
STT model: speech/stt
Language: auto/unset
```

If the endpoint requires authentication, select **API key** and enter its credential in Freehand. The key is stored in Windows Credential Manager; no particular gateway is required.

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

### Configurable request budgets

Each operation uses the budget captured when it starts. Change these values
in Settings for subsequent requests; the shared HTTP transport adds no
separate response-header deadline.

| Operation | Default request budget |
| --- | --- |
| Microphone transcription | 120 seconds |
| Each pause-aware checkpoint | 120 seconds |
| Stored-audio transcription | 360 minutes (6 hours) |
| Transcript cleanup | 120 seconds |
| Speech generation | 180 seconds |

An explicit retry receives a new request budget. If a server has rejected
streaming, a subsequent file attempt can use completed output. Freehand does
not automatically retry an ordinary failed inference request.

### Fixed safety ceilings

These client limits cannot be changed in Settings. A server or reverse proxy
may impose a lower limit.

| Input or response | Maximum |
| --- | --- |
| Microphone WAV | 8 MiB |
| Stored audio file | 2 GiB |
| Completed microphone transcription, metadata, or chat response | 1 MiB |
| Stored-file transcript response | 8 MiB |
| Chat request | 2 MiB |
| Speech playback text | 4,096 characters |
| Generated WAV | 32 MiB |

## Text to speech

On-demand speech uses an independent `POST /audio/speech` capability profile and requests PCM16 WAV for native playback. It has its own endpoint, model, voice, authentication, plaintext-HTTP opt-in, credential, and request budget even when it shares a server with another capability. The compatible API defines no portable voice-list endpoint, so model discovery uses `GET /models` and the voice ID remains explicit.

Example self-hosted values:

```text title="Speech playback model and voice"
TTS model: <model ID served by your speech endpoint>
TTS voice: <voice ID supported by that model>
```

The client must never iterate across the LLM catalog.

## Optional transcript post-processing

The client can pass the raw STT result to `POST /chat/completions` through an independently configured endpoint, model, and credential. This capability can be disabled for verbatim transcription. Processing failure and empty processor output fall back to the raw transcript, subject to cancellation and the normal delivery checks. Retaining both versions after successful cleanup requires enabled session history.

S1-mini v1 requires its exact documented system prompt and control line. Valid values are:

```text title="Supported S1-mini control values"
Styling: casual | semi-casual | semi-formal | formal
Structure: prose | lists
Context: general | email
```

The alpha sends one cleanup request per input, with no sentence chunking or input-relative output limit. Nonempty length-limited completions are accepted. See the [input-length limits](../../guides/post-processing/#input-length-and-alpha-limits) before processing long text.

The default is `semi-casual/prose/general`; `balanced` is not a trained S1-mini v1 value. Thinking must be disabled by the backend route. See [ADR 0001](../../decisions/0001-s1-mini-post-processing/) and the [post-processing setup guide](../../guides/post-processing/).

## Shelved realtime microphone STT research

:::note[Research only]
Realtime microphone transcription is not an active product milestone. The
current pause-aware checkpoint workflow uses the ordinary transcription route,
supports optional cleanup, and produces one final delivery result.

The proposed realtime transport and event contracts remain in
[ADR 0002: Realtime transcription](../../decisions/0002-realtime-transcription/).
:::
