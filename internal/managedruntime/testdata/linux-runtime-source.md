# Linux CPU runtime qualification

The Linux amd64 recipes retain the existing provider pins. No inference runtime
is built from source. Other Linux architectures and accelerators remain unavailable.

- NVIDIA NeMo-Speech.cpp v0.1.0: official release API and downloaded archive
  `nemo-speech-0.1.0-linux-x86_64-cpu.tar.gz`, 4,583,913 bytes,
  SHA256 `0f74131d631ad2c694cf0ec53490866bb6461147959589a69fb6fc231944065b`.
  Executable `nemo-speech-0.1.0-linux-x86_64-cpu/bin/nemo-speech` has ELF RUNPATH
  `$ORIGIN/../lib`. Versioned same-directory `.so` aliases become regular copies.
  [Official release](https://github.com/NVIDIA/NeMo-Speech.cpp/releases/tag/v0.1.0).
- ggml-org/llama.cpp b10809: official release API and downloaded archive
  `llama-b10809-bin-ubuntu-x64.tar.gz`, 16,734,586 bytes,
  SHA256 `5e34434ddc6d03cd1584f403201aff0d4bd1a5793a72ff7e286532dfd1e4b941`.
  Executable `llama-b10809/llama-server` has ELF RUNPATH `$ORIGIN`.
  [Pinned release workflow](https://github.com/ggml-org/llama.cpp/blob/b10809/.github/workflows/release.yml)
  builds the x64 CPU package on Ubuntu 22.04 with runtime CPU dispatch.
- whisper.cpp v1.8.3 has no official Linux server binary archive and remains
  unavailable in managed mode; manual Connections remain usable.
  [Official assets](https://github.com/ggml-org/whisper.cpp/releases/expanded_assets/v1.8.3).

The pinned llama parser applies `/etc/llama.cpp/config.ini` before argv, and
Linux user-config discovery can recover the passwd home when HOME is absent.
Managed startup rejects existing/unreadable system configuration and supplies
private HOME/XDG directories. This is conservative admission, not an OS sandbox
against a privileged actor changing system configuration concurrently.
[Parser](https://github.com/ggml-org/llama.cpp/blob/b10809/common/arg.cpp),
[Directory discovery](https://github.com/ggml-org/llama.cpp/blob/b10809/common/common.cpp).
NeMo's pinned downloader invokes curl without `--disable`; Linux therefore also
uses private HOME/CURL_HOME/XDG paths and rejects unexpected curl configuration.
[Downloader](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/app/model_store.cpp).

`TestLinuxOfficialRuntimeArchives` accepts only explicitly supplied pinned
archives. `FREEHAND_LINUX_RUNTIME_EXECUTE=1` separately opts into owned `--help`
execution. `TestLinuxSelectedModelAcceptance` requires an explicit selected
model and is never enabled automatically in CI. Its optional fixture mode
verifies pre-downloaded files and exercises local installation, cache checks,
loading, bounded inference and stop, but does not qualify direct network download.
Synthetic audio is memory-only and response content is discarded.
