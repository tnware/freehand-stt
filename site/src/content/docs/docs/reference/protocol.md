---
title: Protocol profile
description: The OpenAI-compatible request and response contracts Freehand supports.
---

## Compatibility profiles

The persisted `compatibilityProfile` field is independent for STT (at the root),
`postProcessing`, and `textToSpeech`. Missing fields load as `generic`; the
legacy empty value also resolves to Generic. Neither URLs nor model IDs select
a profile automatically. Unavailable, unknown, and wrong-operation selections
are rejected by Go, including disabled feature settings and metadata probes.
Invalid saved selections use the existing configuration recovery flow without
overwriting the document.

| Profile ID | Implemented operations | Contract |
| --- | --- | --- |
| `generic` | STT, post-processing, TTS | Existing bounded multipart JSON, text chat, and buffered PCM16 WAV contracts |
| `speaches` | STT, TTS | Shared request shapes; typed transcription events and legacy per-segment text SSE; buffered WAV speech |
| `llama-cpp` | Post-processing | Shared non-streaming text chat adapter; prompt preset remains independent |

Generic intentionally retains legacy Speaches SSE support for existing
configurations. Both transcription profiles require final text for typed
streams and retain legacy EOF completion only for untyped segment streams.
Selecting Speaches does not weaken typed completion checks or enable automatic
retries. A remembered streaming failure is scoped to normalized endpoint,
model, and effective compatibility profile.

The Go catalog exposes only implemented capabilities. It does not certify
all models on a provider. Optional STT controls are explicitly gated by their
implemented capability flags. The dedicated
profiles share implementations where their wire contracts match. New dialects
must be implemented and tested before their catalog entries become available.
Metadata tests remain GET-only and never discover capabilities through inference.

Disabled placeholders are operation-specific: `openai` and `localai` across
all three roles; `whisper-cpp` for STT; `vllm` for STT and post-processing;
`vllm-omni`, `kokoro-fastapi`, and `openedai-speech` for TTS. A disabled dedicated
profile does not prevent use of a server through the generic contract.

Qualification evidence for the Speaches stream formats comes from the audit's
v0.8.3 and v0.9.0-rc.3 source comparison. Profile fixtures cover these response
shapes and the shared llama.cpp text request; they do not establish live
compatibility with every release or model. Existing user-tested integrations
remain distinct from automated fixture coverage.

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
prompt=<optional recognition context>
hotwords=<optional; Speaches only>
temperature=<optional 0–1; only with override enabled>
response_format=json
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

Streamed mode requests `Accept: text/event-stream` and accepts current typed `transcript.text.delta` and `transcript.text.done` events, plus the untyped `{ "text": ... }` segment events used by older Speaches releases. It also accepts a completed JSON response from peers that ignore `stream=true`, and cleans up an older Speaches SSE body when an intermediary buffers and wraps that body inside the JSON `text` field. The UI identifies that fallback as buffered because client-side parsing cannot recover progressive timing once an intermediary has collected the response. A rejected or incompatible streaming request is never retried automatically because that could duplicate inference or billing. Streaming unavailability is remembered for the endpoint, model, and compatibility profile; resubmission in completed mode requires the user to choose Retry. Provider and reverse-proxy upload limits still apply; client-side splitting of stored files is deferred.

Typed streams require `transcript.text.done` with a string `text` field. Its
text replaces accumulated deltas, including an empty final string. EOF or
`[DONE]` without that final event is a failed response; accepted partial text
is preserved without cleanup or automatic retry. Missing required delta/final
text fields are malformed events. A read failure or server error also keeps
accepted partial text as failed. Empty or keepalive-only SSE is not a successful
transcript. Legacy untyped Speaches segments retain their EOF completion rule;
that dialect cannot distinguish normal closure from a clean premature EOF.

While a stored file is uploading or streaming, its backend-owned status is rendered as an ephemeral live row in the main History panel. Copy remains unavailable while work is active. Once Go finalizes the transcript, the live row is replaced in place by the terminal result. If the 8 MiB transcript-response ceiling is reached, already accepted text remains available under an explicit failed-partial state rather than being silently dropped. When history is enabled and that partial text fits its separate 2 MiB total budget, it may be retained as a failed run for recovery.

The client accepts an OpenAI-compatible base URL ending in `/v1` and joins endpoint paths without duplicating or removing that prefix.
HTTPS is required by default. Plain HTTP is accepted only when **Allow insecure HTTP** is explicitly enabled in the saved or currently tested settings; this sends credentials and audio without transport encryption.

All inference capabilities and metadata-only checks reject HTTP redirects
(including 301, 302, 303, 307, and 308), with no second request or automatic
retry. This includes same-origin redirects and HTTPS-to-HTTP downgrades,
regardless of the insecure-HTTP setting. Configure the final base URL; a
redirect is reported as an HTTP failure without exposing the Location header
or response body.

Successful STT (completed or streamed) and chat responses retain only bounded
optional metadata. Strings containing the literal request credential are
omitted, including request-ID headers, response/model/provider IDs, finish
reason, service tier, fingerprint, detected languages, and usage type. The
check precedes string truncation so a bound cannot retain a prefix of a
reflected credential. Benign text and metrics survive unsafe optional
metadata. Model discovery omits reflected IDs rather than making altered IDs
selectable. These checks do not attempt to detect encoded credentials.

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

1. If the user configured a health path, append it beneath the base URL path and call that target. The required leading slash does not replace the base path: `https://host/v1` plus `/health` requests `https://host/v1/health`. This preserves existing saved configurations. A failed health probe does not fall back to `/models`.
2. Otherwise call `{base_url}/models` with the configured credential.
3. Require a JSON object with a non-null `data` array for model probes, then report whether the configured model appears. An empty array is valid. Malformed or missing inventory is a response failure while HTTP reachability remains available. Health probes accept bounded successful bodies without imposing a model-list schema.

The check uses the currently displayed compatibility profile and endpoint/model values and an optional bounded credential draft without persisting the draft. It returns a structured, window-lifetime result containing the probe URL, reachability, HTTP status, latency, checked time, stable failure kind, bounded model IDs, and configured-model presence. Returned model IDs are metadata only; choosing one updates the settings draft and performs no request.

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

The alpha sends one cleanup request per input, with no sentence chunking or input-relative output limit. A completion explicitly reporting `finish_reason: "length"` fails with `incomplete_response`, even when its text is nonempty. The workflow uses the raw transcript, shows an output-limit notice, and retains safe response metadata when history is enabled. The partial cleaned text is discarded; no automatic cleanup retry occurs. Missing or other finish reasons retain the existing response rules, so unreported omissions cannot be detected. See the [input-length limits](../../guides/post-processing/#input-length-and-alpha-limits) before processing long text.

The default is `semi-casual/prose/general`; `balanced` is not a trained S1-mini v1 value. Thinking must be disabled by the backend route. See [ADR 0001](../../decisions/0001-s1-mini-post-processing/) and the [post-processing setup guide](../../guides/post-processing/).

## Shelved realtime microphone STT research

:::note[Research only]
Realtime microphone transcription is not an active product milestone. The
current pause-aware checkpoint workflow uses the ordinary transcription route,
supports optional cleanup, and produces one final delivery result.

The proposed realtime transport and event contracts remain in
[ADR 0002: Realtime transcription](../../decisions/0002-realtime-transcription/).
:::

## Optional STT control contract

Root `transcriptionOptions` contains `prompt`, `hotwords`,
`temperatureOverride`, and `temperature`. Missing options load as empty strings,
false, and zero, preserving older requests. The boolean distinguishes an omitted
temperature from explicit zero; inactive numeric values are retained locally.

Both implemented STT profiles support optional `prompt` and `temperature` fields;
only Speaches supports `hotwords`. Prompt is bounded to 8,192 UTF-8 bytes and
hotwords to 2,048. Invalid UTF-8 and control characters other than CR, LF, and tab
are rejected. Temperature must be finite and between 0 and 1. Go validates these
rules when saving and again before building requests or reading file audio.
Validation messages do not include hint contents.

These settings are copied by value with the job's connection/credential snapshot.
The same option writer is used for microphone and file multipart bodies and file
Content-Length calculation. File streaming adds only its existing `stream=true`;
there is no automatic retry after a rejection, and the existing typed completion
and response-size rules are unchanged. These controls affect request construction;
they do not establish model capabilities through metadata discovery.
