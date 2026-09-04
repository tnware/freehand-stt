---
title: Privacy and safety
description: What Freehand sends, retains, stores, and inserts on your Windows PC.
---

Freehand is a desktop client for speech infrastructure you choose. Audio and
transcript text leave the PC only when required by a capability you configured.
The destination may be localhost, a private server, or a hosted provider, so
that server's own privacy and retention policy still applies.

## Audio

Microphone and selected-file audio is kept only for the active transcription
request. Freehand does not retain audio in history or write predictable audio
files to disk. Active audio is released after completion, failure,
cancellation, replacement, or shutdown.

## Transcripts and history

Transcript history is disabled by default. When enabled, it is memory-only,
bounded to 20 entries and 2 MiB, and cleared when Freehand exits. It stores
final transcript text and limited non-secret run details—not audio,
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

## Connection checks

Automatic and manual connection checks request a health route or
`GET /v1/models`. They do not submit audio, prompts, or synthetic inference
jobs, and they do not cycle through discovered models. Model selection itself
does not invoke the model.

## Diagnostics

Operational logs include bounded state, timing, and failure categories. They
exclude audio, transcript text, credentials, private headers, full paths,
model IDs, URL paths and queries, and destination-window identity.
