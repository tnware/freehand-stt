import { describe, expect, it } from "vitest";
import { ID as Backend } from "$bindings/compatibility";
import { ID as ModelProfile } from "$bindings/modelprofile";
import { settings } from "$lib/stores/session-fixtures";
import {
  liveInsertionUnavailableReason,
  realtimeDeliverySummary,
} from "./liveInsertion";

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

it("describes effective realtime delivery rather than promising stop-time insertion", () => {
  const value = eligible();
  expect(realtimeDeliverySummary(value)).toContain("while recording");
  value.liveInsert = false;
  expect(realtimeDeliverySummary(value)).toBe(
    "Stop to finish and deliver the transcript.",
  );
  value.liveInsert = true;
  value.postProcessing.enabled = true;
  expect(realtimeDeliverySummary(value)).toContain("clean up");
  value.autoInsert = false;
  expect(realtimeDeliverySummary(value)).toContain("you to copy");
  value.autoInsert = true;
  value.postProcessing.enabled = false;
  value.voiceTranscription.compatibilityProfile = Backend.VLLM;
  expect(realtimeDeliverySummary(value)).toBe(
    "Stop to finish and deliver the transcript.",
  );
});

it("requires explicit Copy on Linux even with saved live insertion enabled", () => {
  const value = eligible();
  value.platform = "linux";
  expect(liveInsertionUnavailableReason(value)).toContain("explicit Copy");
  expect(realtimeDeliverySummary(value)).toContain("you to copy");
  expect(value.autoInsert).toBe(true);
  expect(value.liveInsert).toBe(true);
});
