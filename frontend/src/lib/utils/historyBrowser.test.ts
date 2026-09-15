import { describe, expect, it } from "vitest";
import {
  HistoryOutcome,
  HistoryProcessingStatus,
  HistoryResponseMode,
  HistorySource,
  type HistoryEntry,
} from "$bindings/history";
import { filterHistoryEntries, selectedHistoryEntry } from "./historyBrowser";

function entry(id: number, source: HistorySource, text: string): HistoryEntry {
  return {
    id,
    text,
    rawText: text,
    completedAt: "2026-09-14T14:30:00Z",
    characterCount: Array.from(text).length,
    outcome: HistoryOutcome.HistoryInserted,
    processingStatus: HistoryProcessingStatus.HistoryProcessingNotRequested,
    details: {
      source,
      startedAt: "2026-09-14T14:29:00Z",
      route: "/v1/audio/transcriptions",
      authenticationMode: "none",
      model: "private-model-name",
      responseMode: HistoryResponseMode.HistoryResponseCompleted,
      buffered: false,
      vadEnabled: false,
      silenceTrimming: false,
      speechPaddingMilliseconds: 0,
      autoStopEnabled: false,
      autoStopActive: false,
      autoStopped: false,
      silenceSplitting: false,
      segmentsTruncated: false,
      durationLimitReached: false,
      processing: {
        requested: false,
        status: HistoryProcessingStatus.HistoryProcessingNotRequested,
      },
    },
  };
}

const voice = entry(
  3,
  HistorySource.HistorySourceVoice,
  "Send the updated plan.",
);
const audio = entry(
  2,
  HistorySource.HistorySourceAudioFile,
  "The launch is Friday.",
);
audio.rawText = "Um the launch is maybe Thursday.";
audio.processedText = "The launch is confirmed for Friday.";
audio.details.fileName = "Release review.WAV";
const earlier = entry(
  1,
  HistorySource.HistorySourceVoice,
  "Bonjour Élodie — 東京!",
);
const entries = [voice, audio, earlier];

describe("history browsing", () => {
  it("combines the source filter with search and keeps the retained order", () => {
    expect(filterHistoryEntries(entries, "", "voice")).toEqual([
      voice,
      earlier,
    ]);
    expect(filterHistoryEntries(entries, "launch", "voice")).toEqual([]);
    expect(filterHistoryEntries(entries, "launch", "audio-file")).toEqual([
      audio,
    ]);
    expect(filterHistoryEntries(entries, "\t \n", "all")).toEqual(entries);
    expect(entries).toEqual([voice, audio, earlier]);
  });

  it("requires every case-insensitive term across final, raw, cleaned text and filename", () => {
    expect(
      filterHistoryEntries(
        entries,
        "  RELEASE\tThursday\nCONFIRMED Friday ",
        "all",
      ),
    ).toEqual([audio]);
    expect(filterHistoryEntries(entries, "Thursday Monday", "all")).toEqual([]);
    expect(filterHistoryEntries(entries, "updated plan", "all")).toEqual([
      voice,
    ]);
  });

  it("searches Unicode text and treats punctuation as literal text", () => {
    expect(filterHistoryEntries(entries, "ÉLODIE 東京!", "all")).toEqual([
      earlier,
    ]);
    expect(filterHistoryEntries(entries, ".*", "all")).toEqual([]);
  });

  it("does not search request metadata or treat an unknown source as Voice", () => {
    const unknown = entry(0, HistorySource.$zero, "Unavailable source");
    expect(filterHistoryEntries(entries, "private-model-name", "all")).toEqual(
      [],
    );
    expect(filterHistoryEntries(entries, "2026-09-14", "all")).toEqual([]);
    expect(filterHistoryEntries([unknown], "", "all")).toEqual([unknown]);
    expect(filterHistoryEntries([unknown], "", "voice")).toEqual([]);
    expect(filterHistoryEntries([unknown], "", "audio-file")).toEqual([]);
  });

  it("retains the selected entry by ID when new entries arrive or reorder", () => {
    expect(selectedHistoryEntry(entries, audio.id)).toBe(audio);
    expect(selectedHistoryEntry([earlier, voice, audio], audio.id)).toBe(audio);
    const zero = entry(0, HistorySource.HistorySourceVoice, "Zero ID");
    expect(selectedHistoryEntry([voice, zero], 0)).toBe(zero);
  });

  it("falls back after filtering or deletion and leaves empty results unselected", () => {
    const filtered = filterHistoryEntries(entries, "updated", "all");
    expect(selectedHistoryEntry(filtered, audio.id)).toBe(voice);
    expect(selectedHistoryEntry(entries, null)).toBe(voice);
    expect(selectedHistoryEntry(entries, 999)).toBe(voice);
    expect(selectedHistoryEntry([], audio.id)).toBeUndefined();
    expect(
      selectedHistoryEntry(
        filterHistoryEntries(entries, "absent", "all"),
        null,
      ),
    ).toBeUndefined();
  });
});
