import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import { FileTranscriptionPhase } from "$lib/state";
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
    expect(session.messages.notice).toContain("Streaming can be tried again");
  });
});
