import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import {
  FileTranscriptionPhase,
  type FileTranscriptionStatus,
} from "$lib/state";
import type { SessionServices } from "$lib/stores/session.svelte";
import { idle, serviceWithStatus, createFiles } from "./session-fixtures";

describe("FileTranscriptionState ordering", () => {
  it("asks Go to open the native picker without sending a path", async () => {
    const selected = {
      generation: 1,
      phase: FileTranscriptionPhase.FileTranscriptionSelected,
      fileName: "meeting.wav",
      fileSize: 4,
      streaming: false,
      buffered: false,
      streamingProfileUnavailable: false,
      streamingUnavailable: false,
      transcriptRevision: 0,
      canStart: true,
      canCancel: false,
      canCopy: false,
    };
    const ChooseAudioFile: SessionServices["files"]["ChooseAudioFile"] = vi.fn(
      () => CancellablePromise.resolve(selected),
    );
    const session = createFiles(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        files: { ChooseAudioFile },
      }),
    );

    await session.files.chooseAudioFile();

    expect(ChooseAudioFile).toHaveBeenCalledWith();
    expect(session.files.status).toEqual(selected);
  });

  it("ignores a late status from an older file generation", () => {
    const session = createFiles(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    session.files.applyStatus({
      generation: 4,
      phase: FileTranscriptionPhase.FileTranscriptionSelected,
      fileName: "new.wav",
      streaming: false,
      buffered: false,
      streamingProfileUnavailable: false,
      streamingUnavailable: false,
      transcriptRevision: 0,
      canStart: true,
      canCancel: false,
      canCopy: false,
    });
    session.files.applyStatus({
      generation: 3,
      phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      fileName: "old.wav",
      transcript: "stale",
      streaming: true,
      buffered: false,
      streamingProfileUnavailable: false,
      streamingUnavailable: false,
      transcriptRevision: 1,
      canStart: true,
      canCancel: false,
      canCopy: true,
    });

    expect(session.files.status.fileName).toBe("new.wav");
    expect(session.files.status.phase).toBe(
      FileTranscriptionPhase.FileTranscriptionSelected,
    );
  });

  it("applies ordered deltas, repairs gaps from a snapshot, and reconciles the final text", async () => {
    const recovered = {
      generation: 4,
      phase: FileTranscriptionPhase.FileTranscriptionStreaming,
      fileName: "meeting.wav",
      transcript: "one two three",
      transcriptRevision: 3,
      streaming: true,
      buffered: false,
      streamingProfileUnavailable: false,
      streamingUnavailable: false,
      canStart: false,
      canCancel: true,
      canCopy: false,
    };
    const session = createFiles(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        files: {
          CurrentFileTranscription: () => CancellablePromise.resolve(recovered),
        },
      }),
    );
    session.files.applyStatus({
      ...recovered,
      transcript: "",
      transcriptRevision: 0,
    });

    expect(
      session.files.applyDelta({ generation: 4, revision: 1, text: "one " }),
    ).toBe("applied");
    expect(
      session.files.applyDelta({ generation: 4, revision: 1, text: "one " }),
    ).toBe("ignored");
    expect(
      session.files.applyDelta({ generation: 3, revision: 2, text: "stale " }),
    ).toBe("ignored");
    expect(
      session.files.applyDelta({ generation: 4, revision: 3, text: "three" }),
    ).toBe("gap");
    expect(session.files.status.transcript).toBe("one ");

    await session.files.refresh();
    expect(session.files.status.transcript).toBe("one two three");
    expect(session.files.status.transcriptRevision).toBe(3);
    expect(
      session.files.applyDelta({ generation: 4, revision: 3, text: "three" }),
    ).toBe("ignored");

    session.files.applyStatus({
      ...recovered,
      phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      transcript: "authoritative final transcript",
      transcriptRevision: 4,
      canStart: true,
      canCancel: false,
      canCopy: true,
    });
    expect(
      session.files.applyDelta({ generation: 4, revision: 5, text: "late" }),
    ).toBe("ignored");
    expect(session.files.status.transcript).toBe(
      "authoritative final transcript",
    );
  });

  it("resets the endpoint streaming capability through the backend", async () => {
    const TryFileStreamingAgain: SessionServices["files"]["TryFileStreamingAgain"] =
      vi.fn(() => CancellablePromise.resolve());
    const session = createFiles(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        files: { TryFileStreamingAgain },
      }),
    );

    await session.files.tryFileStreamingAgain();

    expect(TryFileStreamingAgain).toHaveBeenCalledOnce();
    expect(session.messages.notice).toContain("Text updates selected");
  });
});

describe("file streaming preference", () => {
  it.each([true, false])(
    "keeps preference %s across capability and file changes",
    async (preferred) => {
      const StartFileTranscription = vi.fn(() => CancellablePromise.resolve());
      const session = createFiles(
        serviceWithStatus(() => CancellablePromise.resolve(idle), {
          files: { StartFileTranscription },
        }),
      );
      session.files.streamingPreferred = preferred;
      for (const [generation, unavailable] of [
        [1, true],
        [2, false],
        [3, true],
        [4, false],
      ] as const) {
        session.files.applyStatus({
          ...session.files.status,
          generation,
          streamingUnavailable: unavailable,
          canStart: true,
        });
        expect(session.files.streamingPreferred).toBe(preferred);
        expect(session.files.streamingEnabled).toBe(preferred && !unavailable);
        await session.files.startFileTranscription();
        expect(StartFileTranscription).toHaveBeenLastCalledWith(
          preferred && !unavailable,
        );
      }
    },
  );

  it.each([true, false])(
    "enables streaming only after successful reset (%s), without starting work",
    async (success) => {
      const pending = Promise.withResolvers<void>();
      const TryFileStreamingAgain = vi.fn(
        () =>
          new CancellablePromise<void>((resolve, reject) =>
            pending.promise.then(resolve, reject),
          ),
      );
      const StartFileTranscription = vi.fn(() => CancellablePromise.resolve());
      const session = createFiles(
        serviceWithStatus(() => CancellablePromise.resolve(idle), {
          files: { TryFileStreamingAgain, StartFileTranscription },
        }),
      );
      session.files.streamingPreferred = false;
      session.files.status = { ...session.files.status, canStart: true };
      const reset = session.files.tryFileStreamingAgain();
      expect(session.files.resettingStreaming).toBe(true);
      expect(session.files.streamingPreferred).toBe(false);
      await session.files.tryFileStreamingAgain();
      await session.files.startFileTranscription();
      expect(TryFileStreamingAgain).toHaveBeenCalledOnce();
      if (success) pending.resolve();
      else pending.reject(new Error("reset failed"));
      await reset;
      expect(session.files.streamingPreferred).toBe(success);
      expect(session.files.resettingStreaming).toBe(false);
      expect(StartFileTranscription).not.toHaveBeenCalled();
    },
  );
});

describe("file action admission", () => {
  it.each(["start", "clear"] as const)(
    "serializes %s with selection and resets, and recovers after failure",
    async (action) => {
      const pending = Promise.withResolvers<void>();
      const binding = vi.fn(
        () =>
          new CancellablePromise<void>((resolve, reject) =>
            pending.promise.then(resolve, reject),
          ),
      );
      const choose = vi.fn(() =>
        CancellablePromise.reject<FileTranscriptionStatus>(
          new Error("unexpected picker"),
        ),
      );
      const reset = vi.fn(() => CancellablePromise.resolve());
      const other = vi.fn(() => CancellablePromise.resolve());
      const session = createFiles(
        serviceWithStatus(() => CancellablePromise.resolve(idle), {
          files: {
            StartFileTranscription: action === "start" ? binding : other,
            ClearAudioFile: action === "clear" ? binding : other,
            ChooseAudioFile: choose,
            TryFileStreamingAgain: reset,
          },
        }),
      );
      session.files.status = { ...session.files.status, canStart: true };
      const command = () =>
        action === "start"
          ? session.files.startFileTranscription()
          : session.files.clearAudioFile();
      const first = command();
      expect(session.files.selectionBusy).toBe(true);
      await command();
      await session.files.chooseAudioFile();
      await session.files.tryFileStreamingAgain();
      if (action === "start") await session.files.clearAudioFile();
      else await session.files.startFileTranscription();
      expect(binding).toHaveBeenCalledOnce();
      expect(choose).not.toHaveBeenCalled();
      expect(reset).not.toHaveBeenCalled();
      expect(other).not.toHaveBeenCalled();
      pending.reject(new Error("command failed"));
      await first;
      expect(session.files.selectionBusy).toBe(false);
      binding.mockImplementation(() => CancellablePromise.resolve());
      await command();
      expect(binding).toHaveBeenCalledTimes(2);
    },
  );

  it("allows one cancellation once admitted, even while Start still awaits its reply", async () => {
    const start = Promise.withResolvers<void>();
    const cancel = Promise.withResolvers<void>();
    const CancelFileTranscription = vi.fn(
      () =>
        new CancellablePromise<void>((resolve, reject) =>
          cancel.promise.then(resolve, reject),
        ),
    );
    const session = createFiles(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        files: {
          StartFileTranscription: () =>
            new CancellablePromise<void>((resolve, reject) =>
              start.promise.then(resolve, reject),
            ),
          CancelFileTranscription,
        },
      }),
    );
    session.files.status = { ...session.files.status, canStart: true };
    const starting = session.files.startFileTranscription();
    session.files.applyStatus({
      ...session.files.status,
      phase: FileTranscriptionPhase.FileTranscriptionUploading,
      canStart: false,
      canCancel: true,
    });
    const cancelling = session.files.cancelFileTranscription();
    await session.files.cancelFileTranscription();
    expect(CancelFileTranscription).toHaveBeenCalledOnce();
    expect(session.files.cancelling).toBe(true);
    cancel.reject(new Error("cancel failed"));
    await cancelling;
    expect(session.files.cancelling).toBe(false);
    start.resolve();
    await starting;
  });

  it.each(["older generation", "same generation", "newer selection"])(
    "reconciles a delayed picker reply: %s",
    async (scenario) => {
      const pending = Promise.withResolvers<FileTranscriptionStatus>();
      const ChooseAudioFile = vi.fn(
        () =>
          new CancellablePromise<FileTranscriptionStatus>((resolve, reject) =>
            pending.promise.then(resolve, reject),
          ),
      );
      const session = createFiles(
        serviceWithStatus(() => CancellablePromise.resolve(idle), {
          files: { ChooseAudioFile },
        }),
      );
      const choosing = session.files.chooseAudioFile();
      await session.files.chooseAudioFile();
      expect(ChooseAudioFile).toHaveBeenCalledOnce();
      const current = {
        ...session.files.status,
        generation: 3,
        fileName: "current.wav",
        streamingUnavailable: true,
      };
      session.files.applyStatus(current);
      const reply = {
        ...current,
        generation:
          scenario === "older generation"
            ? 2
            : scenario === "newer selection"
              ? 4
              : 3,
        fileName: "picker.wav",
        streamingUnavailable: false,
      };
      pending.resolve(reply);
      await choosing;
      expect(session.files.status).toEqual(
        scenario === "newer selection" ? reply : current,
      );
      expect(session.files.choosing).toBe(false);
    },
  );
});
