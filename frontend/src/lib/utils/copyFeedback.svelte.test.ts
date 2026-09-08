import { afterEach, describe, expect, it, vi } from "vitest";
import { CopyFeedback } from "./copyFeedback.svelte";

afterEach(() => vi.useRealTimers());
describe("copy confirmation", () => {
  it("gives each successful click its full confirmation interval", async () => {
    vi.useFakeTimers();
    const feedback = new CopyFeedback();
    await feedback.copy("result", async () => true);
    vi.advanceTimersByTime(1200);
    await feedback.copy("result", async () => true);
    vi.advanceTimersByTime(1200);
    expect(feedback.key).toBe("result");
    vi.advanceTimersByTime(400);
    expect(feedback.key).toBe("");
    feedback.dispose();
  });
  it("ignores older completions, failures, and completions after teardown", async () => {
    vi.useFakeTimers();
    const feedback = new CopyFeedback();
    let resolve!: (value: boolean) => void;
    const older = feedback.copy(
      "old",
      () =>
        new Promise<boolean>((done) => {
          resolve = done;
        }),
    );
    await feedback.copy("new", async () => true);
    resolve(true);
    await older;
    expect(feedback.key).toBe("new");
    await feedback.copy("failed", async () => false);
    expect(feedback.key).toBe("new");
    const pending = feedback.copy(
      "pending",
      () =>
        new Promise<boolean>((done) => {
          resolve = done;
        }),
    );
    feedback.dispose();
    resolve(true);
    await pending;
    expect(feedback.key).toBe("");
    expect(vi.getTimerCount()).toBe(0);
  });
});
