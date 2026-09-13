import { describe, it, expect } from "vitest";
import { hideThenToggle } from "./tray-actions";

describe("tray recording boundary", () => {
  it("awaits dismissal before invoking capture or stop", async () => {
    const calls: string[] = [];
    let release!: () => void;
    const hidden = new Promise<void>((resolve) => { release = resolve; });
    const pending = hideThenToggle(async () => { calls.push("hide"); await hidden; }, async () => { calls.push("toggle"); });
    expect(calls).toEqual(["hide"]);
    release(); await pending;
    expect(calls).toEqual(["hide", "toggle"]);
  });
  it("does not touch capture if dismissal fails", async () => {
    let toggled = false;
    await expect(hideThenToggle(async () => { throw new Error("hide failed"); }, async () => { toggled = true; })).rejects.toThrow("hide failed");
    expect(toggled).toBe(false);
  });
});
