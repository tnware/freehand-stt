---
title: Privacy and safety
description: What Freehand sends, retains, stores, and inserts on Windows and macOS.
---

Freehand sends audio or text to the services you choose for transcription,
optional cleanup, and on-demand speech. Each service may run on this computer,
a private server, or a hosted provider. **The server's own privacy and retention
policy still applies.**

:::tip[Check every stage]
Local transcription does not make cleanup or speech generation local. Review
the selected connection for each task before sending sensitive audio or text.
:::

## What goes where?

| When you use… | What leaves Freehand | Destination |
| --- | --- | --- |
| Microphone or audio-file transcription | Recorded or selected-file audio | Selected transcription endpoint |
| Optional cleanup | Completed transcript | Selected cleanup endpoint |
| Speak, Preview, or Listen | Your text, a sample phrase, or chosen transcript | Selected speech endpoint |
| Authenticated requests | Configured API key | Selected capability endpoint |
| Update checks | A release-metadata request; no recordings or transcripts | GitHub release service |
| Managed runtime/model installation | Explicit binary or selected-model download requests | Official pinned release and model hosts |

Local retention has separate limits for [audio](#audio),
[current results and history](#transcripts-and-history),
[settings](#saved-settings-and-backups), and [runtime output](#diagnostics).

## Managed local recognition

Managed runtime listeners accept connections from **this computer only**.
They are not exposed to your LAN, but other local software may still access a
loopback service.

| Part of the workflow | Keep it on this computer |
| --- | --- |
| Voice or audio-file recognition | Select the managed transcription Connection |
| Optional cleanup | Select [managed llama.cpp with S1-mini](../local-runtime/#local-cleanup-with-s1-mini), another local cleanup service, or leave cleanup off |
| Text to speech | Select [managed NeMo with MagpieTTS](../local-runtime/#local-speech-with-magpietts), another local speech service, or leave speech off |

- **Downloads:** runtime binaries and selected model weights stay in Freehand's
  application-data directory until removed. Installation needs internet access;
  browsing the catalog does not run inference.
- **Saved manual connections:** remain available with their keys. Those keys are
  never sent to the managed runtime.
- **Runtime failure:** never silently redirects captured audio to a remote server.
  Recover local setup or explicitly select a manual connection for new work.
- **Removal:** runtime/model removal leaves source recordings, manual connections,
  and other applications' model caches intact. Weights are installation data,
  separate from captured audio and transcript history.

<details>
<summary>What happens during selected-model GPU warm-up?</summary>

Starting a selected GPU runtime includes warm-up, including start-at-launch.
llama.cpp and NeMo use built-in warm-up. CUDA whisper.cpp receives one second
of synthetic silence on this computer, using only its selected loaded model;
Freehand discards the response.

This captures no microphone audio, creates no history entry, invokes no other
catalog models, and contacts no remote inference server. Connection checks
remain metadata-only.

</details>

## Audio

| Audio source | Freehand's lifetime |
| --- | --- |
| Microphone capture | Active transcription request only; released and temporary audio deleted after success, failure, or cancellation |
| Selected audio file | Active request only; original file is never modified or deleted |
| Generated speech | Memory until cleared, replaced, a recording begins, or Freehand exits; saving a file is explicit |

Audio is never retained in transcript history. Your chosen inference server
may retain audio or text under its own policy.

## Transcripts and history

| Text | Where it lives | When it clears |
| --- | --- | --- |
| Latest dictation result | Memory; available with history off | Next recording, Clear, or exit |
| Current file result | Memory; available with history off | Clear/replace the file, another transcription, or exit |
| Unsent speech-composer draft | Current window session; survives task/settings navigation | Window reload or exit |
| Optional transcript history | Memory; **off by default**, at most **20 entries and 2 MiB** | Removal, disabled/cleared history, retention limits, or exit |

Composer text is never written to browser storage or the settings database.
History contains raw and cleaned text with limited non-secret run details. It
contains no audio, credentials, request headers, full paths, or destination
identity. **Transcription details** disappear with their history entry; an open
details window does not keep a separate copy.

Keeping both versions after successful cleanup requires enabled history and
space within its limits. Raw fallback after failed cleanup and failed-insertion
recovery work with history off.

File results always need explicit **Copy**. Voice results use focus-safe
insertion or manual copy according to your settings. See
[session history](../history/) to enable, compare, and remove retained results.

## Safe text insertion

Start dictation with the intended app and text field focused. Keep the
destination focused until delivery finishes; Freehand never brings it to the
foreground for you.

| Platform | What must remain focused |
| --- | --- |
| Windows | Original app, window, and focused control |
| macOS | Same app and window; moving to another field in that window delivers to the currently focused field |

Release physical shortcut modifiers before delivery. Windows waits briefly for
held Ctrl, Alt, Shift, or Windows keys; if they stay held, use **Copy**.

:::caution[Secure Input cannot identify every password field]
macOS Secure Input blocks delivery while active. Custom secure fields that do
not enable it may not be detected. Avoid dictating sensitive text into an
uncertain destination.
:::

If the destination changes or safe delivery is unavailable, the result stays
available for explicit **Copy**. **Check for partially inserted text before
pasting** to avoid duplicates.

Clipboard-paste insertion is disabled. Automatic delivery does not silently
replace your clipboard.

## Saved settings and backups

| Platform | Settings and backups | Window geometry |
| --- | --- | --- |
| Windows | `%LOCALAPPDATA%\Freehand\freehand.db` | `%APPDATA%\Freehand\window-state.json` |
| macOS | `~/Library/Application Support/Freehand/freehand.db` | `window-state.json` in the same directory |

The database stores non-secret configuration such as server addresses, model
choices, vocabulary, custom instructions, and request headers. API keys remain
in the native credential store; history stays in memory.

:::caution[Settings and backups are private, not encrypted]
Windows database access is restricted to your user and SYSTEM. macOS uses
per-user file permissions. Neither database is encrypted, and Keychain denial
never triggers plaintext credential storage.
:::

| Upgrade or recovery action | Retained data |
| --- | --- |
| Earlier alpha settings and native credentials | Left untouched, not imported or reused; see the [first-launch reset notice](../../getting-started/#first-launch) |
| Future schema upgrades | Up to three database backups |
| Explicit recovery reset | An archive of the replaced current database |

Read [settings recovery](../troubleshooting/#saved-settings-need-attention)
before restoring or removing these files.

## Credentials and transport

| Protection | What it means |
| --- | --- |
| Native credential storage | Saved keys go to Windows Credential Manager or macOS Keychain, never the settings database, and are not displayed again |
| Inactive connections | Keep their saved keys; deleting a connection removes its key only when no saved connection still uses it |
| HTTPS by default | HTTP requires explicit permission per connection |
| No HTTP redirects | Inference and connection checks never follow redirects, even to another path on the same server; configure the final base URL |

:::caution[HTTP sends content without encryption]
Allow it only for a trusted local or LAN endpoint. Audio, transcript text, and
credentials are unencrypted in transport; do not allow HTTP across an untrusted
network.
:::

Freehand filters literal API-key copies from server response details and rejects
transcript text containing the key, including realtime captions and finals.
Realtime checks span messages and model-specific parsing. This cannot make an
untrusted server safe: it already received the key and could misuse or transform
it. Connect only to services you trust. See [saved connections](../saved-connections/)
for credential management.

## Connection checks

| Trigger | Behavior |
| --- | --- |
| Finish initial **Voice** setup | Requires an explicit connection test; Audio file and Text to speech remain independently usable |
| Launch after Voice setup, or relevant saved connection changes | Automatically checks the saved STT connection |
| Automatic or manual connection check | Reads a health route or `GET /v1/models` |
| Select a model | Does not invoke it |

Checks send no audio, prompts, or synthetic inference jobs and never cycle
through discovered models. Startup warm-up belongs to the selected managed
runtime, as described [above](#managed-local-recognition).

## Update checks

| Update action | Timing or effect |
| --- | --- |
| Automatic checks | On by default; about 30 seconds after startup, then daily after a successful check |
| Failed automatic check | Retries after 15 minutes while Freehand is running |
| Development build | Does not check automatically |
| Manual check | **About → Check now** |
| Disable automatic checks | **Settings → General** |

Checks read GitHub release metadata without sending recordings or transcripts.
When an update is available, the updater can download and checksum-verify the
platform asset, then waits for you to restart. macOS uses an app-bundle ZIP;
checksum verification does not establish Developer ID trust or notarization.
See [macOS updates](../macos-setup/#update) or
[Windows updates](../windows-installer/#upgrade).

## Diagnostics

### Application logs

Operational logs record status, timing, and failure categories. They exclude
audio, transcripts, credentials, private headers, full paths, model IDs, URL
paths/queries, and destination-window identity.

### Update helper logs

When you restart into an update, Wails separately writes
`wails-update-<pid>.log` in the operating system's temporary directory.

| Included | Excluded | Lifetime |
| --- | --- | --- |
| Installation/temporary paths, process IDs, replacement errors; paths may reveal your account name | Recordings, transcripts, inference API keys | Remains until you or the operating system remove it |

Review that file before sharing it for support.

### Managed runtime output

:::caution[Process output can contain sensitive content]
The runtime may write transcripts, prompts, paths, or other private upstream
text. Freehand cannot promise complete redaction. Turning transcript history
off does not prevent that text appearing in output, and opening a viewer during
screen sharing can expose it.
:::

| Output storage | Limit or lifetime |
| --- | --- |
| Private rolling tail | Memory only, at most **256 KiB and 1,024 chunks**; oldest text drops as it fills |
| Collection | Continues with the viewer closed or scrolling paused |
| Clear, next start attempt, runtime removal, or Quit | Discards the private tail |
| Application logs, events, and crash reports | Never receive the captured output |

For llama.cpp, capture includes normal information, warnings, and errors rather
than debug logging. The private tail is not saved as a log file.

| Viewer | When text is shown |
| --- | --- |
| Embedded **Runtime output** tab | Immediately on opening or switching runtime tabs; retains the selected runtime across workflows, Connections, Local runtime, and History |
| Standalone **Process output** window | Requires **Show output** each opening or runtime switch |
| Hidden viewer, different tab/runtime, global Settings, or hidden workspace | Clears displayed text and revokes reads; leaves the runtime and private tail running |
| Reopened embedded tab | Immediately reads the available tail again |

Both viewers are read-only, with search, colors, and progress updates. They have
no command input or file logging. **Copy selection** copies only your selected
text when you ask; other applications may read it, and it may remain on the
clipboard after the viewer closes.

See [runtime output controls](../local-runtime/#startup-and-process-output)
for viewing, clearing, and copying output.
