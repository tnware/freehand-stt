---
title: Privacy and safety
description: What Freehand sends, retains, stores, and inserts on your Windows PC.
---

Freehand is a desktop client for speech infrastructure you choose. Audio and
transcript text leave the PC only when required by a capability you configured.
The destination may be localhost, a private server, or a hosted provider, so
that server's own privacy and retention policy still applies.

## What goes where?

| Data | Destination | What Freehand keeps |
| --- | --- | --- |
| Microphone or selected-file audio | Your configured speech-to-text endpoint | Audio for the active request; released afterward. Existing source files are unchanged. |
| Transcript sent for cleanup | Your separate cleanup endpoint, when enabled | Keeping both versions after successful cleanup requires enabled session history. Raw failure fallback does not require history. |
| API keys | The configured capability endpoint when authentication is enabled | Saved keys in Windows Credential Manager. |
| Transcript history | Memory on your PC | Off by default; at most 20 entries and 2 MiB, cleared on exit. |
| Speech playback text and audio | Your playback endpoint receives text and returns audio | Generated audio in memory until cleared, replaced, a recording begins, or Freehand exits; saving a file is explicit. |
| Update checks | GitHub release service | Update metadata and any downloaded update; no recordings or transcripts are sent. |

The details below describe the lifetime of each kind of data. Your chosen
server's retention policy applies to anything sent to it.

## Audio

Microphone and selected-file audio is kept only for the active transcription
request. Freehand does not retain audio in history or write predictable audio
files to disk. Active audio is released after completion, failure,
cancellation, replacement, or shutdown. Selecting an existing audio file does
not delete or modify the original file.

Optional text-to-speech is separate: generated playback audio remains in
memory until cleared or replaced, a recording begins, or Freehand exits. You
can explicitly save generated audio. Your chosen inference server may retain
audio or text according to its own policy.

## Transcripts and history

Transcript history is disabled by default. When enabled, it is memory-only,
bounded to 20 entries and 2 MiB, and cleared when Freehand exits. It stores
raw and cleaned transcript text and limited non-secret run details—not audio,
credentials, request headers, full file paths, or destination-window identity.

Stored-audio results require an explicit Copy action. Voice dictation can use
focus-safe direct insertion or manual copy, according to your settings.

## Safe text insertion

Freehand records the destination when voice capture begins. Before delivering
the final text, it verifies that the same application, process, and focused
control still own the destination. If that check fails, Freehand does not
switch applications or type into a different window; it leaves the transcript
available for you to copy.

Clipboard-paste insertion is not enabled. Freehand does not silently replace
the clipboard as part of automatic delivery.

## Credentials and transport

API keys are stored in Windows Credential Manager. They are not written to the
JSON settings file or returned to the interface after saving.

HTTPS is required by default. You can explicitly allow HTTP for a trusted local
or LAN endpoint, but doing so sends audio, transcript text, and credentials
without transport encryption. Do not enable it across an untrusted network.

Inference and connection-check requests never follow HTTP redirects, even to
another path on the same server. Configure the final base URL instead of a
redirecting alias; Freehand will not forward your key, audio, or text to the
redirect destination.

If a server echoes the request's API key literally in optional response details
or discovered model IDs, Freehand removes those values before returning them
to the interface or session history. Valid transcript text and unrelated
details remain available. Text containing the key is rejected. This is a guard
against literal reflection, not protection against a malicious server encoding
or otherwise transforming a key it already received.

## Connection checks

First launch requires an explicit connection test. After setup, Freehand
checks the saved speech connection automatically on launch and after relevant
connection settings change. Automatic and manual checks request a health route or
`GET /v1/models`. They do not submit audio, prompts, or synthetic inference
jobs, and they do not cycle through discovered models. Model selection itself
does not invoke the model.

## Update checks

Automatic update checks are on by default. Freehand checks GitHub release
metadata shortly after startup and once per day. If an update is available,
the updater can download and verify it, then waits for you to restart. You can
disable automatic checks under **Settings → General**. These checks do not
send recordings or transcripts to GitHub.

## Diagnostics

Operational logs include bounded state, timing, and failure categories. They
exclude audio, transcript text, credentials, private headers, full paths,
model IDs, URL paths and queries, and destination-window identity.
