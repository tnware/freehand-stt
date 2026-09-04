---
title: "ADR 0002: Realtime transcription as a transport-capable later milestone"
description: Preserve realtime transcription as a separate versioned transport capability.
---

- Status: Superseded as an active roadmap commitment by ADR 0005; retained as protocol and safety research
- Date: 2026-08-29
- Speaches authority: [v0.8.2 realtime guide](https://github.com/speaches-ai/speaches/blob/v0.8.2/docs/usage/realtime-api.md), [WebSocket router](https://github.com/speaches-ai/speaches/blob/v0.8.2/src/speaches/routers/realtime/ws.py), and [example transcription client](https://github.com/speaches-ai/speaches/blob/v0.8.2/scripts/realtime_transcription_client.py)
- Protocol authority: [OpenAI Realtime transcription](https://developers.openai.com/api/docs/guides/realtime-transcription)

## Decision

Implement realtime transcription after basic dictation and optional transcript post-processing are stable:

```text
hotkey or toggle
  -> WASAPI microphone stream
  -> persistent OpenAI-Realtime-compatible transport
  -> server VAD and transcription events
  -> provisional/final overlay
  -> finalized raw utterance
  -> optional S1-mini by Superwhisper
  -> focus-safe insertion
```

Realtime is a separate capability with its own URL, transport, credential reference, audio format, and model. It does not replace file transcription and does not require LiteLLM.

The first desktop transport should be WebSocket. Speaches v0.8.2 also supports WebRTC negotiation at the same path, but WebRTC is unnecessary complexity for a native Go client that already owns microphone capture.

## Two kinds of streaming

### Uploaded-recording response streaming

Speaches `POST /v1/audio/transcriptions` accepts `stream=true` and returns SSE segments while processing an already uploaded recording. Audio is not flowing while the user speaks.

This may become a dictation latency improvement, but it is not realtime capture.

### Realtime audio streaming

Speaches v0.8.2 exposes:

```text
WS   /v1/realtime?model=<session-model>
POST /v1/realtime?model=<session-model>  # WebRTC SDP negotiation
```

The tagged WebSocket example continuously sends 24 kHz, mono, signed 16-bit PCM as base64 `input_audio_buffer.append` events. A transcription-only session sets:

```text
input_audio_transcription.model=<STT model>
turn_detection.create_response=false
```

The client should keep the protocol transport-independent even though WebSocket is implemented first.

## Event model

Normalize supported server messages into an application event with stable correlation fields:

```go
type TranscriptionEvent struct {
    Kind         EventKind
    SessionID    string
    ItemID       string
    ContentIndex int
    Delta        string
    Transcript   string
}
```

Required semantic kinds:

```text
session_ready
speech_started
speech_stopped
turn_committed
transcript_delta
transcript_completed
transcript_failed
transport_failed
```

Retain unknown protocol events only as bounded non-secret diagnostics; do not expose raw payloads indiscriminately.

Correlation is by `item_id` and `content_index`. OpenAI documents that completion order across separate turns is not guaranteed.

## Current Speaches capability

Speaches v0.8.2 supports:

- `session.created` / `session.updated`
- `input_audio_buffer.speech_started`
- `input_audio_buffer.speech_stopped`
- `input_audio_buffer.committed`
- `conversation.item.input_audio_transcription.completed`
- `conversation.item.input_audio_transcription.failed`

Its v0.8.2 server-event union does not include the newer OpenAI event:

```text
conversation.item.input_audio_transcription.delta
```

Therefore the first Speaches experience is turn/chunk-immediate rather than guaranteed word-by-word transcription. The client still models deltas so newer OpenAI-compatible providers can update provisional text without a UI redesign.

Do not fabricate deltas by repeatedly retranscribing overlapping buffers unless a separately evaluated backend contract explicitly requires that approach.

## Presentation contract

- Gray/muted text is provisional and may be revised.
- Normal text is finalized for one correlated item.
- Speech/VAD status is separate from transcript finality.
- Completion replaces or reconciles provisional text for the same item.
- Multiple completed items are ordered by committed-turn identity, not arrival time alone.
- Provisional text is never inserted into another application.
- Only a finalized raw utterance may proceed to optional S1-mini and focus-safe insertion.

For Speaches v0.8.2, the overlay may show `Listening` during VAD and then publish short finalized utterance chunks after pauses.

## Audio contract

Realtime Speaches input uses:

```text
24,000 Hz
mono
signed 16-bit PCM
base64 chunks
```

The MVP file-transcription path uses a 16 kHz mono PCM16 WAV. Therefore audio capture must accept a requested `FormatSpec`; do not hard-code one sample rate into the entire audio domain.

The realtime adapter must:

- bound its outgoing audio queue;
- define backpressure rather than accumulating unbounded microphone data;
- never log base64 audio;
- cancel and discard the current provisional turn on transport failure;
- avoid reconnecting and replaying audio ambiguously;
- stop capture before closing the transport during shutdown.

## Configuration

```yaml
stt:
  base_url: https://litellm.example/v1
  model: speech/stt

realtime_stt:
  enabled: true
  transport: websocket
  url: ws://speaches.internal:8000/v1/realtime
  session_model: transcription
  transcription_model: Systran/faster-distil-whisper-small.en
  credential_ref: optional-separate-reference
  vad:
    enabled: true
    silence_duration_ms: 1500
    threshold: 0.9
```

The exact Speaches v0.8.2 session schema is older than the current OpenAI GA transcription-session schema. Implement a versioned provider/codec adapter behind shared semantic events instead of assuming every OpenAI-Realtime-compatible backend accepts identical `session.update` JSON.

## LiteLLM boundary

LiteLLM exposes `/v1/realtime`, but its documented supported provider set does not currently name Speaches or an arbitrary self-hosted realtime backend. Realtime configuration therefore supports a direct endpoint independently of the ordinary LiteLLM STT base URL.

Routing through LiteLLM can be enabled later only after the exact deployed LiteLLM version proves:

- WebSocket upgrade and authentication;
- self-hosted upstream URL/provider selection;
- bidirectional event transparency;
- cancellation and close semantics;
- no buffering or schema transformation that breaks the selected backend.

Do not make LiteLLM a milestone-3 dependency.

## Relationship to S1-mini

S1-mini runs only after `transcript_completed`, never for each delta. The pipeline preserves:

```text
provisional text
final raw text
optional clean text
```

as distinct states. A cleanup failure follows ADR 0001's visibly reported `fallback_raw` policy.

## Future conversation mode

The same realtime transport may later feed:

```text
realtime STT -> optional S1-mini -> selected LLM -> TTS -> speakers
```

That later milestone adds response generation, playback queues, interruption, echo/half-duplex control, and conversation state. Those concerns do not belong in realtime-dictation milestone 3.

## Native acceptance

Test on Windows with representative microphones and network faults:

- start/stop and long sessions;
- VAD boundaries and pauses;
- 24 kHz PCM chunk integrity;
- queue backpressure;
- completed-only Speaches behavior;
- a delta-capable fixture/provider;
- item correlation and out-of-order completion;
- transport loss during speech;
- cancellation and shutdown;
- no insertion from provisional or stale events;
- optional S1-mini only after finalization.
