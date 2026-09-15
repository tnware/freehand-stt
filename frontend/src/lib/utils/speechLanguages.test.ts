import { expect, it } from "vitest";
import { ID, type Profile } from "$bindings/modelprofile";
import { VoiceScope, type VoicesResult } from "$bindings/inference";
import { settings } from "$lib/stores/session-fixtures";
import { speechLanguageOptions, speechLanguageValue } from "./speechLanguages";

const magpie: Profile = {
  ...settings.modelProfiles.speech![0],
  id: ID.MagpieTTS,
  languages: [
    { code: "en-US", label: "English" },
    { code: "fr-FR", label: "French" },
    { code: "zh-CN", label: "Mandarin" },
    { code: "ja-JP", label: "Japanese" },
  ],
};
const voices: VoicesResult = {
  voices: [],
  scope: VoiceScope.VoiceScopeModel,
  errorKind: "",
  httpStatus: 200,
  latencyMilliseconds: 1,
  truncated: false,
  languages: ["fr", "ja"],
};

it("offers server default and advertised qualified Magpie locales without automatic detection", () => {
  expect(
    speechLanguageOptions(magpie, voices).map((option) => option.code),
  ).toEqual(["", "fr-FR", "ja-JP"]);
  expect(speechLanguageValue(magpie, "")).toBe("");
  expect(speechLanguageValue(magpie, "fr")).toBe("fr-FR");
  expect(speechLanguageValue(magpie, "unsupported")).toBe("unsupported");
});

it("requires discovery for optional language frontends and ignores failed metadata", () => {
  expect(
    speechLanguageOptions(magpie, null).map((option) => option.code),
  ).toEqual(["", "en-US", "fr-FR"]);
  expect(
    speechLanguageOptions(magpie, { ...voices, errorKind: "http" }),
  ).toEqual(speechLanguageOptions(magpie, null));
  const other = { ...magpie, id: ID.Qwen3TTS };
  expect(speechLanguageOptions(other, voices)).toEqual(other.languages);
  expect(speechLanguageValue(other, "")).toBe("auto");
});
