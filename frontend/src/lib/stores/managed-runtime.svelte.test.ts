import { describe, expect, it, vi } from "vitest";
import { ProviderID, type InstanceStatus } from "$bindings/managedruntime";
import {
  ManagedRuntimeState,
  type ManagedRuntimeService,
} from "./managed-runtime.svelte";

// The only fake is the generated Manager boundary; the store is production code.
function row(id: string): InstanceStatus {
  return {
    instance: {
      id,
      name: id,
      provider: ProviderID.NeMoSpeechCPP,
      model: "nemotron-3.5",
      autoStart: false,
    },
    activeModel: "",
    status: {
      supported: true,
      state: "installed",
      enabled: false,
      realtime: false,
      selectedModel: "nemotron-3.5",
      backend: "cpu",
      version: "0.1.0",
      progress: -1,
      phase: "",
      error: "",
      models: [],
    },
  };
}
function boundary(
  overrides: Partial<ManagedRuntimeService> = {},
): ManagedRuntimeService {
  return {
    GetInstances: async () => [row("one"), row("two")],
    GetProviders: async () => [],
    SetInstance: async () => {},
    DeleteInstance: async () => {},
    Install: async () => {},
    Remove: async () => {},
    Start: async () => {},
    Stop: async () => {},
    Cancel: async () => {},
    RefreshCatalog: async () => {},
    DownloadModel: async () => {},
    RemoveModel: async () => {},
    ...overrides,
  };
}
describe("managed runtime inventory", () => {
  it("fences late inventory reads per instance and ignores deleted or disposed events", async () => {
    let resolve!: (rows: InstanceStatus[]) => void;
    let rows = [row("one"), row("two")];
    const service = boundary({ GetInstances: async () => rows });
    const changed = vi.fn();
    const runtime = new ManagedRuntimeState(service, () => true, changed);
    await runtime.load();
    service.GetInstances = () =>
      new Promise((r) => {
        resolve = r;
      });
    const load = runtime.load();
    runtime.applyStatus({
      ...row("one"),
      status: { ...row("one").status, state: "running" },
    });
    resolve([row("one")]);
    await load;
    expect(runtime.statusFor("one")?.status.state).toBe("running");
    expect(runtime.statusFor("two")).toBeUndefined();
    runtime.applyStatus(row("two"));
    expect(runtime.statusFor("two")).toBeUndefined();
    rows = [];
    service.GetInstances = async () => rows;
    expect(await runtime.deleteInstance("one")).toBe(true);
    expect(changed).toHaveBeenCalledOnce();
    runtime.applyStatus(row("one"));
    expect(runtime.instances).toEqual([]);
    runtime.dispose();
    runtime.applyStatus(row("three"));
    expect(runtime.instances).toEqual([]);
  });
  it("targets independent instances while blocking duplicate work on one", async () => {
    const Start = vi.fn(async () => {});
    const runtime = new ManagedRuntimeState(boundary({ Start }));
    await runtime.load();
    runtime.applyStatus({
      ...row("one"),
      status: { ...row("one").status, phase: "download" },
    });
    expect(await runtime.run("one", "Start")).toBe(false);
    expect(await runtime.run("two", "Start")).toBe(true);
    expect(Start).toHaveBeenCalledExactlyOnceWith({ instanceID: "two" });
    expect(runtime.statusFor("one")?.status.phase).toBe("download");
  });
});
