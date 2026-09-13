---
title: Vocabulary
description: Share names and terminology across Voice and audio-file transcription.
---

Open **Settings → Vocabulary**, or choose **Vocabulary** from the transcription controls. Enter one name or phrase per line. Spaces within phrases are preserved; blank lines and exact duplicates are ignored when sending hints.

Enable **Voice transcription**, **Audio-file transcription**, or both, then choose
**Save** (or **Save and return** when opened from a task). The list and these choices stay saved when you change connections or
models. Voice uses the same list in completed and realtime mode where supported.
Changes apply to the next recording or file job, not work already running.

Check the status for each enabled workflow. If it needs attention, open its help
button to see the selected model's restrictions.

Your current model/backend combination determines how hints are sent:

| Selection                                                               | How the list is used                                                           |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Speaches with a compatible transcription model                          | Recognition hotwords, up to 2,048 UTF-8 bytes                                  |
| A transcription model/backend supporting context hints                  | Appended to your existing context, with a combined 8,192-byte limit            |
| NeMo-Speech.cpp v0.1.0 with the explicit Nemotron 3.5 streaming profile | Vocabulary boosting for completed recordings, audio files, and realtime speech |
| Unsupported combinations                                                | The list is omitted; your preference is kept for a future supported selection  |

Nemotron accepts up to 32 phrases, 128 UTF-8 bytes per phrase, and 2,048 bytes
total. Open **Vocabulary tuning** to adjust **Vocabulary strength** from 0 to 5.
Start at 3 and review the result. Stronger hints can increase incorrect matches,
and the server can cap or disable boosting. Voice and audio files share this
strength setting when they use Nemotron.

The saved list can hold 16,384 UTF-8 bytes. Each supported server and model has
its own smaller request limits. If your selection cannot accept the complete
list, Freehand shows the limit and stops the request before sending audio.
Shorten the list or disable vocabulary for that workflow. Freehand never
silently truncates terms.

Vocabulary is a recognition hint, not a guaranteed correction or a model training operation. Generic compatibility does not prove that every deployed model uses its prompt. Cleanup instructions, S1-mini's trained context categories, and text-to-speech pronunciation remain separate; Freehand does not inject vocabulary into those controls.

Terms are stored locally in settings and sent to the selected transcription
server only when enabled and supported. Review the list before transcribing
sensitive names or switching to a different server.

:::note[Upgrading from separate vocabulary lists]
Freehand combines your saved Voice, realtime, and audio-file terms into one
shared list. Workflows that previously had terms remain enabled and can send
the combined list to their selected server. Before your next transcription,
review the phrases and the **Voice transcription** and **Audio-file transcription**
switches under **Settings → Vocabulary**.
:::

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
available. Save to apply your changes.
