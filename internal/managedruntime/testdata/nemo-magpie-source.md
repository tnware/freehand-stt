# Qualified NeMo combined speech bundle

The optional `magpie-tts` selection uses the **v2602** checkpoint in the
[NeMo-Speech.cpp v0.1.0 model index](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/models/index.json#L91-L186).
It is not the later v2607 Hugging Face checkpoint.

- Magpie: `nvidia/magpie_tts_multilingual_357m`, revision
  `452ef560f972c38d5fc16476259aac9456453547`.
- Main GGUF: `magpie_tts_multilingual_357m.v2602.f16.gguf`, 448,604,832 bytes.
- Tokenizer: the first 33,554,432 bytes of `magpie_tts_multilingual_357m.nemo`,
  extracted by the pinned model manager into `tokenizer/`. Its ten members
  have individual size/SHA-256 pins in `nemo_bundle.go`.
- Codec: `nvidia/nemo-nano-codec-22khz-1.89kbps-21.5fps`, revision
  `fc00890b604aa2de298d2641ffc6c5f6caf8c4d7`, 78,823,104 bytes.

The user-requested `model pull` acquires all artifacts and companions.
The [pinned model store](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/app/model_store.cpp#L938-L1017)
verifies the tokenizer tar prefix, extracts only indexed members, and deletes
the prefix after successful extraction. Freehand verifies the main model,
codec, and every final tokenizer member before marking the bundle installed
or starting it; an additional tokenizer input is rejected. Removal deletes
only the exact Magpie and codec revision directories, preserving ASR assets.

[Server startup](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/app/serve.cpp#L455-L525)
loads enabled engines and warms them before binding HTTP. Freehand supplies
one verified ASR path plus optional verified TTS, codec, and tokenizer paths.
Readiness expects exactly the selected capabilities, then publishes their
distinct reported model identities on one process and endpoint generation.

`--access-log --log-format json` enables bounded private request records while
retaining the normal startup text. Global `--json` is omitted because pinned
`serve.cpp` disables ASR status text in that mode. No request body or headers
are enabled by these switches.

The deterministic bundle tests replace artifact bytes and the child boundary.
They do not execute a runtime binary, download weights, or establish native
Windows/macOS inference acceptance.
