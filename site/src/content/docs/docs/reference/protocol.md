---
title: Protocol profile
description: The OpenAI-compatible request and response contracts Freehand supports.
---

## Compatibility profiles

This reference describes Freehand's implemented client contracts. For setup,
start with [Connect a server](../../guides/connect-a-server/); for model-specific
restrictions, see [Model profiles](../../models/).

The persisted `compatibilityProfile` field is independent for microphone STT
(`voiceTranscription`), audio-file STT (at the root), `postProcessing`, and
`textToSpeech`. Voice uses one connection/model/profile for completed and
qualified realtime transcription. Missing profile fields load as `generic`; the
legacy empty value also resolves to Generic. Neither URLs nor model IDs select
a profile automatically. Unavailable, unknown, and wrong-operation selections
are rejected by Go, including disabled feature settings and metadata probes.
Invalid saved selections use the existing configuration recovery flow without
overwriting the document.

| Profile ID    | Implemented operations    | Contract                                                                                               |
| ------------- | ------------------------- | ------------------------------------------------------------------------------------------------------ |
| `generic`     | STT, post-processing, TTS | Existing bounded multipart JSON, text chat, and buffered PCM16 WAV contracts                           |
| `speaches`    | STT, TTS                  | Shared request shapes; typed transcription events and legacy per-segment text SSE; buffered WAV speech |
| `llama-cpp`   | Post-processing           | Shared non-streaming text chat adapter; prompt preset remains independent                              |
| `whisper-cpp` | STT                       | Native `/inference`, server-loaded model, `/health`, completed JSON                                    |
| `vllm`        | STT, realtime, post-processing | Completed JSON, dedicated file-stream decoder, qualified Qwen3-ASR/Voxtral realtime, text cleanup |
| `nemo-speech-v1` | STT, realtime           | Completed multipart and the NeMo-Speech.cpp v0.1.0 WebSocket contract; explicit model qualification |
| `kokoro-fastapi` | TTS                     | Buffered PCM16 WAV with `stream: false` and voice metadata discovery |
| `vllm-omni`   | TTS                       | Buffered WAV, voice discovery, and qualified Qwen3-TTS language/style fields |

Generic intentionally retains legacy Speaches SSE support for existing
configurations. Generic and Speaches require final text for typed
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

Disabled placeholders are operation-specific: `openai` and `localai` for
completed STT, post-processing, and TTS; `openedai-speech` for TTS. A disabled dedicated
profile does not prevent use of a server through the generic contract.

See [Backend compatibility](../../backends/) for available integrations and
[Connect a server](../../guides/connect-a-server/) for setup. The sections below
describe the request fields and response handling for each operation.

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

Audio-file transcription uses the same request shape with its independently configured connection and model. The native picker accepts `flac`, `mp3`, `mp4`, `mpeg`, `mpga`, `m4a`, `ogg`, `wav`, and `webm`; the selected server must support the file's format. Go revalidates the selected regular file and streams it from disk as multipart data without sending audio through the Wails bridge. whisper.cpp uses its native `/inference` route instead, with no model field.

For audio files, a qualified backend/model combination can expose either response contract. Completed microphone requests always use JSON:

- Completed: `response_format=json`, followed by one bounded `{ "text": ... }` response.
- Streamed: `response_format=json` and `stream=true`, followed by `text/event-stream` events.

For Generic and Speaches, streamed mode requests `Accept: text/event-stream` and accepts typed `transcript.text.delta` and `transcript.text.done` events, plus the untyped `{ "text": ... }` segment events used by older Speaches releases. It also accepts a completed JSON response from peers that ignore `stream=true`, and cleans up an older Speaches SSE body when an intermediary buffers and wraps that body inside the JSON `text` field. The UI identifies that fallback as buffered because client-side parsing cannot recover progressive timing once an intermediary has collected the response. A rejected or incompatible streaming request is never retried automatically because that could duplicate inference or billing. Streaming unavailability is remembered for the endpoint, model, and compatibility profile; resubmission in completed mode requires the user to choose Retry. Provider and reverse-proxy upload limits still apply; client-side splitting of stored files is deferred.

Typed streams require `transcript.text.done` with a string `text` field. Its
text replaces accumulated deltas, including an empty final string. EOF or
`[DONE]` without that final event is a failed response; accepted partial text
is preserved without cleanup or automatic retry. Missing required delta/final
text fields are malformed events. A read failure or server error also keeps
accepted partial text as failed. Empty or keepalive-only SSE is not a successful
transcript. Legacy untyped Speaches segments retain their EOF completion rule;
that dialect cannot distinguish normal closure from a clean premature EOF.

vLLM uses a separate transcription-chunk decoder. A per-chunk stop marker does
not complete a file: `[DONE]` must follow a successful final chunk. Provider
errors or incomplete streams preserve accepted text as a failed partial result.
See the [vLLM contract](../../backends/vllm/).

Copy remains unavailable while file work is active. If the 8 MiB transcript-response ceiling is reached, already accepted text remains available under an explicit failed-partial state rather than being silently dropped. When history is enabled and that partial text fits its separate 2 MiB total budget, it may be retained as a failed run for recovery.

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

If the endpoint requires authentication, select **API key** and enter its credential in Freehand. The key is stored in Windows Credential Manager or macOS Keychain; no particular gateway is required.

## Headers

- `Authorization: Bearer ***` is generated from the credential store when API-key authentication is selected.
- Validated non-secret extra headers are supported for compatible private gateways.
- Secret-looking extra header names are rejected; the custom-header settings are not a credential store.
- Hop-by-hop headers, `Host`, `Content-Length`, and a second `Authorization` header are rejected.

## Metadata-only connection check

Preferred order:

1. If the user configured a health path, append it beneath the base URL path and call that target. The required leading slash does not replace the base path: `https://host/v1` plus `/health` requests `https://host/v1/health`. This preserves existing saved configurations. A failed health probe does not fall back to `/models`.
2. Otherwise use the backend's default metadata route: `{base_url}/health` for whisper.cpp, or `{base_url}/models` for other backends, with the configured credential.
3. Require a JSON object with a non-null `data` array for model probes, then report whether the configured model appears. An empty array is valid. Malformed or missing inventory is a response failure while HTTP reachability remains available. Health probes accept bounded successful bodies without imposing a model-list schema.

The check uses the currently displayed compatibility profile and endpoint/model values and an optional bounded credential draft without persisting the draft. It returns a structured, window-lifetime result containing the probe URL, reachability, HTTP status, latency, checked time, stable failure kind, bounded model IDs, and configured-model presence. Returned model IDs are metadata only; choosing one updates the settings draft and performs no request.

This check does not submit audio or inference requests and stops after 15 seconds. Model inventory is capped at 200 distinct IDs of at most 200 bytes each, and the full response remains subject to the 1 MiB metadata limit.

## Request budgets and safety ceilings

### Configurable request budgets

Each operation uses the budget captured when it starts. Change these values
in Settings for subsequent requests; the shared HTTP transport adds no
separate response-header deadline.

| Operation                   | Default request budget |
| --------------------------- | ---------------------- |
| Microphone transcription    | 120 seconds            |
| Each pause-aware checkpoint | 120 seconds            |
| Stored-audio transcription  | 360 minutes (6 hours)  |
| Transcript cleanup          | 120 seconds            |
| Speech generation           | 180 seconds            |

An explicit retry receives a new request budget. If a server has rejected
streaming, a subsequent file attempt can use completed output. Freehand does
not automatically retry an ordinary failed inference request.

### Fixed safety ceilings

These client limits cannot be changed in Settings. A server or reverse proxy
may impose a lower limit.

| Input or response                                              | Maximum          |
| -------------------------------------------------------------- | ---------------- |
| Microphone WAV                                                 | 8 MiB            |
| Stored audio file                                              | 2 GiB            |
| Completed microphone transcription, metadata, or chat response | 1 MiB            |
| Stored-file transcript response                                | 8 MiB            |
| Chat request                                                   | 2 MiB            |
| Speech playback text                                           | 4,096 characters |
| Generated WAV                                                  | 32 MiB           |

## Text to speech

On-demand speech uses an independent `POST /audio/speech` capability profile and requests PCM16 WAV for native playback. Model, voice, options, and request budget are independent of transcription and cleanup. Selecting the same saved connection deliberately shares its endpoint, authentication, plaintext-HTTP policy, and credential. The Generic compatible baseline defines no portable voice-list endpoint. Speaches, Kokoro-FastAPI, and vLLM-Omni add qualified metadata discovery. Manual voice IDs remain valid where the selected model profile permits them; Qwen3-TTS CustomVoice restricts selection to its qualified preset voices.

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

Freehand sends one cleanup request per input, with no sentence chunking or input-relative output limit. A completion explicitly reporting `finish_reason: "length"` fails with `incomplete_response`, even when its text is nonempty. The workflow uses the raw transcript, shows an output-limit notice, and retains safe response metadata when history is enabled. The partial cleaned text is discarded; no automatic cleanup retry occurs. Missing or other finish reasons retain the existing response rules, so unreported omissions cannot be detected. See the [cleanup result handling](../../models/s1-mini/#language-and-results) before processing long text.

The default is `semi-casual/prose/general`; `balanced` is not a trained S1-mini v1 value. Thinking must be disabled: the llama.cpp and vLLM profiles automatically request this for S1-mini; Generic requires the backend route to enforce it. See the [post-processing setup guide](../../guides/post-processing/).

<span id="shelved-realtime-microphone-stt-research"></span>

## Qualified realtime microphone STT

Realtime is an optional mode of the Voice connection/model/profile selection.
The pause-aware completed flow remains the default. Qualified combinations are
Nemotron 3.5 on NeMo-Speech.cpp v0.1.0, and Qwen3-ASR or Voxtral Mini Realtime on
vLLM v0.28.0. Generic does not enable realtime.

NeMo uses binary 16 kHz mono PCM16 audio after a configuration acknowledgement;
vLLM uses JSON/base64 PCM16 and a different session/commit protocol. These are
distinct adapters, not interchangeable OpenAI Realtime dialects. Partial text
is presentation-only. Authoritative finals enter optional cleanup and focus-safe
delivery; cancellation or transport failure does not replay audio automatically.

See [Live transcription](../../guides/live-transcription/) for configuration and
the [NeMo-Speech.cpp](../../backends/nemo-speech/) and
[vLLM](../../backends/vllm/) references for their protocol contracts.

## Optional STT control contract

`voiceTranscription.transcriptionOptions` owns microphone options; root
`transcriptionOptions` owns file options. Each contains `prompt`, `hotwords`,
`temperatureOverride`, and `temperature`. Missing options load as empty strings,
false, and zero, preserving older requests. The boolean distinguishes an omitted
temperature from explicit zero; inactive numeric values are retained locally.

Generic, Speaches, whisper.cpp, and vLLM support optional `prompt` and
`temperature` fields at the backend level; the selected model can restrict them
further. Only Speaches supports the `hotwords` field. Shared vocabulary is
task-owned and projected into the selected adapter's supported hint fields;
see [Vocabulary](../../guides/vocabulary/). Prompt is bounded to 8,192 UTF-8 bytes and
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

## Cleanup generation request fields

`postProcessing.generationOptions` persists `limitOutputTokens`,
`maxOutputTokens`, and `disableReasoning`. Missing fields load as false, zero,
and false. The optional controls therefore preserve old request shapes for
custom processing. A disabled numeric limit may retain 0–65,536; an enabled
limit requires an integer from 1–65,536. Go validates these options even when
post-processing is disabled and again before request I/O.

Generic, llama.cpp, and vLLM send `max_tokens` only when the limit is enabled.
They never send an explicit zero, a second token-limit alias, or a guessed
model-specific field. An enabled reasoning override requires llama.cpp or vLLM and maps
to `reasoning_effort: "none"`. S1-mini through a qualified reasoning-capable
profile derives that override automatically from its model requirement, even
when the saved custom override is false. Generic cannot enforce reasoning via
this request contract and still requires server-side configuration for S1-mini.

The preset layer expresses the thinking-disabled requirement; compatibility
capabilities qualify enforcement and the inference adapter owns the wire field.
No `reasoning_format`, arbitrary template kwargs, or sampling overrides are
added. Options are copied with the job's connection/credential profile. Both
microphone and stored-file cleanup retain length-limit rejection, durable raw
fallback, cancellation, and one request per cleanup attempt.

## Additional qualified provider profiles

whisper.cpp supports completed transcription using its native `/inference`
route and server-loaded model. Its default connection probe is `/health` beneath
the configured server root; it has no client model selection or file streaming.
vLLM v0.28.0 supports completed transcription, its own file-stream dialect, and
text cleanup with optional output limits and reasoning-off requests. The
S1-mini preset requires reasoning off through both qualified cleanup profiles.
See the [whisper.cpp guide](../../backends/whisper-cpp/) and
[vLLM guide](../../backends/vllm/) for setup, contract details, and limitations.

## Language selection contract

`voiceTranscription.language` and root `language` independently select microphone
and file languages. In the common completed contract, empty omits the field;
`auto` omits it for Generic, Speaches, and vLLM and sends `language=auto` for
whisper.cpp. Model profiles further restrict accepted languages and defaults.
NeMo and specialized vLLM profiles follow their qualified model contracts rather
than accepting arbitrary language hints. Realtime vLLM omits language hints.
Both completed microphone and file uploads resolve their selected contract
before multipart construction, including file content-length calculation.

The existing S1-mini preset declares fixed `language: "en"` in its read-only
profile descriptor. Selected or reported non-English input bypasses cleanup
and preserves raw text with `unsupported_language`; unknown language assumes
English as displayed in the controls. The fixed prompt and reasoning-off
contract are unchanged. See [language selection](../../guides/languages/).

## Speech voice discovery

`ListSpeechVoices` resolves one saved connection and credential snapshot in Go.
Only profiles advertising voice discovery make requests. vLLM-Omni reads
`GET /audio/voices` for string voice IDs. Kokoro-FastAPI reads
`GET /audio/voices` and accepts `voices` containing ID/name objects or the older
string entries. Speaches first reads `GET /models` for the selected model's
`voices`; when absent, it falls back to `GET /audio/voices` and identifies that
list as server-wide. An empty model list is not replaced with unrelated voices.
All paths are relative to the configured base URL, preserving reverse-proxy prefixes.

Metadata uses a 15-second operation budget, a 1 MiB response limit, at most 500
unique voice IDs, and IDs of at most 200 UTF-8 bytes without control characters.
Reflected credentials are omitted, redirects are not followed, and raw response
bodies never appear in diagnostics. Names and language labels are bounded display
metadata, not inferred model capabilities. No discovery call invokes inference.
Lists are transient and scoped to the current connection/model. The selected
voice uses the existing per-model setting; absence from a list does not block it.

Kokoro-FastAPI's speech request uses the existing model/input/voice/speed/WAV
fields plus `stream: false`.
Generic and Speaches keep their existing request shape. vLLM-Omni also explicitly
requests buffered speech; the Qwen3-TTS CustomVoice profile adds `task_type`,
language, and optional style instructions under its
[model contract](../../models/qwen3-tts/). Freehand buffers and validates PCM16
WAV before native playback. Voice blending management, Kokoro-specific language
overrides, normalization, and progressive playback are outside this contract.
