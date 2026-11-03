import { describe, expect, it } from "vitest";
import { ID as Backend } from "$bindings/compatibility";
import { ID as ModelProfile } from "$bindings/modelprofile";
import { settings } from "$lib/stores/session-fixtures";
import { liveInsertionUnavailableReason } from "./liveInsertion";

const eligible = () => ({
  ...structuredClone(settings),
  liveInsert: true,
  autoInsert: true,
  postProcessing: { ...settings.postProcessing, enabled: false },
  voiceTranscription: {
    ...settings.voiceTranscription,
    realtime: true,
    compatibilityProfile: Backend.NeMoSpeechV1,
    modelProfile: ModelProfile.Nemotron35,
  },
});

describe("live phrase insertion availability", () => {
  it("allows only the explicitly qualified realtime combination", () => {
    expect(liveInsertionUnavailableReason(eligible())).toBe("");
    const value = eligible();
    value.voiceTranscription.modelProfile = ModelProfile.Generic;
    expect(liveInsertionUnavailableReason(value)).toContain("Nemotron");
  });
  it.each([
    [
      "manual copy",
      (s: ReturnType<typeof eligible>) => {
        s.autoInsert = false;
      },
      "Direct input",
    ],
    [
      "cleanup",
      (s: ReturnType<typeof eligible>) => {
        s.postProcessing.enabled = true;
      },
      "cleanup",
    ],
    [
      "completed mode",
      (s: ReturnType<typeof eligible>) => {
        s.voiceTranscription.realtime = false;
      },
      "Realtime transcription",
    ],
    [
      "vLLM",
      (s: ReturnType<typeof eligible>) => {
        s.voiceTranscription.compatibilityProfile = Backend.VLLM;
      },
      "after Stop",
    ],
  ] as const)(
    "explains %s without erasing the saved choice",
    (_name, change, reason) => {
      const value = eligible();
      change(value);
      expect(liveInsertionUnavailableReason(value)).toContain(reason);
      expect(value.liveInsert).toBe(true);
    },
  );
  it("keeps existing users opted out", () => {
    expect(settings.liveInsert).toBe(false);
  });
});
