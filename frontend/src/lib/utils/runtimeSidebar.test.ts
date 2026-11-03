import { describe, expect, it } from "vitest";
import {
  ProviderID,
  type InstanceStatus,
  type Model,
  type ProviderDescriptor,
} from "$bindings/managedruntime";
import { runtimeSidebarContext } from "./runtimeSidebar";

const recommended: Model = {
  id: "nemotron-3.5",
  name: "Nemotron 3.5 Streaming",
  description: "Streaming transcription",
  sizeBytes: 0,
  installed: false,
  recommended: true,
  realtime: true,
  profile: "nemotron-3.5-streaming",
  contracts: [],
};
const alternative: Model = {
  ...recommended,
  id: "parakeet-tdt",
  name: "Parakeet TDT v3",
  recommended: false,
};
const speech: Model = {
  ...alternative,
  id: "magpie-tts",
  name: "Magpie TTS Multilingual",
};
const entry: ProviderDescriptor = {
  id: ProviderID.NeMoSpeechCPP,
  name: "NeMo-Speech.cpp",
  version: "0.1.0",
  supported: true,
  backends: ["cpu", "metal"],
  models: [recommended, alternative, speech],
};
function installation(): InstanceStatus {
  return {
    instance: {
      id: "nemo-default",
      name: "Local speech",
      provider: entry.id,
      model: alternative.id,
      autoStart: false,
    },
    activeModel: "",
    status: {
      acquisition: { phase: "", bytes: 0, totalBytes: 0 },
      operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
      supported: true,
      state: "installed",
      enabled: false,
      selectedModel: alternative.id,
      realtime: false,
      backend: "metal",
      version: "0.1.0",
      progress: -1,
      phase: "",
      error: "",
      models: [recommended, alternative, speech].map((model) => ({ ...model })),
    },
  };
}

describe("runtime sidebar setup context", () => {
  it("labels a recommendation without claiming selection, installation or hardware availability", () => {
    expect(runtimeSidebarContext(entry)).toMatchObject({
      model: {
        label: "Recommended model",
        name: recommended.name,
        id: recommended.id,
      },
      binary: "",
      heading: "Next step",
      next: "Install runtime",
    });
    expect(runtimeSidebarContext({ ...entry, models: [] }).model).toEqual({
      label: "Model",
      name: "No model selected",
      id: "",
    });
    expect(
      runtimeSidebarContext({ ...entry, models: [alternative] }).model.label,
    ).not.toBe("Recommended model");
  });

  it("keeps the saved non-recommended model and both selection IDs unchanged", () => {
    const row = installation();
    row.instance.speechModel = speech.id;
    const before = structuredClone(row);
    expect(runtimeSidebarContext(entry, row)).toMatchObject({
      model: {
        label: "Selected model",
        name: alternative.name,
        id: alternative.id,
      },
      speechModel: {
        label: "Selected speech model",
        name: speech.name,
        id: speech.id,
      },
      next: "Download selected model",
      binary: "Apple GPU (Metal) · 0.1.0",
    });
    expect(row).toEqual(before);
  });

  it("preserves missing catalog IDs and asks for review instead of substituting the recommendation", () => {
    const row = installation();
    row.instance.model = "saved-model-no-longer-in-catalog";
    expect(runtimeSidebarContext(entry, row)).toMatchObject({
      model: {
        label: "Selected model",
        name: "saved-model-no-longer-in-catalog",
        id: "saved-model-no-longer-in-catalog",
      },
      next: "Choose a model",
    });
  });

  it("does not treat a status-only model as qualified for setup", () => {
    const row = installation();
    row.status.models![1].installed = true;
    expect(
      runtimeSidebarContext({ ...entry, models: [recommended] }, row),
    ).toMatchObject({
      model: { label: "Selected model", id: alternative.id },
      next: "Choose a model",
    });
  });

  it("requires installation, both selected downloads, and explicit Start in order", () => {
    const row = installation();
    row.status.state = "not_installed";
    row.status.backend = "";
    expect(runtimeSidebarContext(entry, row).next).toBe("Install runtime");
    row.status.state = "installed";
    expect(runtimeSidebarContext(entry, row).next).toBe(
      "Download selected model",
    );
    row.status.models![1].installed = true;
    row.instance.speechModel = speech.id;
    expect(runtimeSidebarContext(entry, row).next).toBe(
      "Download speech model",
    );
    row.status.models![2].installed = true;
    expect(runtimeSidebarContext(entry, row).next).toBe("Start runtime");
    row.status.state = "stopped";
    expect(runtimeSidebarContext(entry, row).next).toBe("Start runtime");
    row.status.state = "running";
    expect(runtimeSidebarContext(entry, row)).toMatchObject({
      heading: "Runtime ready",
      next: "Running locally",
    });
  });

  it("shows pending requests and backend stages before any ready or next-step guidance", () => {
    const row = installation();
    row.status.state = "running";
    expect(runtimeSidebarContext(entry, row, "Stop")).toMatchObject({
      heading: "In progress",
      next: "Stopping runtime",
    });
    expect(runtimeSidebarContext(entry, row, "Restart").next).toBe(
      "Restarting runtime",
    );
    row.status.state = "starting";
    row.status.startupProgress = { phase: "warming_up", startedAt: 1000 };
    expect(runtimeSidebarContext(entry, row, "Start").next).toBe(
      "Warming up selected model",
    );
    expect(runtimeSidebarContext(entry, row, "Cancelling").next).toBe(
      "Cancelling operation",
    );
    row.status.state = "installing";
    expect(runtimeSidebarContext(entry, row).next).toBe("Installing runtime");
    expect(runtimeSidebarContext(entry, undefined, "Install").next).toBe(
      "Installing runtime",
    );
  });

  it("names the model actually downloading even when it is not selected", () => {
    const row = installation();
    row.status.phase = "download";
    row.status.operation = {
      id: 1,
      kind: "download",
      model: recommended.id,
      outcome: "running",
      error: "",
    };
    row.status.acquisition = {
      phase: "downloading",
      bytes: 1024,
      totalBytes: 2048,
    };
    const downloading = runtimeSidebarContext(entry, row);
    expect(downloading.model.name).toBe(alternative.name);
    expect(downloading.next).toBe("Downloading model");
    expect(downloading.detail).toContain(recommended.name);
    expect(downloading.detail).toContain(" / ");
    row.status.acquisition.phase = "verifying";
    expect(runtimeSidebarContext(entry, row).next).toBe("Verifying model");
    expect(runtimeSidebarContext(entry, row).detail).not.toContain(" / ");
  });

  it("does not replace errors with install, download, or running instructions", () => {
    const row = installation();
    for (const state of ["not_installed", "installed", "running", "error"]) {
      row.status.state = state;
      expect(
        runtimeSidebarContext(entry, row, "", "Could not complete operation"),
      ).toMatchObject({
        heading: "Needs attention",
        next: "Review runtime error",
      });
    }
    row.status.error = "Could not start";
    expect(runtimeSidebarContext(entry, row, "Start").next).toBe(
      "Starting runtime",
    );
  });

  it("keeps unsupported runtime guidance separate from errors and setup actions", () => {
    const unsupported = {
      ...entry,
      supported: false,
      unavailableReason: "Windows only",
    };
    expect(runtimeSidebarContext(unsupported)).toMatchObject({
      heading: "Unavailable",
      next: "Choose another runtime",
      detail: "Windows only",
    });
  });
});
