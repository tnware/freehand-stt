import { afterEach, describe, expect, it, vi } from "vitest";
import { CPUState, type Snapshot } from "$bindings/resources";
import {
  ResourceState,
  formatMemory,
  memoryPercent,
  gpuPercent,
  gpuMemoryPercent,
} from "./resources.svelte";

const sample: Snapshot = {
  sampledAt: 100,
  cpuState: CPUState.CPUReady,
  cpuPercent: 42,
  memoryAvailable: true,
  memoryTotalBytes: 16 * 1024 ** 3,
  memoryAvailableBytes: 4 * 1024 ** 3,
  gpus: [],
};

afterEach(() => vi.useRealTimers());

describe("resource sampling", () => {
  it("samples only while visible and clears values on hide/dispose", async () => {
    vi.useFakeTimers();
    const Current = vi.fn().mockResolvedValue(sample);
    const state = new ResourceState({ Current });
    expect(Current).not.toHaveBeenCalled();
    state.setVisible(true);
    await vi.advanceTimersByTimeAsync(0);
    expect(state.snapshot).toEqual(sample);
    await vi.advanceTimersByTimeAsync(4_000);
    expect(Current).toHaveBeenCalledTimes(3);
    state.setVisible(false);
    expect(state.snapshot).toBeNull();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(Current).toHaveBeenCalledTimes(3);
    state.setVisible(true);
    await vi.advanceTimersByTimeAsync(0);
    expect(Current).toHaveBeenCalledTimes(4);
    state.dispose();
    state.setVisible(true);
    await vi.advanceTimersByTimeAsync(10_000);
    expect(state.snapshot).toBeNull();
    expect(Current).toHaveBeenCalledTimes(4);
  });

  it("serializes reads and rejects late responses from a hidden workspace", async () => {
    vi.useFakeTimers();
    let resolve!: (value: Snapshot) => void;
    const Current = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise<Snapshot>((done) => {
            resolve = done;
          }),
      )
      .mockResolvedValue(sample);
    const state = new ResourceState({ Current });
    state.setVisible(true);
    state.setVisible(false);
    state.setVisible(true);
    await vi.advanceTimersByTimeAsync(10_000);
    expect(Current).toHaveBeenCalledTimes(1);
    resolve({ ...sample, cpuPercent: 99 });
    await vi.advanceTimersByTimeAsync(0);
    expect(Current).toHaveBeenCalledTimes(2);
    expect(state.snapshot?.cpuPercent).toBe(42);
    state.dispose();
  });

  it("expires stale metrics during a stalled read and recovers after errors", async () => {
    vi.useFakeTimers();
    let reject!: () => void;
    const Current = vi
      .fn()
      .mockResolvedValueOnce(sample)
      .mockImplementationOnce(
        () =>
          new Promise<Snapshot>((_, fail) => {
            reject = () => fail(new Error("unavailable"));
          }),
      )
      .mockResolvedValue(sample);
    const state = new ResourceState({ Current });
    state.setVisible(true);
    await vi.advanceTimersByTimeAsync(6_000);
    expect(Current).toHaveBeenCalledTimes(2);
    expect(state.snapshot).toBeNull();
    expect(state.unavailable).toBe(true);
    reject();
    await vi.advanceTimersByTimeAsync(2_000);
    expect(state.snapshot).toEqual(sample);
    expect(state.unavailable).toBe(false);
    state.dispose();
  });

  it("does not expose a response after disposal", async () => {
    let resolve!: (value: Snapshot) => void;
    const state = new ResourceState({
      Current: () =>
        new Promise((done) => {
          resolve = done;
        }),
    });
    state.setVisible(true);
    state.dispose();
    resolve(sample);
    await Promise.resolve();
    expect(state.snapshot).toBeNull();
  });

  it("keeps unavailable memory distinct from zero use and formats binary capacity", () => {
    expect(memoryPercent(null)).toBeNull();
    expect(memoryPercent({ ...sample, memoryAvailable: false })).toBeNull();
    expect(memoryPercent(sample)).toBe(75);
    expect(
      memoryPercent({
        ...sample,
        memoryAvailableBytes: sample.memoryTotalBytes,
      }),
    ).toBe(0);
    expect(formatMemory(sample.memoryTotalBytes)).toBe("16.0 GiB");
  });
  it("reports the busiest available GPU and keeps unified RAM separate from VRAM capacity", () => {
    const gpu = {
      name: "GPU",
      utilizationAvailable: true,
      utilizationPercent: 90,
      memoryAvailable: true,
      memoryUsedBytes: 8,
      memoryTotalBytes: 10,
      sharedMemoryAvailable: false,
      sharedMemoryUsedBytes: 0,
      unifiedMemory: false,
    };
    expect(
      gpuPercent({
        ...sample,
        gpus: [gpu, { ...gpu, utilizationPercent: 20 }],
      }),
    ).toBe(90);
    expect(
      gpuPercent({
        ...sample,
        gpus: [{ ...gpu, utilizationAvailable: false }],
      }),
    ).toBeNull();
    expect(gpuMemoryPercent(gpu)).toBe(80);
    expect(gpuMemoryPercent({ ...gpu, unifiedMemory: true })).toBeNull();
    expect(gpuMemoryPercent({ ...gpu, memoryTotalBytes: 0 })).toBeNull();
  });
});
