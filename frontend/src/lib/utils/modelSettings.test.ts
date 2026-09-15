import { describe, expect, it } from "vitest";
import { Purpose } from "$bindings/savedconnection";
import { settings } from "$lib/stores/session-fixtures";
import { applyModelOptions, modelOptions } from "./modelSettings";

const purposes = [
  Purpose.Voice,
  Purpose.Transcription,
  Purpose.Cleanup,
  Purpose.Speech,
];

describe("model preference ownership", () => {
  it.each(purposes)(
    "extracts only model-owned wire fields for %s",
    (purpose) => {
      const current = structuredClone(settings);
      current.language = "ja";
      current.vocabulary = {
        terms: "Shared terms",
        boost: 4,
        voice: true,
        files: true,
      };
      const options = modelOptions(current, purpose);
      expect(Object.keys(options).sort()).toEqual([
        "cleanup",
        "profile",
        "speech",
        "transcription",
        "voice",
      ]);
      expect(Object.keys(options.transcription).sort()).toEqual([
        "nemo",
        "prompt",
        "temperature",
        "temperatureOverride",
      ]);
    },
  );

  it.each(purposes)(
    "preserves all task-owned state while applying %s",
    (purpose) => {
      const current = structuredClone(settings);
      current.language = "ja";
      current.voiceTranscription.language = "en";
      current.vocabulary = {
        terms: "Shared terms",
        voice: true,
        files: true,
        boost: 5,
      };
      current.textToSpeech.speed = 1.5;
      const before = structuredClone(current);
      const options = modelOptions(settings, purpose);
      applyModelOptions(current, purpose, "replacement", options);
      expect(current.language).toBe(before.language);
      expect(current.voiceTranscription.language).toBe(
        before.voiceTranscription.language,
      );
      expect(current.vocabulary).toEqual(before.vocabulary);
      for (const field of [
        "systemPrompt",
        "styling",
        "structure",
        "context",
      ] as const) {
        expect(current.postProcessing[field]).toBe(
          before.postProcessing[field],
        );
      }
      expect(current.textToSpeech.speed).toBe(before.textToSpeech.speed);
      expect(modelOptions(current, purpose)).toEqual(options);
    },
  );
});
