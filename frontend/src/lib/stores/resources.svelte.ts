import type { GPU, Snapshot } from "$bindings/resources";

export interface ResourceService {
  Current(): Promise<Snapshot>;
}

/** One serialized reader, active only while the main workspace is visible. */
export class ResourceState {
  snapshot = $state<Snapshot | null>(null);
  unavailable = $state(false);
  #visible = false;
  #disposed = false;
  #reading = false;
  #generation = 0;
  #timer: ReturnType<typeof setTimeout> | undefined;
  #stale: ReturnType<typeof setTimeout> | undefined;

  constructor(private readonly service?: ResourceService) {}

  setVisible(visible: boolean) {
    if (this.#disposed || visible === this.#visible) return;
    this.#visible = visible;
    this.#generation++;
    clearTimeout(this.#timer);
    clearTimeout(this.#stale);
    this.snapshot = null;
    this.unavailable = false;
    if (visible) void this.#poll();
  }

  async #poll() {
    if (this.#disposed || !this.#visible || this.#reading) return;
    if (!this.service) {
      this.unavailable = true;
      return;
    }
    this.#reading = true;
    const generation = this.#generation;
    try {
      const snapshot = await this.service.Current();
      if (this.#disposed || !this.#visible || generation !== this.#generation)
        return;
      this.snapshot = snapshot;
      this.unavailable = false;
      clearTimeout(this.#stale);
      this.#stale = setTimeout(() => {
        this.snapshot = null;
        this.unavailable = true;
      }, 6_000);
    } catch {
      if (this.#disposed || generation !== this.#generation) return;
      this.snapshot = null;
      this.unavailable = true;
    } finally {
      this.#reading = false;
      if (this.#visible && !this.#disposed) {
        this.#timer = setTimeout(
          () => void this.#poll(),
          generation === this.#generation ? 2_000 : 0,
        );
      }
    }
  }

  dispose() {
    this.setVisible(false);
    this.#disposed = true;
  }
}

export function memoryPercent(snapshot: Snapshot | null): number | null {
  if (!snapshot?.memoryAvailable || snapshot.memoryTotalBytes <= 0) return null;
  return Math.max(
    0,
    Math.min(
      100,
      100 * (1 - snapshot.memoryAvailableBytes / snapshot.memoryTotalBytes),
    ),
  );
}

export function formatMemory(bytes: number): string {
  return `${(bytes / 1024 ** 3).toFixed(1)} GiB`;
}

export function gpuPercent(snapshot: Snapshot | null): number | null {
  const readings = (snapshot?.gpus ?? []).filter(
    (gpu) => gpu.utilizationAvailable,
  );
  return readings.length
    ? Math.max(...readings.map((gpu) => gpu.utilizationPercent))
    : null;
}

export function gpuMemoryPercent(gpu: GPU): number | null {
  if (!gpu.memoryAvailable || gpu.unifiedMemory || gpu.memoryTotalBytes <= 0)
    return null;
  return Math.max(
    0,
    Math.min(100, (100 * gpu.memoryUsedBytes) / gpu.memoryTotalBytes),
  );
}
