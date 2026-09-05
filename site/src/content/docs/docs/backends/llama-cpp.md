---
title: llama.cpp
description: Configure llama.cpp for text cleanup, including the separate S1-mini prompt preset.
---

**Status:** available for text post-processing.
**Evidence:** automated client chat-contract fixtures plus a reported working
S1-mini setup. The runtime version for that report was not recorded. Other
models and server versions need their own qualification.

## Run llama.cpp on Windows

Install the Windows package, then open a new PowerShell window:

```powershell
winget install --exact --id ggml.llamacpp
```

Start the known S1-mini setup with explicit host and port, so a future change
to the server's default port does not change the Freehand endpoint:

```powershell
llama-server.exe `
  -hf superwhisper/s1-mini-GGUF:Q4_K_M `
  --host 127.0.0.1 --port 8080 `
  --jinja --reasoning off --temp 0 `
  -ngl 99 --sleep-idle-seconds 60
```

`-hf` downloads the selected GGUF model on first use. The command requests GPU
offload; check startup output for the backend and offloaded layers. For a
CPU-only setup, replace `-ngl 99` with `-ngl 0 --device none`. Package builds and
accelerators vary; use the [upstream server instructions](https://github.com/ggml-org/llama.cpp/tree/master/tools/server)
for other hosts or backends. This command requires a build supporting
`--reasoning off` and idle sleeping.

Keep that terminal open. In a second PowerShell window, check metadata:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/health
Invoke-RestMethod http://127.0.0.1:8080/v1/models
```

In Freehand's post-processing settings, choose compatibility profile
**llama.cpp**, base URL **`http://127.0.0.1:8080/v1`**, authentication **None**,
local HTTP allowed, and the model returned by **Test**. Choose **S1-mini** as the
separate prompt preset. It requires reasoning off even though it is optional
for other cleanup models. Save, then review cleanup of a short transcript.

Press **Ctrl+C** in the server terminal to stop it. Rerun the launch command to
restart; the downloaded model is cached. Idle sleeping releases model memory
after 60 seconds and reloads on the next inference request, so waking can add
latency. Omit that flag if you prefer to keep the model loaded.

For a custom cleanup model, use its own GGUF and template requirements, then
select **Custom instruction** in Freehand. S1-mini's fixed prompt is not a
general chat prompt. The [post-processing guide](../../guides/post-processing/)
explains raw fallback and the trained S1-mini controls.

## Configure Freehand

1. Enable post-processing and choose **llama.cpp** as the compatibility profile.
2. Enter the chat API base URL, normally ending in `/v1`, and the served model ID.
3. Configure any required authentication and test model-list metadata.
4. Choose the prompt preset separately: **Custom instruction** or **S1-mini**.
5. Save, then explicitly review cleanup of a short transcript.

The [post-processing guide](../../guides/post-processing/) covers setup and
raw fallback. For the known model setup, follow
[S1-mini with llama.cpp](../../guides/post-processing/#s1-mini-by-superwhisper-with-llamacpp),
including its exact prompt and required thinking-disabled behavior.

## Implemented contract

This profile shares the Generic non-streaming `chat/completions` adapter.
Freehand sends the model, system/user string messages, temperature zero, and
`stream=false`, plus the configured generation controls described below. It expects a string message in the first choice. A response
reporting `finish_reason=length` is a failed cleanup and cannot expose partial
cleaned text as a successful result.

The **compatibility profile** selects the server contract. The **prompt preset**
selects how the transcript and instruction are prepared. Selecting llama.cpp
does not force S1-mini, and S1-mini can also use a Generic connection that meets
the same contract.

## Scope and limits

- This Freehand profile qualifies text cleanup only. It makes no claim about
  transcription, speech synthesis, or other modalities supported by a runtime.
- Sampling, context size, and template selection remain server configuration.
  Freehand supports only the output-limit and disable-reasoning controls below.
- Cleanup uses one request. There is no automatic long-input chunking or replay.
- A successful model-list test is not proof that the chosen model accepts the
  exact fields or follows the selected instruction.

See [protocol details](../../reference/protocol/) and the
[upstream llama.cpp server documentation](https://github.com/ggml-org/llama.cpp/tree/master/tools/server).

## Cleanup generation controls

In **Settings → Post-processing → Generation controls**:

| Control | Request behavior |
| --- | --- |
| Limit output tokens | Sends `max_tokens` only when enabled, from 1 to 65,536. Off omits the field; a valid number is retained locally. |
| Disable reasoning, Custom instruction | Sends `reasoning_effort: "none"` when enabled. Off leaves reasoning to the server. |
| Disable reasoning, S1-mini | Required and automatically sent on every cleanup request through this profile. It cannot be turned off for this preset. |

The optional controls start off in older settings. S1-mini's required override
is derived from the preset and qualified compatibility profile; it does not
change the saved optional override for Custom instruction. Prompts, trained S1
controls, and temperature zero are preserved.

The reasoning field requests thinking-disabled generation; it is not
`reasoning_format: "none"`, which controls parsing of generated reasoning. A
compatible server build and model template must honor the request. Unsupported
requests fail through the normal raw-transcript fallback, without automatic
retry or silently dropping the field. An older server that ignores an unknown
field cannot be detected by a successful model-list check; keep server-side
`--reasoning off` for S1-mini and explicitly check the chosen model's behavior.

An output limit is a token budget, not a context-window setting or an automatic
long-input strategy. A low limit may cause raw fallback; no sentence chunking or
input-relative budget is added. See [generation controls](../../guides/post-processing/#generation-controls).

### Source qualification

The adapter was inspected against upstream llama.cpp commit
[`6a1a922d2699`](https://github.com/ggml-org/llama.cpp/commit/6a1a922d269908a29cbd4b49c27e6a8e7fd10fae):

- [Server schema](https://github.com/ggml-org/llama.cpp/blob/6a1a922d269908a29cbd4b49c27e6a8e7fd10fae/tools/server/server-schema.cpp) accepts `max_tokens` as a generation-limit alias.
- [Chat request parser](https://github.com/ggml-org/llama.cpp/blob/6a1a922d269908a29cbd4b49c27e6a8e7fd10fae/tools/server/server-common.cpp) maps `reasoning_effort: "none"` to thinking disabled before applying the model template.

This records source and client-fixture evidence, not a minimum supported release
or a live test of every model/template. The reported S1-mini runtime's exact
build is still unknown. Its launch log shows Jinja enabled and recognition of
the newer reasoning option, which alone does not prove the HTTP override works.
