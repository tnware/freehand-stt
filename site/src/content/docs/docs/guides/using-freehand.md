---
title: Use Freehand
description: Voice dictation, stored audio, transcript history, cleanup, and speech playback.
---

Freehand has three first-class workspaces: **Voice**, **Audio file**, and optional **Text to speech**. Dictation and stored-audio transcription share the same speech endpoint but deliberately have different delivery behavior.

## Voice dictation

### Toggle recording

Press the toggle shortcut once to begin and again to stop. When local VAD is enabled, a toggle recording can also finish after speech has been detected and the configured silence countdown expires.

The destination is captured when recording begins. If that destination is no
longer safe when transcription finishes, Freehand leaves the result available
to copy instead of inserting it elsewhere.

### Hold to talk

Press and hold the configured chord, speak, then release it to stop. Automatic
silence ending does not stop a hold-to-talk recording before you release the
shortcut.

A hold-to-talk chord may use the normal modifier-plus-key forms or a supported modifier-only combination. See [Shortcut policy](../../reference/shortcuts/) for the exact accepted key grammar and default bindings.

### Long dictation and VAD

Optional local WebRTC VAD can provide:

- stable live speech/silence feedback;
- leading and trailing silence trimming with retained context;
- speech-armed automatic stop for toggle recordings;
- pause-aware checkpoints for longer dictation.

Checkpoint requests are sequential and assembled in order. Freehand still performs one final focus-safe insertion rather than pasting partial segments while you speak.

## Audio files

The **Audio file** workspace opens a native picker, validates the selected
file, and sends its audio to the configured transcription endpoint.

- Supported formats: FLAC, MP3, MP4, MPEG, MPGA, M4A, OGG, WAV, and WebM.
- Jobs can be cancelled.
- File work and microphone capture are mutually exclusive.
- Completed JSON and compatible server-sent-event responses are supported.
- A stored-file result always requires explicit copy; it is never inserted automatically.

The interface shows the file name, size, progress, status, and transcript. It
does not expose the full path to page content.

## The rack

The column on the left of the main window applies frequently changed values immediately, without a save step. Its first card keeps the most frequently changed behavior controls together:

- **Capture** — microphone, voice detection, checkpoints, and the recording overlay.
- **Delivery** — whether transcripts are typed straight in or wait to be copied, and whether history is kept.

The infrastructure controls follow beneath it:

- **Speech to text** — endpoint, model, and a connection test.
- **Cleanup** — on or off, endpoint, model, processing profile, and the S1-mini styling, structure, and context controls.

Each row states its own name and value, and each group has a door to the matching Settings section. Speech to text and Cleanup can be collapsed independently; their choices persist across launches and their headers continue to show connection health and the selected model.

Credentials, authentication mode, plaintext-HTTP policy, custom instructions, and other advanced values remain in the full Settings window.

Settings edits apply to the next operation and do not change a request already
in progress.

## Transcript cleanup

Freehand can send the raw transcript to an independently configured OpenAI-compatible chat endpoint after speech recognition.

- **Raw mode** leaves transcription untouched.
- **Custom instructions** use a bounded user-authored system instruction.
- **S1-mini** uses the model's exact trained v1 prompt and control vocabulary.

Raw text remains available when cleanup is disabled, fails, times out, or returns no usable output. Read [Transcript post-processing](../post-processing/) before configuring S1-mini or a local llama.cpp route.

## History

Transcript history is off by default. When enabled, it is:

- memory-only;
- limited to 20 entries and 2 MiB total;
- cleared at shutdown;
- restricted to finalized transcript text and bounded non-secret run details.

History never retains audio, credentials, full file paths, or destination
details. Each copy action is explicit.

## Text to speech

Optional speech playback uses its own endpoint, model, voice ID, credential, transport policy, and request budget.

You can listen to retained completed transcripts or use the bounded **Text to
speech** composer. Generated audio remains in memory until you clear or replace
it, start a recording, or exit Freehand. You can pause, resume, replay, save,
clear, or stop the current result.

This is deliberate on-demand playback. Automatic read-aloud, conversation sequencing, barge-in, and echo control remain outside the feature.

## Status overlay and tray

The optional native overlay mirrors bounded operational state such as recording, speech/silence, checkpoints, processing, and copy-required outcomes. It never renders transcript text or changes focus.

The tray can show or hide Freehand, open Settings or About, cancel active work,
copy an available recovery transcript, and quit.

Tray **Quit** is authoritative. Closing the main or Settings window hides it; it does not end the resident utility.
