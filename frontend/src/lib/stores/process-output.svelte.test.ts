import { describe, expect, it, vi } from "vitest";
import {
  ProcessOutputState,
  type ProcessOutputService,
} from "./process-output.svelte";
import type { OutputSnapshot } from "$bindings/managedruntime/models";

function boundary(
  overrides: Partial<ProcessOutputService> = {},
): ProcessOutputService {
  return {
    EnableProcessOutput: vi.fn(async () => {}),
    DisableProcessOutput: vi.fn(async () => {}),
    ClearProcessOutput: vi.fn(async () => {}),
    ReadProcessOutput: vi.fn(async () => ({
      enabled: true,
      chunks: [
        {
          sequence: 1,
          timestamp: 0,
          stream: "stdout",
          text: "<script>private</script>\n",
        },
      ],
      next: 1,
      truncated: false,
    })),
    ...overrides,
  };
}
describe("private process output", () => {
  it("drops prior launch text when the backend resets the tail", async () => {
    let reset = false;
    const state = new ProcessOutputState(
      boundary({
        ReadProcessOutput: async () => ({
          enabled: true,
          chunks: reset
            ? []
            : [
                {
                  sequence: 1,
                  timestamp: 0,
                  stream: "stdout",
                  text: "old launch",
                },
              ],
          next: reset ? 2 : 1,
          truncated: reset,
        }),
      }),
    );
    state.select("one");
    await state.show();
    await state.poll();
    expect(state.chunks).toHaveLength(1);
    const revision = state.revision;
    reset = true;
    await state.poll();
    expect(state.revision).toBeGreaterThan(revision);
    expect(state.chunks).toEqual([]);
    expect(state.accepted).toBe(true);
  });
  it("keeps renewed consent when an earlier enable finishes after reopening", async () => {
    let finish!: () => void;
    let first = true;
    let enabled = false;
    const state = new ProcessOutputState(
      boundary({
        EnableProcessOutput: async () => {
          if (first) {
            first = false;
            await new Promise<void>((resolve) => {
              finish = resolve;
            });
          }
          enabled = true;
        },
        DisableProcessOutput: async () => {
          enabled = false;
        },
      }),
    );
    state.select("one");
    const old = state.show();
    await Promise.resolve();
    state.select("");
    state.select("one");
    const reopened = state.show();
    finish();
    await Promise.all([old, reopened]);
    expect(state.accepted).toBe(true);
    expect(enabled).toBe(true);
  });
  it("revokes an enable that finishes after native close", async () => {
    let finish!: () => void;
    let enabled = false;
    const service = boundary({
      EnableProcessOutput: async () => {
        await new Promise<void>((resolve) => {
          finish = resolve;
        });
        enabled = true;
      },
      DisableProcessOutput: async () => {
        enabled = false;
      },
    });
    const state = new ProcessOutputState(service);
    state.select("one");
    const showing = state.show();
    await Promise.resolve();
    state.select("");
    finish();
    await showing;
    expect(state.accepted).toBe(false);
    expect(enabled).toBe(false);
  });
  it("bounds both UTF-8 bytes and chunk count without damaging Unicode", async () => {
    let sequence = 0;
    const state = new ProcessOutputState(
      boundary({
        ReadProcessOutput: async () => ({
          enabled: true,
          chunks: Array.from({ length: 1100 }, () => ({
            sequence: ++sequence,
            timestamp: 0,
            stream: "stderr",
            text: "🙂".repeat(100),
          })),
          next: sequence,
          truncated: false,
        }),
      }),
    );
    state.select("one");
    await state.show();
    await state.poll();
    expect(state.chunks.length).toBeLessThanOrEqual(1024);
    expect(
      new TextEncoder().encode(state.chunks.map((c) => c.text).join("")).length,
    ).toBeLessThanOrEqual(256 * 1024);
    expect(state.truncated).toBe(true);
    expect(state.chunks.map((c) => c.text).join("")).not.toContain("�");
  });
  it("never overlaps reads and discards late results after clear, close, and switch", async () => {
    let resolve!: (s: OutputSnapshot) => void;
    const service = boundary({
      ReadProcessOutput: vi.fn(
        () =>
          new Promise<OutputSnapshot>((r) => {
            resolve = r;
          }),
      ),
    });
    const state = new ProcessOutputState(service);
    state.select("one");
    await state.show();
    const read = state.poll();
    await state.poll();
    expect(service.ReadProcessOutput).toHaveBeenCalledTimes(1);
    await state.clear();
    resolve({
      enabled: true,
      chunks: [{ sequence: 1, timestamp: 0, stream: "stdout", text: "secret" }],
      next: 1,
      truncated: false,
    });
    await read;
    expect(state.chunks).toEqual([]);
    expect(state.cursor).toBe(0);
    expect(service.ClearProcessOutput).toHaveBeenCalledWith({
      instanceID: "one",
    });
    const next = state.poll();
    state.select("two");
    resolve({
      enabled: true,
      chunks: [
        { sequence: 2, timestamp: 0, stream: "stdout", text: "old secret" },
      ],
      next: 2,
      truncated: false,
    });
    await next;
    expect(state.chunks).toEqual([]);
    await state.show();
    state.dispose();
    await state.poll();
    expect(state.instanceID).toBe("");
    expect(state.accepted).toBe(false);
  });
  it("does not collect while hidden and keeps polling errors content-free", async () => {
    const service = boundary({
      ReadProcessOutput: vi.fn(async () => {
        throw new Error("private path");
      }),
    });
    const state = new ProcessOutputState(service);
    state.select("one");
    await state.show();
    state.visible = false;
    await state.poll();
    expect(service.ReadProcessOutput).not.toHaveBeenCalled();
    state.visible = true;
    await state.poll();
    expect(state.error).toBe("Could not read process output.");
  });
  it("requires explicit consent before enabling or reading, and clears on switch", async () => {
    const service = boundary();
    const state = new ProcessOutputState(service);
    state.select("one");
    await state.poll();
    expect(service.EnableProcessOutput).not.toHaveBeenCalled();
    expect(service.ReadProcessOutput).not.toHaveBeenCalled();
    await state.show();
    await state.poll();
    expect(state.chunks[0].text).toBe("<script>private</script>\n");
    await state.select("two");
    expect(state.chunks).toEqual([]);
    expect(state.accepted).toBe(false);
    expect(state.cursor).toBe(0);
    expect(service.DisableProcessOutput).toHaveBeenCalledWith({
      instanceID: "one",
    });
  });
});
