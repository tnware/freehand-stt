import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import {
  FileTranscriptionPhase,
  State,
  TTSPhase,
  type FileTranscriptionStatus,
  type Status,
} from "$lib/state";
import {
  Session,
  idle,
  recording,
  historyEntry,
  settings,
  serviceWithStatus,
} from "./session-fixtures";
import {
  subscribeSessionEvents,
  type SessionEventMap,
  type SubscribeSessionEvent,
} from "./session-events";

function eventSource() {
  const handlers = new Map<string, Set<(data: unknown) => void>>();
  const on: SubscribeSessionEvent = (name, callback) => {
    const handle = (data: unknown) =>
      callback({ data: data as SessionEventMap[typeof name] });
    const entries = handlers.get(name) ?? new Set();
    entries.add(handle);
    handlers.set(name, entries);
    return () => {
      entries.delete(handle);
    };
  };
  return {
    on,
    emit<K extends keyof SessionEventMap>(name: K, data: SessionEventMap[K]) {
      handlers.get(name)?.forEach((handle) => handle(data));
    },
    count: () =>
      [...handlers.values()].reduce((sum, entries) => sum + entries.size, 0),
  };
}

const streaming: FileTranscriptionStatus = {
  generation: 4,
  phase: FileTranscriptionPhase.FileTranscriptionStreaming,
  streaming: true,
  buffered: false,
  streamingProfileUnavailable: false,
  streamingUnavailable: false,
  transcriptRevision: 2,
  transcript: "first words",
  canStart: false,
  canCancel: true,
  canCopy: false,
};

describe("shared session event composition", () => {
  it("subscribes before loading and keeps a live dictation event over a late snapshot", async () => {
    const snapshot = CancellablePromise.withResolvers<Status>();
    const session = new Session(serviceWithStatus(() => snapshot.promise));
    const events = eventSource();
    const edge = vi.fn();
    const off = subscribeSessionEvents(session, events.on, edge);
    try {
      const loading = session.load();
      events.emit("dictation:status", recording);
      snapshot.resolve(idle);
      await loading;
      expect(session.dictation.status).toEqual(recording);
      expect(edge).toHaveBeenCalledExactlyOnceWith(recording, idle);
      expect(session.editor.applied).toEqual({ ...settings, microphoneID: "" });
    } finally {
      off();
      session.dispose();
    }
  });

  it("refreshes history only for an accepted active-to-terminal file event", async () => {
    const entries = CancellablePromise.withResolvers<(typeof historyEntry)[]>();
    const TranscriptHistory = vi.fn(() => entries.promise);
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        history: { TranscriptHistory },
      }),
    );
    const events = eventSource();
    const off = subscribeSessionEvents(session, events.on);
    try {
      events.emit("file-transcription:status", streaming);
      events.emit("file-transcription:status", {
        ...streaming,
        generation: 3,
        phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      });
      events.emit("file-transcription:status", {
        ...streaming,
        transcriptRevision: 1,
        phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      });
      expect(TranscriptHistory).not.toHaveBeenCalled();
      expect(session.files.status).toEqual(streaming);
      const completed = {
        ...streaming,
        phase: FileTranscriptionPhase.FileTranscriptionCompleted,
        canCancel: false,
        canCopy: true,
      };
      events.emit("file-transcription:status", completed);
      expect(TranscriptHistory).toHaveBeenCalledTimes(1);
      entries.resolve([historyEntry]);
      await vi.waitFor(() =>
        expect(session.history.entries).toEqual([historyEntry]),
      );
      expect(session.files.historyGeneration).toBe(completed.generation);
      events.emit("file-transcription:status", completed);
      expect(TranscriptHistory).toHaveBeenCalledTimes(1);
    } finally {
      off();
      session.dispose();
    }
  });

  it("routes a delta gap to the file owner's authoritative snapshot", async () => {
    const snapshot =
      CancellablePromise.withResolvers<FileTranscriptionStatus>();
    const CurrentFileTranscription = vi.fn(() => snapshot.promise);
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        files: { CurrentFileTranscription },
      }),
    );
    const events = eventSource();
    const off = subscribeSessionEvents(session, events.on);
    try {
      events.emit("file-transcription:status", streaming);
      events.emit("file-transcription:delta", {
        generation: 4,
        revision: 4,
        text: "missing prefix",
      });
      expect(CurrentFileTranscription).toHaveBeenCalledTimes(1);
      const recovered = {
        ...streaming,
        transcriptRevision: 4,
        transcript: "complete recovered text",
      };
      snapshot.resolve(recovered);
      await vi.waitFor(() => expect(session.files.status).toEqual(recovered));
    } finally {
      off();
      session.dispose();
    }
  });

  it("isolates window drafts while sharing event semantics and fully detaches on unmount", async () => {
    const first = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    const second = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    const events = eventSource();
    const offFirst = subscribeSessionEvents(first, events.on);
    const offSecond = subscribeSessionEvents(second, events.on);
    try {
      await Promise.all([first.editor.load(), second.editor.load()]);
      first.editor.apiKey = "transient-draft";
      first.editor.draft!.model = "unsaved";
      events.emit("settings:changed", { ...settings, model: "committed" });
      expect(first.editor.draft?.model).toBe("unsaved");
      expect(second.editor.applied?.model).toBe("committed");
      expect(second.editor.apiKey).toBe("");
      offFirst();
      offFirst();
      events.emit("tts:status", {
        ...second.speech.status,
        generation: 9,
        phase: TTSPhase.Failed,
        message: "playback failed",
      });
      expect(second.messages.error).toBe("playback failed");
      expect(first.speech.status.generation).toBe(0);
      offSecond();
      expect(events.count()).toBe(0);
      const offAgain = subscribeSessionEvents(second, events.on);
      expect(events.count()).toBe(5);
      offAgain();
      events.emit("dictation:status", {
        ...recording,
        state: State.Transcribing,
      });
      expect(second.dictation.status).toEqual(idle);
    } finally {
      offFirst();
      offSecond();
      first.dispose();
      second.dispose();
    }
  });

  it("releases earlier subscriptions if registration fails partway through", () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    const events = eventSource();
    let calls = 0;
    const on: SubscribeSessionEvent = (name, callback) => {
      if (++calls === 3) throw new Error("registration failed");
      return events.on(name, callback);
    };
    expect(() => subscribeSessionEvents(session, on)).toThrow(
      "registration failed",
    );
    expect(events.count()).toBe(0);
    session.dispose();
  });
});
