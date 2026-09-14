# Pinned startup evidence

These sources qualify startup behavior, not measured native latency.

## llama.cpp b10809

- [common/common.h](https://github.com/ggml-org/llama.cpp/blob/b10809/common/common.h#L579) enables warm-up by default.
- [common/common.cpp](https://github.com/ggml-org/llama.cpp/blob/b10809/common/common.cpp#L1511-L1546) runs a small encoder/decoder pass, synchronizes, and resets memory/counters. It does not exercise every request shape, and that block does not check decode/encode return codes.
- [tools/server/server.cpp](https://github.com/ggml-org/llama.cpp/blob/b10809/tools/server/server.cpp#L463-L487) marks initial readiness only after model initialization. The initial owned-process health boundary includes built-in warm-up when enabled.
- GPU startup removes `--no-warmup`; CPU retains its existing policy. Later health responses alone do not prove model residency or all kernels are warmed.

### Normal private process logging

Verified against tag `b10809`, commit `5266f24da75dc449bd56cbed7addb9c8e4a6a73e`:

- [Argument handlers](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/arg.cpp#L3890-L3962): `--log-disable` pauses the common logger. `--log-file FNAME` (also `LLAMA_ARG_LOG_FILE`) explicitly opens a file; `--log-prompts-dir PATH` separately enables prompt files. `--log-verbosity 3` selects normal info/warning/error output, below trace (4) and debug (5); `--verbose`/`--log-verbose`/`-v` enable all levels. `--log-colors on` forces colors, unlike terminal-dependent `auto`.
- [Color selection](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.cpp#L413-L425) forces the enabled palette even for captured pipes. [Display macros](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.h#L5-L14) use CSI K + CR, SGR 0/1 and foreground 31-37; all qualify under the viewer's bounded display policy. [Warning emission](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.cpp#L96-L121) always appends SGR 0 with colors enabled, even when the prefix (and its foreground color) is disabled. These pinned source sections were re-read for the color-policy change.
- [Actual sink defaults and routing](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.cpp#L80-L177): the constructor sets `file = nullptr`; generic output goes to stdout and leveled messages to stderr. There is **no default disk log** in this logger. [Paused capture](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.cpp#L200-L209) discards incoming messages; [set_file](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.cpp#L314-L328) is the explicit `fopen(path, "w")` boundary. Do not add a dummy filename or null-device flag to disable a sink that is already absent.
- [Log levels/macros](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.h#L24-L32) and [callback mapping](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.cpp#L441-L458) qualify the threshold. Low-level GGML info/continuation callbacks map to trace (4), so level 3 is not a promise to show every model-loading detail. [Server initialization](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/tools/server/server.cpp#L88-L112) uses the common initializer/parser, not a default file logger.
- [Prompt log default](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/common.h#L517) is an empty directory; [the server write](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/tools/server/server-context.cpp#L4261-L4273) is conditional on a nonempty directory. Neither file-logging option is passed by Freehand.
- [Config discovery precedes environment and argv](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/arg.cpp#L716-L804): Windows system config uses `PROGRAMDATA`; [user config](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/common.cpp#L1102-L1115) requires `APPDATA` with no Windows fallback. The unchanged `ggmlEnvironment` excludes both, all `LLAMA_ARG_*` logging overrides, PATH, proxies, and credentials. Merely overriding a file option later in argv would be too late to prevent an environment-triggered file open.
- Non-debug is **not content-safe**: [server warning/error handling](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/tools/server/server.cpp#L54-L84) includes upstream exception text. [Request/response body dumps](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/tools/server/server-http.cpp#L30-L47) are debug-only, but this does not guarantee prompt/transcript redaction from normal output. Capture stays in bounded private memory; viewer consent and the no-events/no-application-logs/no-disk boundary remain unchanged.

CPU and CUDA argv use `--log-verbosity 3 --log-colors on`.
The focused argument regression covers both backends and rejects other logging
options. The environment regression covers logging and config-file overrides.
The opt-in `TestGGMLPinnedRuntimeZIPs/llama` uses the checksum-pinned CPU ZIP in
an isolated temporary installation and the production owned Windows launcher
with its sanitized environment. It repeats the existing `--offline` flag before
`--help`: [duplicate-flag parsing](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/arg.cpp#L816-L831)
emits a real `LOG_WRN`, unlike printf-based help. The test requires that warning
in private stderr capture, its SGR reset, and no new files beneath the temporary
installation. The prior color-disabled probe failed with `--log-disable` and
passed with verbosity 3; the updated color-enabled ZIP probe has not been rerun
as part of this change. It performs no model download, load, inference, or
user-runtime mutation. Deterministic capture tests cover split ANSI/UTF-8,
progress, hostile control strings, chunk eviction and reset; the existing owned
test-child lifecycle test also exercises colors and hostile OSC through pipes.
This model-free probe does not qualify CUDA execution, sustained request output,
native viewer consent/window behavior, or absence of sensitive content. Upstream
[Windows logger teardown](https://github.com/ggml-org/llama.cpp/blob/5266f24da75dc449bd56cbed7addb9c8e4a6a73e/common/log.cpp#L373-L385)
also warns that unflushed final messages can be lost; an empty tail is not health.

## whisper.cpp v1.8.3

- [examples/server/server.cpp](https://github.com/ggml-org/whisper.cpp/blob/v1.8.3/examples/server/server.cpp#L797-L857) serializes `/inference`, accepts multipart fields, and decodes the upload in memory when FFmpeg conversion is disabled.
- [request parameters](https://github.com/ggml-org/whisper.cpp/blob/v1.8.3/examples/server/server.cpp#L476-L608) qualify the explicit synthetic request options. `temperature_inc=0` prevents temperature retries; do not assume the parsed CLI `--no-fallback` reaches the inference parameters.
- [inference and abort handling](https://github.com/ggml-org/whisper.cpp/blob/v1.8.3/examples/server/server.cpp#L890-L978) executes the loaded model; there is no built-in server warm-up. A client deadline requires owned-process termination after failure because cancellation is cooperative.
- [short-input handling](https://github.com/ggml-org/whisper.cpp/blob/v1.8.3/src/whisper.cpp#L6841-L6848) and [encoding admission](https://github.com/ggml-org/whisper.cpp/blob/v1.8.3/src/whisper.cpp#L7007-L7022) mean a tiny or empty clip can succeed without exercising inference. Use one second of silent mono 16-kHz PCM16 WAV.
- [response/reset](https://github.com/ggml-org/whisper.cpp/blob/v1.8.3/examples/server/server.cpp#L1103-L1113) returns JSON text and resets shared parameters only on normal completion. Require that response shape, discard text (silence may hallucinate), and kill the process after any warm-up error instead of publishing readiness.

The warm-up is a single request in the existing 120-second post-launch readiness
budget, with a 64-KiB response limit and no retries. It is sent only for the
selected CUDA Whisper startup, not metadata checks or catalog operations. The
fixture tests validate the actual multipart/WAV request, protocol rejection, and
cancellation without real models or user audio.

## NeMo-Speech.cpp v0.1.0

- [app/serve.cpp](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/app/serve.cpp#L522-L536) runs warm-up before constructing and binding the HTTP server.
- [src/asr/recognizer.cpp](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/src/asr/recognizer.cpp#L221-L315) exercises the loaded runner with synthetic silence and qualified configured GPU batch shapes. Removing `--no-warmup` for GPU launches restores that provider-owned policy without browsing model inventories.
