import { CPUState, type Snapshot } from "$bindings/resources";
import type { ResourceService } from "$lib/stores/resources.svelte";

declare global {
  interface Window {
    testResources: {
      reads: number;
      fail: boolean;
      sample: Snapshot;
    };
  }
}

export function createResourceFixture(): ResourceService {
  const control = (window.testResources = {
    reads: 0,
    fail: false,
    sample: {
      sampledAt: Date.now(),
      cpuState: CPUState.CPUReady,
      cpuPercent: 42,
      memoryAvailable: true,
      memoryTotalBytes: 16 * 1024 ** 3,
      memoryAvailableBytes: 4 * 1024 ** 3,
      gpus: [
        {
          name: "Fixture GPU",
          utilizationAvailable: true,
          utilizationPercent: 88,
          memoryAvailable: true,
          memoryUsedBytes: 10 * 1024 ** 3,
          memoryTotalBytes: 12 * 1024 ** 3,
          sharedMemoryAvailable: true,
          sharedMemoryUsedBytes: 1024 ** 3,
          unifiedMemory: false,
        },
      ],
    },
  });
  return {
    Current: async () => {
      control.reads++;
      if (control.fail) throw new Error("Resource sample unavailable");
      return structuredClone({ ...control.sample, sampledAt: Date.now() });
    },
  };
}
