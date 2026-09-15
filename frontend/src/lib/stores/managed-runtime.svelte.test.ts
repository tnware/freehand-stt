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
      acquisition: { phase: "", bytes: 0, totalBytes: 0 },
      operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
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
    GetBinaryOptions: async ({ provider }) => {
      return {
        provider,
        os: "windows",
        architecture: "amd64",
        supported: true,
        recommendedBackend: "cuda",
        reason: "Compatible NVIDIA GPU detected.",
        options: ["cpu", "cuda"].map((backend) => ({
          backend,
          supported: true,
          available: true,
          reason: "Available",
        })),
      };
    },
    OpenProcessOutput: async ({ instanceID }) => {
      void instanceID;
    },
    SetInstance: async () => {},
    DeleteInstance: async () => {},
    Install: async () => {},
    InstallBackend: async () => {},
    Remove: async () => {},
    Start: async () => {},
    Stop: async () => {},
    Restart: async () => {},
    Cancel: async () => {},
    RefreshCatalog: async () => {},
    DownloadModel: async () => {},
    RemoveModel: async () => {},
    ...overrides,
  };
}
describe("managed runtime inventory", () => {
  it("keeps pending commands separate from backend state until acknowledgement", async () => {
    const acknowledgement = Promise.withResolvers<void>();
    const Start = vi.fn(() => acknowledgement.promise);
    const runtime = new ManagedRuntimeState(boundary({ Start }));
    await runtime.load();
    const starting = runtime.run("one", "Start");
    expect(runtime.pendingFor("one")).toBe("Start");
    expect(runtime.statusFor("one")?.status.state).toBe("installed");
    expect(await runtime.run("one", "Start")).toBe(false);
    acknowledgement.resolve();
    await starting;
    expect(runtime.pendingFor("one")).toBe("");
    runtime.applyStatus({
      ...row("one"),
      status: { ...row("one").status, state: "starting", phase: "start" },
    });
    expect(runtime.isBusy("one")).toBe(true);
    runtime.applyStatus({
      ...row("one"),
      status: { ...row("one").status, state: "running" },
    });
    expect(runtime.isBusy("one")).toBe(false);
    expect(Start).toHaveBeenCalledOnce();
  });
  it("requests one backend-owned restart instead of racing Stop admission with Start", async () => {
    const Restart = vi.fn(async () => {});
    const Start = vi.fn(async () => {});
    const Stop = vi.fn(async () => {});
    const runtime = new ManagedRuntimeState(boundary({ Restart, Start, Stop }));
    await runtime.load();
    expect(await runtime.restart("one")).toBe(true);
    expect(Restart).toHaveBeenCalledExactlyOnceWith({ instanceID: "one" });
    expect(Start).not.toHaveBeenCalled();
    expect(Stop).not.toHaveBeenCalled();
  });
  it("allows output while startup is busy and settings are dirty without runtime mutations", async () => {
    const OpenProcessOutput = vi.fn(async () => {});
    const runtime = new ManagedRuntimeState(
      boundary({ OpenProcessOutput }),
      () => false,
    );
    await runtime.load();
    runtime.applyStatus({
      ...row("one"),
      status: { ...row("one").status, state: "starting" },
    });
    expect(await runtime.openOutput("one")).toBe(true);
    expect(OpenProcessOutput).toHaveBeenCalledExactlyOnceWith({
      instanceID: "one",
    });
    expect(await runtime.openOutput("missing")).toBe(false);
    expect(runtime.isBusy("one")).toBe(true);
  });
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
