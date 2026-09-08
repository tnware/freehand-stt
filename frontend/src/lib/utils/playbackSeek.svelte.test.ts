import { describe, expect, it, vi } from "vitest";
import { PlaybackSeek, playbackSteps, playbackSliderPosition } from "./playbackSeek.svelte";
import { SpeechState } from "$lib/stores/speech.svelte";
import { SessionMessages } from "$lib/stores/messages.svelte";
import { CancellablePromise } from "@wailsio/runtime";
import { idle, serviceWithStatus } from "$lib/stores/session-fixtures";
const messages = new SessionMessages();
const status = {
  ...new SpeechState(serviceWithStatus(() => CancellablePromise.resolve(idle)).speech, messages)
    .status,
  generation: 3,
  canSeek: true,
  positionMilliseconds: 1000,
  durationMilliseconds: 10000,
};
describe("playback drag preview", () => {
  it("projects irregular native progress onto valid steps including a partial final step", () => {
    for (const duration of [43, 60000, 60043]) {
      const steps = playbackSteps(duration);
      expect(steps[0]).toBe(0);
      expect(steps.at(-1)).toBe(duration);
      for (const position of [0, 137, 274, 411, 60040, duration]) {
        const bounded = Math.min(position, duration);
        const value = playbackSliderPosition(bounded, duration);
        expect(steps).toContain(value);
        expect(Math.abs(value - bounded)).toBeLessThanOrEqual(50);
      }
      expect(playbackSliderPosition(duration, duration)).toBe(duration);
    }
  });
  it("holds the drag position over progress updates and sends only the committed target", async () => {
    const seek = new PlaybackSeek();
    const send = vi.fn(async () => {});
    seek.change(status, 2500);
    seek.change(status, 3500);
    expect(seek.position({ ...status, positionMilliseconds: 1500 })).toBe(3500);
    expect(send).not.toHaveBeenCalled();
    await seek.commit(status, send);
    expect(send).toHaveBeenCalledExactlyOnceWith({ generation: 3, positionMilliseconds: 3500 });
    expect(seek.position(status)).toBe(1000);
  });
  it("drops a drag from replaced or cleared audio", async () => {
    for (const current of [
      { ...status, generation: 4 },
      { ...status, canSeek: false },
    ]) {
      const seek = new PlaybackSeek();
      const send = vi.fn(async () => {});
      seek.change(status, 6000);
      await seek.commit(current, send);
      expect(send).not.toHaveBeenCalled();
      expect(seek.draft).toBeNull();
    }
  });
  it("serializes commits and releases pending state after rejection", async () => {
    const seek = new PlaybackSeek();
    const pending = Promise.withResolvers<void>();
    const send = vi.fn(() => pending.promise);
    seek.change(status, 2000);
    const commit = seek.commit(status, send);
    seek.change(status, 5000);
    await seek.commit(status, send);
    expect(send).toHaveBeenCalledTimes(1);
    pending.reject(new Error("unavailable"));
    await expect(commit).rejects.toThrow("unavailable");
    expect(seek.pending).toBe(false);
    expect(seek.draft).toBeNull();
  });
});
