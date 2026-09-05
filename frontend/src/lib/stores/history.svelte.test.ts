import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import { FileTranscriptionPhase, type HistoryEntry } from "$lib/state";
import type { SessionServices } from "$lib/stores/session.svelte";
import {
  Session,
  idle,
  recording,
  historyEntry,
  serviceWithStatus,
} from "./session-fixtures";

describe("Session transcript history", () => {
  it("loads history from its dedicated binding", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        history: {
          TranscriptHistory: () => CancellablePromise.resolve([historyEntry]),
        },
      }),
    );

    await session.load();

    expect(session.history.entries).toEqual([historyEntry]);
  });

  it("does not let an older refresh overwrite a newer history result", async () => {
    const older = CancellablePromise.withResolvers<HistoryEntry[] | null>();
    let request = 0;
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        history: {
          TranscriptHistory: () => {
            request++;
            return request === 1
              ? older.promise
              : CancellablePromise.resolve([historyEntry]);
          },
        },
      }),
    );

    const staleRefresh = session.history.refresh();
    await session.history.refresh();
    older.resolve([]);
    await staleRefresh;

    expect(session.history.entries).toEqual([historyEntry]);
  });

  it("marks a completed file generation synchronized only after history refreshes", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        history: {
          TranscriptHistory: () => CancellablePromise.resolve([historyEntry]),
        },
      }),
    );
    session.files.applyStatus({
      generation: 7,
      phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      fileName: "recording.mp3",
      transcript: historyEntry.text,
      streaming: true,
      buffered: false,
      streamingProfileUnavailable: false,
      streamingUnavailable: false,
      transcriptRevision: 1,
      canStart: true,
      canCancel: false,
      canCopy: true,
    });

    expect(session.files.historyGeneration).toBe(0);
    await session.history.refresh();

    expect(session.files.historyGeneration).toBe(7);
    expect(session.history.entries).toEqual([historyEntry]);
  });

  it("copies, removes, and clears only through explicit history actions", async () => {
    const CopyHistoryEntry: SessionServices["history"]["CopyHistoryEntry"] =
      vi.fn(() => CancellablePromise.resolve());
    const ClearHistory: SessionServices["history"]["ClearHistory"] = vi.fn(() =>
      CancellablePromise.resolve(),
    );
    const DeleteHistoryEntry: SessionServices["history"]["DeleteHistoryEntry"] =
      vi.fn(() => CancellablePromise.resolve());
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        history: { CopyHistoryEntry, DeleteHistoryEntry, ClearHistory },
      }),
    );
    session.history.entries = [historyEntry];

    await session.history.copyHistoryEntry(historyEntry.id);
    expect(CopyHistoryEntry).toHaveBeenCalledWith(historyEntry.id);
    expect(session.history.entries).toEqual([historyEntry]);

    await session.history.deleteHistoryEntry(historyEntry.id);
    expect(DeleteHistoryEntry).toHaveBeenCalledWith(historyEntry.id);
    expect(session.history.entries).toEqual([]);

    session.history.entries = [historyEntry];

    await session.history.clearHistory();
    expect(ClearHistory).toHaveBeenCalledOnce();
    expect(session.history.entries).toEqual([]);
  });
});
