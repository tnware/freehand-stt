import { describe, expect, it } from "vitest";
import {
  ProviderID,
  type Model,
  type ProviderDescriptor,
} from "$bindings/managedruntime";
import { Role, ID as CompatibilityID } from "$bindings/compatibility";
import { ID } from "$bindings/modelprofile";
import { settings } from "$lib/stores/session-fixtures";
import { localSpeechSetup } from "./localSpeechSetup";

function provider(
  id: ProviderID,
  supported = true,
  role = Role.Transcription,
  recommended = true,
): ProviderDescriptor {
  const model: Model = {
    id: "model",
    name: "Model",
    description: "Model description",
    sizeBytes: 0,
    installed: false,
    recommended,
    realtime: false,
    profile: ID.Generic,
    contracts: [
      {
        role,
        compatibilityProfile: CompatibilityID.OpenAI,
        modelProfile: ID.Generic,
        behavior: settings.modelProfiles.transcription![0],
      },
    ],
  };
  return {
    id,
    name: id,
    version: "test",
    supported,
    backends: ["cpu"],
    models: [model],
  };
}

describe("local speech setup guidance", () => {
  it("prefers supported NeMo regardless of catalog order", () => {
    const nemo = provider(ProviderID.NeMoSpeechCPP);
    const result = localSpeechSetup([provider(ProviderID.WhisperCPP), nemo]);
    expect(result.availability).toBe("available");
    expect(result.recommendation).toEqual({
      provider: nemo,
      model: nemo.models![0],
    });
  });
  it("falls back to supported whisper when NeMo is unsupported", () => {
    expect(
      localSpeechSetup([
        provider(ProviderID.NeMoSpeechCPP, false),
        provider(ProviderID.WhisperCPP),
      ]).recommendation?.provider.id,
    ).toBe(ProviderID.WhisperCPP);
  });
  it("never recommends unsupported or cleanup-only models", () => {
    for (const entries of [
      [provider(ProviderID.NeMoSpeechCPP, false)],
      [provider(ProviderID.NeMoSpeechCPP, true, Role.PostProcessing)],
      [provider(ProviderID.LlamaCPP, true, Role.PostProcessing)],
    ]) {
      expect(localSpeechSetup(entries)).toEqual({
        availability: "unavailable",
        recommendation: null,
      });
    }
  });
  it("does not claim unsupported while provider metadata is absent", () => {
    expect(localSpeechSetup([])).toEqual({
      availability: "unknown",
      recommendation: null,
    });
  });
  it("keeps supported non-recommended models discoverable without inventing a recommendation", () => {
    expect(
      localSpeechSetup([
        provider(ProviderID.NeMoSpeechCPP, true, Role.Transcription, false),
      ]),
    ).toEqual({ availability: "available", recommendation: null });
  });
  it("rejects missing transcription contracts", () => {
    const entry = provider(ProviderID.NeMoSpeechCPP);
    entry.models![0].contracts = [];
    expect(localSpeechSetup([entry])).toEqual({
      availability: "unavailable",
      recommendation: null,
    });
  });
});
