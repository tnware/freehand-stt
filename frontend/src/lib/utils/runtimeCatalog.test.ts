import { describe, expect, it } from "vitest";
import { ProviderID, type Model } from "$bindings/managedruntime";
import {
  runtimeModelFamilies,
  runtimeModelGroups,
  runtimeVariantLabel,
} from "./runtimeCatalog";
import { Role, ID as CompatibilityID } from "$bindings/compatibility";
import { ID } from "$bindings/modelprofile";
import { settings } from "$lib/stores/session-fixtures";

const model = (id: string): Model => ({
  id,
  name: id,
  description: "",
  contracts: [],
  sizeBytes: 0,
  installed: false,
  recommended: false,
  realtime: false,
  profile: "generic",
});

describe("runtime catalog presentation", () => {
  it("groups task capabilities once per model while preserving families inside each group", () => {
    const withRoles = (id: string, roles: Role[]): Model => ({
      ...model(id),
      contracts: roles.map((role) => ({
        role,
        compatibilityProfile: CompatibilityID.NeMoSpeechV1,
        modelProfile: ID.Generic,
        behavior: settings.modelProfiles.transcription![0],
      })),
    });
    const models = [
      withRoles("tiny", [Role.Transcription, Role.Realtime]),
      withRoles("magpie", [Role.Speech]),
      withRoles("cleanup", [Role.PostProcessing]),
      withRoles("unknown", []),
    ];
    const groups = runtimeModelGroups(ProviderID.WhisperCPP, models);
    expect(groups.map((group) => group.label)).toEqual([
      "Transcription",
      "Text to speech",
      "Cleanup",
      "Other models",
    ]);
    expect(groups[0].families[0].label).toBe("Tiny");
    expect(
      groups.flatMap((group) =>
        group.families.flatMap((family) => family.models),
      ),
    ).toHaveLength(models.length);
    const single = runtimeModelGroups(
      ProviderID.WhisperCPP,
      models.slice(0, 1),
    );
    expect(single[0].label).toBe("");
    expect(single[0].families[0].label).toBe("Tiny");
  });
  it("orders known Whisper families without reclassifying unknown IDs or mutating models", () => {
    const models = [
      model("large-v3-turbo-q8_0"),
      model("small.en"),
      model("base"),
      model("tiny.en-q5_1"),
      model("tiny-custom"),
    ];
    expect(
      runtimeModelFamilies(ProviderID.WhisperCPP, models).map(
        ({ id, models }) => [id, models.map((item) => item.id)],
      ),
    ).toEqual([
      ["tiny", ["tiny.en-q5_1"]],
      ["base", ["base"]],
      ["small", ["small.en"]],
      ["large", ["large-v3-turbo-q8_0"]],
      ["other", ["tiny-custom"]],
    ]);
    expect(models[0].id).toBe("large-v3-turbo-q8_0");
    expect(
      models.every((item) => item.profile === "generic" && !item.realtime),
    ).toBe(true);
  });
  it("leaves other providers flat even when their IDs resemble Whisper families", () => {
    const models = [model("base"), model("tiny")];
    expect(runtimeModelFamilies(ProviderID.NeMoSpeechCPP, models)).toEqual([
      { id: "all", label: "", models },
    ]);
    expect(runtimeVariantLabel(ProviderID.NeMoSpeechCPP, "tiny")).toBe("");
  });
  it("labels known language and quantization variants only as presentation metadata", () => {
    expect(runtimeVariantLabel(ProviderID.WhisperCPP, "base")).toBe(
      "Multilingual · Standard",
    );
    expect(runtimeVariantLabel(ProviderID.WhisperCPP, "base.en-q5_1")).toBe(
      "English only · Q5_1",
    );
    expect(
      runtimeVariantLabel(ProviderID.WhisperCPP, "large-v3-turbo-q8_0"),
    ).toBe("Multilingual · Q8_0");
    expect(runtimeVariantLabel(ProviderID.WhisperCPP, "base-custom")).toBe("");
  });
});
