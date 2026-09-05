---
title: llama.cpp
description: Configure llama.cpp for text cleanup, including the separate S1-mini prompt preset.
---

**Status:** available for text post-processing.
**Evidence:** automated client chat-contract fixtures plus a reported working
S1-mini setup. The runtime version for that report was not recorded. Other
models and server versions need their own qualification.

## Configure Freehand

1. Enable post-processing and choose **llama.cpp** as the compatibility profile.
2. Enter the chat API base URL, normally ending in `/v1`, and the served model ID.
3. Configure any required authentication and test model-list metadata.
4. Choose the prompt preset separately: **Custom instruction** or **S1-mini**.
5. Save, then explicitly review cleanup of a short transcript.

The [post-processing guide](../../guides/post-processing/) covers setup and
raw fallback. For the known model setup, follow
[S1-mini with llama.cpp](../../guides/post-processing/#s1-mini-by-superwhisper-with-llamacpp),
including its exact prompt and server-side reasoning configuration.

## Implemented contract

This profile shares the Generic non-streaming `chat/completions` adapter.
Freehand sends the model, system/user string messages, temperature zero, and
`stream=false`. It expects a string message in the first choice. A response
reporting `finish_reason=length` is a failed cleanup and cannot expose partial
cleaned text as a successful result.

The **compatibility profile** selects the server contract. The **prompt preset**
selects how the transcript and instruction are prepared. Selecting llama.cpp
does not force S1-mini, and S1-mini can also use a Generic connection that meets
the same contract.

## Scope and limits

- This Freehand profile qualifies text cleanup only. It makes no claim about
  transcription, speech synthesis, or other modalities supported by a runtime.
- Model-specific reasoning, sampling, context, and template requirements remain
  server configuration unless Freehand explicitly implements those controls.
- Cleanup uses one request. There is no automatic long-input chunking or replay.
- A successful model-list test is not proof that the chosen model accepts the
  exact fields or follows the selected instruction.

See [protocol details](../../reference/protocol/) and the
[upstream llama.cpp server documentation](https://github.com/ggml-org/llama.cpp/tree/master/tools/server).
