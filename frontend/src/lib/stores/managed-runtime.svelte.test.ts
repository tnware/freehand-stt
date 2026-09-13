import { describe, expect, it, vi } from "vitest";
import type { Status } from "$bindings/managedruntime";
import {
  ManagedRuntimeState,
  type ManagedRuntimeService,
} from "./managed-runtime.svelte";

export const runtimeStatus: Status = {
  supported: true,
  state: "not_installed",
  enabled: false,
  selectedModel: "nemotron-3.5",
  realtime: true,
  backend: "",
  version: "",
  progress: -1,
  phase: "",
  error: "",
  models: [],
};
function service(
  overrides: Partial<ManagedRuntimeService> = {},
): ManagedRuntimeService {
  return {
    GetStatus: async () => runtimeStatus,
    Install: async () => {},
    Remove: async () => {},
    Start: async () => {},
    Stop: async () => {},
    Cancel: async () => {},
    RefreshCatalog: async () => {},
    DownloadModel: async () => {},
    RemoveModel: async () => {},
    SetPreferences: async () => {},
    ...overrides,
  };
}
describe("managed runtime state", () => {
  it("blocks a second action during metadata work even when state is installed", async () => {
    const Start = vi.fn(async () => {});
    const runtime = new ManagedRuntimeState(service({ Start }));
    runtime.applyStatus({
      ...runtimeStatus,
      state: "installed",
      phase: "catalog",
    });
    expect(runtime.busy).toBe(true);
    expect(await runtime.run("Start")).toBe(false);
    expect(Start).not.toHaveBeenCalled();
  });
  it("preserves events over late snapshots and ignores disposed results", async () => {
    let resolve!: (status: Status) => void;
    const runtime = new ManagedRuntimeState(
      service({
        GetStatus: () =>
          new Promise((r) => {
            resolve = r;
          }),
      }),
    );
    const loading = runtime.load();
    runtime.applyStatus({ ...runtimeStatus, state: "running", enabled: true });
    resolve(runtimeStatus);
    await loading;
    expect(runtime.status?.state).toBe("running");
    runtime.dispose();
    runtime.applyStatus(runtimeStatus);
    expect(runtime.status?.state).toBe("running");
  });
  it("enables the recommended realtime model only explicitly and reads back preferences", async () => {
    let current = runtimeStatus;
    const changed = vi.fn();
    const SetPreferences = vi.fn(async (p) => {
      current = {
        ...current,
        enabled: p.enabled,
        selectedModel: p.model,
        realtime: p.realtime,
      };
    });
    const runtime = new ManagedRuntimeState(
      service({ GetStatus: async () => current, SetPreferences }),
      () => true,
      changed,
    );
    await runtime.load();
    await runtime.enable();
    expect(SetPreferences).toHaveBeenCalledWith({
      enabled: true,
      model: "nemotron-3.5",
      realtime: true,
    });
    expect(runtime.status?.enabled).toBe(true);
    expect(changed).toHaveBeenCalledOnce();
  });
  it("blocks mutations for unsaved drafts and unsupported hosts", async () => {
    const Install = vi.fn();
    const runtime = new ManagedRuntimeState(service({ Install }), () => false);
    await runtime.load();
    expect(await runtime.run("Install")).toBe(false);
    expect(Install).not.toHaveBeenCalled();
    expect(runtime.error).toContain("Save or discard");
    const unsupported = new ManagedRuntimeState(
      service({
        Install,
        GetStatus: async () => ({ ...runtimeStatus, supported: false }),
      }),
    );
    await unsupported.load();
    expect(await unsupported.run("Install")).toBe(false);
    expect(Install).not.toHaveBeenCalled();
  });
  it("supports explicit cancel while busy, and retry after a safe failure", async () => {
    const Install = vi
      .fn()
      .mockRejectedValueOnce(new Error("unsafe internal path"))
      .mockResolvedValue(undefined);
    const Cancel = vi.fn(async () => {});
    const runtime = new ManagedRuntimeState(service({ Install, Cancel }));
    await runtime.load();
    expect(await runtime.run("Install")).toBe(false);
    expect(runtime.error).not.toContain("unsafe internal path");
    expect(await runtime.retry()).toBe(true);
    expect(Install).toHaveBeenCalledTimes(2);
    runtime.applyStatus({ ...runtimeStatus, state: "installing" });
    expect(await runtime.run("Start")).toBe(false);
    expect(await runtime.cancel()).toBe(true);
    expect(Cancel).toHaveBeenCalledOnce();
  });
  it("selects only installed models and turns realtime off for completed-only ASR", async () => {
    const SetPreferences = vi.fn(async () => {});
    const runtime = new ManagedRuntimeState(service({ SetPreferences }));
    runtime.applyStatus({
      ...runtimeStatus,
      enabled: true,
      models: [
        {
          id: "parakeet",
          name: "Parakeet",
          description: "",
          sizeBytes: 0,
          installed: false,
          recommended: false,
          realtime: false,
          profile: "parakeet",
        },
      ],
    });
    expect(await runtime.useModel("parakeet")).toBe(false);
    runtime.applyStatus({
      ...runtime.status!,
      models: (runtime.status!.models ?? []).map((m) => ({
        ...m,
        installed: true,
      })),
    });
    await runtime.useModel("parakeet");
    expect(SetPreferences).toHaveBeenCalledWith({
      enabled: true,
      model: "parakeet",
      realtime: false,
    });
  });
  it("loads metadata without installing, downloading or enabling", async () => {
    const Install = vi.fn(),
      DownloadModel = vi.fn(),
      SetPreferences = vi.fn();
    const runtime = new ManagedRuntimeState(
      service({ Install, DownloadModel, SetPreferences }),
    );
    await runtime.load();
    expect(runtime.status).toEqual(runtimeStatus);
    expect(Install).not.toHaveBeenCalled();
    expect(DownloadModel).not.toHaveBeenCalled();
    expect(SetPreferences).not.toHaveBeenCalled();
  });
});
