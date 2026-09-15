import { describe, expect, it } from "vitest";
import {
  runtimePresentation,
  modelSize,
  catalogGroups,
  runtimeModelSelection,
  runtimeModelsDownloaded,
} from "./managedRuntime";
import {
  ProviderID,
  type Instance,
  type Model,
  type Status,
} from "$bindings/managedruntime";
import { Role, ID as CompatibilityID } from "$bindings/compatibility";
import { ID } from "$bindings/modelprofile";
import { settings } from "$lib/stores/session-fixtures";

it("selects and checks both task models without replacing the other task's selection", () => {
  const instance: Instance = {
    id: "nemo",
    name: "NeMo",
    provider: ProviderID.NeMoSpeechCPP,
    model: "asr",
    speechModel: "",
    autoStart: false,
  };
  const speech: Model = {
    id: "magpie-tts",
    name: "Magpie",
    description: "Speech synthesis",
    sizeBytes: 1,
    installed: true,
    recommended: false,
    realtime: false,
    profile: ID.MagpieTTS,
    contracts: [
      {
        role: Role.Speech,
        compatibilityProfile: CompatibilityID.NeMoSpeechV1,
        modelProfile: ID.MagpieTTS,
        behavior: settings.modelProfiles.speech![0],
      },
    ],
  };
  const asr: Model = { ...speech, id: "asr", contracts: [] };
  const selected = runtimeModelSelection(instance, speech);
  expect(selected).toMatchObject({ model: "asr", speechModel: "magpie-tts" });
  expect(
    runtimeModelSelection(selected, { ...asr, id: "other-asr" }),
  ).toMatchObject({ model: "other-asr", speechModel: "magpie-tts" });
  expect(runtimeModelsDownloaded(selected, [asr, speech])).toBe(true);
  expect(
    runtimeModelsDownloaded(selected, [asr, { ...speech, installed: false }]),
  ).toBe(false);
  expect(runtimeModelsDownloaded(instance, [asr])).toBe(true);
  expect(runtimeModelsDownloaded({ ...instance, model: "" }, [asr])).toBe(
    false,
  );
});
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
  it("does not report readiness from initial inspection unless autostart actually reaches running", () => {
    const inspecting = {
      ...status,
      state: "installed",
      backend: "cpu",
      phase: "startup",
      operation: {
        id: 1,
        kind: "startup",
        model: "",
        outcome: "running",
        error: "",
      },
    };
    expect(runtimePresentation(inspecting)).toMatchObject({
      label: "Checking runtime",
      activity: "Checking runtime",
      ready: false,
      completion: "",
    });
    const inspected = {
      ...inspecting,
      phase: "",
      operation: { ...inspecting.operation, outcome: "succeeded" },
    };
    expect(runtimePresentation(inspected)).toMatchObject({
      label: "Installed",
      ready: false,
      completion: "",
    });
    expect(
      runtimePresentation({ ...inspecting, state: "starting" }),
    ).toMatchObject({
      label: "Starting runtime",
      ready: false,
      completion: "",
    });
    expect(
      runtimePresentation({ ...inspected, state: "running" }),
    ).toMatchObject({
      label: "Running",
      ready: true,
      completion: "Runtime ready.",
    });
  });
  it("describes binding admission without reusing an old outcome, progress, or optimistic readiness", () => {
    const completed = {
      ...status,
      state: "installed",
      backend: "cpu",
      progress: 1,
      operation: {
        id: 1,
        kind: "download",
        model: "old-model",
        outcome: "succeeded",
        error: "",
      },
    };
    expect(runtimePresentation(completed, 1000, "Start")).toMatchObject({
      label: "Starting runtime",
      activity: "Starting runtime",
      startup: "",
      ready: false,
      completion: "",
      operationModel: "",
      transferred: "",
      percent: null,
    });
    expect(
      runtimePresentation({ ...completed, state: "running" }, 1000, "Stop"),
    ).toMatchObject({
      label: "Stopping runtime",
      activity: "Stopping runtime",
      completion: "",
      percent: null,
    });
    expect(
      runtimePresentation({ ...completed, state: "running" }, 1000, "Restart"),
    ).toMatchObject({
      label: "Restarting runtime",
      completion: "",
    });
  });
  it("replaces pending startup text with backend stages while keeping cancellation explicit", () => {
    const starting = {
      ...status,
      state: "starting",
      phase: "start",
      operation: {
        id: 2,
        kind: "start",
        model: "",
        outcome: "running",
        error: "",
      },
      startupProgress: { phase: "warming_up", startedAt: 1000 },
    };
    expect(runtimePresentation(starting, 6500, "Start")).toMatchObject({
      label: "Warming up selected model",
      startup: "Warming up selected model · 5s in this stage",
      percent: null,
      completion: "",
    });
    expect(runtimePresentation(starting, 6500, "Cancelling")).toMatchObject({
      label: "Cancelling operation",
      activity: "Cancelling operation",
      startup: "",
      percent: null,
      completion: "",
    });
  });
  it("shows native restart stopping and startup stages before its terminal outcome", () => {
    const restarting = {
      ...status,
      state: "stopping",
      phase: "restart",
      operation: {
        id: 3,
        kind: "restart",
        model: "",
        outcome: "running",
        error: "",
      },
    };
    expect(runtimePresentation(restarting, 1000, "Restart").label).toBe(
      "Stopping runtime",
    );
    expect(
      runtimePresentation(
        {
          ...restarting,
          state: "starting",
          startupProgress: { phase: "launching", startedAt: 1000 },
        },
        1500,
      ).label,
    ).toBe("Launching runtime");
    expect(
      runtimePresentation({
        ...restarting,
        state: "running",
        phase: "",
        operation: { ...restarting.operation, outcome: "succeeded" },
      }),
    ).toMatchObject({
      label: "Running",
      ready: true,
      completion: "Runtime restarted.",
    });
  });
  it("exposes backend failure text after work and suppresses the stale error during retry admission", () => {
    const failed = {
      ...status,
      state: "error",
      operation: {
        id: 4,
        kind: "start",
        model: "",
        outcome: "failed",
        error: "Selected model could not start.",
      },
    };
    expect(runtimePresentation(failed)).toMatchObject({
      label: "Needs attention",
      error: "Selected model could not start.",
      completion: "Selected model could not start.",
    });
    expect(runtimePresentation(failed, 1000, "Start")).toMatchObject({
      label: "Starting runtime",
      error: "",
      completion: "",
      ready: false,
    });
  });
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
    ).toMatchObject({ label: "Unavailable on this platform", ready: false });
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
