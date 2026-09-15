import { describe, expect, it } from "vitest";
import { settings } from "$lib/stores/session-fixtures";
import { copySettings, quickSettingsDraft } from "./settingsDraft";

describe("audio-file language quick settings", () => {
  it.each(["fr", "auto"])(
    "changes file language to %s without changing Voice or the applied snapshot",
    (language) => {
      const applied = structuredClone(settings);
      applied.language = "ja";
      applied.voiceTranscription.language = "en";
      const before = structuredClone(applied);

      const next = quickSettingsDraft(applied, { language });

      expect(next.language).toBe(language);
      expect(next.voiceTranscription).toEqual(before.voiceTranscription);
      expect(next.rememberedModels).toEqual(before.rememberedModels);
      expect(applied).toEqual(before);
    },
  );
});

it("NeMo drafts do not mutate confirmed Voice or file options", () => {
  const applied = structuredClone(settings);
  const before = structuredClone(applied);
  const draft = copySettings(applied);
  draft.voiceTranscription.transcriptionOptions.nemo.normalize = true;
  draft.transcriptionOptions.nemo.disablePunctuation = true;
  expect(applied).toEqual(before);
});
