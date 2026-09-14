import { describe, expect, it } from "vitest";
import {
  runtimePresentation,
  modelSize,
  catalogGroups,
} from "./managedRuntime";
import type { Status } from "$bindings/managedruntime";
const status: Status = {
  acquisition: { phase: "", bytes: 0, totalBytes: 0 },
  operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
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
describe("local runtime presentation", () => {
  it("reports elapsed time for the current startup stage, never a percentage", () => {
    const starting = {
      ...status,
      state: "starting",
      progress: 0.5,
      startupProgress: { phase: "warming_up", startedAt: 1000 },
    };
    expect(runtimePresentation(starting, 6500)).toMatchObject({
      activity: "Warming up selected model",
      startup: "Warming up selected model · 5s in this stage",
      percent: null,
    });
    expect(
      runtimePresentation(
        {
          ...starting,
          startupProgress: { phase: "launching", startedAt: 6000 },
        },
        6500,
      ).startup,
    ).toBe("Launching runtime · 0s in this stage");
  });
  it("tracks transferred bytes without treating a full transfer as verified success", () => {
    const downloading = {
      ...status,
      state: "installing",
      phase: "download",
      operation: {
        id: 1,
        kind: "download",
        model: "nemotron-3.5",
        outcome: "running",
        error: "",
      },
      acquisition: {
        phase: "downloading",
        bytes: 250_000_000,
        totalBytes: 1_000_000_000,
      },
    };
    expect(runtimePresentation(downloading)).toMatchObject({
      percent: 25,
      completion: "",
    });
    expect(
      runtimePresentation({
        ...downloading,
        acquisition: { ...downloading.acquisition, bytes: 750_000_000 },
      }).percent,
    ).toBe(75);
    const verifying = {
      ...downloading,
      acquisition: {
        ...downloading.acquisition,
        phase: "verifying",
        bytes: 1_000_000_000,
      },
    };
    expect(runtimePresentation(verifying)).toMatchObject({
      percent: null,
      completion: "",
    });
    expect(
      runtimePresentation({
        ...verifying,
        phase: "",
        state: "installed",
        operation: { ...verifying.operation, outcome: "succeeded" },
      }).completion,
    ).not.toBe("");
    for (const outcome of ["cancelled", "failed"]) {
      expect(
        runtimePresentation({
          ...verifying,
          operation: { ...verifying.operation, outcome },
        }).completion,
      ).not.toBe(
        runtimePresentation({
          ...verifying,
          operation: { ...verifying.operation, outcome: "succeeded" },
        }).completion,
      );
    }
  });
  it("does not count a failed download as installed just because the release version is known", () => {
    expect(
      runtimePresentation({ ...status, state: "error", version: "v0.1.0" }),
    ).toMatchObject({ installed: false, step: 0, ready: false });
    expect(
      runtimePresentation({
        ...status,
        state: "error",
        version: "v0.1.0",
        backend: "cuda",
      }),
    ).toMatchObject({ installed: true, step: 1, ready: false });
  });
  it("distinguishes not installed, starting, running and unsupported without optimistic readiness", () => {
    expect(runtimePresentation(status)).toMatchObject({
      label: "Not installed",
      installed: false,
      ready: false,
      step: 0,
    });
    expect(
      runtimePresentation({ ...status, state: "starting", enabled: true }),
    ).toMatchObject({ label: "Starting", ready: false, step: 2 });
    expect(
      runtimePresentation({ ...status, state: "running", enabled: false }),
    ).toMatchObject({ label: "Running", ready: true, step: 3 });
    expect(
      runtimePresentation({
        ...status,
        state: "running",
        enabled: true,
        supported: false,
      }),
    ).toMatchObject({ label: "Windows only", ready: false });
  });
  it("shows indeterminate progress and unknown sizes honestly", () => {
    expect(
      runtimePresentation({ ...status, state: "installing", progress: -1 })
        .percent,
    ).toBeNull();
    expect(
      runtimePresentation({ ...status, state: "installing", progress: 0.42 })
        .percent,
    ).toBe(42);
    expect(modelSize(0)).toBe("Size not reported");
    expect(modelSize(1_500_000_000)).toBe("1.5 GB");
  });
  it("groups downloaded and available models, with recommendations first without mutating the catalog", () => {
    const models = [
      {
        id: "b",
        name: "B",
        contracts: [],
        description: "",
        sizeBytes: 0,
        installed: false,
        recommended: false,
        realtime: false,
        profile: "generic",
      },
      {
        id: "n",
        name: "Nemotron 3.5",
        contracts: [],
        description: "",
        sizeBytes: 0,
        installed: false,
        recommended: true,
        realtime: true,
        profile: "nemotron",
      },
      {
        id: "a",
        name: "A",
        contracts: [],
        description: "",
        sizeBytes: 0,
        installed: true,
        recommended: false,
        realtime: false,
        profile: "generic",
      },
    ];
    expect(
      catalogGroups(models).map((g) => [g.label, g.models.map((m) => m.id)]),
    ).toEqual([
      ["Downloaded", ["a"]],
      ["Available to download", ["n", "b"]],
    ]);
    expect(models.map((m) => m.id)).toEqual(["b", "n", "a"]);
  });
});
