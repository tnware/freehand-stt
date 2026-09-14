import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import { TTSPhase, type Status } from "$lib/state";

import {
  Session,
  processingProfiles,
  idle,
  recording,
  serviceWithStatus,
} from "./session-fixtures";

describe("Session status loading", () => {
  it("keeps a live event that arrives while the initial snapshot is pending", async () => {
    const snapshot = CancellablePromise.withResolvers<Status>();
    const session = new Session(serviceWithStatus(() => snapshot.promise));

    const loading = session.load();
    session.dictation.applyStatus(recording);
    snapshot.resolve({ ...idle, generation: recording.generation });
    await loading;

    expect(session.dictation.status).toEqual(recording);
    expect(session.editor.processingProfiles).toEqual(processingProfiles);
  });

  it.each(["older-first", "newer-first"])(
    "keeps the newer dictation snapshot when overlapping loads finish %s",
    async (order) => {
      const older = CancellablePromise.withResolvers<Status>();
      const newer = CancellablePromise.withResolvers<Status>();
      const CurrentStatus = vi
        .fn()
        .mockImplementationOnce(() => older.promise)
        .mockImplementationOnce(() => newer.promise);
      const session = new Session(serviceWithStatus(CurrentStatus));
      try {
        const first = session.dictation.load();
        const second = session.dictation.load();
        const current = { ...recording, generation: 2 };
        if (order === "older-first") {
          older.resolve({ ...idle, generation: 1 });
          await first;
          newer.resolve(current);
          await second;
        } else {
          newer.resolve(current);
          await second;
          older.resolve({ ...idle, generation: 1 });
          await first;
        }
        expect(session.dictation.status).toEqual(current);
      } finally {
        session.dispose();
      }
    },
  );

  it("applies the initial snapshot when no event arrives", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(recording)),
    );

    await session.load();

    expect(session.dictation.status).toEqual(recording);
  });
});

describe("Session owner composition", () => {
  it("keeps identically named generated status methods in separate namespaces", async () => {
    const bindings = serviceWithStatus(() =>
      CancellablePromise.resolve(recording),
    );
    const speechStatus = {
      ...(await bindings.speech.CurrentStatus()),
      generation: 12,
      phase: TTSPhase.Paused,
    };
    bindings.speech.CurrentStatus = vi.fn(() =>
      CancellablePromise.resolve(speechStatus),
    );
    const session = new Session(bindings);
    await session.load();
    expect(session.dictation.status).toEqual(recording);
    expect(session.speech.status).toEqual(speechStatus);
    session.dispose();
  });

  it("derives aggregate busy state without owning duplicate feature flags", () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    expect(session.busy).toBe(false);
    session.editor.saving = true;
    expect(session.busy).toBe(true);
    session.editor.saving = false;
    session.speech.previewing = true;
    expect(session.busy).toBe(true);
    session.speech.previewing = false;
    expect(session.busy).toBe(false);
    session.dispose();
  });

  it.each(["dictation", "speech", "files", "settings", "devices"])(
    "does not start later initialization work after disposal during %s loading",
    async (stage) => {
      const pending = CancellablePromise.withResolvers<void>();
      const session = new Session(
        serviceWithStatus(() => CancellablePromise.resolve(idle)),
      );
      const loads = [
        vi.spyOn(session.dictation, "load").mockImplementation(async () => {
          if (stage === "dictation") await pending.promise;
        }),
        vi.spyOn(session.speech, "load").mockImplementation(async () => {
          if (stage === "speech") await pending.promise;
        }),
        vi.spyOn(session.files, "refresh").mockImplementation(async () => {
          if (stage === "files") await pending.promise;
        }),
        vi.spyOn(session.editor, "load").mockImplementation(async () => {
          if (stage === "settings") await pending.promise;
          return true;
        }),
        vi
          .spyOn(session.editor, "refreshDevices")
          .mockImplementation(async () => {
            if (stage === "devices") await pending.promise;
          }),
        vi.spyOn(session.history, "refresh").mockResolvedValue(),
      ];
      vi.spyOn(session.runtime, "load").mockResolvedValue(true);
      const blocked = [
        "dictation",
        "speech",
        "files",
        "settings",
        "devices",
      ].indexOf(stage);
      const loading = session.load();
      await vi.waitFor(() => expect(loads[blocked]).toHaveBeenCalledOnce());
      session.dispose();
      pending.resolve();
      await loading;
      for (const next of loads.slice(blocked + 1)) {
        expect(next).not.toHaveBeenCalled();
      }
    },
  );

  it("does not publish a late initialization failure after disposal", async () => {
    const pending = CancellablePromise.withResolvers<Status>();
    const session = new Session(serviceWithStatus(() => pending.promise));
    const loading = session.load();
    session.dispose();
    pending.reject(new Error("initialization failed after teardown"));
    await loading;
    expect(session.messages.error).toBe("");
  });

  it("does not initialize a disposed session again", async () => {
    const CurrentStatus = vi.fn(() => CancellablePromise.resolve(idle));
    const session = new Session(serviceWithStatus(CurrentStatus));
    const runtimeLoad = vi.spyOn(session.runtime, "load");
    session.dispose();
    await session.load();
    expect(CurrentStatus).not.toHaveBeenCalled();
    expect(runtimeLoad).not.toHaveBeenCalled();
  });

  it("disposes presentation timers and all credential drafts without stopping backend work", async () => {
    vi.useFakeTimers();
    const bindings = serviceWithStatus(() => CancellablePromise.resolve(idle));
    bindings.dictation.Cancel = vi.fn(bindings.dictation.Cancel);
    bindings.speech.Stop = vi.fn(bindings.speech.Stop);
    const session = new Session(bindings);
    try {
      await session.editor.load();
      await session.editor.updateQuickSettings(
        { autoInsert: false },
        "delivery",
      );
      session.editor.apiKey = "stt-draft";
      session.editor.processingAPIKey = "processing-draft";
      session.editor.ttsAPIKey = "speech-draft";
      session.messages.reportInfo("informational notice");
      session.messages.announce("saved notice");
      expect(vi.getTimerCount()).toBe(3);
      session.dispose();
      expect(vi.getTimerCount()).toBe(0);
      expect([
        session.editor.apiKey,
        session.editor.processingAPIKey,
        session.editor.ttsAPIKey,
      ]).toEqual(["", "", ""]);
      expect(session.messages.info).toBe("");
      expect(session.messages.notice).toBe("");
      expect(bindings.dictation.Cancel).not.toHaveBeenCalled();
      expect(bindings.speech.Stop).not.toHaveBeenCalled();
    } finally {
      session.dispose();
      vi.useRealTimers();
    }
  });
});
