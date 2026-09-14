# NeMo acquisition fixture provenance

Qualified source: NVIDIA/NeMo-Speech.cpp tag `v0.1.0`, commit
`4f9676226f667d14608487df744f375db87127f8`, inspected in the local source checkout
(`git rev-parse HEAD` and `git describe --tags --exact-match`).

- [app/model_store.cpp:680–750](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/4f9676226f667d14608487df744f375db87127f8/app/model_store.cpp#L680-L750):
  curl writes the supplied output file, resumes existing file downloads with
  `--continue-at -`, and retries without resume when curl returns 33.
  Lines 716–722 use `--silent --show-error --stderr <diagnostic>` when JSON mode,
  quiet mode, **or non-terminal stderr** is in use. Removing `--json` does not
  enable progress under Freehand's private redirected child pipes. No progress
  output fixture is fabricated; there is no byte stream to parse in this mode.
- [app/model_store.cpp:981–1062](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/4f9676226f667d14608487df744f375db87127f8/app/model_store.cpp#L981-L1062):
  `materialize()` writes `directory / (artifact.filename + ".partial")`, verifies
  size/SHA-256 and renames the partial to the destination. Full cached files are
  verified before returning. Diagnostic cleanup remains Freehand-owned after
  cancellation. Qualified models are single `file` artifacts, not TAR prefixes.
- `catalog.go` already pins qualified artifact totals and exact cache paths from
  the release's `share/nemo-speech/model-index.json`.

`nemo_acquisition_test.go` replaces only the OS process boundary and payload bytes:
its controlled child writes a short partial, blocks until measured progress is
observed, then either renames a complete fixture, returns an error, is cancelled,
or publishes corrupt bytes. The real adapter, polling, integrity verification,
worker state and status notifications execute. These are deterministic boundary
tests, **not** native NeMo download or inference acceptance.

Production observes exact file lengths at most five times per second while the
owned CLI is active. Counters include resumed bytes and may decrease when curl
restarts. They are acquisition progress, never verification or success evidence.
The explicit `verifying` phase describes Freehand's own checksum verification
following successful CLI exit; upstream's silent checksum phase cannot itself be
observed. No raw child output, diagnostic text, paths or URLs cross the wire.
