import { describe, expect, it } from "vitest";
import { ProviderID, type Model } from "$bindings/managedruntime";
import { runtimeModelFamilies, runtimeVariantLabel } from "./runtimeCatalog";

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
