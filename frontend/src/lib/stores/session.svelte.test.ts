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
    snapshot.resolve(idle);
    await loading;

    expect(session.dictation.status).toEqual(recording);
    expect(session.editor.processingProfiles).toEqual(processingProfiles);
  });

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
