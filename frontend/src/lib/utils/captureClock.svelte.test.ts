import { describe, expect, it } from "vitest";
import { State } from "$lib/state";
import { CaptureClock } from "./captureClock.svelte";

const start = Date.parse("2026-09-08T18:00:00Z");
const recording = {
  generation: 1,
  state: State.Recording,
  startedAt: new Date(start).toISOString(),
};

describe("capture duration", () => {
  it.each(["0001-01-01T00:00:00Z", "", undefined, "invalid"])(
    "freezes the take on a missing/invalid completion start (%s)",
    (startedAt) => {
      const clock = new CaptureClock();
      clock.update(recording, start + 3200);
      expect(clock.seconds).toBe(3);
      clock.update({ ...recording, state: State.Transcribing, startedAt }, start + 4800);
      expect(clock.seconds).toBe(4);
      clock.update({ ...recording, state: State.PostProcessing, startedAt }, start + 90000);
      expect(clock.seconds).toBe(4);
      clock.update({ ...recording, state: State.Failed, startedAt }, start + 120000);
      expect(clock.seconds).toBe(4);
    },
  );
  it("does not carry a previous take into a new generation and resets on idle", () => {
    const clock = new CaptureClock();
    clock.update(recording, start + 5000);
    clock.update(
      { generation: 2, state: State.Recording, startedAt: "0001-01-01T00:00:00Z" },
      start + 6000,
    );
    expect(clock.seconds).toBe(0);
    clock.update(
      { generation: 2, state: State.Recording, startedAt: new Date(start + 6000).toISOString() },
      start + 8000,
    );
    expect(clock.seconds).toBe(2);
    clock.update({ generation: 2, state: State.Idle }, start + 10000);
    expect(clock.seconds).toBe(0);
  });
  it("handles a pane mounted during processing or an invalid recording timestamp", () => {
    for (const startedAt of [
      undefined,
      "invalid",
      "0001-01-01T00:00:00Z",
      new Date(start + 10000).toISOString(),
    ]) {
      const clock = new CaptureClock();
      clock.update({ ...recording, state: State.Transcribing, startedAt }, start);
      expect(clock.seconds).toBe(0);
      clock.update({ ...recording, startedAt }, start);
      expect(clock.seconds).toBe(0);
    }
  });
});
