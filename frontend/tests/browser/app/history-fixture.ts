import { CancellablePromise } from "@wailsio/runtime";
import {
  HistoryOutcome,
  HistoryProcessingStatus,
  HistoryResponseMode,
  HistorySource,
  InsertionMode,
  RecordingMode,
  type HistoryEntry,
} from "$lib/state";
import type { HistoryStateService } from "$lib/stores/history.svelte";
import { historyEntry } from "$lib/stores/session-fixtures-data";

declare global {
  interface Window {
    testHistory: {
      appendVoice: () => Promise<void>;
      failNextClear: () => void;
    };
  }
}

function entry(
  id: number,
  text: string,
  source = HistorySource.HistorySourceVoice,
  rawText = text,
): HistoryEntry {
  const completedAt = `2026-09-14T12:${String(id * 10).padStart(2, "0")}:00Z`;
  const file = source === HistorySource.HistorySourceAudioFile;
  const model = `fixture/${file ? "file" : "voice"}-${id}`;
  const elapsedMilliseconds = file ? 1900 : 8000;
  const processingStatus =
    rawText === text
      ? HistoryProcessingStatus.HistoryProcessingNotRequested
      : HistoryProcessingStatus.HistoryProcessingCompleted;
  return {
    ...structuredClone(historyEntry),
    id,
    text,
    rawText,
    processedText: rawText === text ? undefined : text,
    characterCount: text.length,
    completedAt,
    processingStatus,
    outcome:
      source === HistorySource.HistorySourceAudioFile
        ? HistoryOutcome.HistoryTranscribed
        : HistoryOutcome.HistoryCopyRequired,
    details: {
      ...structuredClone(historyEntry.details),
      source,
      fileName: file ? "Microphone check.wav" : undefined,
      fileSize: file ? 384044 : undefined,
      startedAt: new Date(
        Date.parse(completedAt) - elapsedMilliseconds,
      ).toISOString(),
      completedAt,
      elapsedMilliseconds,
      server: `https://${file ? "files" : "voice"}.example.test/v1`,
      authenticationMode: "none",
      model,
      language: file ? "fr" : "en",
      insertionMode: file
        ? InsertionMode.ManualCopy
        : InsertionMode.DirectInput,
      responseMode: file
        ? HistoryResponseMode.HistoryResponseStreamed
        : HistoryResponseMode.HistoryResponseCompleted,
      microphone: file ? undefined : "Desk microphone",
      recordingMode: file ? undefined : RecordingMode.RecordingToggle,
      captureDurationMilliseconds: file ? undefined : 5000,
      audioDurationMilliseconds: file ? 12000 : 4700,
      uploadMilliseconds: file ? 600 : undefined,
      transcriptionMilliseconds: file ? 1300 : 2300,
      requestTimeoutSeconds: file ? 120 : 90,
      transcription: {
        requestId: `fixture-request-${id}`,
        responseId: `${file ? "file" : "voice"}-response-${id}`,
        effectiveModel: model,
        provider: "Synthetic test server",
        finishReason: "stop",
        detectedLanguages: [file ? "fr" : "en"],
        serverAudioSeconds: file ? 12 : 4.7,
        usage: {
          type: "tokens",
          inputTokens: 24,
          outputTokens: 9,
          totalTokens: 33,
        },
        performance: { generatedTokens: 9, generationMilliseconds: 300 },
        requestCount: 1,
        usageReportCount: 1,
        performanceReportCount: 1,
      },
      processing: {
        ...historyEntry.details.processing,
        requested: rawText !== text,
        status: processingStatus,
        server:
          rawText === text ? undefined : "https://cleanup.example.test/v1",
        model: rawText === text ? undefined : "fixture/cleanup",
        preset: rawText === text ? undefined : "generic",
        elapsedMilliseconds: rawText === text ? undefined : 700,
        rawCharacterCount: rawText.length,
        processedCharacters: rawText === text ? undefined : text.length,
        deliveredCharacters: text.length,
      },
    },
  };
}

/** Memory-only backend data for the real App and History pane. */
export function createHistoryFixture() {
  const readingText =
    "\n\nThis longer paragraph keeps the selected transcript independently scrollable while the run information remains available in its own pane.".repeat(
      18,
    );
  let entries = [
    entry(3, "Prepare the release checklist."),
    entry(
      2,
      "Confirm the microphone settings.",
      HistorySource.HistorySourceAudioFile,
    ),
    entry(
      1,
      "Ship the updated voice workflow." + readingText,
      HistorySource.HistorySourceVoice,
      "um ship the updated voice workflow" + readingText,
    ),
  ];
  let rejectClear = false;
  const service: HistoryStateService = {
    TranscriptHistory: () =>
      CancellablePromise.resolve(structuredClone(entries)),
    CopyHistoryEntry: () => CancellablePromise.resolve(),
    CopyHistoryEntryVersion: () => CancellablePromise.resolve(),
    DeleteHistoryEntry: (id) => {
      entries = entries.filter((item) => item.id !== id);
      return CancellablePromise.resolve();
    },
    ClearHistory: () => {
      if (rejectClear) {
        rejectClear = false;
        return CancellablePromise.reject(
          new Error("Fixture history clear failed. Try again."),
        );
      }
      entries = [];
      return CancellablePromise.resolve();
    },
  };
  return {
    service,
    failNextClear: () => {
      rejectClear = true;
    },
    appendVoice: () => {
      entries = [
        entry(4, "Incoming transcript stays in the sidebar."),
        ...entries,
      ];
    },
  };
}
