import { describe, expect, it } from "vitest";
import { Role, ID as BackendID } from "$bindings/compatibility";
import {
  ProviderID,
  type InstanceStatus,
  type Model,
  type ProviderDescriptor,
} from "$bindings/managedruntime";
import { ID } from "$bindings/modelprofile";
import { settings } from "$lib/stores/session-fixtures";
import { runtimeSetupGuidance } from "./runtimeSetupIntent";

function model(id: string, role: Role): Model {
  return {
    id,
    name: id,
    description: "",
    sizeBytes: 0,
    installed: false,
    recommended: role === Role.Transcription,
    realtime: false,
    profile: ID.Generic,
    contracts: [
      {
        role,
        compatibilityProfile: BackendID.NeMoSpeechV1,
        modelProfile: ID.Generic,
        behavior: settings.modelProfiles.transcription![0],
      },
    ],
  };
}
const models = [
  model("transcription", Role.Transcription),
  model("speech", Role.Speech),
  model("cleanup", Role.PostProcessing),
];
const entry: ProviderDescriptor = {
  id: ProviderID.NeMoSpeechCPP,
  name: "NeMo",
  version: "fixture",
  supported: true,
  backends: ["cpu"],
  models,
};
function row(): InstanceStatus {
  return {
    instance: {
      id: "local",
      name: "Local",
      provider: entry.id,
      model: "transcription",
      autoStart: false,
    },
    activeModel: "",
    status: {
      supported: true,
      state: "installed",
      enabled: false,
      selectedModel: "transcription",
      realtime: false,
      backend: "cpu",
      version: "fixture",
      progress: -1,
      phase: "",
      error: "",
      models: structuredClone(models),
      acquisition: { phase: "", bytes: 0, totalBytes: 0 },
      operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
    },
  };
}

describe("explicit runtime setup intent", () => {
  it("guides speech selection ahead of the transcription download without changing either selection", () => {
    const value = row();
    const before = structuredClone(value);
    expect(runtimeSetupGuidance("speech", entry, value)).toMatchObject({
      role: Role.Speech,
      next: "Choose a speech model",
      chooseModel: true,
      section: "speech",
    });
    expect(value).toEqual(before);
  });
  it("distinguishes speech download, required transcription download, start and explicit connection choice", () => {
    const value = row();
    value.instance.speechModel = "speech";
    expect(runtimeSetupGuidance("speech", entry, value).next).toBe(
      "Download the selected speech model",
    );
    value.status.models![1].installed = true;
    expect(runtimeSetupGuidance("speech", entry, value).next).toBe(
      "Download the remaining selected model",
    );
    value.status.models![0].installed = true;
    expect(runtimeSetupGuidance("speech", entry, value).next).toBe(
      "Start the runtime",
    );
    value.status.state = "running";
    expect(runtimeSetupGuidance("speech", entry, value).next).toBe(
      "Choose its built-in connection in Speech settings",
    );
  });
  it("does not suggest selecting a model while running, busy, failed or unsupported", () => {
    const value = row();
    value.status.state = "running";
    expect(runtimeSetupGuidance("speech", entry, value)).toMatchObject({
      chooseModel: false,
      next: "Stop the runtime to choose a speech model",
    });
    expect(
      runtimeSetupGuidance("speech", entry, value, "Stop").chooseModel,
    ).toBe(false);
    value.status.state = "error";
    expect(runtimeSetupGuidance("speech", entry, value).next).toBe(
      "Review the runtime error",
    );
    expect(
      runtimeSetupGuidance("speech", { ...entry, supported: false }, value)
        .next,
    ).toBe("Choose another runtime for speech");
  });
  it("keeps cleanup intent distinct and only trusts role-qualified models", () => {
    const value = row();
    value.instance.model = "cleanup";
    expect(runtimeSetupGuidance("cleanup", entry, value)).toMatchObject({
      role: Role.PostProcessing,
      section: "processing",
      next: "Download the selected cleanup model",
      settingsLabel: "Cleanup settings",
    });
    expect(
      runtimeSetupGuidance(
        "speech",
        { ...entry, models: models.slice(0, 1) },
        value,
      ).next,
    ).toBe("Choose another runtime for speech");
  });
  it("does not treat initial recommendations as selected models", () => {
    const guide = runtimeSetupGuidance("speech", entry);
    expect(guide.selected).toBeUndefined();
    expect(guide.next).toBe("Install the runtime for speech");
    expect(guide.chooseModel).toBe(false);
  });
});
