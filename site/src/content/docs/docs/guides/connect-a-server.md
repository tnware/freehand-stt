---
title: Connect a speech server
description: Configure local, LAN, private remote, or hosted OpenAI-compatible speech services.
---

Freehand connects to speech services you provide. It does not install or run a
model itself.

## Choose a topology

All capabilities may share one gateway, or each may use a different server.
They remain independently configurable either way.

### One gateway

```text
Freehand
  ├─ STT ───────────────┐
  ├─ transcript cleanup ├─ https://speech.example.com/v1
  └─ TTS ───────────────┘
```

Use this when one compatible gateway exposes every route and model you need.
Freehand still saves and tests each capability separately because support for
one route does not imply support for the others.

### Separate services

```text
Freehand
  ├─ STT ──────────────── Speaches or another compatible speech server
  ├─ transcript cleanup ─ llama.cpp or another compatible chat server
  └─ TTS ──────────────── Speaches or another compatible speech server
```

Use this when models live on different machines or runtimes. A cleanup outage
does not invalidate successful STT: Freehand preserves and delivers the raw
transcript.

## Information Freehand needs

For speech to text, collect:

- the base URL including its API prefix, normally `/v1`;
- the exact transcription model ID expected by the server;
- the authentication mode and API key, if required;
- any non-secret gateway headers;
- whether the endpoint is intentionally using plaintext HTTP.

For example:

```text
Base URL: https://speech.example.com/v1
Model: speech/stt
Authentication: Bearer token
```

Freehand appends `/audio/transcriptions`. Do not enter the complete operation
URL unless the server's base path is itself unusual.

## Local and LAN servers

`http://127.0.0.1:8000/v1` and a private address such as
`http://192.168.1.50:8000/v1` are plaintext connections. Freehand rejects them
until **Allow insecure HTTP** is explicitly enabled for that capability.

Only enable plaintext HTTP on a network you trust. Audio, transcript text, and
credentials sent over HTTP are not encrypted in transit. Prefer HTTPS for a
server reached across machines whenever practical.

## Test without invoking a model

Freehand checks a configured health path or `GET /v1/models`. It reports:

- whether the server was reachable;
- HTTP status and request latency;
- a stable failure category;
- discovered model IDs when the response has a compatible model-list shape;
- whether the currently configured model appears in that list.

The check never sends audio or a prompt. A successful metadata check proves the
route is reachable and credentials were accepted; it does not prove that a
model can transcribe a particular recording.

Freehand performs one automatic STT metadata check during readiness and after
relevant saved STT connection fields change. It does not poll or iterate
through discovered models.

## Validated combinations

The following combinations have been exercised end to end in the Windows app:

| Capability | Backend | Route | Evidence |
| --- | --- | --- | --- |
| Microphone and stored-file STT | [Speaches](https://github.com/speaches-ai/speaches) | `POST /v1/audio/transcriptions` | Native Windows use |
| On-demand TTS | [Speaches](https://github.com/speaches-ai/speaches) | `POST /v1/audio/speech` | Native Windows use |
| S1-mini transcript cleanup | [llama.cpp](https://github.com/ggml-org/llama.cpp) `llama-server` | `POST /v1/chat/completions` | Native Windows use |

The table is intentionally short. It records evidence rather than every server
that advertises compatible routes. See the [protocol profile](../../reference/protocol/)
for the exact wire contracts and [Transcript post-processing](../post-processing/)
for the validated llama.cpp/S1-mini command.

For connection and request failures, see [Troubleshooting](../troubleshooting/).
