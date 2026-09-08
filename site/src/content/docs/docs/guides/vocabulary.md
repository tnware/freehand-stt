---
title: Vocabulary
description: Share names and terminology across Voice and audio-file transcription.
---

Open **Settings → Vocabulary**, or choose **Vocabulary** from the transcription controls. Enter one name or phrase per line. Spaces within phrases are preserved; blank lines and exact duplicates are ignored when sending hints.

Enable **Voice transcription**, **Audio-file transcription**, or both. These preferences and the list stay with your task when you change connections or models. Voice uses the same list in completed and realtime mode. Save settings to apply changes to the next recording or file job; work already running keeps its original snapshot.

The workflow switches sit directly below the phrase editor. Each row shows whether
hints are enabled or the selection needs attention. Open its help button to see
the selected model, supported hint format, and any restriction. The help beside
**Names and phrases** explains how the shared list is used.

Your current model/backend combination determines how hints are sent:

| Selection                                                               | How the list is used                                                           |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Speaches with a compatible transcription model                          | Recognition hotwords, up to 2,048 UTF-8 bytes                                  |
| A transcription model/backend supporting context hints                  | Appended to your existing context, with a combined 8,192-byte limit            |
| NeMo-Speech.cpp v0.1.0 with the explicit Nemotron 3.5 streaming profile | Vocabulary boosting for completed recordings, audio files, and realtime speech |
| Unsupported combinations                                                | The list is omitted; your preference is kept for a future supported selection  |

Nemotron accepts up to 32 phrases, 128 UTF-8 bytes per phrase, and 2,048 bytes total. Open **Vocabulary tuning** to adjust **Vocabulary strength**, shared between the enabled workflows, from 0 to 5. Start around 2–3. Stronger hints can increase incorrect matches, and the server can cap or disable boosting. These request fields are documented in the runtime's [v0.1.0 API](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/api.md) and [recognition configuration](https://github.com/NVIDIA/NeMo-Speech.cpp/blob/v0.1.0/docs/asr/configuration.md#word-boosting).

The saved library can hold 16,384 UTF-8 bytes. Each adapter has its own smaller request limits. If an enabled supported selection cannot accept the complete list, Freehand shows the limit and stops the request before sending audio. Shorten the list or disable vocabulary for that workflow. Terms are never silently truncated.

Vocabulary is a recognition hint, not a guaranteed correction or a model training operation. Generic compatibility does not prove that every deployed model uses its prompt. Cleanup instructions, S1-mini's trained context categories, and text-to-speech pronunciation remain separate; Freehand does not inject vocabulary into those controls.

Terms are stored locally in the settings database and sent to the selected transcription server only when enabled and supported. Existing active hotwords and realtime vocabulary are combined into the shared list on upgrade; review the combined phrases before your next transcription. Workflows that previously had terms remain opted in. Historical inactive model snapshots remain in storage but no longer restore their old terms when selected.

## Review your list

The editor counts unique phrases and identifies exact duplicates after trimming
surrounding whitespace. Repeated phrases are sent once; Freehand does not rewrite
your saved list. Case differences remain distinct.

Open the **lines to review** summary below the editor to see duplicates and source lines that exceed the selected Voice
or Audio file model's phrase count, per-phrase size, or total vocabulary budget.
For context-based hints, the budget also includes your existing context. Select
**Line …** to highlight that phrase in the editor. Feedback for a workflow that is
turned off is labeled **off**; it does not turn the workflow on.

If the support check fails, choose **Try again** beside the message; your draft stays in place.

Byte limits count UTF-8 bytes, so some characters use more than one byte. The
shared list itself must fit within 16,384 bytes before detailed line feedback is
available. Changes take effect after **Save settings**.
