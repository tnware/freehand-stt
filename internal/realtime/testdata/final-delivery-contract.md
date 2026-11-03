# Realtime final-delivery qualification

These are protocol/source checks, not native macOS or Windows acceptance tests.

## NeMo-Speech.cpp v0.1.0

Pinned release commit: `4f9676226f667d14608487df744f375db87127f8`.

- [HTTP server, lines 888–927](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/4f9676226f667d14608487df744f375db87127f8/server/http/http_server.cpp#L888-L927) assigns a unique `event_id` and emits a completed transcription only for an authoritative `result.is_final`. Each final contains that turn's text, not the cumulative session transcript.
- [Audio append, lines 929–945](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/4f9676226f667d14608487df744f375db87127f8/server/http/http_server.cpp#L929-L945) emits finalized turns while audio is still arriving.
- [Commit, lines 1015–1023](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/4f9676226f667d14608487df744f375db87127f8/server/http/http_server.cpp#L1015-L1023) flushes the final result before acknowledging the recording commit.

Freehand deduplicates final IDs, rejects conflicting reuse, and never deduplicates by phrase text. Legacy finals without IDs remain compatible with stop-time delivery, but cannot authorize live insertion. Losing this evidence after a live attempt permanently stops further automatic insertion. Caption deltas never authorize insertion.

## vLLM v0.28.0

- [Realtime connection, lines 154–159](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/realtime/connection.py#L154-L159) ends audio input on `final: true`.
- [Generation, lines 238–258](https://github.com/vllm-project/vllm/blob/v0.28.0/vllm/entrypoints/speech_to_text/realtime/connection.py#L238-L258) emits deltas followed by one completed result. The qualified Freehand contract admits that result only after recording stops; it does not reinterpret token deltas as finalized phrases.

## Delivery boundary

Live typing is opt-in, restricted to qualified NeMo/Nemotron realtime with automatic insertion and without whole-transcript cleanup. One coalescing owner sends only the new suffix of monotonically growing authoritative text to the original captured target. Failed or uncertain dispatch is never retried. Stop joins the owner and does not replay text. Cancellation does not undo already posted text. Platform dispatch success is not an application-level receipt.
