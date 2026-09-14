# Pinned startup evidence

These sources qualify startup behavior, not measured native latency.

## llama.cpp b10809

- [common/common.h](https://github.com/ggml-org/llama.cpp/blob/b10809/common/common.h#L579) enables warm-up by default.
- [common/common.cpp](https://github.com/ggml-org/llama.cpp/blob/b10809/common/common.cpp#L1511-L1546) runs a small encoder/decoder pass, synchronizes, and resets memory/counters. It does not exercise every request shape, and that block does not check decode/encode return codes.
- [tools/server/server.cpp](https://github.com/ggml-org/llama.cpp/blob/b10809/tools/server/server.cpp#L463-L487) marks initial readiness only after model initialization. The initial owned-process health boundary includes built-in warm-up when enabled.
- GPU startup removes `--no-warmup`; CPU retains its existing policy. Later health responses alone do not prove model residency or all kernels are warmed.

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
