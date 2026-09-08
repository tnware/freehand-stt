<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="branding/freehand-readme-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="branding/freehand-readme-light.png" />
    <img src="branding/freehand-readme-light.png" alt="Freehand — Speech to text, anywhere you type." width="100%" />
  </picture>
</p>

Freehand is a lightweight Windows client for speech-to-text and text-to-speech
services you choose. Dictate into apps, transcribe recordings, or generate
spoken audio—with models running on your PC, your network, or a compatible
hosted service.

**Free forever. Open source. No Freehand subscription or account.** Your chosen
provider or hosting may have its own costs. You do not need a dedicated local
GPU: send inference to another machine to keep this PC's memory and GPU
available for your other work, or run your models locally if you prefer.

The current alpha is about an 8 MB installer or a 15 MB portable executable.
Freehand stays compact by leaving model inference and model storage on the
speech infrastructure you choose. These download sizes exclude WebView2 and
your inference services; transcription time depends on the model, hardware,
network, and optional cleanup stage.

[Download the latest alpha](https://github.com/tnware/freehand-stt/releases) ·
[Get started](https://tnware.github.io/freehand-stt/docs/getting-started/) ·
[Read the documentation](https://tnware.github.io/freehand-stt/docs/)

## Choose your workflow

- **Voice dictation** — use a global shortcut to capture speech and insert the
  transcript into the original application when the target remains safe.
  Otherwise, the result stays available to copy.
- **Audio file transcription** — select a recording, transcribe it, and copy
  the result explicitly. No microphone or recording shortcut is required.
- **Text to speech** — enter text, generate spoken audio, and play or explicitly
  save it. This optional task uses its own speech-generation service; it does
  not require a transcription connection.

Dictation is the default task, not a setup requirement for the others. Microphone
and file transcription have independent connection/model settings. Optional
[live dictation](https://tnware.github.io/freehand-stt/docs/guides/live-transcription/)
is a mode of the Voice transcription selection with qualified Nemotron / NeMo-Speech.cpp or Qwen3-ASR / vLLM profiles, showing live
results and single-row overlay captions. Cleanup and text-to-speech
can each use a separate service and model.

### Optional transcript cleanup

Leave transcripts unchanged, use custom instructions with a separate chat
model, or select the **S1-mini by Superwhisper** profile for its English
style, structure, and context controls. You provide the inference services;
Freehand does not download or host these models. If cleanup fails or returns
empty text, Freehand falls back to the raw transcript.

[Configure transcript cleanup](https://tnware.github.io/freehand-stt/docs/guides/post-processing/)

## Highlights

- Toggle recording or hold to talk from any Windows application.
- Use independent OpenAI-compatible endpoints for speech recognition,
  optional transcript cleanup, and optional speech playback.
- Keep the current transcript available to copy with history disabled.
- Use local voice detection for silence trimming, automatic stop, and
  pause-aware checkpoints.
- Insert voice transcripts only when the original target remains safe, or copy
  them explicitly.
- Opt into bounded, memory-only history; configure or disable the native status overlay.

Freehand does not bundle a model, inference server, or heavyweight local
runtime.

## Install

Freehand requires Windows 11 with WebView2 and a reachable service compatible
with the task you want to use. Only dictation needs a microphone and recording
shortcut. Download the per-user installer from
[GitHub Releases](https://github.com/tnware/freehand-stt/releases).

The current alpha is not Authenticode-signed, so Windows may identify its
publisher as unknown. Verify manual downloads against the published
`SHA256SUMS` file.

Follow [Get started](https://tnware.github.io/freehand-stt/docs/getting-started/)
to choose a task, connect its service, and complete your first run.

## Documentation

- [Install and update Freehand](https://tnware.github.io/freehand-stt/docs/guides/windows-installer/)
- [Connect a speech server](https://tnware.github.io/freehand-stt/docs/guides/connect-a-server/)
- [Use Freehand](https://tnware.github.io/freehand-stt/docs/guides/using-freehand/)
- [Privacy and safety](https://tnware.github.io/freehand-stt/docs/guides/privacy-and-safety/)
- [Troubleshooting](https://tnware.github.io/freehand-stt/docs/guides/troubleshooting/)

## Contributing

Bug reports, interoperability results, documentation fixes, design feedback,
and code contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for
development setup and pull-request guidance.

Please report vulnerabilities through GitHub private vulnerability reporting,
not a public issue. Freehand is available under the [MIT License](LICENSE).
