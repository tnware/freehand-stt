import { describe, expect, it } from "vitest";
import { State, type Status } from "$lib/state";
import {
  railPhase,
  showIdleShortcutGuidance,
  statusMessage,
} from "$lib/utils/status";

const status = (state: State, canCopy = false, message?: string): Status => ({
  state,
  generation: 1,
  message,
  canCancel: false,
  canCopy,
});

describe("status guidance", () => {
  it("shows the configured shortcut only while idle", () => {
    const shortcut = "Ctrl+Shift+Space";

    expect(showIdleShortcutGuidance(status(State.Idle), shortcut)).toBe(true);
    expect(showIdleShortcutGuidance(status(State.Idle), "")).toBe(false);
    expect(showIdleShortcutGuidance(status(State.Failed), shortcut)).toBe(
      false,
    );
    expect(showIdleShortcutGuidance(status(State.Failed, true), shortcut)).toBe(
      false,
    );
  });

  // The transport's stage has room for a clause, not a sentence, so the reason
  // moved to the messages channel rather than stretching the bar. It still has
  // to reach the user, which is what these assert.
  it("preserves the coordinator reason for ordinary failures", () => {
    const failed = status(State.Failed, false, "The endpoint timed out.");

    expect(statusMessage(failed)).toBe("The endpoint timed out.");
  });

  it("falls back to a reason when the coordinator gives none", () => {
    expect(statusMessage(status(State.Failed))).toContain("could not be transcribed");
  });

  it("explains that copy-required never inserted into the new focus target", () => {
    const waiting = status(State.Failed, true);

    expect(statusMessage(waiting)).toContain("Focus moved");
    expect(statusMessage(waiting)).toContain("nothing was typed");
  });

  it("says nothing when the transport already accounts for the state", () => {
    expect(statusMessage(status(State.Idle))).toBe("");
    expect(statusMessage(status(State.Recording))).toBe("");
    expect(statusMessage(status(State.Transcribing))).toBe("");
    expect(statusMessage(status(State.PostProcessing))).toBe("");
  });
});

describe("progress rail", () => {
  // The rail covers everything between the end of speech and the transcript
  // landing, because from the outside that is one wait.
  it("runs for the whole post-recording stretch", () => {
    expect(railPhase(status(State.Transcribing))).toBe("working");
    expect(railPhase(status(State.PostProcessing))).toBe("working");
    expect(railPhase(status(State.Ready))).toBe("working");
    expect(railPhase(status(State.Cancelling))).toBe("working");
  });

  it("stays out of the way while idle or recording", () => {
    expect(railPhase(status(State.Idle))).toBe("hidden");
    expect(railPhase(status(State.Recording))).toBe("hidden");
  });

  // A transcript waiting to be copied is a finished run, not a broken one.
  it("completes for copy-required and marks a genuine failure", () => {
    expect(railPhase(status(State.Failed, true))).toBe("done");
    expect(railPhase(status(State.Failed, false))).toBe("error");
  });
});
